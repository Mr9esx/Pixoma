## ADDED Requirements

### Requirement: MCP 消息平台
管理员 MUST 能创建平台类型 `mcp` 的消息平台。该平台 MUST NOT 要求 Bot Token / App Secret，MUST NOT 启动 IM 适配器、MUST NOT 收消息。停用该消息平台 MUST 使挂在其上的 MCP 用户凭据全部失效（请求 401）。

#### Scenario: 后台创建 MCP 平台
- **WHEN** 管理员选择平台 MCP 并填写名称、不填 IM 凭证
- **THEN** 系统保存该消息平台，列表显示平台为 MCP，不启动 Telegram/飞书等适配器

#### Scenario: 停用后 MCP 凭据失效
- **WHEN** 该 MCP 消息平台已停用，客户端仍携带其上用户的有效 Bearer
- **THEN** `/mcp` 与 `/sse` 拒绝请求且不执行工作流
