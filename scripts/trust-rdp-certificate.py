#!/usr/bin/env python3
"""Explicitly pin a verified public RDP certificate. Does not read credentials."""
import argparse, hashlib, json, os, pathlib, re, shutil, socket, sqlite3, ssl, stat, struct, subprocess, tempfile

# Filled from the reviewed native helper when preparing this diagnostic tool.
HELPER_SHA256 = '2a8d9f11db52b33ee010f9c3735a22d9b3eedf409786414dabab29a841a5db04'

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--name',default='win')
    parser.add_argument('--confirm-sha256',required=True,help='SHA256 independently verified on the Windows computer')
    args=parser.parse_args()
    expected=args.confirm_sha256.lower().replace(':','')
    if not re.fullmatch('[0-9a-f]{64}',expected):raise SystemExit('Expected a complete SHA256 fingerprint')
    data=pathlib.Path('/vol1/@appdata/remote-desktop')
    app=pathlib.Path('/vol1/@appcenter/remote-desktop')
    if os.geteuid()!=data.stat().st_uid:
        raise SystemExit('Run with sudo -u remote-desktop; do not run as root')
    os.umask(0o077)
    env=dict(os.environ)
    for name,child in [('HOME','home'),('XDG_CONFIG_HOME','config'),('XDG_CACHE_HOME','cache'),('XDG_DATA_HOME','data')]:
        directory=data/'engine'/child
        info=directory.lstat()
        if not stat.S_ISDIR(info.st_mode) or info.st_uid!=os.geteuid() or info.st_mode&0o077:
            raise SystemExit('Expected private runtime directories from 0.1.11 or later')
        env[name]=str(directory)
    env.update(LD_LIBRARY_PATH=str(app/'lib'),OPENSSL_MODULES=str(app/'lib'/'ossl-modules'))
    helper=data/'pin-rdp-certificate'
    if hashlib.sha256(helper.read_bytes()).hexdigest()!=HELPER_SHA256:
        raise SystemExit('Native helper checksum mismatch; no trust changes made')
    with sqlite3.connect((data/'app.db').as_uri()+'?mode=ro',uri=True,timeout=3) as db:
        connections=[json.loads(row[0]) for row in db.execute("SELECT data FROM objects WHERE kind='connections'")]
    selected=[c for c in connections if c.get('name')==args.name and c.get('protocol')=='rdp']
    if len(selected)!=1:raise SystemExit('Expected exactly one saved RDP connection with that name')
    host,port=selected[0]['host'],int(selected[0]['port'])
    with socket.create_connection((host,port),timeout=5) as conn:
        conn.settimeout(5)
        conn.sendall(bytes.fromhex('030000130ee000000000000100080003000000'))
        def exact(size):
            data=b''
            while len(data)<size:
                part=conn.recv(size-len(data))
                if not part:raise SystemExit('Server closed during negotiation; no trust changes made')
                data+=part
            return data
        head=exact(4)
        length=int.from_bytes(head[2:4],'big')
        if head[:2]!=b'\x03\x00' or length!=19:raise SystemExit('Unexpected RDP response; no trust changes made')
        packet=head+exact(15)
        if packet[5]!=0xd0 or packet[11]!=2 or struct.unpack('<H',packet[13:15])[0]!=8 or int.from_bytes(packet[15:19],'little')!=2:
            raise SystemExit('Server did not select NLA; no trust changes made')
        # Inspection only, never sends CredSSP or RDP credentials.
        context=ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        context.check_hostname=False;context.verify_mode=ssl.CERT_NONE
        with context.wrap_socket(conn,server_hostname=host) as secure:
            der=secure.getpeercert(binary_form=True)
    if hashlib.sha256(der).hexdigest()!=expected:
        raise SystemExit('Server certificate differs from the fingerprint verified on Windows; no trust changes made')
    print('Verified the certificate again. Saving only this host/port/certificate pin; no login attempted.',flush=True)
    with tempfile.TemporaryDirectory(prefix='cert-pin-',dir=data/'engine') as tmp:
        executable=pathlib.Path(tmp)/'pin-rdp-certificate'
        shutil.copyfile(helper,executable);executable.chmod(0o700)
        result=subprocess.run([str(executable),host,str(port),expected],input=der,env=env)
    if result.returncode:raise SystemExit(result.returncode)
    print('完成：已为该主机和端口信任核对过的证书。请回到应用重新连接。')

if __name__=='__main__':main()
