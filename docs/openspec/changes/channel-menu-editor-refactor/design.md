## Context

现状（仅描述与本次方案直接相关的部分；动机见 `proposal.md` Why）：
- `web/admin/src/features/menu/menu-card-editor.tsx`（472 行）以「菜单项 / 卡片 / 按钮」三 section 堆叠为骨架，通过 `selectedItemId` + `path: string[]` + `editingButton` 三个状态切换编辑对象，section 之间仅靠 `border` 与 `<h3>` 区分层级；卡片清单（`card-list-panel.tsx`）混在大纲下，孤儿卡片也展示，职责不清。
- `action-form.tsx`（151 行）以单一 `Select` 平铺 7 种 `ActionType`，并以 `action.type === '...'` 逐个分支渲染参数面板；旧版 `open_workflow` 用多选 Checkbox（沿用了 `Action.workflow_ids?: string[]` 数组语义），但实际投放与运行语义都是单数：
  - `quick-config/lib/menu-payload.ts` 的 `addWorkflowMenuEntry(menu, { workflowId: number })` 每次只产 1 个 case；
  - `internal/channel/capability/open_case.go` 的 `start` 走单 `case_id` 立即开始；
  - `list` 模式走 `case_ids[]` 给用户列表选 1 个，是「主菜单一个按钮聚合多个 case」的设想，但实际数据流没有任何生产端产多元素 `workflow_ids`。
- `phone-simulation.tsx` 与编辑器解耦，未与「当前选中」联动。
- 既有约束（按 `pixoma-design-system` 与 `shadcn` skill）：不重复造轮子，优先复用 `components/ui` 已安装的 shadcn 组件，缺什么就 `pnpm dlx shadcn@latest add`；语义类名、内置 variants、`flex + gap`，不写裸色值。

## Goals / Non-Goals

**Goals**
- 左侧 pane 顶部 tab 切换「大纲」/「卡片库」两个视图，职责分明。
- 大纲 = 纯树形结构（菜单项 → 被引用的卡片 → 按钮），不混孤儿卡片。
- 卡片库 = 平铺所有卡片（含未挂载），2 列卡片网格，每张卡显示 name / 正文摘要 / 按钮数 / 引用状态。
- 三个「+ 新建卡片」入口：顶部 tab 区右侧 / 卡片库工具栏 / `open_card` 动作参数面板的二次入口。
- 动作类型选择器改为**单一 `Select` + `optgroup` 分组**（不强制常驻 Tab）；6 种动作类型各自只显所需字段；切动作时参数面板实时换内容。
- 工作流单选；**彻底删除 list 模式**（`workflow_ids[]` 改 `workflow_id: string` 单数；`mode` 字段删除；`open_case.list` 分支删除）。
- 右侧编辑面板按 `selectedNode.kind` 渲染：菜单项 / 卡片 / 按钮各自有专属视图，按钮与菜单项共用同一套参数面板。
- `PhoneSimulation` 与选中态联动：菜单项 → 主键盘；卡片 → 卡片视图；按钮 → 卡片视图内对应按钮高亮。
- 保留 `validateMenuConfig` 与 `menu-editor.contract.test.ts` 关键断言行为不倒退。

**Non-Goals**
- 不动消息平台管理页本身、不动 quick-config wizard、不动 channel-tg 适配器。
- 不引入新第三方 UI 库；只允许通过 shadcn CLI 添加 shadcn 组件。
- 不重写 i18n 体系；只在现有 `menu.*` 命名空间下新增/调整键。
- 不把编辑器抽象为通用能力（保持菜单/卡片场景特化）。
- 不动 `channel-menu-config` / `channel-menu-interaction` 这两个中立 spec（其 REQUIREMENTS 不变）。

## Decisions

### D1 — 布局采用「左 pane（双视图） + 中 pane（编辑器） + 右 pane（PhoneSimulation）」三栏
- 选型：父组件拆为三栏。
  - 左 pane：顶部 tab「大纲 / 卡片库」，内容随 tab 切换；选中态用 `selectedNode: { kind: 'item' | 'card' | 'button'; id: string } | null` 表达。
  - 中 pane：编辑器，按 `selectedNode.kind` 渲染对应视图（item / card / btn），无选中时给出引导。
  - 右 pane：PhoneSimulation，按 `selectedNode.kind` 切换主键盘 / 卡片视图。
- 备选 A：保留 path 栈，仅做 section 视觉强化。否决：仍需用户在三层间手动跟踪，无法解决"找不到当前在编辑哪一层"的核心痛点。
- 备选 B：把三层各占一个 Tab。否决：跨层引用（菜单项引用卡片）会强制切换 Tab，操作割裂。
- 依据：用户原话"现在布局太难看了，交互太难用了，重新设计，不要被已有的束缚"，三栏 + 大纲/卡片库 分视图是产品上最直观的方案。

### D2 — 左 pane 顶部 tab 切换「大纲 / 卡片库」，职责分明
- 选型：
  - 大纲 = 树形结构，只显示被引用的菜单项 / 卡片 / 按钮；不混孤儿卡片。
  - 卡片库 = 2 列网格，平铺所有卡片（含未挂载），显示引用状态（已引用 ● / 未挂载 ○）。
  - 「+ 新建卡片」入口：① 顶部 tab 区右侧（任意视图可见）② 卡片库工具栏 ③ `open_card` 动作参数面板的二次入口。
- 备选 A：在大纲下加「未挂载的卡片」分组。否决：用户反馈"为什么要单独拿出来"——卡片是菜单项的子项，未挂载的卡片不在主大纲里是产品逻辑。
- 备选 B：把卡片管理做成 modal。否决：modal 不利于浏览与批量管理；平铺卡片库更适合找/选/对比。
- 依据：用户对 v7「未挂载的卡片」分组的明确反对；分视图是产品上更清晰的做法。

### D3 — 动作类型选择器用「单一 `Select` + `optgroup` 分组」，不用常驻 Tab
- 选型：在 `ActionForm` 顶部渲染单一 `Select`，用 `<optgroup label="工作流能力">` / `<optgroup label="TG 平台能力">` 分组；当前项左侧加 9px 圆点（绿/蓝）+ 右侧 `src-tag` 徽章同步能力来源；不使用常驻 Tab。
- 备选 A：常驻两个 Tab。否决：用户反馈"为什么要有两个 Tab"——既然当前编辑已是工作流能力，TG Tab 完全是冗余；optgroup 内部分组是更克制的做法。
- 备选 B：把 7 种动作做成可视卡片网格。否决：占用屏宽大、对只有一种工作流能力的现状不经济。
- 依据：用户原话"打开卡片、发文字这些是 tg 特有的能力，工作流是我们的……需要分类"，但分类不必常驻在 Tab 里——选择器内分组 + 大纲 Badge + 动作大卡 src 徽章三处共同传达即可。

### D4 — 6 种动作类型各自精准字段，切换实时换内容
- 选型：`ActionForm` 用 `switch (action.type)` + 早返回：
  - `open_workflow`：菜单项标签 + 工作流单选下拉（必填提示）
  - `open_card`：菜单项标签 + 卡片下拉（从卡片库全部卡片填充）+ 「+ 新建卡片」入口
  - `send_text`：菜单项标签 + 文本 Textarea
  - `send_media`：菜单项标签 + 媒体 URL 多行 Textarea
  - `open_url`：菜单项标签 + URL Input
  - `copy_text`：菜单项标签 + 文本 Textarea
- 切换动作类型时，整个 `ac-body` 区域从 `<template>` 克隆替换；不需要 React 状态机的复杂 diff。
- 依据：用户要求"demo 需要包含所有动作类型"——6 种动作类型都要有明确视觉对照；切类型不相关字段必须不出现。

### D5 — 彻底删除 list 模式死代码
- 选型：
  - `open_workflow` 动作的 case 选择 MUST 用单选控件（shadcn `Select`）；写入时只放 1 个 id。
  - 后端 `Action.workflow_ids?: string[]` 改 `workflow_id: string`（强类型单数），**后端同步收紧**。
  - 后端 `Action.mode?: 'list' | 'direct'` 字段删除。
  - 后端 `internal/channel/capability/open_case.go` 删除 list 分支、`case_ids[]` 参数、`backCaseIDs` 辅助函数。
  - 前端 `WorkflowMenuMode` 类型删除；`addWorkflowMenuEntry({ mode })` 入参去掉 mode。
  - i18n `openModeList` / `openModeDirect` 文案删除。
  - 既有 `internal/...` test 文件里的 `WorkflowIDs: []string{...}` 改 `WorkflowID: "..."`。
- 备选 A：保留多选 Checkbox + 默认 `list`。否决：与 quick-config 单数投放语义不符，用户反馈明确反对。
- 备选 B：保留 list 模式作为高级场景。否决：用户明确确认「打开工作流永远只会有一个」，多工作流通过 `Card.buttons[]` 承载。
- 依据：用户产品澄清"打开工作流只会有一个，例如用户选择了换脸工作流，那就进入换脸工作流的流程了……不会出现我点击了按钮会出来 N 个工作流。只有一种情况，就是打开卡片……卡片下面会有 4 个工作流按钮"。

### D6 — 大纲节点分类 Badge（能力来源 + 数据结构层级）
- 选型：在大纲项旁用一个 `<Badge>`（shadcn 已有）显示：
  - 菜单项与按钮：能力来源（工作流 / TG 卡片 / TG 文字 / TG 媒体 / TG 链接 / TG 复制）
  - 卡片：数据结构层级（卡片），与菜单项/按钮的语义 Badge 视觉区分（用灰色 #475569 调）
- 备选 A：只分两个大类（工作流 / TG）。否决：TG 内部 5 种动作仍是不同行为，运营扫读时仍需打开才能区分；细分标签成本低、价值高。
- 备选 B：直接显示 `ActionType` 英文。否决：违反 i18n 原则。
- 依据：兼顾"清晰明确"与"扫读全部动作分布"两个目标。

### D7 — 右侧编辑面板按 `selectedNode.kind` 渲染，菜单项/按钮共用参数面板
- 选型：
  - 选中菜单项 → 动作类型 + 参数面板（item-editor）
  - 选中卡片 → 卡片编辑（name / text / media / 按钮列表 + 添加按钮 + 删除卡片，card-editor）
  - 选中按钮 → 按钮标签 + 6 种动作任选（btn-editor，与 item-editor 共用同一套参数面板）
- 共用方式：6 种动作的 `<template>` 抽到独立文件/常量，菜单项与按钮都克隆同一份。
- 依据：用户要求"菜单项和按钮用同一套参数面板"——按钮的字段集与菜单项完全一致，避免维护两套。

### D8 — `PhoneSimulation` 按 `selectedNode.kind` 切换视图
- 选型：把 `selectedNode` 通过 props 传进 `PhoneSimulation`：
  - 选中菜单项（item）→ 主键盘视图，按 label 高亮对应按钮
  - 选中卡片（card）→ 卡片视图（卡片正文 + 按钮列表）
  - 选中按钮（btn）→ 卡片视图，按钮高亮
  - 草稿未保存到预览时降级展示「当前编辑未保存到预览」
- 备选 A：把 `PhoneSimulation` 改用 store 全局读取。否决：与 React Query 单一数据源冲突。
- 备选 B：移除 `PhoneSimulation`。否决：实时预览是「交互顺手」的必要条件。
- 依据：用户原话"交互需要合理、顺手、清晰明确"——预览是「顺手」的关键。

### D9 — 关键 `data-testid` 保留以兼容 contract test
- 选型：保留 `action-form` / `card-picker` / `card-list-panel` 这三个 `data-testid`；新增 `outline-pane` / `card-library-pane` / `editor-pane` / `action-type-select` 等用于新结构的断言。`menu-editor.contract.test.ts` 关键断言继续通过。
- 备选 A：删 testid 重写 contract test。否决：原 spec 已验证的行为契约是稳定基线。
- 依据：BREAKING 标记已明确，但 contract 行为是必须保留的护栏。

## Risks / Trade-offs

- [D1 三栏布局在窄屏（< 1280px）会挤压编辑面板] → 用 `min-w-0` + `flex-1` + 内部 `overflow-auto`；在大纲/卡片库项过多时允许 `ScrollArea` 内部滚动，编辑面板始终占据剩余空间。
- [D2 大纲/卡片库切换时选中态不互通] → 选中态用全局 `selectedNode` 表达，切到卡片库点选卡片再切回大纲时仍定位到同一卡片节点；不存在"丢焦点"。
- [D3 单一 Select 内 optgroup 视觉分组能力弱于 Tab] → 用左侧 9px 圆点 + 右侧 `src-tag` + 动作大卡 src 徽章共同强化分类信号。
- [D4 切换动作时整个 ac-body 替换] → 用户视觉感受是"瞬时换"；如要平滑过渡可加 `framer-motion` 的 fade，但本 change 不引入新动画库，保持 React Query + 原生 transition。
- [D5 后端 `workflow_ids[]` 改 `workflow_id` 是 BREAKING 数据变更] → 既有数据迁移策略：取首元素作为新 `workflow_id`；旧 `mode: 'list'` 数据降级为「无 mode 字段」（direct 默认行为）。迁移脚本纳入 tasks。
- [D5 既有 `internal/...` test 改单数字段] → 同步在 tasks 中作为独立条目处理；不在主 UI 任务里。
- [D6 卡片 Badge 与菜单项/按钮 Badge 视觉冲突] → 卡片用灰色 #475569（中性），菜单项/按钮用绿/蓝（能力来源）；视觉层级分明。
- [D7 菜单项/按钮共用参数面板] → 通过 `<template>` 共享；后续若需要为按钮定制字段（如「长按」），可扩展。
- [D8 `PhoneSimulation` 与 `menuDraft` 同步可能短暂不一致] → 在 `useEffect` 中以 `menuDraft ?? menuQuery.data` 为单一数据源，模拟键盘使用同一数据。
- [D9 既有 `menu-editor.contract.test.ts` 内部结构断言可能误伤] → 优先调整 contract test 的内部结构断言，保留端到端 UI 契约断言。

## Migration Plan

1. **数据模型迁移**（后端先行）：发布一个一次性迁移脚本，把既有 `workflow_ids: []string{"X"}` 改 `workflow_id: "X"`；`mode: 'list'` 数据降级为「无 mode 字段」。脚本纳入 tasks。
2. **后端收紧**：`Action` 类型定义收紧；`open_case.go` 删除 list 分支；test 同步；跑 Go test 全绿。
3. **前端落地**：
   - 第一步：`Action` 类型收紧（前端 API 类型同步）；`addWorkflowMenuEntry` 入参去 mode；`menu-payload.test.ts` 同步。
   - 第二步：拆 `MenuCardEditor` 为三栏 + 双视图 + 6 种动作参数面板（按 template 切换）；引入 `OutlinePane` / `CardLibraryPane` / `CardEditorView` / `ButtonEditorView` 组件；`PhoneSimulation` 按 `selectedNode.kind` 切换。
   - 第三步：i18n 增改；shadcn 组件按需添加；新 testid 与现有 contract test 对齐。
4. **验证**：跑 `pnpm test src/features/menu` 与 `pnpm typecheck` 与 `go test ./...`；视觉层面人工核对 demo 路径全部 6 种动作。
5. **回滚**：UI 改动集中在 `features/menu/*` 内；后端 `Action` 字段收紧可通过迁移脚本回滚（反向回填数组）；如遇阻塞可整体 revert `features/menu/*` + 后端对应包，不影响 quick-config 与消息平台管理页。

## Open Questions

- 卡片被删除时若被菜单项引用，是直接降级（菜单项 action 改为 `send_text`）还是拒绝删除 + 提示「先解除引用」？本期实现为「提示并降级」（更轻量、避免阻塞），后续可考虑 strict 模式。
- `Card.buttons[]` 是否有上限（如 8 个）防止运营配超长？本期不设限；`channel-menu-interaction` 已规定「主键盘直达入口数量 MUST 不超过 6 个」，但卡片内按钮上限未规定；可作为后续增强。
