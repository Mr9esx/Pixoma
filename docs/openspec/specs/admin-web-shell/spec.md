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
系统 MUST 提供后台菜单模块，将菜单项映射到路由，并在侧栏展示 Dashboard 与全部一期资源入口；菜单顺序 MUST 为 Dashboard、实例、Case、主键盘、Task、User、Session。

#### Scenario: 侧栏展示菜单
- **WHEN** 用户打开控制台
- **THEN** 侧栏可见 Dashboard、实例、Case、主键盘、Task、User、Session 菜单项，且顺序如上

#### Scenario: 点击菜单进入对应路由
- **WHEN** 用户点击某一菜单项
- **THEN** 导航到该菜单绑定的页面路由

### Requirement: 默认进入 Dashboard
系统 MUST 将应用默认路由指向 Dashboard 页。

#### Scenario: 打开根路径
- **WHEN** 用户访问控制台根路径
- **THEN** 内容区展示 Dashboard（而非强制登录页或其它资源页）

### Requirement: 布局与基础主题
系统 MUST 提供含侧栏的管理布局：侧栏包含品牌区、导航菜单与底栏工具区；内容区 MUST NOT 再提供仅承载语言/主题切换的顶栏。系统 MUST 继续支持基座自带的基础主题切换能力（若基座提供明暗主题）。语言切换与主题切换控件 MUST 放置在侧栏底栏；当侧栏收起为图标模式时，上述控件 MUST 仍可以图标形式被发现并操作。

#### Scenario: 布局稳定包裹页面
- **WHEN** 用户在资源页之间切换
- **THEN** 侧栏布局保持，内容区切换为目标页，且内容区上方不出现仅含语言/主题的顶栏

#### Scenario: 侧栏底栏可切换语言与主题
- **WHEN** 用户打开控制台并查看侧栏底栏
- **THEN** 可见语言切换与主题切换入口，且操作后界面语言或主题相应变化

#### Scenario: 侧栏收起后仍可操作底栏工具
- **WHEN** 侧栏处于收起（图标）模式
- **THEN** 用户仍可通过底栏对应图标入口切换语言或主题

### Requirement: 中英 i18n
系统 MUST 提供中英两套完整界面文案（壳子、Dashboard、一期资源页），默认语言为中文，并 MUST 提供语言切换能力。

#### Scenario: 默认中文
- **WHEN** 用户首次打开控制台且无已保存语言偏好
- **THEN** 界面以中文展示

#### Scenario: 切换到英文
- **WHEN** 用户将语言切换为英文
- **THEN** 壳子与当前页可见文案切换为英文，且偏好在刷新后仍生效

### Requirement: 中性基色主题
系统 MUST 以 shadcn **neutral** 作为管理控制台亮色与暗色共用的基色（`baseColor`），并通过 CSS 语义变量驱动背景、前景、卡片、主色、muted、边框、侧栏等相关 token。暗色模式 MUST 不以蓝灰（slate）为主调；亮色与暗色 MUST 使用同一套基色，避免日夜切换出现「一套灰、一套蓝」的不一致。

#### Scenario: 暗色不发蓝
- **WHEN** 用户将主题切换为暗色（或系统偏好解析为暗色）并打开任意壳页
- **THEN** 页面主背景与主要表面色呈现中性灰，不以明显蓝灰为主调

#### Scenario: 亮暗共用同一基色
- **WHEN** 用户在亮色与暗色之间切换
- **THEN** 两侧均基于 neutral 语义 token，且主题切换能力（含偏好持久化）仍可用

#### Scenario: 基座配置与 token 对齐
- **WHEN** 开发者查看 `web/admin` 的 shadcn 基座配置与主题 CSS
- **THEN** `baseColor` 为 `neutral`，且亮/暗 CSS 变量与该基色一致

#### Scenario: 主表面中性不影响 chart 色相
- **WHEN** 主题已切换为 neutral，且页面使用 chart 语义色（`chart-1`…`chart-5`）
- **THEN** 主背景与主要表面仍为中性灰；chart 色板 MAY 保留色相（不要求无色）

### Requirement: 表面无投影
系统 MUST 让贴在页面上的卡片、inset 内容画布、按钮与表单控件使用细线框而非 drop shadow。对话框、下拉菜单、popover、sheet 等浮层 MAY 保留投影。

#### Scenario: 卡片与内容区无表面阴影
- **WHEN** 用户打开 Dashboard 或任意壳页
- **THEN** 卡片与 inset 内容画布不以 drop shadow 垫高，而以边框区分层次

#### Scenario: 浮层仍有投影
- **WHEN** 用户打开下拉菜单或对话框
- **THEN** 该浮层仍可使用投影与页面背景区分

### Requirement: 未初始化进入向导而非业务壳
当平台未完成初始化时，管理前端 MUST 引导用户进入登录（若需要）与初始化向导，MUST NOT 将未初始化用户默认送入完整业务资源壳并假装系统已就绪。

#### Scenario: 未初始化打开控制台
- **WHEN** 用户打开管理控制台且平台未初始化
- **THEN** 界面进入初始化/向导流程，而非直接展示完整业务侧栏资源页为唯一内容

### Requirement: 初始化后需登录
平台已初始化后，管理前端 MUST 在无有效会话时展示登录页；使用向导设定（或默认后已改密）的管理员账号登录成功后方可进入业务壳。

#### Scenario: 未登录访问根路径
- **WHEN** 平台已初始化且用户无会话访问控制台根路径
- **THEN** 用户被引导至登录，而非直接进入需鉴权的业务页

### Requirement: 侧栏包含 TG Menu 入口
管理控制台侧栏 MUST 提供 TG Menu（或产品约定中文名，如「主键盘」）入口，并映射到对应路由。本期侧栏在既有资源项基础上增加该项；顺序置于 Case 附近。

#### Scenario: 侧栏可见 Menu 入口
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏可见 TG Menu 管理入口且可点击进入

#### Scenario: 中英切换含 Menu 文案
- **WHEN** 用户切换中/英文
- **THEN** TG Menu 侧栏与页面关键文案随语言切换

### Requirement: 侧栏品牌为 Pixoma
侧栏品牌区 MUST 展示项目名称「Pixoma」，且 MUST NOT 展示 shadcn-admin 模板默认文案（如「Shadcn-Admin」「Vite + ShadcnUI」）。

#### Scenario: 侧栏可见 Pixoma
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏顶部品牌区显示「Pixoma」，且不显示上述模板默认主/副标题文案

