#!/usr/bin/env python3
"""Loopback-only synthetic RDP TLS endpoint for browser certificate-flow tests.
No Windows authentication. Control API rotates between two temporary test certs.
"""
import hashlib,http.server,json,pathlib,socket,ssl,subprocess,tempfile,threading
with tempfile.TemporaryDirectory(prefix='rdp-certificate-qa-') as tmp:
    contexts=[];fingerprints=[];current=[0];lock=threading.Lock()
    for i in range(2):
        cert=pathlib.Path(tmp)/f'{i}.pem';key=pathlib.Path(tmp)/f'{i}.key'
        subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-sha256','-days','1','-subj',f'/CN=local-test-{i}','-keyout',str(key),'-out',str(cert)],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        ctx=ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER);ctx.load_cert_chain(cert,key);contexts.append(ctx)
        fingerprints.append(hashlib.sha256(ssl.PEM_cert_to_DER_cert(cert.read_text())).hexdigest())
    class Control(http.server.BaseHTTPRequestHandler):
        def do_POST(self):
            with lock:
                if self.path=='/rotate':current[0]=1-current[0]
                elif self.path=='/reset':current[0]=0
                else:self.send_error(404);return
                payload=json.dumps({'fingerprint':fingerprints[current[0]]}).encode()
            self.send_response(200);self.send_header('Content-Type','application/json');self.end_headers();self.wfile.write(payload)
        def log_message(self,*args):pass
    control=http.server.ThreadingHTTPServer(('127.0.0.1',3391),Control)
    threading.Thread(target=control.serve_forever,daemon=True).start()
    def client(conn):
        try:
            conn.settimeout(5);request=b''
            while len(request)<19:
                piece=conn.recv(19-len(request))
                if not piece:return
                request+=piece
            if request!=bytes.fromhex('030000130ee000000000000100080003000000'):return
            conn.sendall(bytes.fromhex('030000130ed000000000000200080002000000'))
            with lock:context=contexts[current[0]]
            with context.wrap_socket(conn,server_side=True) as secure:secure.recv(4096)
        except (ssl.SSLError,OSError):pass
        finally:conn.close()
    with socket.socket() as listener:
        listener.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);listener.bind(('127.0.0.1',3390));listener.listen()
        print('Synthetic RDP TLS ready on 127.0.0.1:3390; control on 3391',flush=True)
        while True:
            conn,_=listener.accept();threading.Thread(target=client,args=(conn,),daemon=True).start()
