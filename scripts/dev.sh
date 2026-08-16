#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -d web/admin/node_modules ]]; then
  pnpm --dir web/admin install
fi

echo "管理页面: http://127.0.0.1:5173"
echo "后台接口: http://127.0.0.1:8080"

export COMFY_MOCK="${COMFY_MOCK:-0}"
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
