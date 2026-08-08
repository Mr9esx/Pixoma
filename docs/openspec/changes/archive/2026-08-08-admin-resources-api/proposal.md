## Why

独立 admin-api 宿主与实例 API 落地后，运维仍无法通过 HTTP 管理 Case、User、Session、Task。需要在 admin-api 上补齐这些资源的管理接口，形成完整管理面后端，供后续 `web/admin` 对接。

## What Changes

- 在 admin-api 增加 Case 管理 API（创建/更新/列表/详情/禁用或上下架；复用 catalog 应用/仓储，禁止依赖 `channel/tg`）
- 增加 User 管理 API（列表/详情；写路径以运维可观察字段为准，不强行替代 TG upsert 主路径）
- 增加 Session 运维 API（列表/详情；以排障只读为主，避免破坏对话态机）
- 增加 Task 运维 API（列表/详情；支持取消等运营侧动作，不在 admin 侧发起“代用户 ConfirmRun”）
- 统一错误与分页/过滤约定（与 foundation 实例 API 风格对齐）
- 本期仍无鉴权

## Capabilities

### New Capabilities

- `case-admin-api`: Case/目录资源的管理 HTTP
- `user-admin-api`: 用户资源的查询与基础管理 HTTP
- `session-admin-api`: Session 运维查询 HTTP
- `task-admin-api`: Task 运维查询与取消等管理 HTTP

### Modified Capabilities

- （无）不修改 TG 对话主路径需求；若现有 `workflow-registry` / `dialog-session` / `task-orchestrator` 规格未覆盖 admin 面，以本 change 新 capability 表达

## Impact

- 代码：`apps/admin-api` 路由扩展；`internal/catalog` / `identity` / `conversation` / `runtime` 的 application 查询或命令（按需补薄应用服务）
- API：新增 `/api/v1/cases`、`/users`、`/sessions`、`/tasks`（最终路径在 design 对齐）
- 依赖：`admin-api-foundation` 已提供宿主与无鉴权约定
- 后续：`admin-web-console` 对接本 API
