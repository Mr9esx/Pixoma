## ADDED Requirements

### Requirement: Edge 长轮询领取任务
系统 MUST 提供 Edge 可调用的控制面 Agent API：Edge MUST 能对指定 `instance_id` 发起长轮询（或等价等待）以领取待执行任务；成功领取 MUST 返回含合法 `job_ref` 的任务描述，并将该 Task 置于带租约的已领取状态。默认跨进程派发 MUST NOT 依赖 Redis Streams。

#### Scenario: 在线 Edge 领到可执行任务
- **WHEN** 控制面存在可调度到某实例的待领取任务，且该实例 Edge 正在长轮询领取
- **THEN** Edge 获得含 `job_ref` 的任务，且控制面记录领取租约与实例归属

#### Scenario: 无任务时等待后空返回
- **WHEN** 该实例暂无待领取任务，Edge 发起带等待时限的领取请求
- **THEN** 在等待时限内若仍无任务则返回空结果且不报错为系统故障

### Requirement: 租约、心跳与回收
已领取任务 MUST 具备租约到期时间；Edge MUST 能通过心跳延长租约或申报在线。租约过期且未终态时，系统 MUST 使任务重新可被领取（或按策略失败收口），MUST NOT 永久卡在已领取态。

#### Scenario: Edge 宕机后任务可再领
- **WHEN** 任务已被领取但 Edge 在租约内未续约且未上报终态
- **THEN** 租约到期后任务可被同一或其他合格实例再次领取（或进入产品定义的失败收口）

### Requirement: Edge 回报状态
Edge MUST 能通过控制面 API 上报执行状态（至少 running 与终态）；控制面 MUST 经既有幂等 status 入口更新 Task，MUST NOT 要求 Edge 连接业务库拼装工作流。

#### Scenario: 上报成功终态
- **WHEN** Edge 完成执行并上报合法 succeeded status（含产物 BlobRef）
- **THEN** Task 收敛为成功并触发既有通知意图路径
