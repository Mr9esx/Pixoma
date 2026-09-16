# tg-group-invoke Specification

## Purpose

让人在 Telegram 群里用 @bot 或命令调用已有工作流，成品发回这个群；未点名的闲聊不进 Pixoma。

## Requirements

### Requirement: 群触发仅限点名

在 group / supergroup 中，系统 MUST 仅处理：提及本 bot、以 `/` 开头且指向本 bot 的命令、回复本 bot 消息的更新。未点名的普通文本与媒体 MUST NOT 进入能力调用或主菜单。channel 类型的广播聊天 MUST NOT 当作群调用入口。

#### Scenario: @bot 开工作流

- **WHEN** 群成员发送提及本 bot 的文本且能解析到一个已启用工作流
- **THEN** 系统为该操作者启动该工作流，并在该群回复进度或结果

#### Scenario: 闲聊忽略

- **WHEN** 群成员发送未提及 bot、也不是命令、也不是回复 bot 的文本
- **THEN** 系统不创建会话、不回主菜单、不调用工作流

#### Scenario: 频道消息忽略

- **WHEN** 更新来自 Telegram channel 而非 group/supergroup
- **THEN** 系统不按群调用处理该更新

### Requirement: 群结果发回发起群

从群触发并成功跑完的任务，其终态媒体与状态文案 MUST 投递到发起时所在的群。MUST NOT 改投到操作者私聊，除非本次触发发生在私聊。

#### Scenario: 成品回群

- **WHEN** 群成员通过点名成功 ConfirmRun，任务输出含图片
- **THEN** 该群收到该图片（或等价媒体），操作者私聊不因此多一条终态图

#### Scenario: 私聊仍回私聊

- **WHEN** 用户在私聊 ConfirmRun 成功且输出含图片
- **THEN** 图片发到该私聊，MUST NOT 发到该用户所在的任意群
