# agent-pull-dispatch Specification

## Purpose
TBD - created by archiving change pixoma-guided-deploy. Update Purpose after archive.
## Requirements
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

### Requirement: 同一任务只被一台节点消费
控制面 MUST 保证多个节点并发向同一 Topic 领取时，同一 Task 最多被一台节点成功领取（原子抢占）；领取成功后 MUST 记录持有节点与租约。并发竞争落败的领取 MUST 返回空结果（或等价 no-content），MUST NOT 重复发放同一任务。

#### Scenario: 并发领取互斥
- **WHEN** 两台节点同时向同一 Topic 领取且该 Topic 只有一个可领取任务
- **THEN** 恰好一台节点成功领取；另一台获得空结果，且该任务未被重复分配

### Requirement: 失败重回 Topic 与有界重试
节点执行失败（终态 failed）时，任务 MUST 按有界重试策略重新进入其 `dispatch_topic` 可被再次领取；重试次数达到上限后 MUST 收敛为最终 failed 并记录原因。节点领取后长时间无进展（未续约且租约过期、无终态上报）时，任务 MUST 被回收重新可领取或按策略失败收口，MUST NOT 永久卡在已领取态。

#### Scenario: 失败后重回 Topic
- **WHEN** 节点上报 failed 且重试次数未达上限
- **THEN** 任务重新进入 dispatch_topic 可领取队列，重试计数 +1

#### Scenario: 节点宕机租约回收
- **WHEN** 任务已被领取、节点未续约、租约过期且无终态上报
- **THEN** 任务重新可被领取（或按重试上限失败收口）

#### Scenario: 超限失败收口
- **WHEN** 任务重试次数达到上限
- **THEN** 任务收敛为 failed 并记录含重试次数的原因

