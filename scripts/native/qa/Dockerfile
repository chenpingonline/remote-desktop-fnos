ARG NATIVE_IMAGE
FROM ${NATIVE_IMAGE}
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends xrdp xorgxrdp xserver-xorg-core xterm xauth dbus-x11 xfonts-base && rm -rf /var/lib/apt/lists/*
RUN apt-get update && apt-get install -y --no-install-recommends python3-pil && rm -rf /var/lib/apt/lists/*
RUN useradd --create-home --shell /bin/bash rdpqa && echo 'rdpqa:synthetic-rdp-qa-only' | chpasswd && \
    printf '#!/bin/sh\nexec xterm -geometry 80x24+20+20\n' > /home/rdpqa/.xsession && chmod 755 /home/rdpqa/.xsession && chown rdpqa:rdpqa /home/rdpqa/.xsession && \
    openssl req -x509 -newkey rsa:2048 -nodes -sha256 -days 1 -subj /CN=localhost -keyout /etc/xrdp/key.pem -out /etc/xrdp/cert.pem && \
    chmod 644 /etc/xrdp/key.pem && mkdir -p /run/xrdp /run/dbus
COPY test-full-rdp.py /tmp/test-full-rdp.py
CMD ["python3", "-u", "/tmp/test-full-rdp.py"]
