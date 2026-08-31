## MODIFIED Requirements

### Requirement: 用户列表与详情
系统 MUST 通过 admin-api 提供用户列表与按内部 ID 详情查询。列表和详情表示 MUST 包含内部用户 ID、消息平台 `channel_id` 和该平台内的通用外部用户标识 `external_user_id`，并保留用户名、姓名等资料字段；列表表示 MUST NOT 将 Telegram 专用 ID 作为契约字段或专用过滤参数。

#### Scenario: 列出用户
- **WHEN** 客户端请求用户列表
- **THEN** 返回用户集合，每条记录包含内部 ID、消息平台、通用外部用户标识和用户资料字段

#### Scenario: 按过滤条件列出用户
- **WHEN** 客户端请求用户列表并携带 `q`、时间范围或 `channel_id`/`external_user_id` 等已支持过滤参数
- **THEN** 仅返回匹配条件的用户，并支持 `limit`/`offset` 分页；`q` MUST 能匹配用户标识和用户资料字段

#### Scenario: 获取用户详情
- **WHEN** 客户端请求已存在用户的详情
- **THEN** 返回该用户记录及其消息平台和通用外部用户标识

#### Scenario: 用户不存在
- **WHEN** 客户端请求不存在的用户 ID
- **THEN** 返回未找到错误
