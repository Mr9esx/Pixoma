## Why

企业微信里能在本机跑、又不必公网回调的，是「智能机器人」长连接，不是自建应用 HTTP 回调。飞书那次会先铺好多平台管道；本 change 只接这一种企微形态，让人在私聊或群 @ 里调用已有工作流并收回图。

## What Changes

- 新增 `wecom` 智能机器人适配器：本机连 `WSS`，`aibot_subscribe`（Bot ID + Secret）。
- 后台可创建该类型消息平台；探测走智能机器人通道，不打 Telegram。
- 进会话欢迎/模板卡片作为工作流入口；群里 @ 可触发。
- 出图可能超过流式窗口：超时后用主动推送把成品发回原会话。
- 同一机器人同时只保一条活动长连接：热重载必须先停干净再连。

依赖：`feishu-channel` 的按平台工厂、凭证、探测、后台表单。

### 非目标

- 不做企微自建应用回调、可信 IP、群 Webhook 单向推送当对话入口。
- 不做个人微信、公众号、钉钉。
- 不把官方不存在的 Go SDK 当硬依赖。

## Capabilities

### New Capabilities

- `wecom-aibot-channel`：智能机器人长连接、模板卡片、媒体、单连接生命周期。

### Modified Capabilities

- `channel-management`：平台类型 `wecom` 的创建、凭证与启停 MUST 走智能机器人，不得再填单一 Bot Token 去打 Telegram。

## Impact

- `internal/channels/wecom`（或等价包）+ 工厂注册。
- Admin 创建表单企微字段。
- 架构文档补企微入站（长连接、单 WSS）。
- Mock：`comfy_mock` 下企微主路径仍须出图。
