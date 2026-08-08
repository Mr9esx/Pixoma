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
