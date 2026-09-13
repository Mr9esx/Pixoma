## ADDED Requirements

### Requirement: 入站区分私聊与群
Telegram 适配器 MUST 按聊天类型分支：private 走既有菜单与填表；group / supergroup 走群点名规则。适配器 MUST 使用消息的操作者 user id 作为外部用户 id，MUST NOT 再用群 chat id 充当用户 id。

#### Scenario: 私聊身份仍是对话对方
- **WHEN** 私聊收到文本或回调查询
- **THEN** 外部用户 id 等于该私聊 chat id（与 Telegram 私聊 chat.id == from.id 一致）

#### Scenario: 群身份是点按钮或发消息的人
- **WHEN** 群里两人先后与 bot 交互
- **THEN** 系统分别映射为两个消息平台账户，互不覆盖

### Requirement: 群里不发 ReplyKeyboard
系统 MUST NOT 向 group / supergroup 发送 ReplyKeyboard 主菜单。群入口 MUST 通过点名、命令或回复 bot 完成。

#### Scenario: 群 @ 不弹出底下键盘
- **WHEN** 群成员首次 @bot
- **THEN** 回复为文本、命令提示或 inline/卡片类控件，不带 ReplyKeyboard

#### Scenario: 私聊仍有底下键盘
- **WHEN** 用户在私聊发送 `/start`
- **THEN** 仍展示由该消息平台菜单配置生成的 ReplyKeyboard
