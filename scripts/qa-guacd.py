#!/usr/bin/env python3
"""Loopback-only simulated guacd. Does not exercise RDP/VNC or native libraries."""
import socket,threading,time,base64,json,pathlib,http.server
# Test controls pause actual protocol milestones, without driving UI timers.
handshake_gate=threading.Event();handshake_gate.set()
frame_gate=threading.Event();frame_gate.set()
class Control(http.server.BaseHTTPRequestHandler):
 def do_POST(self):
  if self.path=='/hold-handshake':handshake_gate.clear()
  elif self.path=='/hold-frame':frame_gate.clear()
  elif self.path=='/release-handshake':handshake_gate.set()
  elif self.path=='/release-frame':frame_gate.set()
  elif self.path=='/reset':handshake_gate.set();frame_gate.set()
  else:self.send_error(404);return
  self.send_response(200);self.end_headers();self.wfile.write(b'{}')
 def log_message(self,*args):pass
control=http.server.ThreadingHTTPServer(('127.0.0.1',3392),Control)
threading.Thread(target=control.serve_forever,daemon=True).start()
stats={'mouseEvents':0,'keyEvents':0,'unicodeClipboard':False,'heldKeys':0}
stats_lock=threading.Lock()
def write_stats():
 with stats_lock:
  target=pathlib.Path('build/qa-mock-stats.json')
  temporary=target.with_suffix('.tmp')
  temporary.write_text(json.dumps(stats))
  temporary.replace(target)
def enc(*p):return ''.join((',' if i else '')+str(len(str(v)))+'.'+str(v) for i,v in enumerate(p))+';'
def read(f):
 values=[]
 while True:
  prefix=''
  while True:
   c=f.read(1)
   if not c:raise EOFError()
   if c=='.':break
   prefix+=c
  value=f.read(int(prefix))
  if len(value)!=int(prefix):raise EOFError()
  values.append(value)
  if f.read(1)==';':return values

def client(c):
 try:
  f=c.makefile('r',encoding='utf-8')
  def send(*parts):c.sendall(enc(*parts).encode())
  selected=read(f)
  assert selected in (['select','rdp'],['select','vnc'])
  send('args','VERSION_1_5_0','hostname','port')
  for _ in range(6):read(f)
  if not handshake_gate.wait(30):return
  send('ready','simulated-only')
  if not frame_gate.wait(30):return
  send('size','0','800','600');send('rect','0','0','0','800','600');send('cfill','14','0','40','60','90','255');send('sync',str(int(time.time()*1000)))
  clipboard=b'';held=set()
  while True:
   p=read(f)
   if p[0]=='mouse':stats['mouseEvents']+=1
   elif p[0]=='key':
    stats['keyEvents']+=1
    if p[2]=='1':
     held.add(p[1])
     if p[1]==str(0x76):stats['pasteModifier']='command' if str(0xffe7) in held else 'ctrl' if str(0xffe3) in held else 'none'
    else:held.discard(p[1])
    stats['heldKeys']=len(held)
   elif p[0]=='clipboard':clipboard=b'';send('ack',p[1],'OK','0')
   elif p[0]=='blob':clipboard+=base64.b64decode(p[2]);send('ack',p[1],'OK','0')
   elif p[0]=='end':
    stats['unicodeClipboard']=clipboard.decode()=='中文😀\nclipboard';send('clipboard','77','text/plain');send('blob','77',base64.b64encode(clipboard).decode());send('end','77')
   write_stats()
 except (EOFError,OSError,AssertionError):pass
 finally:c.close()
with socket.socket() as s:
 s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',14822));s.listen()
 while True:
  c,_=s.accept();threading.Thread(target=client,args=(c,),daemon=True).start()
