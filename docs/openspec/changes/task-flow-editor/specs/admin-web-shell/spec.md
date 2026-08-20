## MODIFIED Requirements

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、渠道、Task、User、Session、Topic；「主键盘」不再作为一级菜单项。任务分流画布作为 Case 编辑页内的功能区块，不单独占用一级菜单。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、渠道、Task、User、Session、Topic 菜单项，且顺序如上，且不含「主键盘」

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由（Topic 指向 Topic 管理页）
