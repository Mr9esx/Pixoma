## Context

见 `proposal.md` 的 Why。规格已要求按平台装配，但 `tgChannelFactory.Create` 不看 `snap.Platform`，凭证只有 `BotToken`，探测一律 getMe，Admin 创建表单写死 telegram。飞书官方 Go SDK 提供 `larkws` 长连接与发消息/上传图片。卡片须用新版 `card.action.trigger`（旧版不支持长连接）。

## Goals / Non-Goals

**Goals:**

- 工厂/凭证/探测/表单可插飞书。
- 本机长连接跑通「点工作流 → 回图」。

**Non-Goals:**

- 不在本 change 实现企微协议。
- 不设计 MCP 执行门面。

## Decisions

### 1. 先改管道再写适配器

- **选择**：同一 change 内先让工厂按 platform 分支、凭证改为按平台结构、探测接口化，再实现 `internal/channels/feishu`。
- **理由**：只写飞书包而工厂仍 `bot.New(token)`，启用飞书会直接崩。
- **备选**：管道单独 change——后台能选飞书但点启用即坏，中间态不可用。

### 2. 出站用官方 Open API，入站用 SDK 长连接

- **选择**：入站 `larkws`；发送/上传走 `oapi-sdk-go`。回调 handler 只做翻译与 `CapabilityInvoke`，立即返回。
- **理由**：与 TG「适配器翻译、应用层不引用 SDK」一致。
- **备选**：自写 WS 协议——无必要。HTTP 事件订阅——违反本机无公网前提。

### 3. 菜单映射为卡片，不接 ReplyKeyboard 语义

- **选择**：`SendMenu` / `SendList` 在飞书渲染为卡片按钮；主入口可用欢迎卡片或会话内卡片。能力 id 与 TG 相同（`open_case`）。
- **理由**：飞书没有常驻 ReplyKeyboard。
- **备选**：命令-only——能跑但不接已有菜单配置，运营要配两套入口。

### 4. 身份

- **选择**：`external_user_id` = 飞书 user_id/open_id（实现期锁定一种并贯穿）；投递地址用 chat_id。群与人分开，沿用 `tg-group-invoke` 的拆分。
- **理由**：飞书天然 chat ≠ user。
- **备选**：继续用 chat 当 user——群会再绑错。

## Risks / Trade-offs

- [3 秒超时重推] → 回调只入队/调能力启动，禁止等待 Task。
- [多进程抢连接] 长连接集群只推一个 client → 文档写明单实例；Assembler 保证停旧再启新。
- [权限漏配] 只开 im:message 收不到单聊 → 创建/探测文案提示必开 `p2p_msg` 与群 @ 权限（界面只字段/错误，不写教程长文）。

## Migration Plan

- Telegram 回归：工厂默认分支保持 TG。
- 回滚：停用飞书消息平台；代码回滚工厂分支。

## Open Questions

无。欢迎卡片 vs 仅卡片菜单属于飞书适配器内渲染选择，不改变规格。
