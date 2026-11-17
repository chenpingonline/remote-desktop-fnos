import base64, collections, hashlib, os, pathlib, signal, socket, ssl, subprocess, time
import io, select
from PIL import Image

root=pathlib.Path('/qa');root.mkdir(exist_ok=True)
env=dict(os.environ,LD_LIBRARY_PATH='/out/lib',OPENSSL_MODULES='/out/lib/ossl-modules')
for key,child in [('HOME','home'),('XDG_CONFIG_HOME','config'),('XDG_CACHE_HOME','cache'),('XDG_DATA_HOME','data')]:
    p=root/child;p.mkdir(exist_ok=True);env[key]=str(p)
processes=[];handles=[]
username='rdpqa@QA.INVALID' if os.environ.get('QA_EMAIL_ACCOUNT')=='1' else 'rdpqa'
blackhole=[]
if os.environ.get('QA_BLACKHOLE_KDC')=='1':
    config=root/'krb5.conf'
    config.write_text('[libdefaults]\n default_realm = QA.INVALID\n dns_lookup_kdc = false\n dns_lookup_realm = false\n[realms]\n QA.INVALID = {\n  kdc = 127.0.0.1:1088\n }\n')
    env['KRB5_CONFIG']=str(config)
    # Accept transport connections but never answer Kerberos. No external traffic.
    for kind in (socket.SOCK_DGRAM,socket.SOCK_STREAM):
        sink=socket.socket(socket.AF_INET,kind);sink.bind(('127.0.0.1',1088))
        if kind==socket.SOCK_STREAM:sink.listen(8)
        blackhole.append(sink)
def launch(name,args,custom_env=None):
    out=open(root/(name+'.log'),'wb');handles.append(out)
    p=subprocess.Popen(args,env=custom_env or os.environ,stdout=out,stderr=subprocess.STDOUT,start_new_session=True)
    processes.append(p);return p
def wait_port(port):
    for _ in range(200):
        try:
            c=socket.create_connection(('127.0.0.1',port),timeout=.5);c.close();return
        except OSError:time.sleep(.1)
    raise RuntimeError('port not ready '+str(port))
try:
    shadow=os.environ.get('QA_SERVER')=='shadow'
    if shadow:
        shadow_env=dict(env,LD_LIBRARY_PATH='/qa-shadow/lib',DISPLAY=':91')
        launch('xvfb',['Xvfb',':91','-screen','0','1024x768x24','-nolisten','tcp','-ac'],shadow_env)
        for _ in range(100):
            if pathlib.Path('/tmp/.X11-unix/X91').exists():break
            time.sleep(.1)
        launch('xterm',['xterm','-geometry','80x24+20+20','-fg','white','-bg','black','-e','sh','-c','i=0; while [ "$i" -lt 60 ]; do printf "NLA GFX synthetic desktop tick %s\\n" "$i"; i=$((i+1)); sleep 0.5; done; exec sleep 90'],shadow_env)
        config=subprocess.check_output(['python3','-c',"import ctypes; l=ctypes.CDLL('/qa-shadow/lib/libfreerdp3.so.3'); l.freerdp_settings_get_config_path.restype=ctypes.c_char_p; print(l.freerdp_settings_get_config_path().decode())"],env=shadow_env,text=True).strip()
        cert_path=pathlib.Path(config)/'shadow';cert_path.mkdir(parents=True,exist_ok=True)
        import shutil
        shutil.copyfile('/etc/xrdp/cert.pem',cert_path/'shadow.crt')
        shutil.copyfile('/etc/xrdp/key.pem',cert_path/'shadow.key')
        hashed=subprocess.check_output(['openssl','dgst','-md4','-provider','default','-provider','legacy'],input='synthetic-rdp-qa-only'.encode('utf-16le'),env=env).decode().split()[-1]
        sam=root/'sam';sam.write_text(username+':::'+hashed+':\n');sam.chmod(0o600)
        launch('shadow',['/qa-shadow/bin/freerdp-shadow-cli','/port:3389','/bind-address:127.0.0.1','/sec:nla','/sam-file:'+str(sam),'/log-level:DEBUG'],shadow_env)
    else:
        launch('sesman',['/usr/sbin/xrdp-sesman','--nodaemon'])
        launch('xrdp',['/usr/sbin/xrdp','--nodaemon'])
    wait_port(3389)
    launch('guacd',['/out/bin/guacd','-f','-L','debug','-b','127.0.0.1','-l','14822'],env);wait_port(14822)
    fingerprint=hashlib.sha256(ssl.PEM_cert_to_DER_cert(pathlib.Path('/etc/xrdp/cert.pem').read_text())).hexdigest()
    c=socket.create_connection(('127.0.0.1',14822),timeout=10);c.settimeout(12 if blackhole else 60);f=c.makefile('rb')
    def send(*parts):c.sendall((','.join(str(len(str(p)))+'.'+str(p) for p in parts)+';').encode())
    def read():
        parts=[]
        while True:
            digits=b''
            while True:
                b=f.read(1)
                if not b:raise EOFError()
                if b==b'.':break
                digits+=b
            parts.append(f.read(int(digits)).decode())
            if f.read(1)==b';':return parts
    send('select','rdp');args=read();assert args[0]=='args'
    for ins in [('size','1024','768','96'),('audio',),('video',),('image','image/png','image/jpeg'),('timezone','UTC')]:send(*ins)
    params={'hostname':'127.0.0.1','port':'3389','username':username,'password':'synthetic-rdp-qa-only','security':'tls','ignore-cert':'false','disable-audio':'true','cert-fingerprints':'sha256:'+fingerprint,'resize-method':'display-update'}
    if shadow:params['security']='nla'
    if os.environ.get('QA_DOMAIN'):params['domain']=os.environ['QA_DOMAIN']
    if os.environ.get('QA_DISABLE_GFX')=='1':params['disable-gfx']='true';params['force-lossless']='true'
    send('connect',*(params.get(a,'') for a in args[1:]))
    seen=collections.Counter();started=time.monotonic();deadline=started+60;images={};valid_image=False
    while time.monotonic()<deadline:
        msg=read();op=msg[0];seen[op]+=1
        if op=='error':raise RuntimeError(str(msg))
        if op=='sync':send('sync',msg[1])
        if op=='img':images[msg[1]]={'type':msg[4],'data':bytearray()}
        if op=='blob' and msg[1] in images:images[msg[1]]['data'].extend(base64.b64decode(msg[2]))
        if op=='end' and msg[1] in images:
            result=images.pop(msg[1]);data=bytes(result['data'])
            if len(data)>500 and any(high-low>64 for low,high in Image.open(io.BytesIO(data)).convert('RGB').getextrema()):
                (root/('frame.png' if data.startswith(b'\x89PNG') else 'frame.jpg')).write_bytes(data)
                valid_image=True
        if valid_image and (not shadow or time.monotonic()-started>=8):
            raw=(root/'guacd.log').read_text(errors='replace')
            assert 'winpr_int_assert' not in raw, 'native assertion during connected desktop'
            assert 'RDP_STAGE_' not in raw and 'RDP_AUTH_NTLM_PASSWORD_ACCOUNT' not in raw
            if env.get('WLOG_LEVEL')=='WARN':
                assert 'Using package: Negotiate' not in raw and 'Acquired client credentials' not in raw
            if shadow and params.get('disable-gfx')!='true':assert 'RDPGFX channel will be used' in raw
            if blackhole:
                assert not select.select(blackhole,[],[],0)[0], 'Non-domain account attempted Kerberos'
            print('FULL RDP AUTH + LIVE IMAGE PASS',params['security'],'gfx disabled',params.get('disable-gfx','false'),'seconds',round(time.monotonic()-started,2),dict(seen),flush=True)
            break
    else:raise RuntimeError('no image '+str(dict(seen)))
    c.close()
except BaseException:
    for path in root.glob('*.log'):
        print('\nLOG',path.name, path.read_text(errors='replace')[-30000:],flush=True)
    for path in [pathlib.Path('/var/log/xrdp.log'),pathlib.Path('/var/log/xrdp-sesman.log')]:
        if path.exists():print('\nLOG',path,path.read_text(errors='replace')[-15000:],flush=True)
    raise
finally:
    for p in reversed(processes):
        if p.poll() is None:os.killpg(p.pid,signal.SIGTERM)
    for p in reversed(processes):
        try:p.wait(timeout=5)
        except subprocess.TimeoutExpired:os.killpg(p.pid,signal.SIGKILL)
    for f in handles:f.close()
    for sink in blackhole:sink.close()
