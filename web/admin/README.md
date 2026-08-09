# Admin Web Console

Pixoma 管理控制台。只通过 `apps/admin-api` 取数；**本期无鉴权**，勿对公网暴露。

## 前置

1. 先启动 admin-api（默认 `127.0.0.1:8081`），见 `apps/admin-api/README.md`
2. Node 20+，启用 pnpm（`corepack enable`）

## 安装与启动

```bash
cd web/admin
cp .env.example .env.development
pnpm install
pnpm dev
```

环境变量：`VITE_ADMIN_API_BASE`（例：`http://127.0.0.1:8081`）

## 联调

1. `make run-admin-api`（或 `go run ./apps/admin-api/cmd/admin-api`）
2. `cd web/admin && cp .env.example .env.development && pnpm install && pnpm dev`
3. 浏览器打开 Vite URL；根路径应为 Dashboard
4. DevTools Network：请求前缀为 `VITE_ADMIN_API_BASE` + `/api/v1/...`

Dashboard 数字基于 list 拉取样本，受 `limit` 限制，不是全库精确统计。

## 非目标

无登录鉴权；无前端 mock；无 TG/Menu 管理。
