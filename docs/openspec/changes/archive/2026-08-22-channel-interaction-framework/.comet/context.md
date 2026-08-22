# Comet Design Handoff

- Change: channel-interaction-framework
- Phase: design
- Mode: compact
- Context hash: 1b88e9ae638e451a84d8bfe90540fa75986cd2773094cc8e3dd5ed7efff04308

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/channel-interaction-framework/proposal.md

- Source: docs/openspec/changes/channel-interaction-framework/proposal.md
- Lines: 1-36
- SHA256: 6658425cf84028602093ed1d9aee4581bd4dfe4ae5d0eda2754f447d5e641163

```md
## Why

现有菜单模型把「业务动作」硬编码在菜单里（`kind` 只表达 Case 工作流语义），每加一个业务模块（签到/计费/会员）就要改菜单模型和适配器；平台交互差异（TG 键盘、飞书卡片）以游离的 extras 存放，无法表达「同一能力在不同渠道怎么交互」。需要一个明确的**交互框架与协议**：能力注册表 + 菜单=能力入口树 + 统一交互协议（UI 事件 → 能力调用 → 结果渲染），让新增能力和新渠道都只做「注册/适配」，不改底层。

## What Changes

- **能力注册表（新）**：能力 = {id、展示名、参数 schema、执行 handler、各渠道渲染声明}；业务模块以能力形式注册。本期内置 `open_case`（Case 工作流），预留签到/计费/会员等后续注册。
- **BREAKING** 菜单模型改为能力入口：菜单项 = {capability_id、label、params、children?}；移除菜单节点的业务 kind 语义（folder/open_case/placeholder/reply_media 的业务部分），仅保留通用分组与展示字段。
- **统一交互协议（新）**：渠道 UI 事件（按钮点击/文本/媒体）→ 能力调用 → 执行结果 → 渠道渲染；适配器只负责「事件翻译 + 结果渲染」，不再认识具体业务。
- **渠道菜单交互设计（核心工作流）**：用户端——主键盘由管理员**显式配置**（直达入口 ≤6，平台不自动塞满）、分组最多一层且只在消息按钮中出现、流程全部走消息按钮、任一步可返回/退出；管理端——能力为中心的编辑器（选能力 + schema 表单）、**渠道形态预览**（主键盘/消息按钮模拟渲染）、直白文案。
- **按渠道账户上下文**：能力执行上下文为渠道账户（channel + external_user_id）；本期只定义上下文结构，不实现具体权益业务，也不做跨渠道账户合并。
- **TG 适配器验证**：主键盘=能力入口（≤6 个直达按钮），流程走消息按钮（内联）；渲染差异按能力的渠道声明读取，替代游离 extras。
- **管理台菜单配置改版**：编辑器从「选 kind」改为「选能力 + 填参数 + 渠道展示微调」。
- **用户文案术语约束**：所有面向用户的文案禁用内部术语（inline/callback/extras/capability 等），统一使用「按钮/选项/功能」等大白话；术语表仅内部使用。

## Capabilities

### New Capabilities
- `capability-registry`: 能力注册表与声明（id、展示名、参数 schema、执行 handler、各渠道渲染声明）
- `channel-interaction-protocol`: 统一交互协议（UI 事件 → 能力调用 → 结果渲染；适配器契约）
- `channel-account-context`: 按渠道账户的执行上下文（权益/记录挂渠道账户；不做跨渠道合并）
- `channel-menu-interaction`: 用户端菜单交互规范（主键盘显式配置 ≤6、一层分组、消息按钮流程、返回/退出）

### Modified Capabilities
- `channel-runtime-ports`: 交互动作泛化为「能力调用」（Action 由工作流专属改为 capability + params）
- `channel-menu-config`: 菜单模型改为能力入口（capability_id + params），移除业务 kind
- `channel-tg`: TG 适配器按交互协议渲染（主键盘=能力入口、消息按钮流程、结果渲染）
- `tg-menu-admin-api`: 渠道菜单管理 API 载荷改为能力入口结构
- `admin-web-shell`: 管理台菜单编辑器改版（选能力+参数+渠道展示+**渠道形态预览**）；用户文案术语约束

## Impact

- 领域/包：`internal/channel`（新增 capability 注册表与交互协议）、`internal/menu`（模型改能力入口）、`internal/channel/tg`（渲染重构）、`internal/httpapi/channelmenu`（API 载荷）、`web/admin`（菜单编辑器 + 文案）
- 持久化：`channel_menu_items` 增加能力入口字段（capability_id / params_json），迁移/替换旧 kind 语义
- 行为：现有 TG 主键盘与文件夹交互改为能力入口 + 消息按钮流程；Case 工作流行为等价保留
- 非目标：会员/计费/签到业务实现、飞书/企微适配器实现、跨渠道统一账户、渠道管理改动

```

## docs/openspec/changes/channel-interaction-framework/design.md

- Source: docs/openspec/changes/channel-interaction-framework/design.md
- Lines: 1-159
- SHA256: ee96c79f327efc68a84ff0a69e537a23006ffbed5582b85be95c2c75d1324b1c

[TRUNCATED]

```md
## Context

渠道层重构已落地（渠道壳、渠道作用域菜单、运行时端口、身份渠道化、热生效装配、notify 路由）。现状痛点：菜单 `kind` 把业务动作硬编码进模型（folder/open_case/placeholder/reply_media），每加业务要改菜单与适配器；端口层 `Action` 是 Case 工作流专属；平台差异以游离 extras 存放；面向用户文案可能泄露内部术语。本次引入「能力注册表 + 统一交互协议 + 渠道账户上下文」，把「业务能做什么」与「渠道怎么展示/触发」解耦。

## Goals / Non-Goals

**Goals:**
- 能力注册表：能力 = {id、展示名、参数 schema、执行 handler、各渠道渲染声明}；新增业务=注册能力
- 菜单改为能力入口树：菜单项 = {展示字段 + capability_id + params + children}
- 统一交互协议：UI 事件 → 能力调用 → 结果 → 渠道渲染；适配器不感知具体业务
- 渠道账户上下文：权益挂渠道账户（渠道 + 外部用户 id），不做跨渠道合并
- TG 验证：主键盘=能力入口（≤6），流程走消息按钮；旧 extras 并入渲染声明
- 用户文案术语约束写入规范

**Non-Goals:**
- 会员/计费/签到等具体业务实现（仅框架与 open_case 内置能力）
- 飞书/企微/钉钉适配器实现
- 跨渠道统一账户
- 渠道管理改动

## Decisions

### D1. 能力注册表：Go 代码注册 + 声明式渲染

`internal/channel/capability`：

```go
type Capability interface {
    ID() string
    DisplayName() string
    ParamsSchema() map[string]ParamSpec   // name → {type, required, options, default}
    Invoke(ctx context.Context, acct AccountCtx, params map[string]any) (Result, error)
}
type RenderDecl struct {                  // 每渠道渲染声明
    Entry  string `json:"entry"`          // root | message_button
    Config map[string]any `json:"config"` // 如 {columns: 2}
}
type Registry struct{ caps map[string]Capability }
```

能力以 Go 注册（`Register(cap)`），内置 `open_case`（包装现有 `botapp.Facade` 流程）。渲染声明作为能力的一部分按渠道提供，替代游离 extras。理由：代码注册类型安全、可直接注入 Facade；渲染声明数据化，管理台可读。

备选：能力全部声明式（YAML/DB）。放弃原因：执行 handler 仍需代码，双份配置易漂移。

### D2. 菜单模型：能力入口

`MenuNode` 调整：

```go
type MenuNode struct {
    ID, ParentID, Label string
    Order, Enabled      ...
    IntroText, PlaceholderText string
    Reply *ReplyPayload         // 展示字段保留
    CapabilityID string         // 空 = 纯分组/展示
    Params map[string]any       // 能力参数
    Children []MenuNode
}
```

`kind` 业务语义移除：folder → 纯分组（CapabilityID 空）；open_case → `CapabilityID="open_case"` + params.case_ids/back；placeholder/reply_media → 展示字段（CapabilityID 空或内置展示能力）。校验：未知 capability_id 拒绝；params 按 schema 校验。

### D3. 交互协议：事件 → 能力调用 → 结果

端口层 `Action` 泛化为：

```go
type CapabilityInvoke struct {
    CapabilityID string
    Params       map[string]any
    Account      AccountCtx
}
type Result struct {
    Text    string
    Options []Option   // {label, value} 供适配器渲染为按钮/卡片
    Media   []MediaRef
    Error   *string
}
```


```

Full source: docs/openspec/changes/channel-interaction-framework/design.md

## docs/openspec/changes/channel-interaction-framework/tasks.md

- Source: docs/openspec/changes/channel-interaction-framework/tasks.md
- Lines: 1-48
- SHA256: a6ece2d1b9b3fba90c8c1458c16f1b4321366e2b0c69894cee9fac0763e04018

```md
## 1. 能力注册表

- [ ] 1.1 新增 `internal/channel/capability`：Capability 接口（ID/DisplayName/ParamsSchema/Invoke）、ParamSpec、Registry 与 `Register`
- [ ] 1.2 参数 schema 校验：按 ParamSpec 校验能力调用参数（必填/类型/选项/默认值），非法返回可理解错误
- [ ] 1.3 内置 `open_case` 能力：包装现有 `botapp.Facade`（预览/开始/填表/确认/结果），params 支持 case 选择与 back 引用
- [ ] 1.4 能力清单查询：注册表支持按渠道筛选渲染声明，供管理台与适配器读取

## 2. 菜单模型改为能力入口

- [ ] 2.1 `MenuNode` 移除业务 kind：新增 `CapabilityID` 与 `Params`；folder 语义降为纯分组（CapabilityID 空）；placeholder/reply 保留为展示字段
- [ ] 2.2 校验规则更新：未知 capability_id 拒绝；params 按 schema 校验；分组子项规则（纯分组/能力入口均可）重新定义
- [ ] 2.3 持久化：`channel_menu_items` 增加 capability_id / params_json 列；删除 kind 业务枚举
- [ ] 2.4 数据迁移：旧 kind → 能力入口（open_case→open_case 能力、placeholder/reply→展示字段、folder→分组）；`tg_root_layout` extras → open_case 的 tg 渲染声明
- [ ] 2.5 默认种子改为能力入口（open_case 挂载图片 Case）

## 3. 统一交互协议

- [ ] 3.1 端口 `Action` 泛化为 `CapabilityInvoke{CapabilityID, Params, Account}`；`Result{Text/Options/Media/Error}` 渠道无关结果结构
- [ ] 3.2 `AccountCtx{ChannelID, ExternalUserID, InternalUserID}`：适配器经身份解析填充，能力执行携带
- [ ] 3.3 适配器契约：UI 事件 → CapabilityInvoke → registry 执行 → Result → 渠道渲染；适配器不写业务分支

## 4. TG 适配器按协议渲染

- [ ] 4.1 主键盘：根层能力入口直达按钮 ≤6；超出进「更多」分组或消息按钮
- [ ] 4.2 分组与流程：点分组 → 消息按钮列出子项与 open_case 入口；返回语义保留；每行按钮数按渲染声明
- [ ] 4.3 open_case 行为等价回归：预览/开始/填表/确认/出图通知
- [ ] 4.4 移除适配器内业务 switch（open_folder/open_case 等专属动作），改走 registry

## 5. 管理台改版

- [ ] 5.1 菜单编辑器：入口类型改为「选择能力（来自 registry）→ 按 ParamsSchema 渲染参数表单 → 渠道展示微调」；移除 kind 选择
- [ ] 5.2 移除 extras 独立编辑入口（旧数据已迁移为渲染声明）
- [ ] 5.3 用户文案术语约束：管理台与 bot 文案禁用 inline/callback/extras/capability 等内部术语，统一「按钮/选项/功能」

## 6. 交互设计实现（用户端 + 管理端）

- [ ] 6.1 用户端交互：主键盘显式配置（≤6，平台不自动塞满）；超出入口限制提示
- [ ] 6.2 用户端交互：分组最多一层（消息按钮内只有能力入口，无嵌套分组）
- [ ] 6.3 用户端交互：流程每步提供返回/退出按钮（返回上一级或主菜单）
- [ ] 6.4 管理端预览：按当前配置模拟渲染 TG 主键盘与消息按钮流程（遵循显式配置与一层分组规则）

## 7. 测试与回归

- [ ] 7.1 单元：能力注册/参数校验/Result 结构/AccountCtx
- [ ] 7.2 领域与持久化：菜单能力入口校验、params_json 往返、迁移脚本
- [ ] 7.3 适配器：事件→能力调用翻译、主键盘 ≤6、一层分组消息按钮、返回语义
- [ ] 7.4 API 与前端：菜单 API 载荷（capability_id）、编辑器 schema 表单、预览渲染
- [ ] 7.5 全量 `go test ./...` + `pnpm test/build` + TG 手工回归（Case 流程等价 + 主键盘显式配置/一层分组/返回退出）

```

## docs/openspec/changes/channel-interaction-framework/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/admin-web-shell/spec.md
- Lines: 1-23
- SHA256: 72f07bf13f34106309b96ef9e472554ee83a9ebeefaa5d29eb4367ebfee6400e

```md
## ADDED Requirements

### Requirement: 菜单编辑器按能力配置
渠道详情内的菜单配置页 MUST 以「能力 + 参数 + 渠道展示」方式编辑：每个入口选择已注册能力、按能力参数 schema 填写参数、按渠道调整展示；编辑器 MUST NOT 再以业务 kind（如 open_case/placeholder）作为入口类型。参数表单由能力注册表的 schema 驱动。

#### Scenario: 按能力添加入口
- **WHEN** 管理员在渠道菜单中添加入口
- **THEN** 从已注册能力中选择并填写参数，保存后运行时按能力渲染

#### Scenario: 参数表单按 schema 渲染
- **WHEN** 管理员选中某能力
- **THEN** 表单按该能力参数 schema 展示必填项与选项，非法输入被阻止或提示

### Requirement: 渠道形态预览
菜单配置页 MUST 提供渠道形态预览：由后端复用「菜单 + 能力渲染声明 → 渠道渲染结构」的渲染逻辑返回预览 DTO（主键盘行列、各分组消息按钮），前端仅绘制；预览与真实渲染 MUST 同源，保存前可查看用户实际看到的形态，并反映主键盘显式配置与一层分组规则。

#### Scenario: 预览主键盘形态
- **WHEN** 管理员查看主键盘预览
- **THEN** 展示按当前入口配置渲染的 TG 主键盘（≤6 个直达按钮）

#### Scenario: 预览消息按钮流程
- **WHEN** 管理员查看某个分组的流程预览
- **THEN** 展示点击该入口后的消息按钮列表（一层分组 + 返回/退出）

```

## docs/openspec/changes/channel-interaction-framework/specs/capability-registry/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/capability-registry/spec.md
- Lines: 1-37
- SHA256: df91f575a44f4b8e8469fc79b34ad05758734d19e983f7e2cf02a579f120cfe8

```md
## Purpose

能力注册表把「业务能做什么」从渠道和菜单中抽象出来：每个能力声明其展示名、参数 schema、执行入口与各渠道的渲染方式；新增业务模块只需注册能力，无需改动适配器或菜单模型。

## ADDED Requirements

### Requirement: 能力声明
系统 MUST 提供能力注册表，每个能力至少声明：能力 id（kebab-case）、展示名、参数 schema、执行 handler 标识、各渠道渲染声明。注册表 MUST 支持运行时查询与按渠道筛选，供菜单配置和适配器使用。

#### Scenario: 注册新能力
- **WHEN** 新增一个能力（如 checkin）并完成注册
- **THEN** 无需改动适配器或菜单模型，即可在菜单配置中引用并触发

#### Scenario: 查询能力清单
- **WHEN** 管理台或适配器请求能力清单
- **THEN** 返回已注册能力的 id、展示名、参数 schema 与渠道渲染声明

### Requirement: 参数 schema
能力 MUST 以 **JSON Schema** 声明其参数（draft-07 子集），管理台据此渲染参数表单、适配器据此校验事件载荷；非法参数 MUST 被拒绝并返回可理解错误。

#### Scenario: 按 schema 校验参数
- **WHEN** 菜单项或事件携带不符合 schema 的参数
- **THEN** 系统拒绝并返回参数错误说明

### Requirement: 渠道渲染声明
能力 MUST 在能力定义内按渠道提供默认渲染声明（入口形态：主键盘直达 / 消息按钮 / 分组；展示配置如每行按钮数）；菜单项 MAY 通过渲染覆盖（render override）微调指定渠道的配置，覆盖语义为按 key 合并。渲染声明按渠道隔离；未声明某渠道渲染的能力在该渠道不展示。

#### Scenario: 按渠道展示能力入口
- **WHEN** 同一能力在 TG 与后续飞书渠道消费
- **THEN** 各自按本渠道的渲染声明展示，无需改写能力定义

### Requirement: 内置能力
系统 MUST 内置 `open_case` 能力（执行现有 Case 工作流：预览/开始/填表/确认/结果）；`open_case` 参数 MUST 支持 case 选择上下文（如 back 引用），以承载文件夹/返回语义。

#### Scenario: open_case 走既有工作流
- **WHEN** 用户点击 open_case 能力入口
- **THEN** 进入既有 Case 预览与填表工作流，行为等价保留

```

## docs/openspec/changes/channel-interaction-framework/specs/channel-account-context/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/channel-account-context/spec.md
- Lines: 1-23
- SHA256: 88bc02a86f1448b064ce3cdb2e7b2ab339ac7f07f01ff137a571c5f2f9ef9263

```md
## Purpose

渠道账户上下文定义能力执行时的身份与归属边界：权益（会员、余额、签到记录等）挂在渠道账户（渠道 + 外部用户 id）上，不做跨渠道账户合并；为后续计费/会员/签到能力提供统一执行上下文。

## ADDED Requirements

### Requirement: 渠道账户执行上下文
能力调用 MUST 携带渠道账户上下文（channel_id + external_user_id + 内部用户 id），业务能力据此读写归属该渠道账户的权益数据；不同渠道的同一自然人视为不同渠道账户。

#### Scenario: 能力读取渠道账户权益
- **WHEN** 某能力执行且需要权益上下文
- **THEN** 上下文提供当前渠道账户标识，权益读写限定于该渠道账户

#### Scenario: 跨渠道账户隔离
- **WHEN** 同一自然人在两个渠道各有一个渠道账户
- **THEN** 各自的权益数据互不影响，不自动合并

### Requirement: 权益数据归属
系统 MUST 以渠道账户为权益归属键（渠道 + 外部用户 id，或对应的内部用户 id）；权益表结构 MUST 允许按渠道账户查询与更新。

#### Scenario: 按渠道账户记录权益
- **WHEN** 签到/充值等能力写入权益
- **THEN** 记录归属当前渠道账户，可被同一渠道账户后续查询

```

## docs/openspec/changes/channel-interaction-framework/specs/channel-interaction-protocol/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/channel-interaction-protocol/spec.md
- Lines: 1-44
- SHA256: 06f44bc9f1bc5f3939adc113975f955502350dc46fd6bdbb90b6676c65afdf1d

```md
## Purpose

统一交互协议定义渠道上「用户点了一下」如何变成「业务执行并渲染结果」：适配器把 UI 事件翻译为能力调用，执行后按能力的渠道渲染声明输出结果；适配器不再认识具体业务。

## ADDED Requirements

### Requirement: 事件 → 能力调用协议
系统 MUST 将渠道 UI 事件（按钮点击、文本、媒体、返回）翻译为能力调用（capability_id + params + 渠道账户上下文）；能力调用 MUST 携带执行上下文，其中用户身份为渠道账户（渠道 + 外部用户 id）。

#### Scenario: 按钮点击触发能力
- **WHEN** 用户点击 TG 主键盘或消息按钮
- **THEN** 适配器翻译为对应能力调用并执行，返回结果按渠道渲染

### Requirement: 结果渲染协议
能力执行结果 MUST 以渠道无关的结果结构返回（文本/选项列表/媒体/错误），由适配器按本渠道渲染声明输出；同一结果在不同渠道可呈现不同形态。

#### Scenario: 结果跨渠道形态不同
- **WHEN** 同一能力返回选项列表
- **THEN** TG 渲染为消息按钮，飞书（后续）渲染为卡片按钮，内容语义一致

### Requirement: 适配器不感知具体业务
适配器 MUST 仅依赖能力调用与结果结构，MUST NOT 为具体业务编写分支（如签到、计费）；新增业务能力不得要求修改适配器。

#### Scenario: 新能力不改适配器
- **WHEN** 注册一个新能力并加入菜单
- **THEN** 既有适配器无需改动即可完成事件翻译与结果渲染

### Requirement: 用户文案术语约束
所有面向用户的消息、按钮与错误文案 MUST 使用用户可理解的表达（按钮/选项/功能/返回等），MUST NOT 出现内部术语（inline、callback、extras、capability、参数 schema 等）；内部术语仅允许出现在代码、文档与内部配置。

#### Scenario: 文案无内部术语
- **WHEN** 检查面向用户的文案（bot 消息、管理台引导语）
- **THEN** 不存在 inline/callback/extras/capability 等内部术语

### Requirement: 协议层导航上下文
能力调用 MUST 携带协议层导航上下文（NavContext：back 锚点为 root 或分组/流程步 id），与能力参数分离；返回/退出按钮 MUST 由适配器按导航上下文统一渲染与处理，能力 MUST NOT 感知导航栈。

#### Scenario: 任意流程步可返回
- **WHEN** 用户处于某能力流程中的任一步骤
- **THEN** 适配器按 NavContext 渲染返回按钮（回 root 或上一层分组），无需该能力实现返回逻辑

#### Scenario: 退出回到主菜单
- **WHEN** 用户点击退出
- **THEN** 回到主菜单并清理当前流程上下文（如填表会话），无歧义残留

```

## docs/openspec/changes/channel-interaction-framework/specs/channel-menu-config/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/channel-menu-config/spec.md
- Lines: 1-44
- SHA256: c60a7651300430949c949f38f85fed5fe7fa3c37c9591ace5396b39eb4de2adc

```md
## MODIFIED Requirements

### Requirement: 菜单模型平台中立
菜单领域模型 MUST 使用平台中立的**能力入口**语义：菜单项 = {展示字段（label/intro/占位/回复媒体）、能力入口（capability_id + params）、children}；中立核心字段 MUST NOT 包含平台专有字段（如 Telegram 行/列网格、回调编码、消息长度上限）。平台渲染差异 MUST 由能力的渠道渲染声明承载，MUST NOT 进入中立核心字段，且 MUST NOT 因通用化抹平平台特色能力。

#### Scenario: 中立字段不含 TG 专有字段
- **WHEN** 通过管理 API 读取或提交菜单的中立字段
- **THEN** 中立字段中不存在 row/col、回调前缀或 TG 消息长度约束；平台差异仅存在于能力的渠道渲染声明

#### Scenario: TG 特色能力保留
- **WHEN** 管理员为 TG 渠道配置能力入口的渠道展示参数（如根层每行按钮数）
- **THEN** 参数存于该能力的渠道渲染声明，TG 适配器按声明渲染；其他渠道读取不到且不受影响

#### Scenario: 同一菜单可被多平台适配器渲染
- **WHEN** 同一份渠道菜单（能力入口 + 分组）分别由 TG 与后续平台适配器消费
- **THEN** 各适配器按各自平台形态渲染，无需改写菜单配置

### Requirement: Case 挂载与反查
系统 MUST 支持 open_case 能力入口引用 Case（作为能力参数）并从 Case 反查其出现的菜单路径；引用的 Case 必须存在，否则拒绝保存。

#### Scenario: 保存文件夹并挂载 Case
- **WHEN** 管理面在渠道菜单中创建文件夹项，其 open_case 能力参数引用已存在的 Case 后保存
- **THEN** 持久化成功，后续读取可见该文件夹及其 Case 引用

#### Scenario: 绑定不存在的 Case 被拒绝
- **WHEN** 提交的 open_case 能力参数引用不存在的 `case_id`
- **THEN** 系统拒绝保存并返回可理解的校验错误

#### Scenario: 查询 Case 的菜单挂载
- **WHEN** 调用方请求某 Case 的菜单挂载
- **THEN** 系统返回零条或多条路径；每条能标识所在渠道、菜单项与可读路径

### Requirement: 默认种子按渠道
系统在渠道无自定义菜单时 MUST 提供可用的默认种子菜单；种子根层入口 MUST 使用 open_case 能力并挂载图片类 Case。

#### Scenario: 空配置使用默认种子
- **WHEN** 渠道尚无自定义菜单
- **THEN** 运行时仍能得到可用的根菜单项集合（open_case 能力入口）

## REMOVED Requirements

### Requirement: 渠道差异数据（extras）管理
**Reason**: 平台差异数据并入能力的渠道渲染声明，不再以游离 extras 存储。
**Migration**: 既有 extras（如 tg_root_layout）迁移为 open_case 等能力的渠道渲染声明参数；管理台不再提供 extras 独立编辑入口。

```

## docs/openspec/changes/channel-interaction-framework/specs/channel-menu-interaction/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/channel-menu-interaction/spec.md
- Lines: 1-34
- SHA256: be93b16674378dab681ee4eea1e28446867eee5235f24ba66261ea621185b3d0

```md
## Purpose

渠道菜单交互规范定义用户在 bot 里「看到什么、怎么走」的体验边界：主键盘由管理员显式配置、直达入口有限、分组最多一层、流程走消息按钮且随时可返回退出；它是本次渠道菜单交互设计重建的核心。

## ADDED Requirements

### Requirement: 主键盘显式配置
主键盘（常驻入口）MUST 由管理员显式配置，系统 MUST NOT 自动把未配置的能力塞进主键盘；主键盘直达入口数量 MUST 不超过 6 个。

#### Scenario: 管理员显式配置主键盘
- **WHEN** 管理员在菜单配置中只添加 3 个能力入口
- **THEN** 主键盘只显示这 3 个入口，其余已注册能力不自动出现

#### Scenario: 超过 6 个入口被限制
- **WHEN** 管理员尝试为主键盘配置超过 6 个直达入口
- **THEN** 系统提示数量上限，其余入口需放入分组或消息按钮

### Requirement: 分组最多一层
菜单分组 MUST 最多一层：主键盘入口点击后进入的分组内只能包含能力入口（或下一层列表），MUST NOT 出现嵌套分组。

#### Scenario: 一层分组内只有能力入口
- **WHEN** 用户点击某分组
- **THEN** 消息按钮中出现的是能力入口列表，不再出现子分组

### Requirement: 流程走消息按钮
所有流程步骤（选择、预览、确认、退出）MUST 通过消息按钮呈现，用户无需理解菜单层级；任一步骤 MUST 提供返回或退出入口。

#### Scenario: 流程每步可返回
- **WHEN** 用户在流程中任一步骤
- **THEN** 都有返回上一级或回到主菜单的按钮

#### Scenario: 流程每步可退出
- **WHEN** 用户在流程中任一步骤
- **THEN** 都有明确的退出按钮，退出后回到主菜单且不留下进行中状态歧义

```

## docs/openspec/changes/channel-interaction-framework/specs/channel-runtime-ports/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/channel-runtime-ports/spec.md
- Lines: 1-17
- SHA256: 0c419f11c8ac7aac5b7b6aca4fe2ec1411e08d21688b55eba9cd8316ad2685a9

```md
## MODIFIED Requirements

### Requirement: 渠道端口契约
系统 MUST 定义渠道运行时端口，至少覆盖：入站事件（消息/回调查询/媒体/按钮点击）、出站消息（文本/菜单/选项列表/媒体）、媒体桥（上传/下载/转存）与身份映射（外部用户 → 内部用户）。入站事件中的动作 MUST 为能力调用形态（capability_id + params），MUST NOT 是具体业务专属动作。应用层 MUST 仅依赖端口类型，MUST NOT 依赖具体平台 SDK。

#### Scenario: 应用层不感知平台
- **WHEN** 新增一个平台适配器且不改动业务能力应用层
- **THEN** 应用层代码不出现该平台 SDK 类型

## ADDED Requirements

### Requirement: 交互动作泛化为能力调用
系统 MUST 将端口层的交互动作由工作流专属（如打开 Case、确认）泛化为能力调用（capability_id + params + 渠道账户上下文）；适配器把 UI 事件翻译为能力调用，业务结果以渠道无关结构返回。

#### Scenario: 事件翻译为能力调用
- **WHEN** 适配器收到按钮点击事件
- **THEN** 输出包含 capability_id 与 params 的能力调用，而非业务专属动作

```

## docs/openspec/changes/channel-interaction-framework/specs/channel-tg/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/channel-tg/spec.md
- Lines: 1-32
- SHA256: 9f50bb008db811de8aadb9c0465304f47606469ac24ed1bc97e6e890fca6227b

```md
## MODIFIED Requirements

### Requirement: 主菜单来自 Menu 配置
Telegram 适配器展示的主 ReplyKeyboard MUST 由**渠道作用域**菜单的**能力入口**根项生成（含默认种子），根项数量 MUST 不超过 6 个直达入口；MUST NOT 再以源码常量作为唯一长期配置源；菜单读取以渠道 id 为作用域。

#### Scenario: /start 展示配置中的主键盘
- **WHEN** 用户发送 `/start` 或打开主菜单，且该渠道菜单配置可用
- **THEN** 用户收到与配置能力入口顺序/文案一致的 ReplyKeyboard（直达入口 ≤6）

### Requirement: 文件夹下钻浏览 Case
当用户点击分组（文件夹）菜单项时，适配器 MUST 以消息按钮展示该分组的子项与 open_case 能力入口，并提供返回上一级或主菜单的入口。点击能力入口 MUST 进入对应能力流程（open_case 进入既有 Case 预览或填表工作流）。

#### Scenario: 进入文件夹看到子项与 Case
- **WHEN** 用户点击根层分组项，且该分组配置了子分组与/或 open_case 能力入口
- **THEN** 用户收到消息按钮，其中可区分分组与 Case 入口，且可返回

#### Scenario: 进入子分组并可返回
- **WHEN** 用户在消息按钮中点击子分组，再点击返回
- **THEN** 用户回到上一层分组视图或主菜单（按设计的返回语义），过程 MUST NOT 崩溃

#### Scenario: 占位与回复媒体
- **WHEN** 用户点击占位或回复媒体展示项
- **THEN** 行为与既有占位提示 / 发文本与图片 URL 语义一致；单张图片失败 MUST NOT 导致进程崩溃

## ADDED Requirements

### Requirement: 能力入口与流程渲染
TG 适配器 MUST 按能力的渠道渲染声明渲染入口与流程（消息按钮、返回、结果），MUST NOT 为具体业务编写分支；新注册能力加入菜单后，适配器无需改动即可完成事件翻译与结果渲染。

#### Scenario: 新能力入口直接可用
- **WHEN** 管理台把新注册能力加入 TG 菜单并保存
- **THEN** 主键盘或消息按钮出现该入口，点击触发对应能力，无需改适配器

```

## docs/openspec/changes/channel-interaction-framework/specs/tg-menu-admin-api/spec.md

- Source: docs/openspec/changes/channel-interaction-framework/specs/tg-menu-admin-api/spec.md
- Lines: 1-20
- SHA256: ddbd22f39a64323f41d437f9da2f44e19539fca59bca7fac8b49a599531b6688

```md
## MODIFIED Requirements

### Requirement: Menu 树管理 HTTP 接口
admin-api MUST 暴露渠道菜单管理接口（路径约定 `/api/v1/channels/{id}/menu`）：支持按渠道获取完整树形配置，以及整棵树写回。菜单载荷使用能力入口结构（capability_id + params + 展示字段）；未知 capability_id MUST 被拒绝。接口鉴权策略与现有 admin-api 一致（仅内网约定）。

#### Scenario: 获取 Menu 树
- **WHEN** 客户端 GET `/api/v1/channels/{id}/menu` 且服务可用
- **THEN** 返回 200 与当前树形菜单 JSON（含能力入口与展示字段）

#### Scenario: 更新 Menu 树
- **WHEN** 客户端提交合法菜单树进行更新
- **THEN** 返回成功，且随后 GET 可见更新结果

#### Scenario: 非法配置被拒绝
- **WHEN** 客户端提交缺必填、未知 capability_id、非法能力参数、无效 `case_id`、空 reply_media 或违反同层 label 唯一等规则的配置
- **THEN** 返回 4xx 与错误说明，且不破坏既有已保存配置

#### Scenario: 回复媒体非法 URL 被拒绝
- **WHEN** 客户端提交回复媒体展示字段且图片列表含非 http(s) 绝对 URL
- **THEN** 返回 4xx，且不破坏既有已保存配置

```
