#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -d web/admin/node_modules ]]; then
  pnpm --dir web/admin install
fi

# 控制面默认监听局域网，供远程 Edge 主动连入（HTTP_ADDR 可覆盖）。
# PUBLIC_URL 仅用于启动横幅展示；未设置时自动取本机局域网 IP。
LAN_IP="$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || true)"
export HTTP_ADDR="${HTTP_ADDR:-0.0.0.0:8082}"
if [[ -n "$LAN_IP" ]]; then
  export PUBLIC_URL="${PUBLIC_URL:-http://$LAN_IP:8082}"
fi

echo "管理页面: http://127.0.0.1:5173"
echo "后台接口: ${PUBLIC_URL:-http://127.0.0.1:8082}"

go run ./apps/pixoma/cmd/pixoma &
pixoma_pid=$!

export VITE_ADMIN_API_BASE=
pnpm --dir web/admin dev &
vite_pid=$!

cleanup() {
  trap - INT TERM EXIT
  kill "$pixoma_pid" "$vite_pid" 2>/dev/null || true
  wait "$pixoma_pid" "$vite_pid" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

wait
