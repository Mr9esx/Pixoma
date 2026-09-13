## ADDED Requirements

### Requirement: 按平台类型选择适配器
装配器 MUST 读取消息平台的 `platform` 再选择适配器工厂。`platform` 为 `telegram` 时装配 TG 适配器；为 `feishu` 时装配飞书适配器。MUST NOT 对非 telegram 实例调用 Telegram Bot 构造或 `api.telegram.org` 探测。

#### Scenario: 飞书实例不启动 TG Bot
- **WHEN** 存在启用且凭证完整的飞书消息平台，同时没有启用的 Telegram 消息平台
- **THEN** 运行时启动飞书长连接，且不创建 Telegram Bot、不请求 getMe

#### Scenario: Telegram 实例仍走 TG
- **WHEN** 存在启用的 Telegram 消息平台
- **THEN** 仍装配 TG 适配器，既有私聊行为不变
