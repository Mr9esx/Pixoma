# dialog-session Specification

## Purpose
TBD - created by archiving change workflow-engine-core. Update Purpose after archive.
## Requirements
### Requirement: 每聊天最多一个进行中的填表会话
系统 MUST 以 Telegram `chat_id` 为键，保证同一时刻最多一个非终态 Dialog Session。浏览菜单与 Case 列表 MUST NOT 创建 Session。仅 `StartCase` 成功后进入 `collecting` 并上锁。

#### Scenario: 浏览菜单不创建会话
- **WHEN** 用户仅打开菜单或 Case 列表
- **THEN** 系统不创建 Dialog Session，用户仍可自由导航

#### Scenario: StartCase 上锁
- **WHEN** 用户对某 active Case 执行 StartCase 且当前无进行中 Session
- **THEN** 系统创建 `collecting` 会话并禁止再次 StartCase，直至退出或提交

### Requirement: 上锁期间拦截新 Case 并提供退出
当 Session 处于 `collecting` 或 `confirming` 时，系统 MUST 拒绝新的 StartCase，并返回当前 Case 摘要及「继续 / 退出并重选」类选项语义。用户执行 ExitSession 后，会话 MUST 进入 `exited`（或等价清除），允许新开 Case。

#### Scenario: 上锁时开新 Case 被拒绝
- **WHEN** 用户已在 collecting 中又请求 StartCase
- **THEN** 系统返回锁定错误/视图，不创建第二个 Session

#### Scenario: 退出后可重新开始
- **WHEN** 用户 ExitSession 后再 StartCase
- **THEN** 系统允许创建新的 collecting 会话

### Requirement: 确认执行后结束填表会话
ConfirmRun 成功创建 Task 后，系统 MUST 结束填表 Session（`submitted` 或清除 active），从而解锁。进行中的执行 Task MUST NOT 阻止用户开启新的填表 Session。

#### Scenario: 提交后解锁
- **WHEN** ConfirmRun 成功
- **THEN** 该 chat 不再处于填表锁，可 StartCase

#### Scenario: 生成中可开新 Case
- **WHEN** 用户已有 running Task 且无填表 Session
- **THEN** StartCase 被允许

### Requirement: Session 草稿支持图片 Blob 值
对话会话 MUST 允许将某个 input 键的 Draft 存为媒体 Blob 引用（与文本/数值/布尔并存）。跳过规则对允许跳过的非必填图片字段仍然有效；必填图片缺少 Blob 时不得进入可成功 Confirm 的状态。

#### Scenario: 提交图片草稿后索引前进
- **WHEN** 会话处于 collecting，当前键类型语义为图片，且提交带 Blob 的 Draft
- **THEN** 该键被记录，会话前进到下一输入或 confirming

#### Scenario: 跳过可选图片字段
- **WHEN** 当前键为非必填且允许跳过的图片字段，用户选择跳过
- **THEN** 会话不要求该 Blob，并前进到下一输入或 confirming

