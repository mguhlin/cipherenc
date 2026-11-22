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
 print('Native X11 protocol drop: password first, multiple files/folders, spaces and Tcl metacharacters, adjacent archives, default keep-original PASS')
 def chord(modifier,name):
  code=x.XKeysymToKeycode(d,x.XStringToKeysym(modifier.encode()));xt.XTestFakeKeyEvent(d,code,1,0);key(name);xt.XTestFakeKeyEvent(d,code,0,0);x.XFlush(d)
 def select_all():
  key('Home');chord('Shift_L','End')
 def paste(text):
  subprocess.run(['xclip','-selection','clipboard'],input=text.encode(),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,check=True)
  chord('Control_L','v');time.sleep(.1)
 def select_first(path,folder=False):
  click(xx+(245 if folder else 100),yy+380);locate('Choose folder' if folder else 'Choose files')
  chord('Alt_L','s' if folder else 'n');select_all();key('BackSpace');paste(str(path));key('Return');time.sleep(.3)
  if folder:chord('Alt_L','o');time.sleep(.3)
 def fill_password(confirm='syntheticpassword123456',paste_confirm=False):
  type_password(xx+110,yy+160)
  click(xx+540,yy+160)
  if paste_confirm:paste(confirm)
  else:
   for char in confirm:key(char)
 def wait_saved(paths):
  for _ in range(180):
   for w,name in windows():
    if 'Shred the original' in name:key('Return');time.sleep(.1)
   if all(Path(str(path)+'.cen').exists() for path in paths):
    time.sleep(.7)
    for w,name in windows():
     if 'Shred the original' in name:key('Return')
    time.sleep(.3);return
   time.sleep(.1)
  subprocess.run(['import','-window',str(main),'/tmp/cipherenc-autostart-failure.png'],check=True)
  raise AssertionError('No automatic adjacent archive for '+str(paths))
 def restart():
  global app,main,xx,yy
  app.terminate();app.wait();app=subprocess.Popen(['build/CipherEnc-native'],env=env,stdout=log,stderr=log)
  main=locate('Offline encryption');time.sleep(.7);xx,yy=origin(main)
 # Reproduce the reported order using the actual native file picker.
 restart()
 picked=scratch/'picked.txt';picked.write_text('File selected before its password\n')
 select_first(picked);assert not Path(str(picked)+'.cen').exists()
 fill_password('notmatchingpassword');time.sleep(.65);assert not Path(str(picked)+'.cen').exists(),'Mismatching confirmation started encryption'
 click(xx+540,yy+160);select_all();key('BackSpace');paste('syntheticpassword123456');wait_saved([picked]);assert picked.read_text()=='File selected before its password\n'
 print('Native file picker first: mismatch waits, pasted matching confirmation auto-encrypts and saves beside original without Process selected PASS')
 restart()
 picked_folder=scratch/'picked-folder';picked_folder.mkdir();(picked_folder/'note.txt').write_text('Folder selected before password\n')
 select_first(picked_folder,folder=True);fill_password();wait_saved([picked_folder]);assert (picked_folder/'note.txt').read_text()=='Folder selected before password\n'
 print('Native folder picker first: typing the matching confirmation auto-encrypts beside the original without Process selected PASS')
 # Clear must invalidate an armed start, and never bring back selected work.
 restart()
 cancelled=scratch/'cancelled.txt';cancelled.write_text('Synthetic cancelled selection')
 select_first(cancelled);fill_password(paste_confirm=True);click(xx+235,yy+675);time.sleep(.7);assert not Path(str(cancelled)+'.cen').exists(),'Clear failed to cancel pending encryption'
 print('Native pending automatic encryption cancelled by Clear PASS')
 restart()
 generated=scratch/'generated.txt';generated.write_text('Synthetic generated-password selection')
 select_first(generated);click(xx+120,yy+205);time.sleep(.65);assert not Path(str(generated)+'.cen').exists(),'Generated password used before confirmation'
 click(xx+100,yy+160);select_all();chord('Control_L','c');time.sleep(.1)
 generated_password=subprocess.check_output(['xclip','-selection','clipboard','-o']).decode();assert len(generated_password)==28
 click(xx+540,yy+160);paste(generated_password);wait_saved([generated]);assert generated.read_text()=='Synthetic generated-password selection'
 print('Native generated password after selection: waits for confirmation, then auto-saves without Process selected PASS')
 print('Synthetic archive directory:',scratch)
finally:
 if source:source.terminate();source.wait()
 app.terminate();app.wait()
