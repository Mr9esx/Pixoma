## ADDED Requirements

### Requirement: Case 路由投放配置读写
Case 创建/更新 API MUST 接受可选的路由投放配置字段，并按 `topic-routing-config` 校验；非法配置 MUST 被拒绝并返回可定位错误。Case 详情接口 MUST 返回路由配置。

#### Scenario: 保存合法路由配置
- **WHEN** 更新 Case 携带引用已启用 Topic 的路由规则
- **THEN** 更新成功且详情返回该路由配置

#### Scenario: 非法路由配置被拒绝
- **WHEN** 更新 Case 携带引用不存在 Topic 的路由规则
- **THEN** 更新被拒绝，返回含 Topic 相关信息的错误
