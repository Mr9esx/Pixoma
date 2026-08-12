# dispatch-topic-routing Specification

## Purpose
TBD - created by archiving change edge-agent-topic-routing. Update Purpose after archive.
## Requirements
### Requirement: 本期不实现投放表达式选路
本期实现 MUST NOT 将「用户会员等级 × Case 分类投放表达式」作为调度选路的验收路径。调度 MUST 继续按实例维度选择目标，并向 `dispatch.<instance_id>`（或实例配置的 DispatchTopic）投递。Topic 表达式分流延后到后续 change。

#### Scenario: 调度按实例 Topic 投递
- **WHEN** 某 pending Task 被成功调度到实例 `gpu-1`
- **THEN** dispatch 发布到该实例对应 Topic，且不依赖会员/分类表达式求值

### Requirement: Topic 无消费者时不假装投递成功
当目标实例的 dispatch Topic 在 split 模式下无在线 Edge（或等价消费者）时，系统 MUST 保持 Task 可重试并记录原因，MUST NOT 假装已可靠交付执行。

#### Scenario: split 下无在线 Edge
- **WHEN** runtime_mode 为 split 且目标实例无在线 Edge
- **THEN** Task 保持 pending（或未被标记为已可靠 queued）

