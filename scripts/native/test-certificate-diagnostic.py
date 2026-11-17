#!/usr/bin/env python3
"""Synthetic local RDP TLS certificate rejection; no Windows or real credentials."""
import os, pathlib, socket, ssl, subprocess, tempfile, threading, time
with tempfile.TemporaryDirectory() as tmp:
    cert, key = tmp+'/cert.pem', tmp+'/key.pem'
    subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-sha256','-days','1','-subj','/CN=qa.invalid','-keyout',key,'-out',cert],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    listener=socket.socket();listener.bind(('127.0.0.1',0));listener.listen()
    port=listener.getsockname()[1]
    pin_mode=os.environ.get('RD_TEST_PIN_MODE','')
    received_auth=[]
    context=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER);context.load_cert_chain(cert,key)
    def serve():
        while True:
            try:
                conn,_=listener.accept();conn.settimeout(5)
                request=conn.recv(4096)
                if not request:conn.close();continue
                # TPKT/X.224 connection confirm selecting TLS + CredSSP (HYBRID).
                conn.sendall(bytes.fromhex('030000130ed000001234000200080002000000'))
                try:
                    with context.wrap_socket(conn,server_side=True) as secure:
                        received_auth.append(len(secure.recv(4096)))
                except (ssl.SSLError,OSError):conn.close()
            except OSError:return
    threading.Thread(target=serve,daemon=True).start()
    env=dict(os.environ,LD_LIBRARY_PATH='/out/lib',OPENSSL_MODULES='/out/lib/ossl-modules')
    bad_home=os.environ.get('RD_TEST_HOME_MODE')=='unwritable'
    for name,child in [('HOME','home'),('XDG_CONFIG_HOME','config'),('XDG_CACHE_HOME','cache'),('XDG_DATA_HOME','data')]:
        if bad_home:
            env.pop(name,None)
        else:
            directory=tmp+'/'+child;os.mkdir(directory,0o700);env[name]=directory
    if bad_home:env['HOME']='/root' 
    session_fingerprint=__import__('hashlib').sha256(ssl.PEM_cert_to_DER_cert(pathlib.Path(cert).read_text())).hexdigest()
    if pin_mode in ('changed','legacy','other-port','trusted-mismatch'):
        pin_cert=cert
        if pin_mode=='changed':
            pin_cert=tmp+'/other.pem'
            subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-sha256','-days','1','-subj','/CN=other.invalid','-keyout',tmp+'/other.key','-out',pin_cert],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        der=ssl.PEM_cert_to_DER_cert(pathlib.Path(pin_cert).read_text())
        import hashlib
        fingerprint=hashlib.sha256(der).hexdigest()
        if pin_mode=='changed':session_fingerprint=fingerprint
        else:subprocess.run([os.environ['RD_TEST_PIN_TOOL'],'127.0.0.1',str(port+1 if pin_mode=='other-port' else port),fingerprint],input=der,env=env,check=True,stdout=subprocess.DEVNULL)
    daemon=subprocess.Popen(['/out/bin/guacd','-f','-L','debug','-b','127.0.0.1','-l','14822'],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.PIPE,start_new_session=True)
    logs=[]
    def collect():
        for line in daemon.stderr:
            if len(logs)<5000:logs.append(line)
    threading.Thread(target=collect,daemon=True).start()
    try:
        for _ in range(50):
            try:c=socket.create_connection(('127.0.0.1',14822),timeout=1);break
            except OSError:time.sleep(.1)
        c.settimeout(10);reader=c.makefile('rb')
        def read():
            parts=[]
            while True:
                digits=b''
                while True:
                    b=reader.read(1)
                    if not b:raise EOFError()
                    if b==b'.':break
                    digits+=b
                parts.append(reader.read(int(digits)).decode())
                if reader.read(1)==b';':return parts
        def send(*parts):
            c.sendall((','.join(str(len(p))+'.'+p for p in parts)+';').encode())
        send('select','rdp');args=read();assert args[0]=='args'
        for instruction in [('size','800','600','96'),('audio',),('video',),('image','image/png'),('timezone','UTC')]:send(*instruction)
        params={'hostname':'127.0.0.1','port':str(port),'username':'synthetic-user','password':'synthetic-password','security':'nla','ignore-cert':'false','disable-audio':'true'}
        if pin_mode in ('correct','changed','trusted-mismatch'):
            params['cert-fingerprints']='sha256:'+('0'*64 if pin_mode=='trusted-mismatch' else session_fingerprint)
        send('connect',*(params.get(a,'') for a in args[1:]))
        error=None
        for _ in range(100):
            message=read()
            if message[0]=='error':error=message;break
        time.sleep(.3)
        data=b''.join(logs)
        if pin_mode=='correct':
            assert b'Certificate validation failed' not in data
            assert any(received_auth), 'matching pin did not reach NLA negotiation'
            print('Matching host/port/SHA256: certificate accepted and synthetic NLA negotiation reached')
        elif pin_mode in ('changed','trusted-mismatch'):
            assert b'Session certificate fingerprint mismatch or expired' in data
            assert not any(received_auth)
            print('Changed certificate: rejected before synthetic NLA data')
        elif bad_home:
            assert b'certificate store initialization failed' in data
            assert error and error[2]=='769'
            print('Unwritable inherited HOME: reproduced certificate-store failure and status=769 before certificate verification')
        else:
            assert b'Session certificate authorization missing' in data, 'missing session pin was not rejected'
            assert b'certificate store initialization failed' not in data
            assert error and error[2] in ('769','519'), 'expected TLS/security rejection'
            assert not any(received_auth)
            print('Missing session authorization, including legacy shared trust: rejected; status='+error[2])
        c.close()
    finally:
        import signal
        os.killpg(daemon.pid,signal.SIGTERM)
        daemon.wait(timeout=5);listener.close()
