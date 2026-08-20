## REMOVED Requirements

### Requirement: 本期不实现 Topic 与投放规则管理验收
**Reason**: 本期交付 Topic 目录管理与投放规则管理，延后声明作废。
**Migration**: 以本 delta 新增的「Topic 管理与默认 Topic」要求为准。

## ADDED Requirements

### Requirement: Topic 管理与默认 Topic
系统 MUST 提供 Topic 的创建、查询（单个与列表）、更新、启用/禁用。Topic MUST 包含稳定 `key`、显示名、启用状态与创建时间。应用启动初始化 MUST 幂等确保默认 Topic（`default`）存在；默认 Topic MUST NOT 可删除。被禁用 Topic MUST NOT 作为新的路由目标，也不可作为新订阅的绑定目标。

#### Scenario: 创建后列表可见
- **WHEN** 管理员创建 Topic `fast-gpu` 成功
- **THEN** 列表接口返回该 Topic，且可被 Case 路由规则引用

#### Scenario: 默认 Topic 自动创建且不可删
- **WHEN** 业务库为空时启动控制面
- **THEN** 自动创建 `default` Topic；对其删除请求被拒绝

#### Scenario: 禁用 Topic 不可路由
- **WHEN** Topic `fast-gpu` 被禁用
- **THEN** 引用它的新规则保存被拒绝（存量任务按既有终态/对账路径处理）
