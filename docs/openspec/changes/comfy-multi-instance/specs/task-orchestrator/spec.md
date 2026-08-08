## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底，以免创建事件丢失导致任务无人认领。调度选择实例时 MUST 仅从健康且未被熔断禁止的实例中选择，并 MUST 在可选集合上使用轮询（round-robin）分配，避免总是固定选择同一台；选定后 MUST 投递 `dispatch.<instance_id>` 并将 Task 置为 `queued`（幂等）。当没有任何可选实例时 MUST 保持 Task 为 `pending`（或等价未投递状态）并记录可诊断原因，MUST NOT 假装已排队到某实例。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其投递（若实例可用）

#### Scenario: 多健康实例轮询
- **WHEN** 存在至少两台健康且未熔断的实例，连续调度多个 pending Task
- **THEN** 连续选定的 InstanceID 在轮询意义上分散到这些实例，而非每次都是同一台

#### Scenario: 全部不可用时不投递
- **WHEN** 没有任何健康且未熔断的实例
- **THEN** Task 仍保持 pending（或未被标记为 queued），且不向任意实例投递 dispatch
