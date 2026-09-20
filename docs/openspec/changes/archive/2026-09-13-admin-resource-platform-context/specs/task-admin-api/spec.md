## MODIFIED Requirements

### Requirement: Task 列表与详情
系统 MUST 通过 admin-api 提供 Task 列表与按 ID 详情查询。列表和详情表示 MUST 包含 `session_id`、`user_id` 和 `channel_id`，其中 `channel_id` 表示产生该任务的消息平台，`user_id` 表示与该任务关联的用户。

#### Scenario: 列出 Task
- **WHEN** 客户端请求 Task 列表
- **THEN** 返回任务集合及状态、消息平台、关联用户和关联 Session 等关键字段

#### Scenario: 按过滤条件列出 Task
- **WHEN** 客户端请求 Task 列表并携带 `q`、时间范围或 `status`/`instance_id`/`chat_id`/`channel_id`/`session_id`/`case_id` 等已支持过滤参数
- **THEN** 仅返回匹配条件的任务，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Task 详情
- **WHEN** 客户端请求已存在 Task 的详情
- **THEN** 返回该任务记录及其消息平台、关联用户和关联 Session 字段
