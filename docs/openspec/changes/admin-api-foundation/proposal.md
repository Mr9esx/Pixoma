## Why

管理能力目前临时挂在 bot 同进程 HTTP 上，且 `apps/admin-api` 仅有占位目录，无法作为独立管理入口。需要先落地可运行的 admin-api 骨架，并把已有实例管理接口从 bot 迁出，为后续资源 API 与 `web/admin` 提供稳定后端边界。

## What Changes

- 新建可独立启动的 `apps/admin-api`（配置、wire、health、CORS；本期无鉴权）
- 将 bot 上的 `/api/v1/comfy-instances*` 管理/观测 HTTP **迁移**到 admin-api（复用现有 handler/应用能力，禁止依赖 `channel/tg`）
- **BREAKING**：bot 进程不再挂载管理 CRUD/观测路由；运维改连 admin-api
- 明确 admin-api 与 bot 共用同一持久化（DB/配置约定），实例池刷新语义与现有行为对齐
- 文档与示例配置补充 admin-api 启动与 curl 用法

## Capabilities

### New Capabilities

- `admin-api-host`: 独立 admin-api 进程的启动、健康检查、CORS、无鉴权管理入口与 bot 职责边界
- `comfy-instance-admin-api`: Comfy 实例在 admin-api 上的 CRUD 与 system/queue/tasks 观测 HTTP

### Modified Capabilities

- （无）主规格中实例管理行为不改语义，仅变更承载进程；若后续发现既有 spec 写死“挂在 bot”，再补 delta

## Impact

- 代码：`apps/admin-api/`、`apps/bot/cmd/comfyui-bot`（卸下管理路由）、`internal/httpapi/comfyinstances`（挂载点迁移）、`configs/`（admin-api 配置）
- API：管理客户端改连 admin-api 基址；bot HTTP 仅保留非管理用途（如 `/healthz`，若仍需要）
- 依赖：与 bot 共享 DB / 实例仓储；不引入鉴权依赖
- 后续 change：`admin-resources-api`、`admin-web-console` 依赖本 change
