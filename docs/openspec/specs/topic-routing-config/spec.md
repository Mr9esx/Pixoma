# topic-routing-config Specification

## Purpose
定义 Case 与 Topic 之间的路由投放配置：一个 Case 声明一对 N 目标 Topic 与条件规则、首个命中即投、无命中回退默认 Topic，并规定配置的持久化与校验契约。
## Requirements
### Requirement: Case 路由配置结构
Case 路由配置 MUST 包含有序规则列表（每条 = 条件 + 目标 Topic key）；规则求值 MUST 采用首个命中即投；无任何规则命中或未配置路由时 MUST 回退系统默认 Topic（Case 不单独声明默认 Topic）。Case 可配置的 Topic 数量 MUST 为一个到多个。

#### Scenario: 首个命中即投
- **WHEN** Case 配置规则为 [条件1→Topic A, 条件2→Topic B] 且任务上下文同时满足两条规则
- **THEN** 任务投递到第一条命中规则的目标 Topic A

#### Scenario: 无命中回退默认
- **WHEN** 任务上下文不满足任何规则（或 Case 未配置路由）
- **THEN** 任务投递到默认 Topic

### Requirement: 路由配置校验
路由配置 MUST 在保存时校验：目标 Topic 必须存在且启用；条件必须通过条件协议校验；每条规则 MUST 含条件与目标 Topic；Case 至少有一条可达路径（规则命中或默认 Topic）。

#### Scenario: 引用不存在的 Topic 被拒绝
- **WHEN** 保存引用不存在或已禁用 Topic 的路由规则
- **THEN** 保存被拒绝并返回可定位错误

