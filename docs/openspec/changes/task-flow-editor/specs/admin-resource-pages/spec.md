## ADDED Requirements

### Requirement: Case 任务分流画布区
Case 管理页 MUST 在创建/编辑流程中提供任务分流画布区（React Flow），用于编辑与查看该 Case 的路由投放配置；保存时随 Case 一并提交 `routing`，详情页 MUST 以只读画布展示。

#### Scenario: 编辑并保存路由
- **WHEN** 用户在 Case 编辑页修改画布分支并保存 Case
- **THEN** Case 请求携带合法 `routing`，详情页只读画布展示相同配置

### Requirement: 节点订阅 Topic 编辑
计算节点详情页 MUST 支持查看与编辑订阅 Topic 列表；未设置时展示并提交为默认 Topic 语义（空列表），保存后调用节点更新 API。

#### Scenario: 修改节点订阅
- **WHEN** 管理员在节点详情页把订阅修改为 `["default","fast-gpu"]` 并保存
- **THEN** 请求发往节点更新 API 且载荷包含该订阅列表

### Requirement: Topic 管理页
管理控制台 MUST 提供 Topic 管理页：列表、新建、编辑、启用/禁用，数据来自 `/api/v1/topics`；默认 Topic（`default`）在 UI 上不可删除。

#### Scenario: 新建 Topic
- **WHEN** 管理员在 Topic 页提交合法新建
- **THEN** UI 展示更新后的列表，数据来自 Topic 管理 API

#### Scenario: 默认 Topic 不可删除
- **WHEN** 管理员在 Topic 页对 `default` 执行删除
- **THEN** UI 不提供成功删除路径（操作被禁用或服务端拒绝被展示为错误）
