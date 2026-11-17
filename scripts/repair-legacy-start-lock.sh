#!/bin/bash
# One-time repair for remote-desktop 0.1.1. No data or configuration is removed.
set -euo pipefail
appbase=/var/apps/remote-desktop
target=$(readlink -f "$appbase/target")
vardir=$(readlink -f "$appbase/var")
expected=$(readlink -f "$target/bin/remote-desktop")
read -r pid stamp < "$vardir/server.pid"
case "$pid" in ''|*[!0-9]*) echo '拒绝操作：PID 文件无效。' >&2; exit 1;; esac
[ "$pid" -gt 1 ]
[ "$(readlink "/proc/$pid/exe")" = "$expected" ] || { echo '拒绝操作：进程并非本应用。' >&2; exit 1; }
[ "$(awk '{print $22}' "/proc/$pid/stat")" = "$stamp" ] || { echo '拒绝操作：PID 已被复用。' >&2; exit 1; }
[ "$(readlink "/proc/$pid/fd/9")" = "$vardir/lifecycle.lock" ] || { echo '拒绝操作：未检测到旧版启动锁遗留。' >&2; exit 1; }
kill -TERM "$pid"
for ((i=0;i<12;i++)); do
 if [ "$(readlink "/proc/$pid/exe" 2>/dev/null || true)" != "$expected" ]; then
  echo '已停止经过身份核对的旧版远程桌面进程，数据与密钥已保留。现在可以原位升级。'
  exit 0
 fi
 sleep 1
done
echo '旧版进程尚未退出，未执行强制终止。' >&2
exit 1
