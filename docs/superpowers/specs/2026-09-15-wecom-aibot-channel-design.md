---
comet_change: wecom-aibot-channel
role: technical-design
canonical_spec: openspec
---

# 企业微信智能机器人渠道设计

## 目标与边界

本设计为 Pixoma 增加企业微信「智能机器人」长连接渠道。管理员配置 Bot ID、Secret 与可选 WSS 地址后，机器人可在单聊或群聊中复用 Telegram 已有的工作流会话能力，并将任务结果投递回发起会话。

本次不实现企业微信自建应用 HTTP 回调、可信 IP 或个人微信、公众号、钉钉渠道。入站能力严格对齐当前 Telegram：文本、图片与图片文档；语音、视频和普通文件不进入 `open_case`。出站按 MIME 支持图片、动图、视频、文件和文本回退。

## 架构

### 共享会话核心与平台薄适配层

将当前 Telegram 适配器内的平台无关逻辑迁入 `internal/channels/conversation`。该核心负责：

- 身份解析、`open_case` 与 `list_tasks` 调用；
- 会话输入、跳过、退出、确认运行与返回导航；
- 菜单树和卡片动作解析；
- 回调动作令牌的生成、消费与过期处理；
- `protocol.Result`、任务通知到逻辑出站效果的转换。

核心不解析任何平台事件，也不处理 WebSocket、Bot API 或媒体加密。平台在下载并存储入站媒体后，将 `BlobRef` 交给核心；核心向一个窄渲染端口输出文本、媒体、菜单和卡片效果。

Telegram、飞书与企微均改为薄适配层：

| 平台 | 适配层职责 |
| --- | --- |
| Telegram | 将 Bot update 和 callback 解码为核心输入，调用 Telegram Bot API 渲染效果。 |
| 飞书 | 将长连接事件解码为核心输入，调用飞书 IM API 与长连接事件回复渲染效果。 |
| 企微 | 将智能机器人帧与事件解码为核心输入，调用企业微信长连接协议渲染效果和媒体操作。 |

先从 Telegram 提炼核心并迁移全部既有测试，再迁移飞书。企微在共享核心稳定后接入。这样「对齐 Telegram」由同一段业务代码保证，而不是靠三套实现长期同步。

### 企微传输封装

在 `internal/channels/wecom` 中定义 Pixoma 自己的 `Client` 接口和事件模型。适配器仅依赖该接口；`wecom-aibot-go` 的客户端包装实现该接口，并固定到一个提交版本。SDK 负责：

- `aibot_subscribe`、心跳、ACK、断线重连和关闭；
- 消息与事件帧解析；
- 图片/文件下载与 AES 解密；
- 临时素材上传，以及被动回复和主动推送。

SDK 不承载 Pixoma 的菜单、会话、权限、任务或状态语义。测试使用假 `Client`，不直接依赖网络和 SDK 的定时行为。

企业微信默认端点为 `wss://openws.work.weixin.qq.com`；私有化部署可通过加密凭证中的 `wecom_ws_url` 覆盖。官方维护的 Node SDK提供相同的长连接、事件、媒体解密与上传协议能力；Go SDK 仅作为 Go 层协议实现，不引入 Node sidecar。[企业微信维护的协议参考](https://github.com/WecomTeam/aibot-node-sdk)

### 工厂、凭证与运行时

将 `apps/pixoma/internal/telegram` 中的 `tgChannelFactory` 改为按平台显式分发的渠道工厂：

- `telegram` 创建 Telegram 薄适配层；
- `feishu` 创建飞书薄适配层并注册通知处理器；
- `wecom` 创建企微薄适配层并注册通知处理器；
- `mcp` 不创建 IM 适配器。

`ChannelSnapshot` 改为承载平台专属、已解密的运行时配置，并以完整凭证和 WSS 地址计算哈希。更新 Bot ID、Secret 或 WSS 地址都会触发重建。

`domain.Credential` 新增 `wecom_bot_id`、`wecom_secret` 和 `wecom_ws_url`。Secret 始终加密存储和掩码回显；Bot ID 与 WSS 地址可在详情 API 回显。创建与更新 API 使用专属字段，不把 Bot ID 或 Secret 伪装成 Telegram Token 或飞书 App ID。

企微探测不得新建业务 WSS。`CheckReachability` 对企微读取运行中适配器的认证、心跳与最后错误状态；未认证、重连中或状态未知均持久化为非 `ok`，以确保后台不会在健康信息未就绪时显示绿色。

### 单连接生命周期

企业微信限制同一 Bot ID 同时只有一条有效长连接。Assembler 的替换流程改为：

1. 标记旧适配器为 stopping，并脱离当前快照；
2. 等待 `Stop` 返回成功；
3. 仅在停止成功后创建并启动新适配器；
4. 停止失败或超时时写入 adapter error，跳过本轮新建，等待下一次协调。

该规则对全部 IM 平台生效。企微客户端收到被新连接抢占的断开事件后停止重连，避免两个实例互相踢线。

## 数据流

### 入站消息与卡片事件

```text
平台事件 → 薄适配层解码 → 图片下载/解密并写 Blob
        → conversation 核心 → Capability Registry
        → 逻辑出站效果 → 平台薄适配层渲染
```

- 单聊使用会话地址 `single/<userid>`；群聊使用 `group/<chatid>`。地址仍放入既有 `ChannelAddr.ExternalChatID`，因此异步任务通知在进程重启后仍可恢复正确的投递类型，无需修改会话表。
- 群聊仅处理 @ 机器人的文本或图片输入；结果仍投递到该群。
- `enter_chat` 在 4 秒预算内以欢迎语或入口模板卡片响应。
- 模板卡片事件在 4 秒预算内确认并更新卡片。按钮动作由核心令牌解析；令牌过期时渲染明确的重新选择提示。
- 入站图片和图片文档下载后以现有 `open_case` 的 `step=media` 提交。没有活动会话时，核心返回主菜单或入口卡片，与 Telegram 保持一致。

### 出站消息与任务通知

同步能力结果优先使用回调帧被动回复。媒体先上传为企微临时素材，再按 MIME 发送图片、动图、视频或文件。长任务只在回调窗口内确认已提交；`task_succeeded`、`task_failed` 与 `task_cancelled` 由既有 `NotifyRouter` 交给渠道适配器，再用 `aibot_send_msg` 主动投递到原始单聊或群聊。

主动投递地址始终从持久化的 `ChatID` 解析，不依赖内存中的回调帧。单聊以用户 ID 寻址，群聊以群 ID 寻址。

## 管理端

渠道创建表单新增「企业微信智能机器人」：名称、Bot ID、Secret，以及可选 WSS 地址。表单和详情编辑沿用已安装 shadcn/ui 组件与 Pixoma 设计令牌；Secret 使用现有 `SecretInput`，不在列表或详情明文展示。

平台探测、列表、详情头部与健康告警都读取同一健康结果。企微适配器未认证、停止中、重连中或状态不可用时显示黄色，不显示绿色。状态文案使用现有 `StatusDot` 和「已启用 / 已停用」约定。

## 错误处理与安全

- Bot ID 或 Secret 缺失时拒绝启动并暴露可诊断错误；不回退到 Telegram。
- 企微回调、媒体下载、解密、上传和发送失败分别记录平台、渠道和操作类型，不记录 Secret、AES key、媒体 URL 或完整消息内容。
- 下载使用与 Telegram 相同的大小上限和超时策略；解密或 MIME 校验失败不写入 Blob。
- SDK 依赖固定版本并隔离在 `Client` 包装层；任何协议升级先通过包装层契约测试。
- 适配器停止失败时禁止替换连接，优先保持单连接约束。

## 验证策略

1. 为共享核心迁移 Telegram 的菜单、卡片、会话输入、回调、结果与通知测试，确保行为不变。
2. 为 Telegram、飞书、企微建立同一组跨平台行为契约：文本、图片、图片文档、无活动会话、菜单、卡片按钮、确认运行、任务通知和媒体 MIME 路由。
3. 为企微薄适配层使用假 `Client` 覆盖订阅、欢迎事件、卡片事件、群 @、AES 图片、上传、被动回复与主动推送。
4. 为 Assembler 覆盖凭证重建时「旧连接先停」、停止失败不新建、平台工厂三路分发，以及企微探测不拨第二条 WSS。
5. 为 HTTP API 与后台表单覆盖企微凭证字段、Secret 掩码和健康状态映射。
6. 在 `comfy_mock` 下验证企微单聊和群 @ 创建任务、出图回传，以及超过被动回复窗口后仍主动投递成品。

## 风险与回滚

共享核心迁移的主要风险是 Telegram 行为回归；先迁移测试和保留 Telegram 端到端验证降低该风险。社区 Go SDK 的维护规模有限，因此只经本地接口使用并固定版本。单连接停止失败会暂时降低可用性，但比建立第二条连接并踢掉活动连接更安全。

回滚方式是停用企微渠道；适配器将关闭长连接，已存在的 Telegram 和飞书渠道不受影响。
