## MODIFIED Requirements

### Requirement: 不替代 TG upsert 主路径
admin-api MUST NOT 成为 Telegram 等 IM 用户的创建入口；这些用户仍由消息路径 upsert。MCP 消息平台上的用户 MUST 允许由管理端创建（MCP 没有入站消息）。

#### Scenario: 管理面不为 IM 提供创建
- **WHEN** 运维对非 MCP 消息平台使用用户管理 API
- **THEN** 不存在创建该类用户的管理写入口

#### Scenario: MCP 平台可生成用户
- **WHEN** 管理员在已有 MCP 消息平台上生成用户并填写显示名
- **THEN** 创建 `channel_users` 行（权限为允许）、签发该用户 Bearer，用户页可按该渠道列出

#### Scenario: MCP 平台可删除用户
- **WHEN** 管理员删除某 MCP 消息平台上的用户
- **THEN** 该用户从渠道名单消失，其 Bearer 立刻失效；MUST NOT 通过该入口删除非 MCP 消息平台用户
