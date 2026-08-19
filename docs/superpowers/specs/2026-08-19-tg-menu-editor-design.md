---
role: technical-design
status: accepted
---

# TG 菜单配置与预览：所见即所得编辑器 + 能力驱动参数表单

## 1. 背景与问题

渠道详情页的「菜单」Tab 是左树右表单的开发者视角：

- 暴露 `ID`、`order`、`placeholder_text`、`reply`、`intro_text`、`capability_id`、`render_override` 等内部字段，用户必须理解菜单模型才能配置；
- 「无（展示/分组）」等开发话术直接出现在功能选择里；
- 预览在另一个 Tab，是按钮 chips + 分组列表的开发视图，不像真实的 Telegram 聊天；改完要切 Tab 刷新。

交互框架（channel-interaction-framework）已定义方向：菜单 = 能力入口树、主键盘显式配置（≤6）、一层分组、管理端「选能力 → schema 参数表单 → 渠道形态预览」。本次按 Case 编辑器改造的思路落地为具体交互。

## 2. 目标与边界

**目标：**

- 不熟悉 Telegram / 菜单内部概念的人也能把机器人菜单配明白；
- 配置和预览合一：所见即所得，左边 TG 手机模拟、右边配当前选中的按钮，实时联动；
- 按钮行为统一为「能力」：选能力 → 由能力注册表的参数 schema 驱动表单（不写死 open_case 特判）；
- 「提示文字」「回复图文」注册为内置展示能力，编辑器逻辑完全统一；
- 保存前校验：主键盘 ≤6、一层分组、按钮名必填、open_case 至少 1 个工作流、能力参数按 schema 校验。

**非目标：**

- 不做用户端深流程模拟（手机模拟只到「点按钮后看到什么」这一层；填表/确认/执行结果不在模拟范围）；
- 不做画布式树编辑（不拖拽连线，仍用列表 + 配置面板）；
- 不实现签到/计费/会员等新能力；
- 不做飞书/企微等其它渠道适配器；
- 不保留旧 `kind` 语义（folder/open_case/placeholder/reply_media 业务 kind 移除；旧数据经一次性兼容转换）。

## 3. 已确认决定

| 项 | 选择 |
|---|---|
| 布局 | 所见即所得双栏：左 TG 手机模拟（实时、可点击），右配置面板（当前选中项）；顶栏 = 渠道名 + 启用 + 保存（右上角） |
| 按钮行为 | 「显示子按钮（分组）」特殊项 + 注册表能力列表；选能力后由 `params_schema` 驱动参数表单 |
| 内置展示能力 | `reply_text`（提示文字，params: text）、`reply_media`（回复图文，params: text + images[]） |
| open_case 展示名 | 「开 Case」→「打开工作流」（与 Case→工作流改名一致） |
| 预览 | 前端本地模拟（纯函数复刻 `BuildKeyboardLayout` / `BuildPreview`），不改服务端预览端点 |
| 保存校验 | 主键盘 ≤6；一层分组；按钮名必填；非分组按钮必须有能力；open_case 至少 1 个 case_id；能力参数按 schema 校验 |
| 旧数据 | Get 时把旧 `placeholder_text` / `reply` 菜单项兼容转换为 `reply_text` / `reply_media` 能力条目；Put 后以新形态落库 |
| 高级模式 | ID / 排序 / 原始 JSON 折叠在配置面板底部，默认只读，二次确认可编辑 |

## 4. 交互设计

### 4.1 页面结构

```
┌────────────────────────────────────────────────────────────┐
│ 图片助手        ● 启用        未保存：2 处    [保存]        │
├──────────────────────────────┬─────────────────────────────┤
│ 用户视角（Telegram）          │ 当前编辑：📂 工具            │
│ ┌──────────────────────┐     │ 按钮上显示的名字             │
│ │ 聊天消息区            │     │ [📂 工具]                   │
│ │ 📂 工具               │     │ 这个按钮做什么               │
│ │ 处理图片的小工具们👇    │     │ [显示子按钮 ▾]              │
│ │ [AI 修图] [抠图]      │     │ 点进来先看到的文字           │
│ │ ‹ 返回主菜单           │     │ [……]                       │
│ │ [输入条]              │     │ 子按钮（按顺序显示）         │
│ │ [🖼 图片] [🎬 视频]    │     │ [AI 修图 ›][抠图 ›]         │
│ │ [📂 工具] [💰 充值]    │     │ [＋ 加一个子按钮]            │
│ └──────────────────────┘     │ ────────────────            │
│ ⚠ 只支持一层子按钮           │ 主键盘整体：每行按钮数 [2 ▾] │
│ 主键盘建议 ≤6 个             │ 高级（ID / 排序）▾           │
└──────────────────────────────┴─────────────────────────────┘
```

### 4.2 交互规则

- **点哪配哪**：点手机里的主键盘按钮 / 子按钮，右侧切换到该项配置；右侧改动实时反映到手机模拟。
- **功能选择**：下拉 = 「显示子按钮（分组）」 + 注册表能力（如「打开工作流」「提示文字」「回复图文」）。选择能力后下方出现该能力的参数表单（由后端 `params_schema` 渲染）。
- **主键盘整体**：每行按钮数（`render_override.columns`，1-8，默认 2），顶部实时重排。
- **保存**：顶栏右侧；有未保存改动时显示「未保存：N 处」；校验不通过显示红条 + 联动标红，问题清空才可点。
- **高级模式**：ID / 排序 / 原始 JSON 折叠，默认只读；进入编辑需二次确认。

### 4.3 能力参数表单（schema 驱动）

- `open_case`：schema 声明 `case_ids: string[]` → 表单渲染为「选择工作流」（可搜索、多选，显示名称 + 价格）。
- `reply_text`：schema 声明 `text: string` → 表单渲染为单行/多行文本。
- `reply_media`：schema 声明 `text: string` + `images: string[]` → 表单渲染为回复文字 + 图片链接列表。
- 前端维护一个 JSON Schema 子集 → 表单控件的映射（`string` / `number` / `boolean` / `array of string` / `enum`），未知类型降级为原始输入框。
- 表单只渲染**管理端参数 schema**（`AdminParamsSchema`）：`open_case` 的完整 schema 含运行期参数（`step`/`text`/`blob` 等），管理端只暴露 `case_ids`。

### 4.4 保存校验

| 规则 | 表现 |
|---|---|
| 主键盘按钮 > 6 | 红条提示，手机模拟主键盘标红 |
| 分组内嵌套分组 | 校验拒绝；编辑器层面对分组项不提供「添加子分组」入口 |
| 按钮名为空 | 该项配置面板标红 |
| 非分组按钮未选能力 | 功能选择标红 |
| open_case 未选任何工作流 | 参数表单标红 |
| 能力参数不符合 schema | 对应字段标红 + 后端兜底拒绝 |

## 5. 后端改动（A+C 范围）

### 5.1 能力注册表

- 新增内置展示能力（`internal/channel/capability`）：
  - `ReplyText`：`ID() = "reply_text"`，`DisplayName() = "提示文字"`，`ParamsSchema()` = `{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`；`Invoke` 返回文本 Result。
  - `ReplyMedia`：`ID() = "reply_media"`，`DisplayName() = "回复图文"`，`ParamsSchema()` = `{"type":"object","properties":{"text":{"type":"string"},"images":{"type":"array","items":{"type":"string"}}},...}`；`Invoke` 返回媒体 Result。
  - 两个能力在 admin-api 与 pixoma 启动时注册（与 open_case 同位置）。
- `open_case` 的 `DisplayName()` 从「开 Case」改为「打开工作流」。
- 能力接口新增 `AdminParamsSchema() json.RawMessage`：管理端表单按此渲染；默认返回完整 `ParamsSchema()`，`open_case` 覆写为仅 `case_ids`（运行期参数不暴露）。

### 5.2 管理 API

`ListCapabilities` DTO 增加 `params_schema`（= 各能力的 `AdminParamsSchema()`，raw JSON 透传）：

```json
{ "id": "open_case", "display_name": "打开工作流", "params_schema": { "type": "object", "properties": { "case_ids": { "type": "array", "items": { "type": "string" } } } } }
```

### 5.3 菜单运行时与旧数据

- `menuItemDispatch` 统一走能力：`capability_id != ""` → `Registry.Invoke`（open_case / reply_text / reply_media 各自渲染）；有 children 且无能力 → 分组（`showGroup`）；两者皆无 → 校验层拒绝（旧数据除外）。
- 兼容转换：在 `application.Service.Get` 读路径对旧条目做一次性转换——`placeholder_text` → `reply_text` 能力 + `params.text`；`reply` → `reply_media` 能力 + `params.text/images`；转换结果仅在响应中生效，Put 后落库为新形态。
- `menu/domain/validate.go` 增加：root ≤6；一层分组；非分组节点必须有 `capability_id`；能力参数按注册表 schema 校验（`ValidateParams` 已存在）。

### 5.4 前端技术要点

- 纯函数 `buildTgSimulation(tree)`（`web/admin/src/features/channels/menu-simulation.ts`）：复刻 `BuildKeyboardLayout` 与分组消息渲染，供手机模拟使用，含单测。
- schema → 表单渲染器 `renderParamsForm(schema, value)`（`web/admin/src/features/channels/params-form.tsx`）：按 JSON Schema 子集渲染控件。
- 编辑器改造：`channel-menu-editor.tsx` 重构为双栏；保留树数据模型与 PUT 全量保存；`NodeEditor` 改为 schema 驱动。
- i18n：新增「这个按钮做什么」「用户点这个按钮后…」「未保存：{{count}} 处」「主键盘建议不超过 6 个按钮」等大白话文案，zh/en 成对。

## 6. 测试策略

- 前端：`menu-simulation` 单测（主键盘布局 / 分组消息 / 禁用项过滤）；`params-form` 契约测试（schema 子集 → 控件、未知类型降级）；编辑器 contract 测试（双栏结构、能力选择来自 registry、保存校验文案）。
- 后端：`reply_text` / `reply_media` Invoke 测试；`ListCapabilities` 返回 schema 的 handler 测试；旧数据兼容转换测试；validate 新规则测试（root ≤6 / 一层分组 / 无能力拒绝）。
- 端到端：mock TG 下「打开工作流 → 工作流列表 → 填表 → 确认 → 结果」回归不变。

## 7. 风险

- JSON Schema 子集渲染覆盖不全：未知控件降级为原始输入，不阻塞配置。
- 旧数据转换语义差异：`placeholder_text` 原本「留空显示 XX：暂未开放」，转换后 `reply_text` 需要保留该回退文案（缺 text 时用「{{label}}：暂未开放」）。
- 深流程模拟（填表/确认）不做，可能导致用户以为点按钮能看到全流程；在手机模拟区注明「模拟到这一步，真实流程以保存后为准」。
