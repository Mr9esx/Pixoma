## ADDED Requirements

### Requirement: 企微智能机器人消息平台
管理员 MUST 能选择平台类型 `wecom` 并提交智能机器人 Bot ID 与 Secret。MUST NOT 把该类型当作 Telegram Bot Token 或企微自建应用回调配置。探测 MUST 验证智能机器人通道，MUST NOT 调用 Telegram getMe。

#### Scenario: 后台创建智能机器人
- **WHEN** 管理员选择企业微信智能机器人并填写 Bot ID 与 Secret
- **THEN** 系统保存该消息平台，列表显示对应平台类型，Secret 不在列表明文回显

#### Scenario: 缺少 Secret 不启动
- **WHEN** 消息平台启用但 Secret 为空
- **THEN** 适配器不启动，并记录可诊断错误
