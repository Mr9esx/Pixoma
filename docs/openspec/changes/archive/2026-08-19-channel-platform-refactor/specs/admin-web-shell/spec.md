## MODIFIED Requirements

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、渠道、Task、User、Session；「主键盘」不再作为一级菜单项。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、渠道、Task、User、Session 菜单项，且顺序如上，且不含「主键盘」

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

## REMOVED Requirements

### Requirement: 侧栏包含 TG Menu 入口
**Reason**: 管理台改版为「渠道」一级实体，主键盘菜单配置收进渠道详情。
**Migration**: 侧栏「渠道」入口替代；原 TG Menu 编辑能力迁移至渠道详情内的菜单配置页。

## ADDED Requirements

### Requirement: 侧栏包含渠道入口
管理控制台侧栏 MUST 提供「渠道」入口并映射到渠道列表路由；渠道详情内 MUST 提供该渠道的菜单配置入口（原「主键盘」编辑能力）。

#### Scenario: 侧栏可见渠道入口
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏可见「渠道」管理入口且可点击进入渠道列表

#### Scenario: 渠道详情可编辑菜单
- **WHEN** 用户进入某渠道详情
- **THEN** 可进入该渠道的菜单配置页进行编辑

#### Scenario: 中英切换含渠道文案
- **WHEN** 用户切换中/英文
- **THEN** 「渠道」相关侧栏与页面关键文案随语言切换
