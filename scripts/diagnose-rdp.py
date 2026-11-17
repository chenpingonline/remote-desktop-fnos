#!/usr/bin/env python3
"""Read-only RDP transport probe. Never reads secrets or sends login credentials."""
import argparse, hashlib, json, pathlib, socket, sqlite3, ssl, struct, sys

def probe(host, port):
    result = {'stage': 'TCP'}
    try:
        with socket.create_connection((host, port), timeout=5) as conn:
            conn.settimeout(5)
            # X.224 connection request: TLS + NLA, no username/cookie/credentials.
            conn.sendall(bytes.fromhex('030000130ee000000000000100080003000000'))
            def exact(size):
                data = b''
                while len(data) < size:
                    part = conn.recv(size-len(data))
                    if not part: raise EOFError('peer closed before full response')
                    data += part
                return data
            header = exact(4)
            size = int.from_bytes(header[2:4], 'big')
            if header[:2] != b'\x03\x00' or not 11 <= size <= 4096:
                return {'stage':'X224', 'result':'INVALID_RDP_RESPONSE'}
            packet = header+exact(size-4)
            if packet[5] != 0xd0:
                return {'stage':'X224', 'result':'NOT_CONNECTION_CONFIRM'}
            if len(packet) == 11:
                return {'stage':'NEGOTIATION', 'result':'LEGACY_RDP_ONLY'}
            if len(packet) != 19 or struct.unpack('<H',packet[13:15])[0] != 8:
                return {'stage':'NEGOTIATION', 'result':'UNEXPECTED_RESPONSE_LENGTH'}
            value = int.from_bytes(packet[15:19], 'little')
            if packet[11] == 3:
                names = {1:'TLS_REQUIRED',2:'TLS_NOT_ALLOWED',3:'SERVER_CERT_MISSING',4:'INCONSISTENT_FLAGS',5:'NLA_REQUIRED',6:'NLA_EXTENDED_REQUIRED'}
                return {'stage':'NEGOTIATION','result':'SERVER_REJECTED','code':value,'detail':names.get(value,'UNKNOWN')}
            if packet[11] != 2:
                return {'stage':'NEGOTIATION','result':'UNKNOWN_RESPONSE_TYPE','type':packet[11]}
            result = {'stage':'TLS','selectedProtocol':{0:'LEGACY_RDP',1:'TLS',2:'NLA',8:'NLA_EXTENDED'}.get(value,'UNKNOWN'),'selectedValue':value}
            if value not in (1,2,8):return result
            # Certificate inspection ONLY. No CredSSP/RDP authentication follows.
            # Unverified certificates are never trusted or saved by this probe.
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
            context.check_hostname = False
            context.verify_mode = ssl.CERT_NONE
            with context.wrap_socket(conn,server_hostname=host) as secure:
                der = secure.getpeercert(binary_form=True)
                result.update(result='TLS_HANDSHAKE_OK_NO_LOGIN_ATTEMPT',tlsVersion=secure.version(),cipher=secure.cipher()[0],certificateSHA256=hashlib.sha256(der).hexdigest())
            return result
    except ssl.SSLError as error:
        result.update(result='TLS_HANDSHAKE_FAILED',sslReason=error.reason or 'UNKNOWN')
    except (OSError,EOFError) as error:
        result.update(result='TRANSPORT_FAILED',errorType=type(error).__name__,errno=getattr(error,'errno',None))
    return result

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--db',default='/vol1/@appdata/remote-desktop/app.db')
    parser.add_argument('--name',default='win')
    args=parser.parse_args()
    dbpath=pathlib.Path(args.db).resolve()
    with sqlite3.connect(dbpath.as_uri()+'?mode=ro',uri=True,timeout=3) as db:
        rows=db.execute("SELECT data FROM objects WHERE kind='connections'").fetchall()
        matches=[]
        for row in rows:
            connection=json.loads(row[0])
            if connection.get('name')==args.name and connection.get('protocol')=='rdp':matches.append(connection)
    if len(matches)!=1:raise SystemExit('Expected exactly one saved RDP connection with that name; no network probe performed')
    connection=matches[0]
    print('Probe only: no credential read, no login attempt, no certificate trust changes')
    print(json.dumps(probe(connection['host'],int(connection['port'])),ensure_ascii=False,indent=2))
if __name__=='__main__':main()
