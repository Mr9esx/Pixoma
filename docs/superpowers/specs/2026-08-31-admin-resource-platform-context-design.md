---
comet_change: admin-resource-platform-context
role: technical-design
canonical_spec: openspec
---

# Admin Resource Platform Context — Technical Design

## Context

Task、Session、User 三个后台资源页需要让运维直接识别消息平台来源和跨资源关系。当前 Task 只能通过 `session_id` 间接推断来源，Session 未在管理 DTO 中暴露 `channel_id`，User 前端仍使用 Telegram 专用 `tg_user_id`。

OpenSpec delta specs 是需求事实源。本设计只细化管理端投影、前端展示和测试边界，不改变任务执行、会话态机或用户 upsert 主路径。

## Goals / Non-Goals

**Goals**

- Task、Session、User 管理列表能直接识别消息平台来源。
- Task 能同时识别关联用户和关联 Session，并支持跳转。
- User 列表使用消息平台无关的标识和统一「用户信息」展示。
- 管理查询一次返回展示所需上下文，避免前端逐行补查。

**Non-Goals**

- 不合并不同消息平台的用户身份。
- 不改任务执行、会话状态机或用户写入主路径。
- 不新增数据库迁移或冗余平台列。
- 不把 UI 绑定到 Telegram 平台名或 Telegram 专用 ID。

## Data Flow

### Task 管理投影

Task admin 使用专用读取投影，不复用 runtime 领域聚合承载展示上下文。查询按以下路径关联：

```text
tasks
  LEFT JOIN sessions ON tasks.session_id = sessions.id
  LEFT JOIN channel_users ON sessions.user_id = channel_users.id
  LEFT JOIN channel_user_external_identities
    ON channel_users.id = channel_user_external_identities.user_id
  LEFT JOIN channels ON sessions.channel_id = channels.id
```

`channel_user_external_identities` 与 `channels` 存在一对多可能时，按当前用户目录语义取任务所属 `sessions.channel_id` 对应的身份记录；如果未来允许多平台身份绑定同一内部用户，应改为按 `channel_id` 精确匹配，避免笛卡尔展开。列表排序、过滤和分页仍以 Task 主表为锚。

Task admin DTO 返回：

- 既有 Task 字段。
- `session_id`：已有，继续保留。
- `channel_id`：任务来源消息平台实例。
- `user_id`：Session 关联的内部用户 ID。
- `user`：与 User 展示一致的上下文对象，包含内部 ID、消息平台、外部用户 ID、用户名和姓名等字段；缺失时为空。

取消动作和 runtime 领域转换不读取该投影，保持原有 `TaskRepository` 语义。

### Session 管理投影

Session 持久化已包含 `channel_id`。管理 DTO 直接返回该字段，并在需要展示名称时通过 `channels` 补充 `channel_name`。Session 领域模型只增加管理展示所需上下文或由专用查询投影承载，业务写入路径不变。

### User 管理投影

User admin list/detail 返回：

- `id`
- `channel_id`
- `external_user_id`
- `username`
- `first_name`
- `last_name`
- 其他既有资料和访问字段

`channel_id` 和 `external_user_id` 来自 `channel_user_external_identities`。由于当前每个消息平台身份创建独立内部用户，通常为一对一；若未来同一内部用户绑定多个平台身份，管理列表必须按身份行展开或改为显式身份集合，避免随机选择一条身份。搜索 `q` 覆盖内部 ID、外部用户 ID、用户名和姓名。

`tg_user_id` 从 User API 契约、前端类型、列表、详情和搜索入参中移除。Telegram 的外部 ID 在多平台语境下改用 `external_user_id: string` 表示，避免前端数字精度丢失。

## API Shape

`channel_id` 是稳定关联键与筛选键。展示层返回消息平台名称时，可补充 `channel_name`；名称缺失时前端回退 `channel_id`。平台字段缺失时使用空态，不根据 ID 前缀猜测 Telegram。

Task、Session、User 管理查询保持既有分页、时间范围和状态过滤语义。Task 新增平台与用户上下文不改变过滤结果的默认行为；既有 `channel_id` 过滤继续通过 Session 关联生效。

## Frontend

### Task Table

Task 列表在状态和 Case 相关列附近展示：

- 消息平台：优先 `channel_name`，缺失回退 `channel_id`。
- 关联用户：与 User Table 一致的完整用户信息，链接到 `/users/$userId`。
- 关联 Session：Session 标识，链接到 `/sessions/$sessionId`。

链接使用路由器语义跳转，不手动拼浏览器 URL，也不用弹窗替代资源详情页。

### Session Table

Session 列表在用户和 Case 附近新增「消息平台」列，详情头部同步展示来源。列表与详情均来自同一 admin DTO，避免两处口径不一致。

### User Table

User 列表展示：

- 消息平台：优先 `channel_name`，缺失回退 `channel_id`。
- 用户信息：一列承载主行与次行；主行优先用户名，缺少用户名时组合姓名，次行显示 `external_user_id`。

删除 `TG User ID` 列、详情字段和查询参数。单一搜索框继续使用 `q`，覆盖内部 ID、外部用户 ID、用户名和姓名。类型定义为 `channel_id: string`、`external_user_id: string`。

### Visual Rules

- 复用现有 DataTable、链接、Badge、空态和语义令牌；不新增裸 hex。
- 标识类字段使用等宽字体和 `tabular-nums`。
- 列文案进入中英文 i18n：「消息平台」「关联用户」「关联会话」「用户信息」。
- 字段缺失显示「—」，请求失败继续显示资源页既有错误态。

## Error And Empty States

Task 查询使用 LEFT JOIN，Session、User 或平台记录缺失时不吞掉主任务行。缺失字段显示空态；API 仍返回 Task 主字段。分页计数、状态筛选和取消动作继续以 Task 本身为准。

Session 查询同样保留无平台匹配的历史数据行。User 查询找不到外部身份时保留内部用户资料，但必须显示空态而不是伪造平台。

## Testing

### Backend

按 TDD 先补失败测试，再实现投影：

- Task admin list/detail 返回 `channel_id`、`user_id`、`session_id` 和嵌套用户信息。
- Task 平台和用户来自 Session 关联；缺失 Session 时主行仍返回且关联字段为空。
- Session list/detail 返回 `channel_id` 和可展示的 `channel_name`。
- User list/detail 返回 `channel_id`、`external_user_id` 和资料字段。
- User `q` 命中内部 ID、外部用户 ID、用户名和姓名。
- User API 不再将 `tg_user_id` 作为字段或查询参数。

### Frontend

更新列表契约测试：

- Task 列表包含消息平台、关联用户和关联 Session。
- Task 用户与 Session 使用正确路由链接。
- Session 列表包含消息平台。
- User 列表包含消息平台和统一用户信息列。
- Task/Session/User 前端不再出现 `tg_user_id`。
- 中英文文案键同步。

### Validation

运行受影响 Go 测试、受影响 Vitest 测试和静态检查，再运行 OpenSpec 校验。交付前核对三个列表的核心场景：Task 来源与关联可识别、Session 来源可识别、User 无 Telegram 专用字段。

## Risks / Trade-offs

- Task 全量管理列表总是关联 Session、User 和平台表 → 管理查询保持明确 limit/offset，只选择展示字段，并依赖既有主外键索引。
- 用户与外部身份需要额外查询 → 在管理投影中一次完成或按结果批量查询，避免 N+1。
- `tg_user_id` 查询参数移除可能影响旧客户端 → 前端与 admin-api 同批发布；平台无关参数为 `channel_id`/`external_user_id`。
- 平台名称展示依赖 `channels` 表 → 名称缺失时回退 `channel_id`，不阻塞主资源列表。

## Migration Plan

无数据库迁移。先实现并验证后端投影，再同步更新前端类型和列表展示。后端新增字段对旧前端向后兼容；移除 Telegram 专用 User 查询字段与前端切换同批完成。
