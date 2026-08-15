## ADDED Requirements

### Requirement: 侧栏包含 TG Menu 入口
管理控制台侧栏 MUST 提供 TG Menu（或产品约定中文名，如「TG 菜单」）入口，并映射到对应路由。本期侧栏在既有六项基础上增加该项；顺序建议置于 Case 附近（具体顺序在实现中与 i18n 文案一并固定）。

#### Scenario: 侧栏可见 Menu 入口
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏可见 TG Menu 管理入口且可点击进入

#### Scenario: 中英切换含 Menu 文案
- **WHEN** 用户切换中/英文
- **THEN** TG Menu 侧栏与页面关键文案随语言切换
