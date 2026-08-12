## ADDED Requirements

### Requirement: 投递前组装 job 并携带 job_ref
Orchestrator（或其 prep 步骤）在 Publish dispatch 之前 MUST 组装方案 A 任务包并获得 `job_ref`。dispatch 载荷 MUST 包含该 `job_ref`。成功主路径 MUST NOT 再依赖执行面读取业务 Case/Task 库拼装 workflow。该要求在 allinone 与 split 下均生效。

#### Scenario: 调度发出的 dispatch 含 job_ref
- **WHEN** 某 pending Task 被成功调度
- **THEN** 对应 dispatch 消息含有效 `job_ref`，且 Blob 中存在可读任务包

## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底。本期调度 MUST 按实例选择目标（健康/熔断/在线消费者等条件），在可选集合上使用轮询（round-robin）分配，向 `dispatch.<instance_id>`（或实例 DispatchTopic）Publish 含 `job_ref` 的 dispatch，并将 Task 置为 `queued`（幂等）。本期 MUST NOT 以实现投放表达式选 Topic 为验收条件。当没有可投递实例/消费者时 MUST 保持 Task 为 `pending` 并记录原因，MUST NOT 假装已排队。split 模式下 MUST NOT 假设云侧一定能直连家里 ComfyUI。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其投递（若存在可投递实例/消费者）

#### Scenario: 多健康实例轮询
- **WHEN** 存在至少两台健康且未熔断的可投递实例，连续调度多个 pending Task
- **THEN** 连续选定的 InstanceID 在轮询意义上分散到这些实例，而非每次都是同一台

#### Scenario: 按实例 Topic 投递
- **WHEN** 选定实例 `gpu-1` 且可投递
- **THEN** dispatch 发布到该实例 Topic，且含 `job_ref`

#### Scenario: 全部不可用时不投递
- **WHEN** 没有任何健康且未熔断的可投递实例（或 split 下无在线 Edge）
- **THEN** Task 仍保持 pending（或未被标记为 queued），且不向任意实例投递 dispatch

#### Scenario: 无可投递目标时不投递
- **WHEN** 无健康可投递实例（或 split 下无在线 Edge）
- **THEN** Task 仍保持 pending，且不假装成功投递
