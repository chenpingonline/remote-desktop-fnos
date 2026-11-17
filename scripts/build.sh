#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(sed -n 's/^version=//p' packaging/remote-desktop/manifest)
mkdir -p build/vendor dist
python3 scripts/vendor-client.py
npm --prefix web ci
npm --prefix web run build
node --experimental-strip-types --test web/unit/*.test.ts
go test -race ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=$version" -o build/remote-desktop-linux-amd64 ./cmd/server
archive=build/vendor/guacamole-server-1.6.0.tar.gz
if [ ! -f "$archive" ]; then curl -fL 'https://archive.apache.org/dist/guacamole/1.6.0/source/guacamole-server-1.6.0.tar.gz' -o "$archive"; fi
python3 - "$archive" <<'PY'
import hashlib,sys
assert hashlib.sha256(open(sys.argv[1],'rb').read()).hexdigest()=='8bc45675da96d7b6f39728160181e3d4ff3c08f460f6d26de5805b642bf13f2b','native source hash mismatch'
PY
cp "$archive" scripts/native/
docker build --platform linux/amd64 -t remote-desktop-native:1.6.0 scripts/native
container=$(docker create --platform linux/amd64 remote-desktop-native:1.6.0)
trap 'docker rm "$container" >/dev/null' EXIT
rm -rf build/native
mkdir -p build/native
docker cp "$container:/out/." build/native/
target=packaging/remote-desktop/app
rm -rf "$target/lib" "$target/ca-certificates" "$target/licenses" "$target/web"
mkdir -p "$target/bin" "$target/lib" "$target/web" "$target/licenses"
cp build/remote-desktop-linux-amd64 "$target/bin/remote-desktop"
cp -a build/native/bin/. "$target/bin/"
cp -aL build/native/lib/. "$target/lib/"
cp -a build/native/ca-certificates "$target/"
cp -aL build/native/licenses/. "$target/licenses/"
cp build/native/dependencies.tsv "$target/"
cp -a web/dist/. "$target/web/"
cp web/src/vendor/LICENSE "$target/licenses/guacamole-common-js-LICENSE"
cp web/src/vendor/NOTICE "$target/licenses/guacamole-common-js-NOTICE"
python3 scripts/collect-licenses.py
go version -m build/remote-desktop-linux-amd64 > "$target/go-dependencies.txt"
cp doc/fnos_remote_desktop_development_v1.0.md docs/COMPATIBILITY.md "$target/"
python3 scripts/validate-package-tree.py
(cd dist && fnpack build --directory ../packaging/remote-desktop)
mv dist/remote-desktop.fpk "dist/remote-desktop-${version}-x86.fpk"
python3 - "dist/remote-desktop-${version}-x86.fpk" <<'PY'
import hashlib,pathlib,sys
p=pathlib.Path(sys.argv[1]);p.with_suffix(p.suffix+'.sha256').write_text(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n')
PY
