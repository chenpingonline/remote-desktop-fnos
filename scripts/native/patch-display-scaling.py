#!/usr/bin/env python3
"""Pass Guacamole DPI through to FreeRDP and preserve it on display updates.

Guacamole 1.6 uses DPI to choose pixel dimensions, but leaves Windows desktop
scale unset and sends zero scale factors in the display-control channel.
"""
from pathlib import Path
import sys

root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path('.')
settings = root / 'src/protocols/rdp/settings.c'
s = settings.read_text()
anchor = '    rdpSettings* rdp_settings = GUAC_RDP_CONTEXT(rdp)->settings;'
assert s.count(anchor) == 1
s = s.replace(anchor, anchor + '''

    /* Match Windows UI scaling to the negotiated display density. */
    UINT32 desktop_scale = (guac_settings->resolution * 100 + 48) / 96;
    if (desktop_scale < 100) desktop_scale = 100;
    if (desktop_scale > 200) desktop_scale = 200;
    UINT32 physical_width = guac_settings->width * 254 / (guac_settings->resolution * 10);
    UINT32 physical_height = guac_settings->height * 254 / (guac_settings->resolution * 10);
''')
anchor = '    freerdp_settings_set_uint32(rdp_settings, FreeRDP_DesktopHeight, guac_settings->height);'
assert s.count(anchor) == 1
s = s.replace(anchor, anchor + '''
    freerdp_settings_set_uint32(rdp_settings, FreeRDP_DesktopScaleFactor, desktop_scale);
    freerdp_settings_set_uint32(rdp_settings, FreeRDP_DeviceScaleFactor, 100);
    freerdp_settings_set_uint32(rdp_settings, FreeRDP_DesktopPhysicalWidth, physical_width);
    freerdp_settings_set_uint32(rdp_settings, FreeRDP_DesktopPhysicalHeight, physical_height);''')
anchor = '    rdp_settings->DesktopHeight = guac_settings->height;'
assert s.count(anchor) == 1
s = s.replace(anchor, anchor + '''
    rdp_settings->DesktopScaleFactor = desktop_scale;
    rdp_settings->DeviceScaleFactor = 100;
    rdp_settings->DesktopPhysicalWidth = physical_width;
    rdp_settings->DesktopPhysicalHeight = physical_height;''')
settings.write_text(s)

disp = root / 'src/protocols/rdp/channels/disp.c'
s = disp.read_text()
anchor = '        DISPLAY_CONTROL_MONITOR_LAYOUT monitors[1] = {{'
assert s.count(anchor) == 1
s = s.replace(anchor, '''        UINT32 desktop_scale = (settings->resolution * 100 + 48) / 96;
        if (desktop_scale < 100) desktop_scale = 100;
        if (desktop_scale > 200) desktop_scale = 200;
''' + anchor)
for old, new in [('.PhysicalWidth = 0', '.PhysicalWidth = width * 254 / (settings->resolution * 10)'), ('.PhysicalHeight = 0', '.PhysicalHeight = height * 254 / (settings->resolution * 10)'), ('.DesktopScaleFactor = 0', '.DesktopScaleFactor = desktop_scale'), ('.DeviceScaleFactor = 0', '.DeviceScaleFactor = 100')]:
    assert s.count(old) == 1
    s = s.replace(old, new)
disp.write_text(s)
