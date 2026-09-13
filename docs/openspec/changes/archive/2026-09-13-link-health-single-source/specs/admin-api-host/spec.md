## ADDED Requirements

### Requirement: 控制面挂载链路健康查询
`pixoma` 控制面托管的管理 HTTP MUST 提供链路健康只读查询。该接口 MUST 走与其他管理 API 相同的会话与权限校验。MUST NOT 把该查询放到独立进程。MUST NOT 在该查询的热路径上对外部消息平台发起探测。

#### Scenario: 一体入口可访问健康查询
- **WHEN** 用户仅启动 `pixoma` 且已初始化、已登录
- **THEN** 可通过该进程的管理地址读取链路健康查询

#### Scenario: 无会话访问被拒
- **WHEN** 平台已初始化，客户端未提供管理员会话调用该查询
- **THEN** 请求因未认证被拒绝

### Requirement: 异步探测踢脚
`pixoma` 控制面 MUST 提供对已启用通道的异步连通探测踢脚。该接口 MUST 在外部消息平台往返完成前返回。MUST NOT 让 `GET /api/v1/channels`、`GET /api/v1/channels/{id}` 或 `GET /api/v1/link-health` 等待这次探测。

#### Scenario: 踢脚返回早于 Telegram
- **WHEN** 已登录客户端调用探测踢脚，且 Telegram getMe 被阻塞
- **THEN** 踢脚在 getMe 返回前结束，探测仍在后台对已启用通道执行并写入 last_check
