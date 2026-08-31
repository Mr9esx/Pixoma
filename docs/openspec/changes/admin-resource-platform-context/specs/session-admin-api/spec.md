## MODIFIED Requirements

### Requirement: Session 列表与详情
系统 MUST 通过 admin-api 提供 Session 列表与详情查询。列表和详情表示 MUST 包含 `channel_id`，以标识产生该 Session 的消息平台。

#### Scenario: 列出 Session
- **WHEN** 客户端请求 Session 列表
- **THEN** 返回 Session 集合及状态和消息平台等关键字段

#### Scenario: 按过滤条件列出 Session
- **WHEN** 客户端请求 Session 列表并携带 `q`、时间范围或 `user_id`/`chat_id`/`channel_id`/`status` 等已支持过滤参数
- **THEN** 仅返回匹配条件的 Session，并支持 `limit`/`offset` 分页

#### Scenario: 获取 Session 详情
- **WHEN** 客户端请求已存在 Session 的详情
- **THEN** 返回含消息平台、草稿/进度等排障所需字段的表示
