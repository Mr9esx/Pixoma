# Admin Web Console

Pixoma 管理控制台。只通过 `apps/admin-api` 取数；**本期无鉴权**，勿对公网暴露。

## 前置

1. Node 20+，启用 pnpm（`corepack enable`）
2. 与 bot 共用同一库时，先准备好 `configs/admin-api.yaml`（`make run-admin` 缺省会从 example 复制）

## 安装与启动

```bash
cd web/admin
cp .env.example .env.development
pnpm install
pnpm dev
```

环境变量：`VITE_ADMIN_API_BASE`（例：`http://127.0.0.1:8081`）

## 联调

推荐在仓库根：

```bash
cp web/admin/.env.example web/admin/.env.development   # 首次
make run-admin    # admin-api + 本前端；Ctrl-C 一起停
# 或全家桶：make run-all
```

也可拆开：`make run-admin-api` + `cd web/admin && pnpm dev`。

浏览器打开 Vite URL；根路径应为 Dashboard。DevTools Network：请求前缀为 `VITE_ADMIN_API_BASE` + `/api/v1/...`。

Dashboard 数字基于 list 拉取样本，受 `limit` 限制，不是全库精确统计。

## 非目标

无登录鉴权；无前端 mock。侧栏「主键盘」页管理机器人底部按钮（`/api/v1/tg-menu`）。
