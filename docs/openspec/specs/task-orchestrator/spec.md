# task-orchestrator Specification

## Purpose
TBD - created by archiving change workflow-engine-core. Update Purpose after archive.
## Requirements
### Requirement: ConfirmRun 创建异步 Task 并物化输入
系统 MUST 在校验通过后创建 Task（`pending`），将输入物化到对象存储，发布可编排信号（如 `task.created`），并立即向调用方返回 `task_id`，MUST NOT 在 ConfirmRun 调用栈内阻塞等待 ComfyUI 完成。

#### Scenario: 确认后立即返回任务号
- **WHEN** 用户 ConfirmRun 且校验通过
- **THEN** 系统持久化 pending Task，返回 task_id，并异步进入编排

### Requirement: Orchestrator 独占 Task 终态写入
系统 MUST 仅由 Orchestrator 将 Task 迁移至 `succeeded`/`failed`/`cancelled`（以及中间态 `queued`/`running` 的控制面确认）。Actuator 上报的 status 与对账结果 MUST 经统一幂等入口 `applyStatus` 写库。

#### Scenario: status 驱动成功
- **WHEN** Orchestrator 收到合法 succeeded status
- **THEN** Task 变为 succeeded，并触发用户通知意图

#### Scenario: 重复 status 不重复副作用
- **WHEN** 同一终态 status 被重复投递
- **THEN** Task 保持已收敛状态，且不重复发送终态通知

### Requirement: 调度与 pending 扫描双触发
系统 MUST 支持事件触发调度，并 MUST 提供定时扫描 `pending` 的兜底。调度 MUST 先求值 Case 路由规则确定目标 Topic（无命中回退默认 Topic），再仅在订阅该 Topic 的可用节点（健康/熔断/在线 Edge 等条件）上抢占领取，将 Task 置为可被该 Topic 订阅节点领取的状态（并准备含 `job_ref` 的任务包），MUST NOT 默认向 Redis `dispatch.<instance_id>` Publish 作为唯一派发手段。当目标 Topic 没有可投递节点/在线 Edge 时 MUST 保持 Task 为 `pending` 并记录原因，MUST NOT 假装已排队。MUST NOT 假设控制面一定能直连家里 ComfyUI。

#### Scenario: 创建事件丢失仍被扫到
- **WHEN** Task 长期处于 pending 且无成功调度
- **THEN** SchedulePending 仍能将其变为可领取（若目标 Topic 存在可投递节点/在线 Edge）

#### Scenario: 多健康实例轮询
- **WHEN** 目标 Topic 存在多个在线订阅节点并发领取
- **THEN** 领取按先到先得分散到这些节点，同一任务只被一台消费，而非预绑定轮询某台

#### Scenario: 按实例 Topic 投递
- **WHEN** pending Task 的路由规则命中 Topic `fast-gpu`
- **THEN** 任务置为可领取、`dispatch_topic=fast-gpu`、含有效 `job_ref`，且 Blob 中存在可读任务包

#### Scenario: 按实例可领取
- **WHEN** pending Task 的路由规则命中 Topic `fast-gpu`
- **THEN** 订阅 fast-gpu 的节点可领取该任务，且领取载荷含有效 `job_ref`

#### Scenario: 全部不可用时不投递
- **WHEN** 目标 Topic 没有任何可投递订阅节点（或无在线 Edge）
- **THEN** Task 仍保持 pending（或未被标记为可领取），且不向任意节点交付任务

#### Scenario: 无可投递目标时不投递
- **WHEN** 目标 Topic 无健康可投递订阅节点（或无在线 Edge）
- **THEN** Task 仍保持 pending，且不假装成功投递

#### Scenario: 条件路由到目标 Topic
- **WHEN** pending Task 的路由规则命中 Topic `fast-gpu`
- **THEN** 任务置为可领取、`dispatch_topic=fast-gpu`、含有效 `job_ref`，且 Blob 中存在可读任务包

#### Scenario: 同 Topic 多节点并发领取互斥
- **WHEN** 目标 Topic 存在多个在线订阅节点并发领取同一个任务
- **THEN** 恰好一台节点成功领取（带租约），其余获得空结果，任务不被重复分配

#### Scenario: 目标 Topic 全部不可用时不投递
- **WHEN** 目标 Topic 没有任何可投递订阅节点（或无在线 Edge）
- **THEN** Task 仍保持 pending（或未被标记为可领取），且不向任意节点交付任务

### Requirement: Status 丢失时对账兜底
对处于 `queued`/`running` 且超时的 Task，Orchestrator MUST 通过 ExecutionQuery（或等价真相源）探测执行结果，并经 `applyStatus` 补写；多次失败或超过最大存活时间 MUST 收口为 failed 并通知。

#### Scenario: 对账补写成功
- **WHEN** status 丢失但 Query 显示已成功
- **THEN** Task 被标记 succeeded 并通知用户

### Requirement: 温和取消
系统 MUST 允许取消仍处于 `pending` 或 `queued` 的 Task。MUST NOT 在本期对 `running` 任务执行 ComfyUI interrupt。

#### Scenario: 取消排队中任务
- **WHEN** 用户取消 queued Task
- **THEN** Task 变为 cancelled，且不再被 Actuator 成功执行（或执行结果被控制面忽略为取消策略所定义）

### Requirement: 调用风暴防护
Orchestrator MUST 对调度、对账、重派、实例探测与通知实施限流、退避（含抖动）、错误分类、实例熔断，以及对账与调度分池，避免大面积故障时放大重试。

#### Scenario: 实例连续失败触发熔断
- **WHEN** 某实例连续调度/探测失败超过阈值
- **THEN** 系统在熔断期内停止向该实例投递新任务，并记录可观测状态

### Requirement: 通过 notify 驱动渠道回用户
Task 到达终态（及可选进度）时，Orchestrator MUST 发布与渠道无关的 notify 意图；MUST NOT 在 Orchestrator 内直接调用 Telegram API。

#### Scenario: 成功后发出通知意图
- **WHEN** Task 变为 succeeded
- **THEN** 系统发布含 chat_id、task_id、输出引用的 notify，供渠道模块投递

### Requirement: 投递前组装 job 并携带 job_ref
Orchestrator（或其 prep 步骤）在使任务可被 Edge 领取之前 MUST 组装方案 A 任务包并获得 `job_ref`。领取下发的任务描述 MUST 包含该 `job_ref`。成功主路径 MUST NOT 再依赖执行面读取业务 Case/Task 库拼装 workflow。该要求在本机与远程部署下均生效。

#### Scenario: 调度发出的 dispatch 含 job_ref
- **WHEN** 某 pending Task 被成功调度为可领取
- **THEN** 对应领取载荷含有效 `job_ref`，且 Blob 中存在可读任务包

#### Scenario: 可领取任务含 job_ref
- **WHEN** 某 pending Task 被成功调度为可领取
- **THEN** 对应领取载荷含有效 `job_ref`，且 Blob 中存在可读任务包

