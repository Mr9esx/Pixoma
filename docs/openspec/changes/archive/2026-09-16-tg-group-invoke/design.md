## Context

见 `proposal.md` 的 Why。当前 `Adapter.account()` 把 `ExternalChatID` 当作 `ExternalUserID`，会话按 `channelID:telegramChatID` 上锁。私聊时 chat.id == from.id，能工作；群里两者不同。入站不看 `Chat.Type`。Notify 按 ChatID 投递，群触发时只要 ChatID 仍是群，回图方向是对的——先修身份与过滤，不要把 ChatID 改成 user id。

## Goals / Non-Goals

**Goals:**

- 群点名可跑工作流，图回该群。
- 会话锁按「群 + 人」；私聊锁不变。
- 入站过滤可测：闲聊进不来。

**Non-Goals:**

- 本文件不设计飞书/企微适配器。
- 不在本 change 扩凭证模型或工厂按平台分支。

## Decisions

### 1. 聊天地址与用户身份分成两个字段

- **选择**：`ChannelAddr` / notify 的 ChatID 继续表示「把消息发到哪」（群或私聊）。`AccountCtx.ExternalUserID` 只用 `from.id`（callback 用 `from`）。群会话键 = `channelID + chatID + userID`。
- **理由**：回图必须回群；账户与锁必须跟人走。
- **备选**：整群一把锁——两人会互抢。只把 ChatID 改成 user id——图会打进私聊，违背「发回这个群」。

### 2. 群填表走「回复 bot」，不强制私聊

- **选择**：缺参时在群里问下一题；只接受该操作者对 bot 消息的回复（或再次 @）。Telegram 默认隐私模式本来就只把命令、@、回复 bot 交给 bot，与点名规则一致。
- **理由**：用户要成品回群；把填表赶到私聊会拆成两条会话，notify 还要决定回哪。
- **备选**：一条 @ 必须带齐参数，否则失败——工作流常有图字段，群里回图给 bot 更实际。强制私聊补全——下次再做，不写入本 change 验收。

### 3. 命令形态最小可用

- **选择**：`/run <case_id或名称>` 与 `@bot <名称或提示>` 两种；名称匹配已启用 Case（精确优先，否则唯一前缀）。群里不发 ReplyKeyboard。
- **理由**：不必先做群菜单编辑器。
- **备选**：群里复用底下键盘——群体验差且会误吃输入，已排除。

### 4. 隐私模式保持默认开

- **选择**：不要求运营关掉 Group Privacy Mode；实现不依赖「收群里每一句」。
- **理由**：与「只处理点名」一致，也减少误触发。
- **备选**：文档要求关隐私模式收全量群消息——和闲聊忽略冲突。

## Risks / Trade-offs

- [群刷屏] 逐步填表会在群里多几条 bot 消息 → 只回复操作者、文案短；不在群里重发主菜单。
- [历史会话] 已用群 chat id 当用户写入的 `users` 行可能脏 → 新交互按 from.id upsert；不自动合并旧行。
- [超大群] 无白名单时任何加了 bot 的群都能调 → 接受为 v1；运营可用停用消息平台止血。

## Migration Plan

- 兼容私聊：私聊路径测试必须保持绿。
- 回滚：关掉群分支（仅 private）即可回到「群消息可能误触发但身份仍错」的旧行为；完整回滚需还原会话键。

## Open Questions

无。填表留在群里用回复 bot，不阻塞本 change 的规格与任务拆分。
