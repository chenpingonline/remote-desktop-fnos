#!/usr/bin/env python3
"""Native FreeRDP pin-store checks using public synthetic certificates only."""
import hashlib,os,pathlib,ssl,subprocess,tempfile
with tempfile.TemporaryDirectory() as tmp:
    root=pathlib.Path(tmp)
    env=dict(os.environ,LD_LIBRARY_PATH='/out/lib',OPENSSL_MODULES='/out/lib/ossl-modules')
    for name,child in [('HOME','home'),('XDG_CONFIG_HOME','config')]:
        directory=root/child;directory.mkdir(mode=0o700);env[name]=str(directory)
    certs=[]
    for i in range(2):
        subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-sha256','-days','1','-subj','/CN=synthetic-'+str(i),'-keyout',str(root/('key'+str(i))),'-out',str(root/('cert'+str(i)))],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        der=ssl.PEM_cert_to_DER_cert((root/('cert'+str(i))).read_text());certs.append((der,hashlib.sha256(der).hexdigest()))
    def pin(host,index,expected=None):
        der,fingerprint=certs[index]
        return subprocess.run(['/pin',host,'3389',fingerprint if expected is None else expected],input=der,env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode
    assert pin('target.example',0,'0'*64)==5
    def saved():
        return {str(p.relative_to(root)):p.read_bytes() for p in root.rglob('server/*') if p.is_file()}
    assert not saved()
    assert pin('other.example',1)==0
    other=saved()
    assert len(other)==1
    assert pin('target.example',0)==0
    baseline=saved()
    assert len(baseline)==2 and all(baseline[k]==v for k,v in other.items())
    assert all(p.stat().st_mode&0o777==0o600 for p in root.rglob('server/*') if p.is_file())
    assert pin('target.example',0)==0 and saved()==baseline
    assert pin('target.example',1)==7 and saved()==baseline
    print('Native pin tool: wrong SHA rejected, existing entries preserved, duplicate idempotent, changed pin refused, private file permissions verified')
