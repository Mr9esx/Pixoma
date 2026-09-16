## Why

原计划覆盖企业微信智能机器人，但 Pixoma 不再接入企业微信或个人微信。保留已经完成的跨平台基础优化，使 Telegram、飞书和后续消息平台共享稳定的会话与运行时边界。

## What Changes

- 提取平台无关的会话控制器，统一处理文本、媒体、菜单、卡片、导航和任务通知。
- 将 Telegram 调整为事件解码与平台渲染层，保持既有交互行为。
- 将飞书纳入显式平台工厂，并正确注册、注销通知处理器。
- 调整适配器重建顺序：旧实例停止失败时不得启动替代实例。
- 删除企业微信凭证、管理 API 字段、协议客户端和第三方依赖。

### 非目标

- 不新增企业微信、个人微信或其他消息平台。
- 不新增后台平台能力目录或会话范围配置。
- 不改变 Telegram、飞书和 MCP 的既有业务语义。

## Capabilities

### Modified Capabilities

- `channel-runtime-ports`：共享会话逻辑与平台适配器职责分离。
- `channel-interaction-protocol`：菜单、卡片、导航与结果渲染由共享控制器决策。
- `channel-management`：适配器替换先停止旧实例；企业微信不再是可创建平台。

## Impact

- `internal/channels/conversation`、`internal/channels/tg`、`internal/channels/feishu`。
- `apps/pixoma/internal/telegram` 的平台工厂与通知注册。
- `internal/channels/application/assembler.go` 的安全重建顺序。
- 企业微信专属运行时代码、凭证字段、HTTP 字段和 Go 依赖已移除。
