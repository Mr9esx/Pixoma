# Brainstorm Summary

- Change: admin-resources-api
- Date: 2026-08-08

## 确认的技术方案

- **HTTP 适配器**：`internal/httpapi/{cases,users,sessions,tasks}`，风格对齐 `comfyinstances`；`apps/admin-api` 组装依赖并挂载。
- **宿主整理**：admin-api 显式拆 router / middleware；bot 轻量抽出 `apps/bot/internal/server`（仅 healthz + 中间件，不改 TG、不搬回管理 API）。
- **Case**：CRUD + `POST .../disable` + `POST .../enable`；仓储补 `Enable`。
- **User**：仅 `GET` 列表/详情，无写接口。
- **Session**：仅只读列表/详情。
- **Task**：列表/详情 + `POST /api/v1/tasks/{id}/cancel`（复用 `RequestCancel`）。
- **列表过滤**：尽量全但仍为固定查询参数——各资源字段过滤 + `q` + `created_from`/`created_to` + `limit`/`offset`；无动态查询语言。
- **边界**：无鉴权；禁止 `httpapi` → `channel/tg`；无 ConfirmRun；Session 无通用 Update。

## 关键取舍与风险

- 过滤做「尽量全」→ User/Session 需新 List，用显式 `ListQuery`，禁止任意字段名拼接。
- 取消竞态 → 只走领域规则；不可取消返回明确错误。
- Case 大 JSON → 校验失败可读；深度 UX 留给前端 change。
- bot server 抽出 → 行为不变，仅结构对齐。

## 测试策略

- 各 httpapi handler 表驱动：过滤、404、校验失败、enable/disable、cancel 成功/失败。
- 仓储 List：关键字/时间/多字段主路径。
- admin-api / bot server：healthz 与挂载冒烟；bot 确认无管理 CRUD 路由。
- README curl 四类资源主路径。

## Spec Patch

- `case-admin-api`：重新启用；列表过滤场景。
- `user-admin-api`：本期无写接口；列表过滤。
- `session-admin-api`：列表过滤明确化。
- `task-admin-api`：取消路径语义；列表过滤；无 ConfirmRun 保持。
