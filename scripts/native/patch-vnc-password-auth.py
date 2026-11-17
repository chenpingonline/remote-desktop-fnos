#!/usr/bin/env python3
"""Match the app's password-only VNC contract; do not advertise account auth."""
from pathlib import Path

source = Path('src/protocols/vnc/vnc.c')
text = source.read_text()
marker = '    rfb_client->GetCredential = guac_vnc_get_credentials;'
assert text.count(marker) == 1, 'Guacamole VNC credential hook changed'
text = text.replace(marker, '''    /* remote-desktop-fnos supports VNC passwords, not ARD/user accounts.
     * LibVNCClient skips account security types when GetCredential is NULL.
     * Otherwise macOS offers ARD first and Guacamole waits for a username
     * our password-only client cannot supply. Keep GetPassword below, and
     * let LibVNCClient negotiate supported password/TLS security normally. */
    rfb_client->GetCredential = NULL;''')
source.write_text(text)
