#!/usr/bin/env python3
"""Check the exported private runtime, including the actual loaded FreeRDP."""
import ctypes
import json
from pathlib import Path
import subprocess

lock = json.loads(Path('/tmp/freerdp-source.json').read_text())
library = ctypes.CDLL('/out/lib/libfreerdp3.so.3')
library.freerdp_get_version_string.restype = ctypes.c_char_p
assert library.freerdp_get_version_string().decode() == lock['version']
library.freerdp_get_build_config.restype = ctypes.c_char_p
config = library.freerdp_get_build_config().decode()
assert 'WITH_FFMPEG=ON' in config
client = ctypes.CDLL('/out/lib/libfreerdp-client3.so.3')
entry = client.freerdp_channels_load_static_addin_entry
entry.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.c_char_p, ctypes.c_uint32]
entry.restype = ctypes.c_void_p
for channel, flags in ((b'cliprdr', 0x1000 | 0x8000), (b'drdynvc', 0x1000 | 0x8000),
                       (b'disp', 0x2000), (b'rdpgfx', 0x2000)):
    assert entry(channel, None, None, flags), f'Missing built-in channel: {channel!r}'
for path in sorted(Path('/out/lib').glob('*.so*')):
    assert 'freerdp2' not in path.name and 'winpr2' not in path.name, path
    result = subprocess.run(['ldd', str(path)], check=True, capture_output=True, text=True)
    assert 'not found' not in result.stdout, result.stdout
subprocess.run(['/out/bin/nla-crypto-check'], check=True)
print(f'Private runtime: FreeRDP {lock["version"]}, FFmpeg, clipboard/display/GFX channels, library closure and NLA crypto verified')
