# Admin API

独立管理 HTTP 进程：健康检查、CORS、计算节点 CRUD/观测，以及 Case / User / Session / Task / TG Menu 管理接口。

**本期无鉴权。** 只在本机或可信内网使用，不要对公网暴露。默认监听 `127.0.0.1:8081`。

## 运行

新部署与日常调试请用仓库根目录 `make dev`（pixoma + 管理页面）。本进程仅过渡期保留。

```bash
# 可选：复制示例配置（DSN 必须与 bot 同一库）

make run-admin-api   # 只起独立 API（默认 127.0.0.1:8081）
# 或：go run ./apps/admin-api/cmd/admin-api
```

默认监听 `127.0.0.1:8081`。配置路径：`ADMIN_CONFIG` 或 `configs/admin-api.yaml`；`HTTP_ADDR`、`DATABASE_DSN`、`COMFY_MOCK` 可覆盖对应配置。

与 bot 共用同一 `database_dsn`（例如 `data/app.db`）。`comfy_mock` 与观测路径对齐 bot。

## 健康检查

```bash
curl -s localhost:8081/healthz
# ok
```

## 计算节点管理与观测

路径：`/api/v1/edges*`。

```bash
curl -s localhost:8081/api/v1/edges

curl -s -X POST localhost:8081/api/v1/edges \
  -H 'Content-Type: application/json' \
  -d '{"name":"gpu-1","base_url":"http://127.0.0.1:8188","enabled":true}'

curl -s localhost:8081/api/v1/edges/gpu-1/system
curl -s localhost:8081/api/v1/edges/gpu-1/queue
curl -s 'localhost:8081/api/v1/edges/gpu-1/tasks?limit=20'
curl -s localhost:8081/api/v1/edges/gpu-1/stats
```

写成功后本进程会 `Pool.Refresh`。bot 侧名单同步依赖后续探活周期 Refresh。

## Case / User / Session / Task

统一前缀 `/api/v1`；列表通用查询：`q`、`created_from`/`created_to`（RFC3339）、`limit`/`offset`。

```bash
# Case（可写 + 上下架）
curl -s 'localhost:8081/api/v1/cases?limit=20&enabled=true'
curl -s localhost:8081/api/v1/cases/<id>
curl -s -X POST localhost:8081/api/v1/cases/<id>/disable
curl -s -X POST localhost:8081/api/v1/cases/<id>/enable

# User（只读；用户仍由 bot/TG upsert）
curl -s 'localhost:8081/api/v1/users?tg_user_id=123&limit=20'
curl -s localhost:8081/api/v1/users/<id>

# Session（只读排障；无通用 Update）
curl -s 'localhost:8081/api/v1/sessions?user_id=<uid>&status=collecting'
curl -s localhost:8081/api/v1/sessions/<id>

# Task（列表/详情 + 取消；无代用户 ConfirmRun）
curl -s 'localhost:8081/api/v1/tasks?status=pending&limit=20'
curl -s localhost:8081/api/v1/tasks/<id>
curl -s -X POST localhost:8081/api/v1/tasks/<id>/cancel
```

不可取消的任务返回明确错误（HTTP 409）；缺失资源为 404。

## 渠道与渠道菜单

渠道管理（`/api/v1/channels`）：创建渠道（平台 + 名称 + Bot Token，凭证加密存储、回显掩码）、列表、启停、受限删除；菜单按渠道作用域读写（`/api/v1/channels/{id}/menu`），平台差异数据走 extras（`/api/v1/channels/{id}/menu/extras`）。

```bash
# 创建 Telegram 渠道
curl -s -X POST localhost:8081/api/v1/channels \
  -H 'Content-Type: application/json' \
  -d '{"platform":"telegram","name":"主机器人","token":"<bot-token>"}'

# 读写某渠道菜单（整树替换；kind: folder / open_case / placeholder / reply_media）
curl -s localhost:8081/api/v1/channels/<channel-id>/menu
curl -s -X PUT localhost:8081/api/v1/channels/<channel-id>/menu \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"id":"btn-image","label":"🖼 图片","order":0,"enabled":true,"kind":"folder","intro_text":"点模板先看预览图","case_ids":["<case-id>"]}]}'

# Case 在菜单中的挂载路径
curl -s localhost:8081/api/v1/cases/<case-id>/menu-placements
```

## 与 bot 的边界

| 进程 | 端口（默认） | 职责 |
|---|---|---|
| bot | `:8080` | 对话 / 编排 / TG；仅保留 `/healthz` |
| admin-api | `127.0.0.1:8081` | 计算节点 + Case/User/Session/Task/TG Menu 管理 HTTP（无鉴权） |

admin-api **不依赖** `channel/tg`。bot 上旧的管理路径已卸下。

## 验收记录

2026-08-08 `admin-resources-api`：四类资源路由挂载与聚焦测试通过；bot 仅 healthz。

```bash
go test ./apps/admin-api/... ./apps/bot/internal/server/ \
  ./internal/httpapi/... \
  ./internal/catalog/... ./internal/identity/... \
  ./internal/conversation/... ./internal/runtime/... \
  ./internal/platform/notify/ -count=1
```
