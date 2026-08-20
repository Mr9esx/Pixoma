## REMOVED Requirements

### Requirement: 本期不实现投放表达式选路
**Reason**: 本期交付按条件表达式选 Topic 投放，延后声明作废。
**Migration**: 以本 delta 新增的「按条件表达式选 Topic 投放」要求为准。

## ADDED Requirements

### Requirement: 按条件表达式选 Topic 投放
调度 MUST 在任务调度时求值 Case 路由规则，选中目标 Topic，并仅使订阅该 Topic 的节点可领取；求值结果 MUST 记录在 Task（`dispatch_topic`）。无规则命中时 MUST 回退默认 Topic。

#### Scenario: 条件命中高规格 Topic
- **WHEN** 任务上下文命中 Case 规则指向 Topic `fast-gpu`
- **THEN** Task 的 `dispatch_topic=fast-gpu`，且只有订阅 fast-gpu 的节点可领取

#### Scenario: 回退默认 Topic
- **WHEN** 任务上下文不命中任何规则
- **THEN** Task 的 `dispatch_topic=default`，默认 Topic 订阅者可领取

#### Scenario: 求值错误保持 pending
- **WHEN** 规则求值出现错误（如 provider 读取用户属性失败）
- **THEN** Task 保持 pending 并记录原因，不投递到默认 Topic，后续调度周期重试
