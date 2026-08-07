## ADDED Requirements

### Requirement: TG 适配器将应用 DTO 渲染为 Bot API 消息
系统 MUST 提供 Telegram 适配器：接收 Bot Update，调用渠道无关的应用用例，并将菜单/Case 列表/会话提示/错误/结果渲染为 Telegram 支持的消息形态（文本、Photo、InlineKeyboard 等）。应用层 MUST NOT 依赖 Telegram SDK 类型。

#### Scenario: Case 列表以按钮呈现
- **WHEN** 用户进入某分类的 Case 列表
- **THEN** 适配器发送包含 Case 入口的 InlineKeyboard（可分页）

#### Scenario: 完成后发送图片结果
- **WHEN** 适配器收到 succeeded 的用户通知且输出含 image BlobRef
- **THEN** 适配器向对应用户发送图片（或等价媒体消息）

### Requirement: 适配器执行 Session 锁拦截文案
当应用层返回会话锁定时，适配器 MUST 向用户展示当前流程提示，并提供继续与退出当前流程的操作入口。

#### Scenario: 锁定时展示退出选项
- **WHEN** 用户在填表中尝试开新 Case
- **THEN** 用户收到锁定说明及退出/继续类可点击操作

### Requirement: 消费 notify 完成对用户投递
适配器 MUST 订阅或接收 Orchestrator 的 notify 意图，并完成对 `chat_id` 的消息投递。终态通知 MUST 幂等处理，避免重复刷屏。

#### Scenario: 重复 notify 不重复刷终态消息策略
- **WHEN** 同一 task 终态 notify 重复到达
- **THEN** 适配器不产生重复的骚扰性终态推送（按去重策略合并或忽略）
