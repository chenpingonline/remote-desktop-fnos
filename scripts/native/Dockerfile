# Build-only container. The installed FPK runs native processes without Docker.
FROM debian:bookworm-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    sed -i 's|http://deb.debian.org|https://deb.debian.org|g' /etc/apt/sources.list.d/debian.sources
RUN apt-get update && apt-get install -y --no-install-recommends build-essential cmake ninja-build python3 pkg-config libcairo2-dev libjpeg62-turbo-dev libpng-dev uuid-dev libvncserver-dev libssl-dev libwebp-dev libpango1.0-dev libavcodec-dev libavutil-dev libswresample-dev libswscale-dev libjson-c-dev libkrb5-dev libopus-dev libusb-1.0-0-dev && rm -rf /var/lib/apt/lists/*
RUN apt-get update && apt-get install -y --no-install-recommends clang && rm -rf /var/lib/apt/lists/*
COPY freerdp-source.json /tmp/freerdp-source.json
COPY freerdp.tar.gz /tmp/freerdp.tar.gz
RUN python3 -c 'import hashlib,json; p="/tmp/freerdp.tar.gz"; assert hashlib.sha256(open(p,"rb").read()).hexdigest()==json.load(open("/tmp/freerdp-source.json"))["sha256"], "FreeRDP source hash mismatch"' && mkdir /tmp/freerdp && tar -xzf /tmp/freerdp.tar.gz --strip-components=1 -C /tmp/freerdp && find /tmp/freerdp -type f -exec touch {} +
# Keep the bookworm ABI; build only the libraries/channels used by guacd.
# Bound compiler memory for small Docker VMs and cross-architecture builds.
ARG TARGETARCH
RUN --mount=type=cache,target=/tmp/freerdp-build,id=remote-desktop-freerdp3-${TARGETARCH},sharing=locked \
    cmake -S /tmp/freerdp -B /tmp/freerdp-build -G Ninja \
    -DCMAKE_C_COMPILER=clang -DCMAKE_CXX_COMPILER=clang++ \
    -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/remote-desktop -DCMAKE_INSTALL_LIBDIR=lib \
    -DCMAKE_C_FLAGS_RELEASE="-O2 -DNDEBUG" \
    -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=OFF \
    -DWITH_CLIENT=OFF -DWITH_CLIENT_COMMON=ON -DWITH_CLIENT_CHANNELS=ON \
    -DWITH_SERVER=OFF -DWITH_SHADOW=OFF -DWITH_X11=OFF -DWITH_WAYLAND=OFF \
    -DWITH_CLIENT_SDL=OFF -DWITH_WINPR_TOOLS=OFF -DWITH_SAMPLE=OFF -DWITH_MANPAGES=OFF -DBUILD_TESTING=OFF \
    -DWITH_ALSA=OFF -DWITH_PULSE=OFF -DWITH_CUPS=OFF -DWITH_PCSC=OFF -DWITH_FUSE=OFF \
    -DWITH_FFMPEG=ON -DWITH_SWSCALE=ON -DWITH_OPUS=ON && \
    cmake --build /tmp/freerdp-build -j2 && cmake --install /tmp/freerdp-build
ENV PKG_CONFIG_PATH=/opt/remote-desktop/lib/pkgconfig
ENV LD_LIBRARY_PATH=/opt/remote-desktop/lib
COPY guacamole-server-1.6.0.tar.gz /tmp/source.tar.gz
RUN echo '8bc45675da96d7b6f39728160181e3d4ff3c08f460f6d26de5805b642bf13f2b  /tmp/source.tar.gz' | sha256sum -c - && tar -xzf /tmp/source.tar.gz -C /tmp
WORKDIR /tmp/guacamole-server-1.6.0
# Keep native debug text off the host syslog; the owned supervisor filters stderr.
# FreeRDP 3.33 headers warn on their own retained 3.x deprecated declarations.
# Keep other -Werror diagnostics; allow configure to detect the real 3.x APIs.
RUN CFLAGS="-O2 -Wno-error=deprecated-declarations" ./configure --prefix=/opt/remote-desktop --with-freerdp-plugin-dir=/opt/remote-desktop/lib/freerdp3 --disable-guacenc --disable-guaclog --without-ssh --without-telnet --without-kubernetes && grep -q '^RDP_LIBS = .*freerdp3' src/protocols/rdp/Makefile
RUN make -C src/libguac -j4 && make -C src/libguac install && make -C src/common -j4
RUN make -j4 && make install
RUN grep -Fq 'syslog(priority, "%s", message);' src/guacd/log.c && sed -i 's/syslog(priority, "%s", message);/\/\* Host syslog disabled: owned supervisor filters stderr. \*\/ (void)priority;/' src/guacd/log.c && make -C src/guacd -j4 && make -C src/guacd install
COPY patch-certificate-verifier.py /tmp/patch-certificate-verifier.py
RUN python3 /tmp/patch-certificate-verifier.py && make -C src/protocols/rdp -j4 && make -C src/protocols/rdp install
COPY patch-display-scaling.py /tmp/patch-display-scaling.py
RUN python3 /tmp/patch-display-scaling.py && make -C src/protocols/rdp -j4 && make -C src/protocols/rdp install
COPY patch-vnc-password-auth.py /tmp/patch-vnc-password-auth.py
RUN python3 /tmp/patch-vnc-password-auth.py && make -C src/protocols/vnc -j4 && make -C src/protocols/vnc install
COPY patch-freerdp3-lifecycle.py /tmp/patch-freerdp3-lifecycle.py
RUN python3 /tmp/patch-freerdp3-lifecycle.py && make -C src/protocols/rdp -j4 && make -C src/protocols/rdp install
COPY patch-nla-account-auth.py /tmp/patch-nla-account-auth.py
RUN python3 /tmp/patch-nla-account-auth.py && make -C src/protocols/rdp -j4 && make -C src/protocols/rdp install
RUN mkdir -p /out/bin /out/lib /out/licenses && cp /opt/remote-desktop/sbin/guacd /out/bin/ && cp -aL /opt/remote-desktop/lib/*.so* /out/lib/ && cp LICENSE /out/licenses/Guacamole-LICENSE && cp NOTICE /out/licenses/Guacamole-NOTICE && dpkg-query -W > /out/dependencies.tsv && \
 find /opt/remote-desktop -type f -name '*.so*' -o -name guacd | xargs ldd | awk '/=> \/(usr\/)?lib/{print $3}' | sort -u | while read lib; do case "$lib" in */libc.so.*|*/libm.so.*|*/libpthread.so.*|*/libdl.so.*|*/librt.so.*|*/ld-linux*) ;; *) cp -L "$lib" /out/lib/ ;; esac; done && \
 mkdir -p /out/licenses/debian-copyrights /out/licenses/common-licenses && \
 for copyright in /usr/share/doc/*/copyright; do package=$(basename "$(dirname "$copyright")"); install -D -m 0644 "$copyright" "/out/licenses/debian-copyrights/$package/copyright"; done && \
 cp -aL /usr/share/common-licenses/. /out/licenses/common-licenses/ && cp -aL /etc/ssl/certs /out/ca-certificates
RUN cp /tmp/freerdp/LICENSE /out/licenses/FreeRDP-LICENSE && \
    python3 -c 'import json; d=json.load(open("/tmp/freerdp-source.json")); f=open("/out/dependencies.tsv","a"); f.write("freerdp-source\t"+d["version"]+"\nwinpr-source\t"+d["version"]+"\nfreerdp-source-sha256\t"+d["sha256"]+"\n")' && \
    cp /tmp/freerdp-source.json /out/licenses/FreeRDP-source.json
COPY check-crypto.c /tmp/check-crypto.c
RUN mkdir -p /out/lib/ossl-modules && cp -L /usr/lib/$(dpkg-architecture -qDEB_HOST_MULTIARCH)/ossl-modules/legacy.so /out/lib/ossl-modules/ && \
 cc /tmp/check-crypto.c $(pkg-config --cflags --libs winpr3) -o /out/bin/nla-crypto-check && \
 LD_LIBRARY_PATH=/out/lib OPENSSL_MODULES=/out/lib/ossl-modules /out/bin/nla-crypto-check
COPY test-vnc-password-auth.py /tmp/test-vnc-password-auth.py
RUN python3 /tmp/test-vnc-password-auth.py
COPY pin-rdp-certificate.c /tmp/pin-rdp-certificate.c
COPY test-certificate-pin.py /tmp/test-certificate-pin.py
COPY test-certificate-diagnostic.py /tmp/test-certificate-diagnostic.py
COPY test-runtime.py /tmp/test-runtime.py
RUN cc /tmp/pin-rdp-certificate.c $(pkg-config --cflags --libs freerdp3 winpr3 openssl) -o /pin && \
    LD_LIBRARY_PATH=/out/lib OPENSSL_MODULES=/out/lib/ossl-modules python3 /tmp/test-runtime.py && \
    python3 /tmp/test-certificate-pin.py && \
    for mode in missing correct changed legacy other-port trusted-mismatch; do \
        RD_TEST_PIN_MODE="$mode" RD_TEST_PIN_TOOL=/pin python3 /tmp/test-certificate-diagnostic.py || exit 1; \
    done
CMD ["/bin/true"]
