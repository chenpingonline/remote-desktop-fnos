#!/usr/bin/env python3
"""Reproduce the unmodified Apache 1.6.0 browser client, checking its release hash."""
import hashlib,pathlib,tarfile,urllib.request
root=pathlib.Path(__file__).resolve().parent.parent
archive=root/'build/vendor/guacamole-client-1.6.0.tar.gz'
archive.parent.mkdir(parents=True,exist_ok=True)
if not archive.exists():urllib.request.urlretrieve('https://archive.apache.org/dist/guacamole/1.6.0/source/guacamole-client-1.6.0.tar.gz',archive)
assert hashlib.sha256(archive.read_bytes()).hexdigest()=='81f9fd5a7b4377fb0ee295d0d4fec92e9667f2aafaa3d0ed8937f535deabdee4','source hash mismatch'
with tarfile.open(archive) as t:
 prefix='guacamole-client-1.6.0/guacamole-common-js/'
 files=sorted(n for n in t.getnames() if n.startswith(prefix+'src/main/webapp/modules/') and n.endswith('.js'))
 out=root/'web/src/vendor';out.mkdir(exist_ok=True)
 (out/'guacamole.js').write_bytes(b'\n'.join(t.extractfile(n).read() for n in files)+b'\nexport default Guacamole;\n')
 for name in ['LICENSE','NOTICE']:(out/name).write_bytes(t.extractfile("guacamole-client-1.6.0/"+name).read())
