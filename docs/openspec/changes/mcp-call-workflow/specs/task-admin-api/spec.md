## MODIFIED Requirements

### Requirement: 不在 admin 发起代用户 ConfirmRun
admin-api MUST NOT 提供“代替用户确认生成并创建任务”的管理入口作为本期能力。通过 MCP 调用工作流 MUST 走独立入口，MUST NOT 实现为 admin-api 上的 ConfirmRun。

#### Scenario: 无 ConfirmRun 管理入口
- **WHEN** 客户端查找代用户 ConfirmRun 的管理 API
- **THEN** 本期不提供该能力

#### Scenario: MCP 不是 admin ConfirmRun
- **WHEN** MCP 客户端成功调用工作流
- **THEN** 该调用不经过 admin-api 的 ConfirmRun 管理入口，管理 API 契约保持无代跑
