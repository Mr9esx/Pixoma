## ADDED Requirements

### Requirement: 本期不实现 Topic 与投放规则管理验收
本期 MUST NOT 将 Topic 目录 CRUD、投放表达式管理、试算 API 作为交付验收项。相关后台能力延后到后续 change。若代码中预留表结构，MUST NOT 阻塞 allinone/split 与方案 A 主路径。

#### Scenario: 主路径不依赖 Topic Admin
- **WHEN** 未配置任何 Topic Admin 资源
- **THEN** allinone 或按实例 dispatch 的 split 主路径仍可完成任务（在实例/Edge 可用时）
