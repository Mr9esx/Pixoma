## MODIFIED Requirements

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、消息平台、Task、User、Session；「主键盘」不再作为一级菜单项。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、消息平台、Task、User、Session 菜单项，且顺序如上，且不含「主键盘」

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

## REMOVED Requirements

### Requirement: 侧栏包含 TG Menu 入口
**Reason**: 管理台改版为「消息平台」一级实体，主键盘菜单配置收进消息平台详情。
**Migration**: 侧栏「消息平台」入口替代；原 TG Menu 编辑能力迁移至消息平台详情内的菜单配置页。

## ADDED Requirements

### Requirement: 侧栏包含消息平台入口
管理控制台侧栏 MUST 提供「消息平台」入口并映射到消息平台列表路由；消息平台详情内 MUST 提供该消息平台的菜单配置入口（原「主键盘」编辑能力）。

#### Scenario: 侧栏可见消息平台入口
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏可见「消息平台」管理入口且可点击进入消息平台列表

#### Scenario: 消息平台详情可编辑菜单
- **WHEN** 用户进入某消息平台详情
- **THEN** 可进入该消息平台的菜单配置页进行编辑

#### Scenario: 中英切换含消息平台文案
- **WHEN** 用户切换中/英文
- **THEN** 「消息平台」相关侧栏与页面关键文案随语言切换
