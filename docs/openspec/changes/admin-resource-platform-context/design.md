## Context

Session 持久化已包含消息平台标识；用户模型已按「消息平台 + 外部用户 ID」维护身份。当前 admin DTO 和前端 Table 没有把这些上下文完整带回，Task 仅能通过 session id 间接推断来源。

## Goals / Non-Goals

**Goals:**

- 让三个后台列表在无需跳转详情的情况下识别数据来源和跨表关系。
- Task 的平台和用户上下文一次列表查询返回，避免前端逐行补查。
- 将用户展示抽象为平台无关的「用户信息」，移除 Telegram 专用列表字段。

**Non-Goals:**

- 不合并不同消息平台的用户账号。
- 不修改任务、会话或用户业务写入路径。
- 不新增数据库迁移，也不把平台展示硬编码为 Telegram。

## Decisions

### API 投影

Task admin DTO 新增 `channel_id` 和 `user_id`。Task repository 的 admin list/get 投影通过既有 `tasks.session_id = sessions.id` 关联读取 Session 的 `channel_id` 和 `user_id`，而不是在 Task 表新增冗余列。

Session DTO 直接暴露持久化层已有的 `channel_id`；Session 领域模型补充该只读上下文字段并从行映射填充。Session admin 投影与 Task 采用同一模式：通过既有 `sessions.user_id = channel_users.id` 关联读取用户资料，并关联通用外部身份和 Channel 名称，一次查询返回嵌套 `user` 与 `channel_name`。

User DTO 新增 `channel_id` 和 `external_user_id`。由于用户与外部身份是两个持久化模型，repository 在 list/get 时返回平台身份投影；多平台用户目录中同一记录仍以单一用户身份展示其所属平台上下文。

备选方案是在业务表冗余平台字段。该方案会引入双写一致性问题，且本需求只是管理投影，因此不采用。

### 用户信息展示

前端 User Table 用一列「用户信息」渲染：优先显示用户名，缺少用户名时显示组合姓名，再补充平台外部用户标识作为次级信息。搜索继续走单一 `q`，后端覆盖内部 ID、外部用户 ID、用户名和姓名。

`tg_user_id` 从前端类型、Table、搜索和 i18n 展示中移除；API 查询契约改为平台无关的 `channel_id` 与 `external_user_id`。

### 资源页结构

三个列表沿用现有 DataTable 和 Master–Detail 结构。Task 增加「消息平台 / 用户 / Session」列，Session 增加「平台 / 关联用户」列，User 用平台列加用户信息列替换 TG 专用列。三类列表都把操作列固定在右侧，并显示可读列名。列名与值统一走 i18n；时间值使用本地时区格式化；字段缺失时显示空态而不是回退到平台猜测。

## Risks / Trade-offs

- Task 全量 admin 列表总是 join Session → 限制查询字段并在既有索引上使用 `session_id` 关联；如果出现孤行，平台/用户字段返回空态并保留任务行。
- 移除 `tg_user_id` 查询参数可能影响旧客户端 → 前端与 admin-api 同步发布；不再将该 Telegram 专用参数纳入本期能力契约。
- 用户与外部身份需要额外查询 → 在同一 repository 查询中完成或按结果批量读取，避免 N+1。

## Migration Plan

无数据库迁移。先补后端 DTO/查询与测试，再更新前端类型和资源页。发布顺序上后端新增字段向后兼容；移除 TG 专用查询参数与前端切换同批完成。
