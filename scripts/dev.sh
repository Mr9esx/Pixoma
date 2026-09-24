#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -d web/admin/node_modules ]]; then
  pnpm --dir web/admin install
fi

# 控制面默认监听局域网，端口由 PORT 统一传入；HTTP_ADDR 可覆盖完整监听地址。
# PUBLIC_URL 未设置时自动使用本机局域网 IP（或回环地址）和同一端口。
LAN_IP="$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || true)"
export HTTP_ADDR="${HTTP_ADDR:-0.0.0.0:${PORT:-30808}}"
HTTP_PORT="${HTTP_ADDR##*:}"
export PUBLIC_URL="${PUBLIC_URL:-http://${LAN_IP:-127.0.0.1}:$HTTP_PORT}"

echo "管理页面: http://127.0.0.1:5173"
echo "后台接口: $PUBLIC_URL"

start_pixoma() {
  go run ./apps/pixoma/cmd/pixoma &
  pixoma_pid=$!
}

start_pixoma

vite_pid=""
stopping=0
cleanup() {
  stopping=1
  trap - INT TERM EXIT
  kill "$pixoma_pid" 2>/dev/null || true
  if [[ -n "$vite_pid" ]]; then
    kill "$vite_pid" 2>/dev/null || true
    wait "$vite_pid" 2>/dev/null || true
  fi
  wait "$pixoma_pid" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

until curl -fsS --max-time 1 "http://127.0.0.1:${HTTP_PORT}/healthz" >/dev/null 2>&1; do
  if ! kill -0 "$pixoma_pid" 2>/dev/null; then
    echo "pixoma 启动失败" >&2
    exit 1
  fi
  sleep 0.2
done

export VITE_ADMIN_API_BASE=
pnpm --dir web/admin dev &
vite_pid=$!

while kill -0 "$vite_pid" 2>/dev/null; do
  if [[ "$stopping" -eq 1 ]]; then
    break
  fi
  if ! kill -0 "$pixoma_pid" 2>/dev/null; then
    echo "pixoma 退出了，正在重新拉起控制面…"
    start_pixoma
  fi
  wait -n "$pixoma_pid" "$vite_pid" 2>/dev/null || true
done
