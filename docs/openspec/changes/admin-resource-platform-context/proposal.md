## Why

Task、Session 和 User 管理列表目前缺少清晰的消息平台来源：Task 只显示 session id，Session 未直接暴露消息平台，User 还暴露 `tg_user_id` 这类 Telegram 专用字段。这会让多消息平台数据的排障和跨表关联依赖内部 id 推断。

## What Changes

- Task 列表与详情补充消息平台来源，并展示关联的用户与 Session。
- Session 列表与详情补充消息平台来源。
- User 列表与详情补充消息平台来源；列表用统一的「用户信息」列承载用户名/姓名等可读信息，不再把 `tg_user_id` 作为列表字段或筛选语义暴露。
- admin-api 返回消息平台和跨表关联所需字段；Task 返回关联用户标识。
- 后台资源页使用这些字段更新三个 Table 与详情展示。

## Capabilities

### New Capabilities

- 无

### Modified Capabilities

- `task-admin-api`: Task 列表和详情 MUST 暴露消息平台、用户与 Session 关联字段。
- `session-admin-api`: Session 列表和详情 MUST 暴露消息平台字段。
- `user-admin-api`: User 列表和详情 MUST 暴露消息平台与通用外部用户标识，列表查询 MUST 支持跨平台用户标识搜索且不再以 TG 专用过滤为核心契约。
- `admin-resource-pages`: Task、Session、User 三个管理页 MUST 展示消息平台来源与统一用户信息，Task MUST 能定位关联用户和 Session。

## Impact

- 后端 admin-api：Task、Session、User DTO 和列表查询需要补充或调整字段；Task 平台信息通过 Session 关联读取。
- 前端：`web/admin/src/features/tasks`、`sessions`、`users` 的 Table、搜索、详情与文案/i18n 需要更新。
- 数据：现有 Session 已有消息平台标识，User 已有消息平台和外部用户标识；不需要新建迁移表。
- 兼容性：前端不再读取或展示 `tg_user_id` 列表字段；admin-api 移除该专用筛选语义属于管理端契约调整，不涉及业务库数据迁移。
