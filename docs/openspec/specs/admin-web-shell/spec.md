# admin-web-shell Specification

## Purpose
基于 shadcn-admin 提供管理后台应用壳：布局、主题、可配置侧栏菜单、默认 Dashboard 与中英 i18n，作为所有资源页的导航入口。
## Requirements
### Requirement: 可运行的管理前端应用
系统 MUST 在 `web/admin` 提供可本地启动的 React 管理前端，技术栈基于 shadcn/ui 与 shadcn-admin 基座，包管理器为 pnpm。

#### Scenario: 本地启动控制台
- **WHEN** 开发者按文档使用 pnpm 安装依赖并启动开发服务器，且已配置可到达的 admin-api
- **THEN** 浏览器可打开管理控制台壳页面

### Requirement: 菜单模块
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部一期资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、Task、User、Session。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、Task、User、Session 菜单项，且顺序如上

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

### Requirement: 默认进入 Dashboard
系统 MUST 将应用默认路由指向 Dashboard 页。

#### Scenario: 打开根路径
- **WHEN** 用户访问控制台根路径
- **THEN** 内容区展示 Dashboard（而非强制登录页或其它资源页）

### Requirement: 布局与基础主题
系统 MUST 提供含侧栏与顶栏的管理布局，并支持基座自带的基础主题切换能力（若基座提供明暗主题）。

#### Scenario: 布局稳定包裹页面
- **WHEN** 用户在资源页之间切换
- **THEN** 侧栏/顶栏布局保持，内容区切换为目标页

### Requirement: 中英 i18n
系统 MUST 提供中英两套完整界面文案（壳子、Dashboard、一期资源页），默认语言为中文，并 MUST 提供语言切换能力。

#### Scenario: 默认中文
- **WHEN** 用户首次打开控制台且无已保存语言偏好
- **THEN** 界面以中文展示

#### Scenario: 切换到英文
- **WHEN** 用户将语言切换为英文
- **THEN** 壳子与当前页可见文案切换为英文，且偏好在刷新后仍生效

