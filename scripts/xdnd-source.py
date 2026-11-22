import sys,time
from pathlib import Path
from Xlib import X,display,protocol
from urllib.parse import quote
D=display.Display();root=D.screen().root
source=root.create_window(930,20,320,140,0,X.CopyFromParent,X.InputOutput,X.CopyFromParent,event_mask=X.PropertyChangeMask)
source.set_wm_name('CipherEnc synthetic drag source');source.map();D.sync()
def atom(s):return D.intern_atom(s)
def find_aware(w):
 if w.get_full_property(atom('XdndAware'),X.AnyPropertyType):return w
 for child in w.query_tree().children:
  out=find_aware(child)
  if out:return out
main=D.create_resource_object('window',int(sys.argv[1]));target=find_aware(main)
if not target:raise Exception('target missing XdndAware')
uri=atom('text/uri-list');selection=atom('XdndSelection');source.set_selection_owner(selection,X.CurrentTime)
payload=('\r\n'.join('file://'+quote(str(Path(p).absolute()),safe='/') for p in sys.argv[4:])+'\r\n').encode()
def send(kind,data):target.send_event(protocol.event.ClientMessage(window=target,client_type=atom(kind),data=(32,data)),event_mask=0);D.flush()
x,y=map(int,sys.argv[2:4]);send('XdndEnter',[source.id,5<<24,uri,0,0]);send('XdndPosition',[source.id,0,(x<<16)|y,X.CurrentTime,atom('XdndActionCopy')])
start=time.monotonic();dropped=False
while time.monotonic()-start<20:
 if not D.pending_events():time.sleep(.01);continue
 e=D.next_event()
 if e.type==X.ClientMessage:
  if e.client_type==atom('XdndStatus') and not dropped:
   if not e.data[1][1]&1:raise Exception('target refused files')
   send('XdndDrop',[source.id,0,X.CurrentTime,0,0]);dropped=True
  if e.client_type==atom('XdndFinished'):print('X11 protocol drop accepted',flush=True);break
 elif e.type==X.SelectionRequest:
  prop=e.property or e.target
  e.requestor.change_property(prop,e.target,8,payload)
  e.requestor.send_event(protocol.event.SelectionNotify(time=e.time,requestor=e.requestor,selection=e.selection,target=e.target,property=prop),event_mask=0)
  D.flush()
else:raise Exception('drop timed out')
# Allow the integration driver to inspect the app before this process exits.
time.sleep(5)
