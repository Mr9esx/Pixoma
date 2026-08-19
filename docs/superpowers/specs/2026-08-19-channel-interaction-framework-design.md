---
comet_change: channel-interaction-framework
role: technical-design
canonical_spec: openspec
---

# 渠道交互框架：深度技术设计（channel-interaction-framework）

> 上游事实源：OpenSpec change `channel-interaction-framework` 的 proposal.md / design.md / specs/**。本文是设计阶段的深度技术细化，重点覆盖能力注册表、交互协议、菜单模型与 TG 交互落地。

## 1. 包结构与分层

```
internal/channel/capability/
  capability.go        Capability 接口、ParamSpec 包装、Result、AccountCtx、NavContext
  registry.go          Registry：注册/查询/按渠道筛选渲染声明
  open_case.go         内置 open_case 能力（包装 botapp.Facade）
internal/channel/protocol/
  invoke.go            CapabilityInvoke / Result / Nav 类型
  render.go            渠道渲染 DTO（主键盘/消息按钮）与预览入口
internal/menu/         菜单模型改能力入口（MenuNode.CapabilityID + Params）
internal/channel/tg/   适配器：事件 → CapabilityInvoke；Result → 消息按钮；Nav 统一返回/退出
internal/httpapi/channelmenu/  菜单 API（capability_id 载荷）+ 预览 DTO API
web/admin/src/features/channels/  编辑器（能力选择 + JSON Schema 表单 + 预览）
```

边界：`capability` 不 import `channel/tg`；适配器只做协议翻译；菜单模型不感知具体能力。

## 2. 能力注册表

```go
type Capability interface {
    ID() string
    DisplayName() string
    ParamsSchema() json.RawMessage            // JSON Schema（draft-07 子集）
    Render(channelID string, overrides map[string]any) (RenderDecl, error)
    Invoke(ctx context.Context, acct AccountCtx, nav Nav, params map[string]any) (Result, error)
}
type RenderDecl struct {
    Entry  string         `json:"entry"`      // root | message_button | group
    Config map[string]any `json:"config"`     // 如 {"columns":2}
}
type AccountCtx struct {
    ChannelID      string `json:"channel_id"`
    ExternalUserID string `json:"external_user_id"`
    InternalUserID string `json:"internal_user_id"`
}
```

- Registry：`Register(cap)` + `Get(id)` + `List()` + `ForChannel(channelID)`；重复注册报错
- 参数校验：`github.com/santhosh-tekuri/jsonschema/v6`（或等价库）编译能力 schema，调用前校验 params；错误映射为可读参数错误
- 渲染声明：能力定义内按渠道提供默认值，菜单项可通过 `render_override` 覆盖（合并语义：override 只替换指定 key）
- open_case：包装现有 Facade；params schema：`{case_ids: [string] 或 case_ref，列表锚点}`；Invoke 按流程状态分发（列表→预览→开始→填表→确认），back 由 Nav 承担

## 3. 交互协议

```go
type Nav struct {
    Back  string `json:"back"`   // "root" | 分组 id | 流程步锚点
    Step  string `json:"step,omitempty"`
}
type CapabilityInvoke struct {
    CapabilityID string         `json:"capability_id"`
    Params       map[string]any `json:"params"`
    Account      AccountCtx     `json:"account"`
    Nav          Nav            `json:"nav"`
}
type Result struct {
    Text    string     `json:"text"`
    Options []Option   `json:"options"`
    Media   []MediaRef `json:"media"`
    Error   *string    `json:"error,omitempty"`
}
type Option struct {
    Label string         `json:"label"`
    Value map[string]any `json:"value"` // 作为下一跳 params（可含下一 Nav）
}
```

- 适配器翻译：UI 事件（按钮/文本/媒体）→ `CapabilityInvoke` → `Registry.Invoke` → `Result` → 渠道渲染
- 导航：返回/退出按钮由适配器在渲染层统一生成（Nav.Back = root / 分组 id）；能力不感知导航栈
- 执行上下文：适配器经 `IdentityResolver` 填充 `AccountCtx`（渠道账户，无跨渠道合并）

## 4. 菜单模型与持久化

```go
type MenuNode struct {
    ID, ParentID, Label string
    Order, Enabled      ...
    IntroText, PlaceholderText string
    Reply *ReplyPayload          // 展示字段
    CapabilityID string           `json:"capability_id,omitempty"`
    Params       map[string]any   `json:"params,omitempty"`
    RenderOverride map[string]any `json:"render_override,omitempty"`
    Children []MenuNode
}
```

- 校验：未知 capability_id 拒绝；params 按能力 JSON Schema 校验；一层分组（分组节点下不允许再嵌套分组）；根层直达入口 ≤6
- 持久化：`channel_menu_items` 增加 `capability_id`、`params_json`、`render_override_json`；`kind` 列废弃
- 迁移（无存量数据但保留脚本能力）：旧 kind → 能力入口映射；`tg_root_layout` extras → open_case 的 tg 渲染覆盖
- 种子：根层 open_case 入口（挂载图片 Case）+ 可选「更多」分组

## 5. TG 渲染（交互设计落地）

- 主键盘：由管理员显式配置的根层能力入口生成，≤6；超出在管理端提示放入分组
- 分组：一层；点击分组后发消息按钮（能力入口 + 返回）
- 流程：消息按钮分步；每步由 Nav 生成返回/退出；退出回到主菜单
- 每行按钮数：读取能力 tg 渲染声明（默认 2），菜单项 RenderOverride 可覆盖
- 预览 DTO 与真实渲染共用同一构建函数（`BuildKeyboardPreview(tree, decls)` / `BuildMessageButtons(flow)`），保证同源

## 6. 管理端

- 编辑器：入口类型 = 能力下拉（来自 registry 清单）+ JSON Schema 表单（自动生成）+ 渠道展示微调（render override）
- 预览：`GET /api/v1/channels/{id}/menu/preview` 返回结构化预览（主键盘行列 + 各分组消息按钮）；前端绘制；文案直白
- 移除 kind 选择与 extras 独立入口；i18n 文案术语约束

## 7. 预览 DTO API

```json
{
  "main_keyboard": [["开 Case","签到"],["充值","帮助"]],
  "groups": { "video": { "title":"视频专区", "buttons":[{"label":"图片 B · ¥15","action":"capability:open_case:..."}] } }
}
```

后端渲染函数与真实适配器共用（同一份菜单+能力声明 → 同一结构），预览即真实。

## 8. 错误处理与边界

- 参数校验失败：可读参数错误（不暴露内部术语）
- 未知能力：菜单校验拒绝；运行时事件无法翻译 → 通用「操作无效」+ 返回主菜单
- 能力执行失败：Result.Error 渲染为直白提示 + 重试/返回
- Nav 指向不存在分组：回到主菜单并记录告警

## 9. 测试策略

- 单元：JSON Schema 校验、Render 合并（override）、Nav 生成返回按钮、AccountCtx
- 适配器：事件→CapabilityInvoke 翻译、主键盘 ≤6、一层分组、返回/退出、预览 DTO 与真实渲染一致性
- open_case 回归：预览/开始/填表/确认/出图等价
- API/前端：菜单载荷、schema 表单、预览绘制
- 全量 `go test ./...` + `pnpm test/build`

## 10. 实施顺序（与 tasks 对齐）

1. capability 注册表 + open_case（tasks 1）
2. 菜单模型能力入口 + 迁移（tasks 2）
3. 交互协议 CapabilityInvoke/Result/Nav/AccountCtx（tasks 3）
4. TG 适配器按协议渲染 + 交互设计落地（tasks 4/6）
5. 管理端编辑器 + 预览 DTO（tasks 5/6.4）
6. 测试与回归（tasks 7）

## 11. 风险与缓解

| 风险 | 缓解 |
|---|---|
| open_case 流程回归 | 等价回归测试（预览/填表/确认/出图） |
| 预览与真实渲染漂移 | 共用构建函数 + 一致性测试 |
| JSON Schema 依赖引入 | 限定 draft-07 子集，表单与校验共用一份 schema |
| 菜单模型破坏性变更 | 无存量数据，直接替换；迁移脚本保留 |
