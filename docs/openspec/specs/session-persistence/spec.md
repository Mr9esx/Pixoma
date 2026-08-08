# session-persistence Specification

## Purpose
将 Case 填表会话（Session）持久化到数据库，并与 User、聊天上下文关联；作为 Task 创建前的输入采集态，生命周期不包含 Task 执行阶段。
## Requirements
### Requirement: Session 持久化
系统 MUST 将 Session 持久化到数据库（替换仅内存实现作为主路径）。记录 MUST 至少包含：session id、关联的 `user_id`、`chat_id`、`case_id`、状态、当前输入进度、草稿输入、时间戳。进程重启后 MUST 仍能按既有查询方式恢复活跃会话（若状态仍为活跃）。

#### Scenario: 重启后活跃 Session 可恢复
- **WHEN** 用户处于 collecting 的 Session 已写入数据库后进程重启
- **THEN** 按该 chat（及实现约定的活跃查询）仍能读出该 Session 及其草稿进度

### Requirement: Session 同时关联 User 与 Chat
每条 Session MUST 同时持久化 `user_id` 与 `chat_id`。系统 MUST NOT 仅用 chat_id 推断用户身份而不写入 `user_id`。

#### Scenario: 创建 Session 时写入 user_id
- **WHEN** 已 upsert 的用户在某 chat 开始 Case
- **THEN** 新建 Session 的 user_id 指向该用户，chat_id 为该聊天

### Requirement: Session 与 Task 生命周期分离且可被 Task 引用
Session 状态机 MUST 表示填表过程（如 collecting / confirming / submitted / exited），MUST NOT 用 Session 状态表示 Comfy 执行中的 running。当 ConfirmRun 成功创建 Task 后，Session MUST 进入已提交（或等价终态），且该 Session 行 MUST 保持可查询以便 Task 通过 `session_id` 关联。系统 MUST NOT 在 Task 仍可能引用时物理删除对应 Session。

#### Scenario: 确认后 Session 结束填表且行仍在
- **WHEN** ConfirmRun 成功
- **THEN** Session 不再处于活跃填表态，且数据库中仍能按 session id 读出该行供 Task 关联

