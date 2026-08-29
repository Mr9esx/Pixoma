## Purpose

让「快速配置」向导在选择「哪台电脑运行工作流」时复用或新建 Edge 节点作为运行节点，并在无需特殊规则时把该节点直接绑定 default Topic，使该 Case 回退 default 的任务能被该节点消费。

## ADDED Requirements

### Requirement: 运行节点选择
向导 MUST 提供「哪台电脑运行你的工作流」步骤，用户 MUST 可选择已有 Edge 节点或新建节点（复用既有节点表单与部署凭证流程）；本步 MUST 仅把选定节点记录为运行节点的向导状态，MUST NOT 写入 Case 模型，MUST NOT 立即持久化节点订阅。

#### Scenario: 选择已有节点
- **WHEN** 用户在运行节点步骤选择一个已有 Edge 节点并推进
- **THEN** 该节点被记录为向导的运行节点，不产生任何持久化副作用

#### Scenario: 新建节点并选定
- **WHEN** 没有匹配节点时用户走新建流程完成注册并部署就绪
- **THEN** 新建节点被记录为运行节点，部署就绪状态对用户可见

### Requirement: Default 分支绑定与派发
「不需要特殊规则」时，向导 MUST 对选定运行节点执行 `PATCH /edges/{id}`，将节点 `subscribe_topics` 加入 `default`（复用该项既有 default 订阅能力），使该 Case 回退 default 的任务由订阅 default 的节点参与竞争消费；向导 MUST NOT 引入 Case 级节点归属字段。

#### Scenario: 默认分支绑定 default
- **WHEN** 用户选择「不需要特殊规则」并在完成页提交
- **THEN** 选定节点 `subscribe_topics` 被写入包含 `default`，任务回退 default 可由该节点消费

#### Scenario: 竞争消费语义
- **WHEN** 另一台节点原本就订阅 default
- **THEN** 该 Case 的回退 default 任务亦可被其消费（非独占），不因本次绑定而受影响
