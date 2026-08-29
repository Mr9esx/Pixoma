# Admin Web Console

Pixoma 管理控制台。开发时由 Vite 把 `/api` 转到本机 pixoma（`:8080`）。

## 前置

1. Node 20+，启用 pnpm（`corepack enable`）
2. 仓库根目录能跑 `pixoma`（见根 README「本地调试」）

## 安装与启动

日常在仓库根目录：

```bash
make dev
```

浏览器打开日志里的 **管理页面**（`http://127.0.0.1:5173`）。不必设置 `VITE_ADMIN_API_BASE`，也不必先 `cp .env.example`。

只起前端时：

```bash
cd web/admin
pnpm install   # 首次
pnpm dev
```

此时仍需另开 pixoma（默认 `:8080`）。`pnpm dev` 会把 `/api` 代理到 `http://127.0.0.1:8080`。

## 联调

根目录 `make dev`：后端和页面一起起，Ctrl-C 一起停。改页面会热更新；改 Go 再跑一次 `make dev`。

DevTools Network：请求路径是 `/api/v1/...`（同源 5173，由 Vite 转发）。

Dashboard 数字基于 list 拉取样本，受 `limit` 限制，不是全库精确统计。

## UI 设计原则

左栏列表筛选（搜索合并、横向 Segment 等）见仓库  
[docs/frontend/admin-list-filters.md](../../docs/frontend/admin-list-filters.md)。新资源页 MUST 遵循该文档。

## 非目标

无前端 mock。侧栏「消息平台」页管理平台接入（`/api/v1/channels`），消息平台详情内配置菜单（`/api/v1/channels/{id}/menu`）。
