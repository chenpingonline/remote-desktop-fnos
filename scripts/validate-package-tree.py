#!/usr/bin/env python3
"""Validate and normalize the public FPK payload before fnpack.

fnOS may dereference links while applying installation permissions. No links,
private data, macOS metadata or special files are permitted in this payload.
"""
import json
import os
import pathlib
import stat
import sys

root = pathlib.Path(__file__).resolve().parent.parent / 'packaging/remote-desktop'
errors = []
for current, dirs, files in os.walk(root, followlinks=False):
    current = pathlib.Path(current)
    for name in dirs + files:
        path = current / name
        relative = path.relative_to(root)
        if name == '.DS_Store' or name.startswith('._'):
            if path.is_file() and not path.is_symlink():
                path.unlink()
            continue
        mode = path.lstat().st_mode
        if stat.S_ISLNK(mode):
            errors.append(f'symbolic link is not permitted: {relative}')
        elif stat.S_ISDIR(mode):
            path.chmod(0o755)
        elif stat.S_ISREG(mode):
            if path.suffix in ('.key', '.db', '.sqlite'):
                errors.append(f'private data is not permitted: {relative}')
            executable = relative.parts[0] == 'cmd' or relative.parts[:2] == ('app', 'bin')
            path.chmod(0o755 if executable else 0o644)
        else:
            errors.append(f'special file is not permitted: {relative}')
privilege = json.loads((root / 'config/privilege').read_text())
if privilege.get('defaults', {}).get('run-as') != 'package':
    errors.append('service must use the package identity')
for field in ('username', 'groupname'):
    if privilege.get(field) != 'remote-desktop':
        errors.append(f'explicit {field} must be remote-desktop')
if errors:
    print('\n'.join(errors), file=sys.stderr)
    sys.exit(1)
print('FPK tree verified: explicit package identity; no links/private data/special files; normalized file modes')
