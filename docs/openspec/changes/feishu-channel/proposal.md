## Why

本机 Pixoma 只能接 Telegram。飞书企业自建应用可以用官方长连接收消息、发图片，不必把 Pixoma 暴露到公网。要落地飞书，还必须先改装配：现在任何启用的消息平台都会被当成 TG Bot 启动。

## What Changes

- 装配工厂、凭证、可达性探测、后台创建表单按 **平台类型** 分支；非 telegram MUST NOT 再走 `getMe` / Bot Token。
- 后台可创建并启用 **飞书** 消息平台（App ID + App Secret），本机挂 WebSocket 长连接。
- 飞书适配器实现运行时端口：收私聊与群 @、卡片按钮打开已有工作流、上传图片后异步回图。
- 事件回调 3 秒内应答；出图走既有 Task，完成后再调发消息 API。
- 菜单在飞书侧渲染为卡片按钮，不复制 TG ReplyKeyboard。

### 非目标

- 不做商店应用、不保证国际版 Lark 长连接。
- 不做企微/钉钉适配器实现（管道预留给 `wecom-aibot-channel`）。
- 不做 MCP、不做公众号。
- 不把飞书交互做成 TG 键盘的像素级复制。

## Capabilities

### New Capabilities

- `feishu-channel`：飞书长连接适配器、卡片交互、媒体桥、异步回图。

### Modified Capabilities

- `channel-runtime-ports`：装配 MUST 按 `platform` 选择适配器，不得把非 TG 实例交给 TG 工厂。
- `channel-management`：创建/探测/热启停 MUST 支持飞书凭证形态；后台可选平台飞书。

## Impact

- `apps/pixoma/internal/telegram` 工厂按平台分支（或迁到中性组装包）。
- `internal/channels/domain` 凭证；`application/reachability` / probe。
- 新包 `internal/channels/feishu`。
- `web/admin` 创建表单：平台选择 + 飞书字段。
- 架构：`overview.md`、`runtime.md`、`bounded-contexts.md`、`data-model.md`（凭证）。
- Mock：`comfy_mock` 下飞书主路径仍须出图。
