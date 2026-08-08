# Admin API

独立管理 HTTP 进程：健康检查、CORS、Comfy 实例 CRUD/观测。

**本期无鉴权。** 只在本机或可信内网使用，不要对公网暴露。

## 运行

```bash
# 可选：复制示例配置（DSN 必须与 bot 同一库）
cp configs/admin-api.example.yaml configs/admin-api.yaml

make run-admin-api
# 或
go run ./apps/admin-api/cmd/admin-api
```

默认监听 `127.0.0.1:8081`。配置路径：`ADMIN_CONFIG` 或 `configs/admin-api.yaml`；`HTTP_ADDR`、`DATABASE_DSN`、`COMFY_MOCK` 可覆盖对应配置。

与 bot 共用同一 `database_dsn`（例如 `data/app.db`）。`comfy_mock` 与观测路径对齐 bot。

## 健康检查

```bash
curl -s localhost:8081/healthz
# ok
```

## 实例管理与观测

路径与迁出前一致：`/api/v1/comfy-instances*`（只是宿主换成 admin-api）。

```bash
curl -s localhost:8081/api/v1/comfy-instances

curl -s -X POST localhost:8081/api/v1/comfy-instances \
  -H 'Content-Type: application/json' \
  -d '{"id":"gpu-1","base_url":"http://127.0.0.1:8188","enabled":true}'

curl -s localhost:8081/api/v1/comfy-instances/gpu-1/system
curl -s localhost:8081/api/v1/comfy-instances/gpu-1/queue
curl -s 'localhost:8081/api/v1/comfy-instances/gpu-1/tasks?limit=20'
```

写成功后本进程会 `Pool.Refresh`。bot 侧名单同步依赖后续探活周期 Refresh（另一任务）。

## 与 bot 的边界

| 进程 | 端口（默认） | 职责 |
|---|---|---|
| bot | `:8080` | 对话 / 编排 / TG；仅保留 `/healthz` |
| admin-api | `127.0.0.1:8081` | 实例管理 HTTP（无鉴权） |

admin-api **不依赖** `channel/tg`。bot 上旧的 `/api/v1/comfy-instances*` 已卸下。

## 验收记录

2026-08-08 在 `feature/20260808/admin-api-foundation` 上完成：

```bash
go test ./apps/admin-api/... ./internal/httpapi/comfyinstances \
  ./apps/bot/cmd/comfyui-bot ./internal/platform/appboot \
  ./internal/platform/instance/...

go build -o "$TMPDIR/admin-api" ./apps/admin-api/cmd/admin-api
go build -o "$TMPDIR/comfyui-bot" ./apps/bot/cmd/comfyui-bot

HTTP_ADDR="127.0.0.1:<free-port>" \
DATABASE_DSN="file:<temp-dir>/admin.db?cache=shared&_pragma=foreign_keys(1)" \
COMFY_MOCK=1 "$TMPDIR/admin-api"

curl -fsS "http://127.0.0.1:<free-port>/healthz"
curl -fsS "http://127.0.0.1:<free-port>/api/v1/comfy-instances"
```

结果：聚焦测试和两个二进制构建通过；临时库首次列表为 `[]`，健康检查为 `ok`。创建 `smoke-gpu` 后，`system` 观测返回 `"mock": true`。源码检查确认 bot 主程序不再引用 `comfyinstances` 或挂载 `/api/v1/comfy-instances`。
