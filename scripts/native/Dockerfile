# Build-only container. The installed FPK runs native processes without Docker.
FROM debian:bookworm-slim@sha256:734a72b3d6142a71993b4bc580ca01d8ca687f77a2c422d96a961d3376b179bc
RUN apt-get update && apt-get install -y --no-install-recommends build-essential pkg-config ca-certificates libcairo2-dev libjpeg62-turbo-dev libpng-dev uuid-dev freerdp2-dev libvncserver-dev libssl-dev libwebp-dev libpango1.0-dev && rm -rf /var/lib/apt/lists/*
COPY guacamole-server-1.6.0.tar.gz /tmp/source.tar.gz
RUN echo '8bc45675da96d7b6f39728160181e3d4ff3c08f460f6d26de5805b642bf13f2b  /tmp/source.tar.gz' | sha256sum -c - && tar -xzf /tmp/source.tar.gz -C /tmp
WORKDIR /tmp/guacamole-server-1.6.0
# Keep native debug text off the host syslog; the owned supervisor filters stderr.
RUN ./configure --prefix=/opt/remote-desktop --with-freerdp-plugin-dir=/opt/remote-desktop/lib/freerdp2 --disable-guacenc --disable-guaclog --without-ssh --without-telnet --without-kubernetes && make -j4 && make install
RUN grep -Fq 'syslog(priority, "%s", message);' src/guacd/log.c && sed -i 's/syslog(priority, "%s", message);/\/\* Host syslog disabled: owned supervisor filters stderr. \*\/ (void)priority;/' src/guacd/log.c && make -C src/guacd -j4 && make -C src/guacd install
COPY patch-certificate-verifier.py /tmp/patch-certificate-verifier.py
RUN python3 /tmp/patch-certificate-verifier.py && make -C src/protocols/rdp -j4 && make -C src/protocols/rdp install
COPY patch-display-scaling.py /tmp/patch-display-scaling.py
RUN python3 /tmp/patch-display-scaling.py && make -C src/protocols/rdp -j4 && make -C src/protocols/rdp install
COPY patch-vnc-password-auth.py /tmp/patch-vnc-password-auth.py
RUN python3 /tmp/patch-vnc-password-auth.py && make -C src/protocols/vnc -j4 && make -C src/protocols/vnc install
RUN mkdir -p /out/bin /out/lib /out/licenses && cp /opt/remote-desktop/sbin/guacd /out/bin/ && cp -aL /opt/remote-desktop/lib/*.so* /out/lib/ && cp LICENSE /out/licenses/Guacamole-LICENSE && cp NOTICE /out/licenses/Guacamole-NOTICE && dpkg-query -W > /out/dependencies.tsv && \
 find /opt/remote-desktop -type f -name '*.so*' -o -name guacd | xargs ldd | awk '/=> \/(usr\/)?lib/{print $3}' | sort -u | while read lib; do case "$lib" in */libc.so.*|*/libm.so.*|*/libpthread.so.*|*/libdl.so.*|*/librt.so.*|*/ld-linux*) ;; *) cp -L "$lib" /out/lib/ ;; esac; done && \
 mkdir -p /out/licenses/debian-copyrights /out/licenses/common-licenses && \
 for copyright in /usr/share/doc/*/copyright; do package=$(basename "$(dirname "$copyright")"); install -D -m 0644 "$copyright" "/out/licenses/debian-copyrights/$package/copyright"; done && \
 cp -aL /usr/share/common-licenses/. /out/licenses/common-licenses/ && cp -aL /etc/ssl/certs /out/ca-certificates
COPY check-crypto.c /tmp/check-crypto.c
RUN mkdir -p /out/lib/ossl-modules && cp -L /usr/lib/$(dpkg-architecture -qDEB_HOST_MULTIARCH)/ossl-modules/legacy.so /out/lib/ossl-modules/ && \
 cc /tmp/check-crypto.c $(pkg-config --cflags --libs winpr2) -o /out/bin/nla-crypto-check && \
 LD_LIBRARY_PATH=/out/lib OPENSSL_MODULES=/out/lib/ossl-modules /out/bin/nla-crypto-check
COPY test-vnc-password-auth.py /tmp/test-vnc-password-auth.py
RUN python3 /tmp/test-vnc-password-auth.py
CMD ["/bin/true"]
