---
comet_change: admin-resources-api
role: technical-design
canonical_spec: openspec
---

# admin-resources-api 技术设计

## 1. 目标与边界

在已落地的 `apps/admin-api` 上补齐 Case / User / Session / Task 管理 HTTP，供后续 `web/admin` 对接。顺带把 admin-api 与 bot 的 HTTP 宿主目录拆清（router / middleware），bot 仅保留 healthz 形态。

非目标：鉴权、管理前端、代用户 ConfirmRun、Session 通用写、多租户、任意动态查询语言。

OpenSpec 能力（canonical）：`case-admin-api`、`user-admin-api`、`session-admin-api`、`task-admin-api`。

## 2. 架构

```
浏览器 / web/admin
        │
        ▼
apps/admin-api（组合根）
  server/router + middleware + healthz
        │ Mount
        ▼
internal/httpapi/{cases,users,sessions,tasks}
        │
        ├─ catalog（Case CRUD / Enable / Disable / List）
        ├─ identity（User Get / List）
        ├─ conversation（Session Get / List）
        └─ runtime（Task Get / List + orchestrator.RequestCancel）
```

- Handler 放在 `internal/httpapi`，与 `comfyinstances` 同模式；**禁止**依赖 `channel/tg`。
- `apps/admin-api/cmd/.../main.go` 只负责配置、DB（`appboot`）、组装依赖、Listen。
- bot：从 `main` 抽出 `apps/bot/internal/server`（chi + 既有中间件 + `/healthz`）；**不**恢复管理 CRUD 路由。

## 3. HTTP 契约

统一前缀：`/api/v1`。分页：`limit` / `offset`。错误 JSON 风格对齐 `comfyinstances`。

| 资源 | 路由 |
|---|---|
| Case | `GET/POST /cases`；`GET/PATCH /cases/{id}`；`POST /cases/{id}/disable`；`POST /cases/{id}/enable` |
| User | `GET /users`；`GET /users/{id}` |
| Session | `GET /sessions`；`GET /sessions/{id}` |
| Task | `GET /tasks`；`GET /tasks/{id}`；`POST /tasks/{id}/cancel` |

### 列表过滤（固定参数，尽量全）

公共：`q`（模糊关键字）、`created_from`、`created_to`、`limit`、`offset`。

| 资源 | 额外过滤 |
|---|---|
| Case | `enabled`、`category`、`tag`、`menu_key` |
| User | `tg_user_id`（精确）；其它可观察标识进 `q` |
| Session | `user_id`、`chat_id`、`status` |
| Task | `status`、`instance_id`、`chat_id`、`session_id`、`case_id` |

## 4. 领域 / 仓储改动

| 上下文 | 改动 |
|---|---|
| catalog | 确保 Create/Update/校验可被 admin 调用；补 `Enable`；扩展 `ListQuery`（关键字/时间等） |
| identity | 补管理用 `List`（现仅 Upsert/GetByID） |
| conversation | 补管理用 `List`（现偏 GetActiveByChat/GetByID） |
| runtime | 管理端 Task `List` 查询扩展；取消 **只** 调 `orchestrator.RequestCancel`，映射 `ErrCancelNotAllowed` 等为明确 HTTP 错误 |

不在 admin 侧发明第二套取消/态机规则。

## 5. 宿主整理（admin-api + bot）

**admin-api**

- `internal/server/middleware.go`：CORS 等
- `internal/server/router.go`（或等价拆分）：注册 healthz、挂载各 `httpapi` Mount
- 保留 `Options` 注入各 Handler

**bot**

- `apps/bot/internal/server`：抽出当前 chi 装配；行为与现网一致（几乎仅 healthz）
- TG 仍在 `channel/tg`，不改成 REST handler 树

## 6. 风险与边界条件

- 富过滤 → 显式 `ListQuery` 结构体，仓储内参数化查询，禁止把客户端键名直接拼进 SQL。
- Case 大文档更新 → 校验失败 4xx + 可读信息；不做半写入。
- 取消竞态 → 以领域状态为准；已终态/不可取消 → 明确失败。
- 无鉴权 → 文档强调仅本机/受信网络；默认绑定沿用 foundation（如 `127.0.0.1`）。

## 7. 测试策略

1. Handler 表驱动：过滤组合、404、校验失败、enable/disable、cancel 成功与失败。
2. 仓储 List：关键字、时间窗、多字段交集的主路径测试。
3. `apps/admin-api` server：四资源挂载 + healthz 冒烟。
4. `apps/bot` server：healthz；断言未挂载 `/api/v1/cases` 等管理 CRUD。
5. README curl：四类资源主路径示例与无鉴权警示。

## 8. 实现顺序建议

1. 仓储/应用补齐（List、Enable、Cancel 可调用）
2. 四个 httpapi 包 + admin-api 路由挂载
3. admin-api / bot server 目录整理
4. 测试 + README 验收
