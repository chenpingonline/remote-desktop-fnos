#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
RD_DEV_TOKEN=${RD_DEV_TOKEN:-}
if [ "${#RD_DEV_TOKEN}" -lt 24 ]; then echo '请先设置至少 24 字符的 RD_DEV_TOKEN 环境变量。' >&2; exit 1; fi
version=$(sed -n 's/^version=//p' packaging/remote-desktop/manifest)
mkdir -p build
npm --prefix web ci
npm --prefix web run build
go build -ldflags "-X main.version=$version" -o build/remote-desktop ./cmd/server
exec ./build/remote-desktop --dev "$@"
