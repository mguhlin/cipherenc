#!/usr/bin/env python3
"""An X11 file drop between two processes, using synthetic files only.
Run under xvfb-run after building build/CipherEnc-native; install requirements-native-tests.txt.
"""
import ctypes as C
from pathlib import Path
import os, subprocess, tempfile, time, sys
x=C.CDLL('libX11.so.6');xt=C.CDLL('libXtst.so.6')
x.XOpenDisplay.argtypes=[C.c_char_p];x.XOpenDisplay.restype=C.c_void_p
d=x.XOpenDisplay(None)
if not d:raise SystemExit('Run with xvfb-run -a python3 scripts/check-native-drop.py')
x.XDefaultRootWindow.argtypes=[C.c_void_p];x.XDefaultRootWindow.restype=C.c_ulong
root=x.XDefaultRootWindow(d)
x.XQueryTree.argtypes=[C.c_void_p,C.c_ulong,C.POINTER(C.c_ulong),C.POINTER(C.c_ulong),C.POINTER(C.POINTER(C.c_ulong)),C.POINTER(C.c_uint)]
x.XFetchName.argtypes=[C.c_void_p,C.c_ulong,C.POINTER(C.c_char_p)];x.XFree.argtypes=[C.c_void_p]
x.XTranslateCoordinates.argtypes=[C.c_void_p,C.c_ulong,C.c_ulong,C.c_int,C.c_int,C.POINTER(C.c_int),C.POINTER(C.c_int),C.POINTER(C.c_ulong)]
x.XStringToKeysym.argtypes=[C.c_char_p];x.XStringToKeysym.restype=C.c_ulong
x.XKeysymToKeycode.argtypes=[C.c_void_p,C.c_ulong];x.XKeysymToKeycode.restype=C.c_uint
x.XFlush.argtypes=[C.c_void_p]
xt.XTestFakeMotionEvent.argtypes=[C.c_void_p,C.c_int,C.c_int,C.c_int,C.c_ulong]
xt.XTestFakeButtonEvent.argtypes=[C.c_void_p,C.c_uint,C.c_int,C.c_ulong]
xt.XTestFakeKeyEvent.argtypes=[C.c_void_p,C.c_uint,C.c_int,C.c_ulong]
def windows(w=root):
 rr=C.c_ulong();parent=C.c_ulong();children=C.POINTER(C.c_ulong)();count=C.c_uint()
 x.XQueryTree(d,w,C.byref(rr),C.byref(parent),C.byref(children),C.byref(count))
 result=[]
 for i in range(count.value):
  child=children[i];name=C.c_char_p();x.XFetchName(d,child,C.byref(name))
  if name.value:result.append((child,name.value.decode(errors='replace')))
  if name:x.XFree(name)
  result+=windows(child)
 if children:x.XFree(children)
 return result
def locate(part):
 for _ in range(80):
  for w,name in windows():
   if part in name:return w
  time.sleep(.1)
 raise AssertionError('Window missing: '+part)
def origin(w):
 xx=C.c_int();yy=C.c_int();child=C.c_ulong();x.XTranslateCoordinates(d,w,root,0,0,C.byref(xx),C.byref(yy),C.byref(child));return xx.value,yy.value
def motion(xx,yy):xt.XTestFakeMotionEvent(d,-1,xx,yy,0);x.XFlush(d);time.sleep(.05)
def click(xx,yy):
 motion(xx,yy);xt.XTestFakeButtonEvent(d,1,1,0);xt.XTestFakeButtonEvent(d,1,0,0);x.XFlush(d);time.sleep(.1)
def key(name):
 code=x.XKeysymToKeycode(d,x.XStringToKeysym(name.encode()));xt.XTestFakeKeyEvent(d,code,1,0);xt.XTestFakeKeyEvent(d,code,0,0);x.XFlush(d)
def type_password(xx,yy):
 click(xx,yy)
 for c in 'syntheticpassword123456':key(c)
 time.sleep(.2)
scratch=Path(tempfile.mkdtemp(prefix='cipherenc-real-drop-'))
file=scratch/'Lesson [private] $notes.txt';file.write_text('Exact private lesson\n\n')
folder=scratch/'Folder with spaces';(folder/'Empty').mkdir(parents=True);(folder/'data.bin').write_bytes(bytes([0,255,7]))
log=open('/tmp/cipherenc-native-dnd.log','w');env=os.environ.copy()
app=subprocess.Popen(['build/CipherEnc-native'],env=env,stdout=log,stderr=log);source=None
try:
 main=locate('Offline encryption');time.sleep(1);xx,yy=origin(main)
 type_password(xx+110,yy+160);type_password(xx+540,yy+160)
 subprocess.run(['import','-window',str(main),'/tmp/cipherenc-native-before-drop.png'],check=True)
 target=(xx+440,yy+275)
 source=subprocess.Popen([sys.executable,str(Path(__file__).with_name('xdnd-source.py')),str(main),str(target[0]),str(target[1]),str(file),str(folder)],env=env,stdout=log,stderr=log)
 # Keep-original is the default in each post-verification confirmation.
 seen=set()
 for _ in range(160):
  for w,name in windows():
   if 'Shred the original' in name:
    subprocess.run(['import','-window',str(w),'/tmp/cipherenc-shred-prompt.png'],check=True)
    key('Return');time.sleep(.2)
  if Path(str(file)+'.cen').exists() and Path(str(folder)+'.cen').exists():
   time.sleep(1);key('Return');time.sleep(.2);break
  time.sleep(.1)
 else:
  subprocess.run(['import','-window',str(main),'/tmp/cipherenc-native-drop-failure.png'],check=True)
  raise AssertionError('Native drop did not create both adjacent encrypted archives')
 assert file.read_text()=='Exact private lesson\n\n'
 assert (folder/'data.bin').read_bytes()==bytes([0,255,7])
 if source.poll() is None:source.terminate();source.wait()
 source=None
 subprocess.run(['import','-window',str(main),'/tmp/cipherenc-native-after-drop.png'],check=True)
 print('Native X11 protocol drop: multiple files/folders, spaces and Tcl metacharacters, adjacent archives, default keep-original PASS')
 print('Synthetic archive directory:',scratch)
finally:
 if source:source.terminate();source.wait()
 app.terminate();app.wait()
