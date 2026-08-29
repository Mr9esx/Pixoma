## Why

现有菜单模型把「业务动作」硬编码在菜单里（`kind` 只表达 Case 工作流语义），每加一个业务模块（签到/计费/会员）就要改菜单模型和适配器；平台交互差异（TG 键盘、飞书卡片）以游离的 extras 存放，无法表达「同一能力在不同消息平台怎么交互」。需要一个明确的**交互框架与协议**：能力注册表 + 菜单=能力入口树 + 统一交互协议（UI 事件 → 能力调用 → 结果渲染），让新增能力和新消息平台都只做「注册/适配」，不改底层。

## What Changes

- **能力注册表（新）**：能力 = {id、展示名、参数 schema、执行 handler、各消息平台渲染声明}；业务模块以能力形式注册。本期内置 `open_case`（Case 工作流），预留签到/计费/会员等后续注册。
- **BREAKING** 菜单模型改为能力入口：菜单项 = {capability_id、label、params、children?}；移除菜单节点的业务 kind 语义（folder/open_case/placeholder/reply_media 的业务部分），仅保留通用分组与展示字段。
- **统一交互协议（新）**：消息平台 UI 事件（按钮点击/文本/媒体）→ 能力调用 → 执行结果 → 消息平台渲染；适配器只负责「事件翻译 + 结果渲染」，不再认识具体业务。
- **消息平台菜单交互设计（核心工作流）**：用户端——主键盘由管理员**显式配置**（直达入口 ≤6，平台不自动塞满）、分组最多一层且只在消息按钮中出现、流程全部走消息按钮、任一步可返回/退出；管理端——能力为中心的编辑器（选能力 + schema 表单）、**消息平台形态预览**（主键盘/消息按钮模拟渲染）、直白文案。
- **按消息平台账户上下文**：能力执行上下文为消息平台账户（channel + external_user_id）；本期只定义上下文结构，不实现具体权益业务，也不做跨消息平台账户合并。
- **TG 适配器验证**：主键盘=能力入口（≤6 个直达按钮），流程走消息按钮（内联）；渲染差异按能力的消息平台声明读取，替代游离 extras。
- **管理台菜单配置改版**：编辑器从「选 kind」改为「选能力 + 填参数 + 消息平台展示微调」。
- **用户文案术语约束**：所有面向用户的文案禁用内部术语（inline/callback/extras/capability 等），统一使用「按钮/选项/功能」等大白话；术语表仅内部使用。

## Capabilities

### New Capabilities
- `capability-registry`: 能力注册表与声明（id、展示名、参数 schema、执行 handler、各消息平台渲染声明）
- `channel-interaction-protocol`: 统一交互协议（UI 事件 → 能力调用 → 结果渲染；适配器契约）
- `channel-account-context`: 按消息平台账户的执行上下文（权益/记录挂消息平台账户；不做跨消息平台合并）
- `channel-menu-interaction`: 用户端菜单交互规范（主键盘显式配置 ≤6、一层分组、消息按钮流程、返回/退出）

### Modified Capabilities
- `channel-runtime-ports`: 交互动作泛化为「能力调用」（Action 由工作流专属改为 capability + params）
- `channel-menu-config`: 菜单模型改为能力入口（capability_id + params），移除业务 kind
- `channel-tg`: TG 适配器按交互协议渲染（主键盘=能力入口、消息按钮流程、结果渲染）
- `tg-menu-admin-api`: 消息平台菜单管理 API 载荷改为能力入口结构
- `admin-web-shell`: 管理台菜单编辑器改版（选能力+参数+消息平台展示+**消息平台形态预览**）；用户文案术语约束

## Impact

- 领域/包：`internal/channel`（新增 capability 注册表与交互协议）、`internal/menu`（模型改能力入口）、`internal/channel/tg`（渲染重构）、`internal/httpapi/channelmenu`（API 载荷）、`web/admin`（菜单编辑器 + 文案）
- 持久化：`channel_menu_items` 增加能力入口字段（capability_id / params_json），迁移/替换旧 kind 语义
- 行为：现有 TG 主键盘与文件夹交互改为能力入口 + 消息按钮流程；Case 工作流行为等价保留
- 非目标：会员/计费/签到业务实现、飞书/企微适配器实现、跨消息平台统一账户、消息平台管理改动
