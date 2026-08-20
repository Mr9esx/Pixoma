# task-persistence Specification

## Purpose
将生成工单（Task）持久化到数据库，并通过 `session_id` 关联填表会话，从而间接关联 User 与 chat；支持按实例查询已派发任务。
## Requirements
### Requirement: Task 持久化
系统 MUST 将 Task 持久化到数据库（替换仅内存实现作为主路径）。记录 MUST 至少包含：task id、`session_id`、`case_id`、状态、可选 `instance_id` / `prompt_id`、输入前缀、产物与错误信息、时间戳。进程重启后 MUST 仍能按 id 读出未完成与历史任务。

#### Scenario: 重启后 Task 仍在
- **WHEN** 已创建的 Task 写入数据库后进程重启
- **THEN** 仍可按 task id 读出其状态与 session_id

### Requirement: Task 通过 Session 关联用户与聊天
创建 Task 时 MUST 写入有效的 `session_id`。Task 行 MUST NOT 将 `user_id` / `chat_id` 作为必需冗余字段；需要通知聊天或用户归属时，系统 MUST 通过 Session（及 User）关联获得。`ListMyTasks` 或等价「我的任务」查询 MUST 在持久化模型上仍按用户或聊天可列出（允许 join）。

#### Scenario: ConfirmRun 写入 session_id
- **WHEN** ConfirmRun 成功创建 Task
- **THEN** 该 Task 的 session_id 指向当次确认所用 Session

#### Scenario: 经 Session 解析通知 chat
- **WHEN** 任务成功需要向 TG 发通知
- **THEN** 系统能通过 Task.session_id → Session.chat_id 得到投递目标（或不依赖已删除的 Session）

### Requirement: 派发后可按 instance 查询
当 Task 被调度并写入 `instance_id` 后，系统 MUST 支持按 `instance_id` 列出这些任务，供实例观测 API 使用。

#### Scenario: 按实例列出已派发任务
- **WHEN** Task 的 instance_id 为 `gpu-1`
- **THEN** 按实例 `gpu-1` 的任务查询包含该 Task

### Requirement: Task 为执行态唯一真相源
系统 MUST 将 Comfy 执行相关的可恢复状态（含 `prompt_id`、任务状态、产物与错误）持久在 Task 上。系统 MUST NOT 再依赖独立的 Actuator Ledger（`LocalRun` / `MemoryLedger`）作为执行进度或对账的真相源。Orchestrator 对账与执行态查询 MUST 读取 Task 仓储（或以其为唯一后端的适配）。

#### Scenario: 无 Ledger 仍可对账
- **WHEN** 进程内未使用 MemoryLedger，且 Task 已写入 prompt_id 与 running/终态
- **THEN** Orchestrator 仍能仅通过 Task 仓储完成对账或执行态查询

### Requirement: Task 记录投递 Topic 与重试/租约信息
Task 持久化记录 MUST 包含：`dispatch_topic`（实际路由 Topic）、重试次数、当前领取节点与租约截止时间（可复用既有字段语义）。进程重启后 MUST 仍能依据这些字段恢复调度（任务按其 Topic 重新可领取）。

#### Scenario: 重启后按 Topic 恢复
- **WHEN** 进程重启，存在 `dispatch_topic=fast-gpu` 且状态为 pending/可领取的任务
- **THEN** 订阅 fast-gpu 的节点仍可领取该任务

