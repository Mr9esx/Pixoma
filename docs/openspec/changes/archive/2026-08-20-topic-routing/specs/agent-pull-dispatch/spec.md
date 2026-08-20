## ADDED Requirements

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
