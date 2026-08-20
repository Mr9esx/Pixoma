## MODIFIED Requirements

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底。调度 MUST 先求值 Case 路由规则确定目标 Topic（无命中回退默认 Topic），再仅在订阅该 Topic 的可用节点（健康/熔断/在线 Edge 等条件）上抢占领取，将 Task 置为可被该 Topic 订阅节点领取的状态（并准备含 `job_ref` 的任务包），MUST NOT 默认向 Redis `dispatch.<instance_id>` Publish 作为唯一派发手段。当目标 Topic 没有可投递节点/在线 Edge 时 MUST 保持 Task 为 `pending` 并记录原因，MUST NOT 假装已排队。MUST NOT 假设控制面一定能直连家里 ComfyUI。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其变为可领取（若目标 Topic 存在可投递节点/在线 Edge）

#### Scenario: 条件路由到目标 Topic
- **WHEN** pending Task 的路由规则命中 Topic `fast-gpu`
- **THEN** 任务置为可领取、`dispatch_topic=fast-gpu`、含有效 `job_ref`，且 Blob 中存在可读任务包

#### Scenario: 同 Topic 多节点并发领取互斥
- **WHEN** 目标 Topic 存在多个在线订阅节点并发领取同一个任务
- **THEN** 恰好一台节点成功领取（带租约），其余获得空结果，任务不被重复分配

#### Scenario: 目标 Topic 全部不可用时不投递
- **WHEN** 目标 Topic 没有任何可投递订阅节点（或无在线 Edge）
- **THEN** Task 仍保持 pending（或未被标记为可领取），且不向任意节点交付任务
