## Context

消息平台层重构已落地（消息平台壳、消息平台作用域菜单、运行时端口、身份消息平台化、热生效装配、notify 路由）。现状痛点：菜单 `kind` 把业务动作硬编码进模型（folder/open_case/placeholder/reply_media），每加业务要改菜单与适配器；端口层 `Action` 是 Case 工作流专属；平台差异以游离 extras 存放；面向用户文案可能泄露内部术语。本次引入「能力注册表 + 统一交互协议 + 消息平台账户上下文」，把「业务能做什么」与「消息平台怎么展示/触发」解耦。

## Goals / Non-Goals

**Goals:**
- 能力注册表：能力 = {id、展示名、参数 schema、执行 handler、各消息平台渲染声明}；新增业务=注册能力
- 菜单改为能力入口树：菜单项 = {展示字段 + capability_id + params + children}
- 统一交互协议：UI 事件 → 能力调用 → 结果 → 消息平台渲染；适配器不感知具体业务
- 消息平台账户上下文：权益挂消息平台账户（消息平台 + 外部用户 id），不做跨消息平台合并
- TG 验证：主键盘=能力入口（≤6），流程走消息按钮；旧 extras 并入渲染声明
- 用户文案术语约束写入规范

**Non-Goals:**
- 会员/计费/签到等具体业务实现（仅框架与 open_case 内置能力）
- 飞书/企微/钉钉适配器实现
- 跨消息平台统一账户
- 消息平台管理改动

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
type RenderDecl struct {                  // 每消息平台渲染声明
    Entry  string `json:"entry"`          // root | message_button
    Config map[string]any `json:"config"` // 如 {columns: 2}
}
type Registry struct{ caps map[string]Capability }
```

能力以 Go 注册（`Register(cap)`），内置 `open_case`（包装现有 `botapp.Facade` 流程）。渲染声明作为能力的一部分按消息平台提供，替代游离 extras。理由：代码注册类型安全、可直接注入 Facade；渲染声明数据化，管理台可读。

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

适配器：UI 事件 → `CapabilityInvoke` → registry 执行 → `Result` → 按消息平台渲染。返回/分组等导航由适配器在消息按钮层处理（不进入能力）。open_case 的「返回列表」通过 params.back 表达。

### D4. 消息平台账户上下文

```go
type AccountCtx struct {
    ChannelID      string
    ExternalUserID string
    InternalUserID string
}
```

能力执行时由适配器（经身份解析）填充；权益读写以（channel_id, external_user_id）为键。不做跨消息平台合并（用户已决策按消息平台账户）。

### D5. TG 渲染

- 根层（无父项）为能力入口，主键盘直达按钮 ≤6（超出部分进「更多」分组或消息按钮）
- 分组（文件夹）不在主键盘展开，点进入后以消息按钮列出子项与 open_case 入口，可返回
- 每行按钮数等 TG 展示参数：并入能力渲染声明（`tg` 消息平台），不再有独立 extras 表编辑
- open_case 保留现有 Case 预览/填表/确认/结果，行为等价

### D6. 管理台

菜单编辑器：入口类型改为「选择能力（下拉，来自 registry）→ 按 ParamsSchema 渲染参数表单 → 消息平台展示微调」。移除 kind 选择与 extras 独立 tab；旧 extras 数据迁移为渲染声明参数。

### D7. 文案术语约束

在 `channel-interaction-protocol` 与 `admin-web-shell` 规格中加入文案约束；管理台与 bot 文案禁用 inline/callback/extras/capability 等内部术语，统一「按钮/选项/功能/返回」。

### D8. 交互设计（用户端 + 管理端）

**用户端（bot 体验）——已确认三项决策：**

- **主键盘显式配置**：主键盘直达入口由管理员在菜单配置中显式添加，≤6 个；平台不自动塞满。超出部分必须放分组或消息按钮。
- **分组最多一层**：分组不在主键盘展开；点击分组后消息按钮只列出能力入口（无嵌套分组）。
- **流程走消息按钮**：选择/预览/确认/退出全部用消息按钮；任一步骤有返回（上一级或主菜单）与退出按钮。

```
主键盘（显式配置，≤6）：
  [开 Case] [签到] [充值] [会员] [帮助]
        │
        ▼
分组（一层，消息按钮）：
  [图片 A · ¥10] [图片 B · ¥15]
  [⬅️ 返回]
        │
        ▼
流程（消息按钮）：
  预览 → [▶ 开始] [« 返回]
  输入 → [✕ 退出]
  确认 → [✅ 确认] [✕ 退出]
```

**管理端（配置体验）：**

- 能力为中心的编辑器：入口类型 = 选能力（下拉，来自注册表）→ schema 参数表单 → 消息平台展示微调
- **消息平台形态预览**：按当前配置模拟渲染主键盘与消息按钮（TG），保存前可见效果；预览遵循主键盘显式配置与一层分组规则
- 直白文案：不出现 inline/capability/extras 等内部术语（见 D7）

## Risks / Trade-offs

- [菜单模型破坏性变更] → 无存量数据，直接替换；迁移脚本把旧 kind 映射为能力入口（folder→分组、open_case→能力、placeholder/reply→展示字段）
- [适配器重构回归（Case 流程）] → open_case 能力包装现有 Facade，回归测试覆盖预览/填表/确认/结果
- [能力注册与渲染声明耦合] → 渲染声明数据化 + 按消息平台隔离，未声明消息平台不展示
- [extras 迁移遗漏] → tg_root_layout 迁移为 open_case 的 tg 渲染声明；管理台移除 extras 入口

## Migration Plan

1. `internal/channel/capability` 注册表 + `open_case` 能力（包装现有 Facade）
2. 菜单模型改能力入口（domain/persistence/API 载荷）
3. 端口 Action → CapabilityInvoke；TG 适配器按协议渲染（主键盘入口 + 消息按钮流程）
4. 管理台菜单编辑器改版（能力选择 + schema 表单）
5. 数据迁移：旧 kind → 能力入口；tg_root_layout extras → 渲染声明
6. 全量测试 + TG 手工回归（Case 流程等价）

## Open Questions

- 渲染声明的存储位置（能力定义内 vs 菜单项内）可在实现期定：倾向「能力定义内按消息平台 + 菜单项可覆盖」
- 参数 schema 采用 JSON Schema 子集还是轻量自定义（管理台表单驱动）：实现期确定，不影响规格
