# Comet Design Handoff

- Change: admin-resources-api
- Phase: design
- Mode: compact
- Context hash: b5488d2d3b8b3faac20118e9a753f22455c1f082628f6511632cc4625729bbf4

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/admin-resources-api/proposal.md

- Source: docs/openspec/changes/admin-resources-api/proposal.md
- Lines: 1-32
- SHA256: e461fdd0e2b521a2c261b65ab02dc9c153e09272091d6e062d36876231907914

```md
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

```

## docs/openspec/changes/admin-resources-api/design.md

- Source: docs/openspec/changes/admin-resources-api/design.md
- Lines: 1-39
- SHA256: 04bcf72968980aa4fbb219ddca0b1e750522fc888e4dc47cf986a2479c6d21b1

```md
## Context

参见 `proposal.md`。依赖 `admin-api-foundation` 的宿主、CORS、无鉴权与实例 API 先例。User 仓储目前偏 TG upsert；Session/Task 写路径敏感，管理面默认只读 + 有限运维动作。

## Goals / Non-Goals

**Goals:**
- Case 完整管理写路径（校验后持久化 + 禁用）
- User/Session 运维查询
- Task 查询 + 取消
- 路由与错误风格与实例 API 对齐

**Non-Goals:**
- 鉴权；前端；ConfirmRun；任意篡改 Session 态机；多租户

## Decisions

1. **资源路径前缀** `/api/v1/{cases|users|sessions|tasks}`，与实例 API 同版本前缀。  
2. **Case**：走 catalog Repository/校验；禁用用既有 `Disable` 语义。  
3. **User**：先 List/Get；若仓储缺 List，本 change 补仓储查询，不引入“管理创建用户为主路径”。  
4. **Session**：只读 List/Get；不提供通用 Update。  
5. **Task**：List/Get + Cancel（复用 runtime 取消应用服务，若已有）。  
6. **分页**：Limit/Offset 或等价查询参数，与现有 ListQuery 风格一致。

## Risks / Trade-offs

- [仓储缺 List 需扩展] → 任务中显式补齐，避免 handler 直查未导出方法。  
- [取消与编排竞态] → 复用领域取消规则，失败返回明确错误。  
- [Case 大 JSON 编辑易错] → 校验失败返回可读错误；深度编辑 UX 留给前端 change。

## Migration Plan

1. 在 admin-api 增加路由与测试。  
2. 文档补充 curl 示例。  
3. 无 bot 路由回迁需求。

## Open Questions

- User List 的排序/过滤字段最小集（实现期按表结构选定，不改变“可列表/详情”规格）。

```

## docs/openspec/changes/admin-resources-api/tasks.md

- Source: docs/openspec/changes/admin-resources-api/tasks.md
- Lines: 1-23
- SHA256: 396fc828b6ed10f8d1bf0c78056e9cb021da099dea6057cb80d144e29d5df31a

```md
## 1. 仓储与应用服务补齐

- [ ] 1.1 确认/补齐 User、Session 的 List/Get 仓储（含过滤 ListQuery）
- [ ] 1.2 确认 Task Cancel（`RequestCancel`）可被 admin-api 复用；扩展管理端 Task List
- [ ] 1.3 Case：管理命令/查询/校验可调用；补齐 Enable；扩展 List 过滤

## 2. HTTP 接口

- [ ] 2.1 实现 `/api/v1/cases` 列表/详情/创建/更新/禁用/启用
- [ ] 2.2 实现 `/api/v1/users` 列表/详情（只读）
- [ ] 2.3 实现 `/api/v1/sessions` 列表/详情（只读）
- [ ] 2.4 实现 `/api/v1/tasks` 列表/详情与 `POST .../cancel`
- [ ] 2.5 统一错误响应与分页/过滤参数风格，并补充 handler 测试

## 3. 宿主整理

- [ ] 3.1 admin-api：拆分 router / middleware，挂载四类 httpapi
- [ ] 3.2 bot：抽出 `apps/bot/internal/server`（healthz + 中间件），不恢复管理 CRUD

## 4. 文档与验收

- [ ] 4.1 更新 admin-api README：四类资源 curl 示例与无鉴权警示
- [ ] 4.2 本地联调：对四类资源走通主路径验收场景

```

## docs/openspec/changes/admin-resources-api/specs/case-admin-api/spec.md

- Source: docs/openspec/changes/admin-resources-api/specs/case-admin-api/spec.md
- Lines: 1-46
- SHA256: a71ce22df5463fb920fd25be7c020835fdec211dea1a3562f9dc8db26dad649e

```md
## Purpose

为管理后台提供 Case（工作流目录）资源的 HTTP 管理能力，含列表、详情与写操作。

## ADDED Requirements

### Requirement: Case 列表与详情
系统 MUST 通过 admin-api 提供 Case 列表与按 ID 详情查询。

#### Scenario: 列出 Case
- **WHEN** 客户端请求 Case 列表（可带过滤参数）
- **THEN** 返回匹配的 Case 集合及启用状态等关键字段

#### Scenario: 按过滤条件列出 Case
- **WHEN** 客户端请求 Case 列表并携带 `q`、时间范围或 `enabled`/`category`/`tag`/`menu_key` 等已支持过滤参数
- **THEN** 仅返回匹配条件的 Case，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Case 详情
- **WHEN** 客户端请求已存在 Case 的详情
- **THEN** 返回可用于管理编辑的 Case 表示（含协议/输入相关必要字段）

### Requirement: Case 创建与更新
系统 MUST 允许通过 admin-api 创建与更新 Case，并持久化；非法协议 MUST 被拒绝。

#### Scenario: 创建合法 Case
- **WHEN** 客户端提交通过校验的 Case 载荷
- **THEN** Case 被持久化且可在列表/详情中查到

#### Scenario: 更新 Case
- **WHEN** 客户端提交已存在 Case 的合法更新
- **THEN** 持久化反映更新内容

#### Scenario: 非法 Case 被拒绝
- **WHEN** 客户端提交未通过校验的 Case 载荷
- **THEN** 返回明确的校验错误且不写入无效数据

### Requirement: Case 禁用与重新启用
系统 MUST 支持禁用/下架 Case，使 bot 菜单主路径不再将其作为可用入口；亦 MUST 支持重新启用（与 catalog 启用语义对齐）。

#### Scenario: 禁用后列表可观察
- **WHEN** 运维禁用某 Case
- **THEN** 后续列表/详情反映禁用状态

#### Scenario: 重新启用后可观察
- **WHEN** 运维对已禁用 Case 发起重新启用
- **THEN** 后续列表/详情反映启用状态，且可作为可用入口（在启用语义下）

```

## docs/openspec/changes/admin-resources-api/specs/session-admin-api/spec.md

- Source: docs/openspec/changes/admin-resources-api/specs/session-admin-api/spec.md
- Lines: 1-27
- SHA256: 07c2a40575458d5a69fec7eeb0646a310072402f7cdc78dc9da7b5c0299bbfff

```md
## Purpose

为运维排障提供 Session 只读管理查询，避免通过 admin 破坏对话采集态机。

## ADDED Requirements

### Requirement: Session 列表与详情
系统 MUST 通过 admin-api 提供 Session 列表与详情查询。

#### Scenario: 列出 Session
- **WHEN** 客户端请求 Session 列表
- **THEN** 返回 Session 集合及状态等关键字段

#### Scenario: 按过滤条件列出 Session
- **WHEN** 客户端请求 Session 列表并携带 `q`、时间范围或 `user_id`/`chat_id`/`status` 等已支持过滤参数
- **THEN** 仅返回匹配条件的 Session，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Session 详情
- **WHEN** 客户端请求已存在 Session 的详情
- **THEN** 返回含草稿/进度等排障所需字段的表示

### Requirement: 不以完整 CRUD 破坏态机
本期 Session 管理 MUST NOT 提供任意改写对话态机关键字段的通用 Update/Delete（除非显式定义的安全运维动作）；默认以只读查询交付。

#### Scenario: 默认无通用写接口
- **WHEN** 客户端尝试对 Session 做未定义的通用更新
- **THEN** 系统不提供该通用写能力或返回方法不允许

```

## docs/openspec/changes/admin-resources-api/specs/task-admin-api/spec.md

- Source: docs/openspec/changes/admin-resources-api/specs/task-admin-api/spec.md
- Lines: 1-38
- SHA256: 716e63ec7a9550946d1ec80887d041609ba71a1eb8b9fbcb3ab2f2c865e618ea

```md
## Purpose

为运营侧提供 Task 列表、详情与取消等管理 HTTP，不在 admin 替代用户 ConfirmRun。

## ADDED Requirements

### Requirement: Task 列表与详情
系统 MUST 通过 admin-api 提供 Task 列表与按 ID 详情查询。

#### Scenario: 列出 Task
- **WHEN** 客户端请求 Task 列表
- **THEN** 返回任务集合及状态、关联 session/instance 等关键字段

#### Scenario: 按过滤条件列出 Task
- **WHEN** 客户端请求 Task 列表并携带 `q`、时间范围或 `status`/`instance_id`/`chat_id`/`session_id`/`case_id` 等已支持过滤参数
- **THEN** 仅返回匹配条件的任务，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Task 详情
- **WHEN** 客户端请求已存在 Task 的详情
- **THEN** 返回该任务记录

### Requirement: Task 取消
系统 MUST 支持运营侧通过 `POST /api/v1/tasks/{id}/cancel`（或等价动作路径）取消仍可取消的任务，并反映取消后状态；取消语义 MUST 复用 runtime 领域取消规则。

#### Scenario: 取消可取消任务
- **WHEN** 运维对可取消状态的任务发起取消
- **THEN** 任务进入取消相关终态或约定的取消中状态，且详情可观察

#### Scenario: 不可取消时明确失败
- **WHEN** 运维对不可取消状态的任务发起取消
- **THEN** 返回明确错误且不伪造成功

### Requirement: 不在 admin 发起代用户 ConfirmRun
admin-api MUST NOT 提供“代替用户确认生成并创建任务”的管理入口作为本期能力。

#### Scenario: 无 ConfirmRun 管理入口
- **WHEN** 客户端查找代用户 ConfirmRun 的管理 API
- **THEN** 本期不提供该能力

```

## docs/openspec/changes/admin-resources-api/specs/user-admin-api/spec.md

- Source: docs/openspec/changes/admin-resources-api/specs/user-admin-api/spec.md
- Lines: 1-31
- SHA256: 1ff970df9ea0c01f1444e151f66261bbfceabf8bf1d077b7d27d11d95fa5763c

```md
## Purpose

为管理后台提供用户资源的查询 HTTP，支撑运维查看 TG 关联用户。

## ADDED Requirements

### Requirement: 用户列表与详情
系统 MUST 通过 admin-api 提供用户列表与按内部 ID 详情查询。

#### Scenario: 列出用户
- **WHEN** 客户端请求用户列表
- **THEN** 返回用户集合（含可观察标识字段，如内部 id、tg 用户相关字段）

#### Scenario: 按过滤条件列出用户
- **WHEN** 客户端请求用户列表并携带 `q`、时间范围或 `tg_user_id` 等已支持过滤参数
- **THEN** 仅返回匹配条件的用户，并支持 `limit`/`offset` 分页

#### Scenario: 获取用户详情
- **WHEN** 客户端请求已存在用户的详情
- **THEN** 返回该用户记录

#### Scenario: 用户不存在
- **WHEN** 客户端请求不存在的用户 ID
- **THEN** 返回未找到错误

### Requirement: 不替代 TG upsert 主路径
admin-api MUST NOT 成为用户创建的主业务入口；用户仍主要由 bot/TG 路径 upsert。本期 MUST NOT 提供用户写接口。

#### Scenario: 管理面本期只读
- **WHEN** 运维使用本期用户管理 API
- **THEN** 可完成列表与详情，且不存在创建/更新/删除用户的管理写入口

```
