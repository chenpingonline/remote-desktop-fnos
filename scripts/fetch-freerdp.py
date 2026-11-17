#!/usr/bin/env python3
"""Fetch the locked upstream release; verify cached and downloaded source alike."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

root = Path(__file__).resolve().parent.parent
lock = json.loads((root / 'scripts/native/freerdp-source.json').read_text())
archive = root / f'build/vendor/freerdp-{lock["version"]}.tar.gz'
archive.parent.mkdir(parents=True, exist_ok=True)
if not archive.exists():
    temporary = archive.with_suffix('.download')
    try:
        for url in (lock['url'], lock['mirror']):
            result = subprocess.run(['curl', '-fL', '--retry', '3', '--retry-all-errors', '--connect-timeout', '20', url, '-o', str(temporary)])
            if result.returncode == 0:
                if hashlib.sha256(temporary.read_bytes()).hexdigest() != lock['sha256']:
                    raise SystemExit('FreeRDP source hash mismatch')
                temporary.replace(archive)
                break
        else:
            raise SystemExit('Unable to download locked FreeRDP source')
    finally:
        temporary.unlink(missing_ok=True)
if hashlib.sha256(archive.read_bytes()).hexdigest() != lock['sha256']:
    raise SystemExit('Cached FreeRDP source hash mismatch')
shutil.copyfile(archive, root / 'scripts/native/freerdp.tar.gz')
print(f'FreeRDP {lock["version"]}: source SHA256 verified')
