## Why

Telegram 今天按私聊写：把聊天 ID 当成操作者，群里一人开工作流会锁整群，闲聊也可能被当成菜单。路线图要「群组调用」：人在群里 @bot 或发命令就能跑已有工作流，成品发回这个群；私聊菜单不变。

## What Changes

- 拆开 **群（chat）** 和 **人（from）**：账户用操作者 user id；会话锁在群里按「这个群 + 这个人」。
- 群里只处理被点名的消息：`@bot`、以 `/` 开头的命令、回复 bot 的消息。未点名闲聊 MUST NOT 进 Pixoma。
- 群里不用 ReplyKeyboard。入口是命令或 @；缺参时在群里短问或引导去私聊补，成品仍发回该群。
- 任务终态 notify 投递到发起时所在的群，而不是误投到操作者私聊（除非本次就是私聊）。
- 私聊路径（`/start`、底下键盘、填表）行为 MUST 与改前一致。

### 非目标

- 不做群里完整底下键盘菜单。
- 不做飞书、企微、MCP（见同批 `feishu-channel` / `wecom-aibot-channel` / `mcp-call-workflow`）。
- 不改 Task 调度/Comfy 执行。
- 不做群白名单管理后台（若实现期需要最小「忽略未知群」开关，不得做成完整运营产品）。

## Capabilities

### New Capabilities

- `tg-group-invoke`：群聊触发规则、群会话键、群结果投递。

### Modified Capabilities

- `channel-tg`：入站 MUST 区分 private / group / supergroup；身份 MUST 用操作者而不是 chat id；群里 MUST NOT 发 ReplyKeyboard。
- `dialog-session`：填表锁键从「仅 chat_id」改为私聊仍按聊天、群里按「聊天 + 操作者」。
- `channel-account-context`：Telegram 外部用户 id MUST 是操作者 user id，群 chat id MUST NOT 再当用户 id。

## Impact

- `internal/channels/tg`：bot 入站过滤、`account()` 身份、群消息路由、notify 目标。
- `internal/sessions` / dialog 键：会话主键语义。
- `internal/users`：Upsert 用 from.id。
- 架构文档：`docs/architecture/runtime.md`（入站身份与 notify 地址）。
- Mock：`comfy_mock` 下群触发主路径仍须出图。
