## ADDED Requirements

### Requirement: Task 记录投递 Topic 与重试/租约信息
Task 持久化记录 MUST 包含：`dispatch_topic`（实际路由 Topic）、重试次数、当前领取节点与租约截止时间（可复用既有字段语义）。进程重启后 MUST 仍能依据这些字段恢复调度（任务按其 Topic 重新可领取）。

#### Scenario: 重启后按 Topic 恢复
- **WHEN** 进程重启，存在 `dispatch_topic=fast-gpu` 且状态为 pending/可领取的任务
- **THEN** 订阅 fast-gpu 的节点仍可领取该任务
