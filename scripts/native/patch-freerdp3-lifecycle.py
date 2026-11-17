#!/usr/bin/env python3
"""Backport Guacamole's GDI PostConnect lifecycle fix for FreeRDP >= 3.31.

Reference: https://github.com/apache/guacamole-server/blob/bb1ba8ed1414ec6a239cea265d4e9c7e23b9c6a3/src/protocols/rdp/rdp.c
Move the existing 1.6.0 rendering setup intact, preserving our other patches.
FreeRDP creates its cache after PreConnect; initializing GDI there aborts.
"""
from pathlib import Path

p = Path('src/protocols/rdp/rdp.c')
s = p.read_text()
start = s.index('static BOOL rdp_freerdp_pre_connect(freerdp* instance) {')
end = s.index('\n}\n', start) + len('\n}\n')
original = s[start:end]
first = original.index('    /* Init FreeRDP internal GDI implementation */')
last_line = '    GUAC_RDP_CONTEXT(instance)->update->altsec->FrameMarker = guac_rdp_gdi_frame_marker;\n\n'
assert original.count(last_line) == 1
last = original.index(last_line) + len(last_line)
rendering = original[first:last]
graphics = '    rdpGraphics* graphics = context->graphics;\n'
assert original.count(graphics) == 1
pre = (original[:first] + original[last:]).replace(graphics, '')
post = '''
/* FreeRDP >= 3.31 creates the cache during connection activation. Initialize
 * GDI and install our update handlers afterwards, as in upstream Guacamole. */
static BOOL rdp_freerdp_post_connect(freerdp* instance) {
    rdpContext* context = GUAC_RDP_CONTEXT(instance);
    rdpGraphics* graphics = context->graphics;

''' + rendering + '''    return TRUE;
}
'''
s = s[:start] + pre + post + s[end:]
anchor = '    rdp_inst->PreConnect = rdp_freerdp_pre_connect;'
assert s.count(anchor) == 1
s = s.replace(anchor, anchor + '\n    rdp_inst->PostConnect = rdp_freerdp_post_connect;')
p.write_text(s)
