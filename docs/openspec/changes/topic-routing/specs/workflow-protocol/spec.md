## ADDED Requirements

### Requirement: Case 协议包含路由投放配置
Case 协议 MUST 支持可选的路由投放配置（结构见 `topic-routing-config`）：有序规则列表（条件 + 目标 Topic key）；未配置时行为等价于全部回退系统默认 Topic。路由配置 MUST 随 Case 一起注册、更新与查询。

#### Scenario: 注册带路由的 Case
- **WHEN** 提交含路由配置的 Case（规则引用已启用 Topic）
- **THEN** Case 注册成功且路由配置可查询

#### Scenario: 旧 Case 无路由配置兼容
- **WHEN** 查询未配置路由的历史 Case
- **THEN** 返回无路由配置的表示，调度回退默认 Topic，不报错
