# user-admin-api Specification

## Purpose
为管理后台提供用户资源的查询 HTTP，支撑运维查看 TG 关联用户。
## Requirements
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
admin-api MUST NOT 成为 Telegram 等 IM 用户的创建入口；这些用户仍由消息路径 upsert。MCP 消息平台上的用户 MUST 允许由管理端创建（MCP 没有入站消息）。

#### Scenario: 管理面本期只读
- **WHEN** 运维使用本期用户管理 API
- **THEN** 可完成列表与详情；对 Telegram 等 IM 用户不存在创建/更新/删除入口

#### Scenario: 管理面不为 IM 提供创建
- **WHEN** 运维对非 MCP 消息平台使用用户管理 API
- **THEN** 不存在创建该类用户的管理写入口

#### Scenario: MCP 平台可生成用户
- **WHEN** 管理员在已有 MCP 消息平台上生成用户并填写显示名
- **THEN** 创建 `channel_users` 行（权限为允许）、签发该用户 Bearer，用户页可按该渠道列出

#### Scenario: MCP 平台可删除用户
- **WHEN** 管理员删除某 MCP 消息平台上的用户
- **THEN** 该用户从渠道名单消失，其 Bearer 立刻失效；MUST NOT 通过该入口删除非 MCP 消息平台用户

