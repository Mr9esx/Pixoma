## Purpose

为管理控制台配置链路（菜单 → 工作流 → Topic → 节点）提供跨模块上下文关联：统一的关联面板展示上下游引用、就绪状态与跳转，降低配置时的心智负担。

## ADDED Requirements

### Requirement: 四模块关联上下文面板
Case / Topic / 节点 / 菜单详情页 MUST 提供统一的关联上下文面板，展示上下游引用、就绪状态并提供跳转。

#### Scenario: Case 详情展示关联链路
- **WHEN** 运维打开 Case 详情
- **THEN** 可看到该 Case 路由到的 Topic、订阅这些 Topic 的节点、以及挂载该 Case 的菜单入口；任一关联项可点击跳转

#### Scenario: Topic 详情展示关联链路
- **WHEN** 运维打开 Topic 详情
- **THEN** 可看到路由到该 Topic 的 Case 与订阅该 Topic 的节点，并显示是否就绪（存在在线订阅节点）

#### Scenario: 节点详情展示关联摘要
- **WHEN** 运维打开节点详情
- **THEN** 可看到该节点订阅的 Topic 与经由这些 Topic 可达的 Case 摘要

#### Scenario: 菜单挂载展示就绪摘要
- **WHEN** 运维在菜单编辑中挂载一个 Case
- **THEN** 可看到该 Case 的路由就绪状态与可执行节点摘要（规则完整且目标 Topic 有在线订阅节点）
