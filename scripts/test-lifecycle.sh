#!/bin/bash
# Linux-only lifecycle contract test. Requires an isolated writable test root,
# a Linux-native service binary and a test engine that owns its listener.
set -euo pipefail
: "${RD_LIFECYCLE_ROOT:?isolated test root required}" "${RD_LIFECYCLE_BIN:?native Linux service binary required}" "${RD_LIFECYCLE_ENGINE:?test engine required}" "${RD_LIFECYCLE_SOURCE:?package source required}"
mkdir -p "$RD_LIFECYCLE_ROOT/target/bin" "$RD_LIFECYCLE_ROOT/var" "$RD_LIFECYCLE_ROOT/etc"
cp "$RD_LIFECYCLE_BIN" "$RD_LIFECYCLE_ROOT/target/bin/remote-desktop"
cp "$RD_LIFECYCLE_ENGINE" "$RD_LIFECYCLE_ROOT/target/bin/guacd"
# Lifecycle tests stub crypto; the packaged native implementation has its own known-answer test.
printf '#!/bin/sh\nexit 0\n' > "$RD_LIFECYCLE_ROOT/target/bin/nla-crypto-check"
chmod 755 "$RD_LIFECYCLE_ROOT/target/bin/nla-crypto-check"
cp -a "$RD_LIFECYCLE_SOURCE/cmd" "$RD_LIFECYCLE_ROOT/cmd"
ln -s "$RD_LIFECYCLE_ROOT/target" "$RD_LIFECYCLE_ROOT/target-link"
export TRIM_APPDEST="$RD_LIFECYCLE_ROOT/target-link" TRIM_PKGVAR="$RD_LIFECYCLE_ROOT/var" TRIM_PKGETC="$RD_LIFECYCLE_ROOT/etc" TRIM_TEMP_LOGFILE="$RD_LIFECYCLE_ROOT/error.log"
unset wizard_public_origin wizard_gateway_gid || true
main="$RD_LIFECYCLE_ROOT/cmd/main"
cleanup() { "$main" stop >/dev/null 2>&1 || true; }
trap cleanup EXIT
"$RD_LIFECYCLE_ROOT/cmd/install_callback"
[ ! -e "$TRIM_PKGETC/public-origin" ] && [ ! -e "$TRIM_PKGETC/gateway-gid" ]
timeout 8 "$main" start
read -r first_pid first_stamp < "$TRIM_PKGVAR/server.pid"
[ "$(readlink "/proc/$first_pid/fd/9" 2>/dev/null || true)" != "$TRIM_PKGVAR/lifecycle.lock" ] || { echo 'Service inherited lifecycle lock' >&2; exit 1; }
timeout 3 "$main" status
timeout 3 "$main" start
read -r second_pid second_stamp < "$TRIM_PKGVAR/server.pid"
[ "$first_pid $first_stamp" = "$second_pid $second_stamp" ]
echo 'Start, immediate status, repeated start and closed lock descriptor: PASS'
timeout 8 "$main" stop
timeout 3 "$main" stop
result=0; timeout 3 "$main" status || result=$?
[ "$result" = 3 ]
echo 'Stop, repeated stop and stopped status=3: PASS'
# In the minimal runtime image, curl is deliberately unavailable.
[ -z "$(command -v curl || true)" ]
python3 - "$TRIM_APPDEST/app.sock" <<'PY'
import socket,sys
s=socket.socket(socket.AF_UNIX);s.bind(sys.argv[1]);s.close()
PY
timeout 8 "$main" start
timeout 3 "$main" status
timeout 8 "$main" stop
echo 'Stale socket recovery without curl: PASS'
# Reproduce the old inherited descriptor without touching unrelated processes.
(
 exec 9>"$TRIM_PKGVAR/lifecycle.lock"
 flock 9
 "$TRIM_APPDEST/bin/remote-desktop" --data "$TRIM_PKGVAR" --key "$TRIM_PKGETC/master.key" --web "$TRIM_APPDEST/web" --origin https://nas.test --socket "$TRIM_APPDEST/app.sock" --guacd 127.0.0.1:14822 --engine "$TRIM_APPDEST/bin/guacd" </dev/null >> "$TRIM_PKGVAR/server.log" 2>&1 &
 legacy_pid=$!
 legacy_stamp=$(awk '{print $22}' "/proc/$legacy_pid/stat")
 printf '%s %s\n' "$legacy_pid" "$legacy_stamp" > "$TRIM_PKGVAR/server.pid"
)
for ((i=0;i<10;i++)); do
 if "$TRIM_APPDEST/bin/remote-desktop" --healthcheck "$TRIM_APPDEST/app.sock" >/dev/null 2>&1; then break; fi
 sleep 0.2
done
read -r legacy_pid legacy_stamp < "$TRIM_PKGVAR/server.pid"
[ "$(readlink "/proc/$legacy_pid/fd/9")" = "$TRIM_PKGVAR/lifecycle.lock" ]
timeout 8 "$main" stop
echo 'Verified legacy service lock recovery on stop: PASS'
# Legacy parameters must no longer influence startup after an upgrade.
printf 'invalid-legacy-origin' > "$TRIM_PKGETC/public-origin"
printf 'invalid-legacy-gid' > "$TRIM_PKGETC/gateway-gid"
timeout 8 "$main" start
timeout 3 "$main" status
timeout 8 "$main" stop
echo 'Unattended fresh startup and ignored legacy deployment parameters: PASS'

# Use the wizard values fnOS would submit, and exercise uninstall without a UI value.
uninstall="$RD_LIFECYCLE_ROOT/cmd/uninstall_init"
install="$RD_LIFECYCLE_ROOT/cmd/install_callback"
choices=$(python3 - "$RD_LIFECYCLE_SOURCE/wizard/uninstall" <<'PY'
import json, sys
field, = json.load(open(sys.argv[1]))[0]['items']
assert field['type'] == 'radio' and field['initValue'] == 'false'
assert [o['value'] for o in field['options']] == ['false', 'true']
print(field['field'], *(o['value'] for o in field['options']))
PY
)
read -r field keep purge <<< "$choices"
unset "$field"
cp "$TRIM_PKGETC/master.key" "$RD_LIFECYCLE_ROOT/original.key"
db_checksum=$(sha256sum "$TRIM_PKGVAR/app.db")
mkdir -p "$TRIM_PKGVAR/backups/previous"
touch "$TRIM_PKGVAR/backups/previous/app.db"
ln -s "$RD_LIFECYCLE_ROOT/original.key" "$TRIM_PKGVAR/external-link"
for choice in default "$keep"; do
 if [ "$choice" = default ]; then "$uninstall"; else env "$field=$choice" "$uninstall"; fi
 # Model fnOS removing the configuration directory and remounting package data.
 rm -f "$TRIM_PKGETC/master.key"
 "$install"
 "$install"
 cmp "$RD_LIFECYCLE_ROOT/original.key" "$TRIM_PKGETC/master.key"
 [ "$(stat -c %a "$TRIM_PKGETC/master.key")" = 600 ]
 [ "$(sha256sum "$TRIM_PKGVAR/app.db")" = "$db_checksum" ]
 [ -f "$TRIM_PKGVAR/backups/previous/app.db" ]
done
env "$field=$keep" "$uninstall"
head -c 32 /dev/urandom > "$TRIM_PKGETC/master.key"
if "$install"; then echo 'Conflicting credential keys accepted' >&2; exit 1; fi
cmp "$RD_LIFECYCLE_ROOT/original.key" "$TRIM_PKGVAR/.uninstall-master.key"
rm "$TRIM_PKGETC/master.key"
if env "$field=$keep" "$uninstall"; then echo 'Missing credential key accepted' >&2; exit 1; fi
"$install"
cmp "$RD_LIFECYCLE_ROOT/original.key" "$TRIM_PKGETC/master.key"
timeout 8 "$main" start
timeout 3 "$main" status
echo 'Default and explicit retention, paired key restore and service restart: PASS'
env "$field=$purge" "$uninstall"
[ -z "$(find "$TRIM_PKGVAR" "$TRIM_PKGETC" -mindepth 1 -print -quit)" ]
[ -s "$RD_LIFECYCLE_ROOT/original.key" ]
echo 'Explicit purge removes private data and backups, preserving external files: PASS'
