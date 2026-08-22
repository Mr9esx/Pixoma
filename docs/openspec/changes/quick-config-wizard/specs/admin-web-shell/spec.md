## MODIFIED Requirements

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部资源入口；菜单顺序 MUST 为 Dashboard、快速配置、实例、Case、渠道、Task、User、Session、设置；「主键盘」不再作为一级菜单项；「快速配置」MUST 可导航到三步快速配置向导页，并随语言切换提供中英文案。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、快速配置、实例、Case、渠道、Task、User、Session、设置菜单项，且顺序如上，且不含「主键盘」

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

#### Scenario: 快速配置入口打开向导
- **WHEN** 用户点击侧栏「快速配置」
- **THEN** 内容区展示三步快速配置向导页，且第一步可操作

#### Scenario: 快速配置文案随语言切换
- **WHEN** 用户切换中/英文
- **THEN** 侧栏「快速配置」入口与向导页关键文案随语言切换
