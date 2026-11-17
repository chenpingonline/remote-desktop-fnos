#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(sed -n 's/^version=//p' packaging/remote-desktop/manifest)
arch=${1:-amd64}
case "$arch" in
  amd64|x86) arch=amd64; package_platform=x86 ;;
  arm64|arm) arch=arm64; package_platform=arm ;;
  *) echo "Usage: $0 [amd64|arm64]" >&2; exit 1 ;;
esac
mkdir -p build/vendor dist
manifest_backup=$(mktemp)
cp packaging/remote-desktop/manifest "$manifest_backup"
container=
cleanup() {
  if [ -n "$container" ]; then docker rm "$container" >/dev/null; fi
  cp "$manifest_backup" packaging/remote-desktop/manifest
  rm -f "$manifest_backup"
}
trap cleanup EXIT
python3 scripts/vendor-client.py
npm --prefix web ci
npm --prefix web run build
node --experimental-strip-types --test web/unit/*.test.ts
go test -race ./...
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags "-s -w -X main.version=$version" -o build/remote-desktop-linux-$arch ./cmd/server
archive=build/vendor/guacamole-server-1.6.0.tar.gz
if [ ! -f "$archive" ]; then curl -fL 'https://archive.apache.org/dist/guacamole/1.6.0/source/guacamole-server-1.6.0.tar.gz' -o "$archive"; fi
python3 - "$archive" <<'PY'
import hashlib,sys
assert hashlib.sha256(open(sys.argv[1],'rb').read()).hexdigest()=='8bc45675da96d7b6f39728160181e3d4ff3c08f460f6d26de5805b642bf13f2b','native source hash mismatch'
PY
cp "$archive" scripts/native/
python3 scripts/fetch-freerdp.py
freerdp_version=$(python3 -c 'import json; print(json.load(open("scripts/native/freerdp-source.json"))["version"])')
native_image="remote-desktop-native:guac1.6.0-freerdp${freerdp_version}-$arch"
docker build --platform "linux/$arch" -t "$native_image" scripts/native
container=$(docker create --platform "linux/$arch" "$native_image")
rm -rf build/native
mkdir -p build/native
docker cp "$container:/out/." build/native/
target=packaging/remote-desktop/app
rm -rf "$target/lib" "$target/ca-certificates" "$target/licenses" "$target/web"
mkdir -p "$target/bin" "$target/lib" "$target/web" "$target/licenses"
cp build/remote-desktop-linux-$arch "$target/bin/remote-desktop"
cp -a build/native/bin/. "$target/bin/"
cp -aL build/native/lib/. "$target/lib/"
cp -a build/native/ca-certificates "$target/"
cp -aL build/native/licenses/. "$target/licenses/"
cp build/native/dependencies.tsv "$target/"
cp -a web/dist/. "$target/web/"
cp web/src/vendor/LICENSE "$target/licenses/guacamole-common-js-LICENSE"
cp web/src/vendor/NOTICE "$target/licenses/guacamole-common-js-NOTICE"
python3 scripts/collect-licenses.py
go version -m build/remote-desktop-linux-$arch > "$target/go-dependencies.txt"
cp doc/fnos_remote_desktop_development_v1.0.md docs/COMPATIBILITY.md "$target/"
python3 scripts/validate-package-tree.py
python3 - "$arch" "$target" <<'PYARCH'
import pathlib, struct, sys
expected = {'amd64': 62, 'arm64': 183}[sys.argv[1]]
root = pathlib.Path(sys.argv[2])
count = 0
for folder in ('bin', 'lib'):
    for path in (root / folder).rglob('*'):
        if not path.is_file(): continue
        header = path.read_bytes()[:20]
        if header[:4] != b'\x7fELF': continue
        assert header[4:6] == b'\x02\x01', f'Unsupported ELF: {path}'
        assert struct.unpack('<H', header[18:20])[0] == expected, f'Wrong architecture: {path}'
        count += 1
assert count, 'No native binaries found'
print(f'Verified {count} ELF files for {sys.argv[1]}')
PYARCH
python3 - "$package_platform" <<'PYPLATFORM'
import pathlib,re,sys
p=pathlib.Path('packaging/remote-desktop/manifest')
p.write_text(re.sub(r'^platform=.*$', 'platform='+sys.argv[1], p.read_text(), flags=re.M))
PYPLATFORM
(cd dist && fnpack build --directory ../packaging/remote-desktop)
mv dist/remote-desktop.fpk "dist/remote-desktop-${version}-${package_platform}.fpk"
python3 - "dist/remote-desktop-${version}-${package_platform}.fpk" <<'PY'
import hashlib,pathlib,sys
p=pathlib.Path(sys.argv[1]);p.with_suffix(p.suffix+'.sha256').write_text(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n')
PY
