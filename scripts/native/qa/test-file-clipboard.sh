#!/bin/sh
# Run in the built native image, from the configured Guacamole source tree.
set -eu
cc -I. -Isrc/protocols/rdp -Isrc/common -Isrc/libguac -ffunction-sections -fdata-sections \
    -Wno-deprecated-declarations /qa/test-file-clipboard.c src/common/.libs/libguac_common.a \
    -Wl,--gc-sections -L/opt/remote-desktop/lib -lguac -lpthread \
    $(pkg-config --cflags --libs freerdp-client3 freerdp3 winpr3) -o /tmp/test-file-clipboard
/tmp/test-file-clipboard
