# Comet Design Handoff

- Change: channel-menu-editor-refactor
- Phase: design
- Mode: compact
- Context hash: 2d36b12aa71a9914485f50f147ed9bb181fdf8ee1eafb762d3d78c7bc7f25b0b

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/channel-menu-editor-refactor/proposal.md

- Source: docs/openspec/changes/channel-menu-editor-refactor/proposal.md
- Lines: 1-73
- SHA256: 0c91bbca3736d07609621a46af89d7d59f6b8526be34af40f58f92b7f2fc8129

```md
## Why

消息平台菜单/卡片编辑器（`web/admin/src/features/menu`）当前面临三类问题：

1. **布局/交互陈旧**：菜单项 / 卡片 / 卡片按钮三层切换靠「path 栈 + 三 section 堆叠」，没有大纲视图，找不到当前在编辑哪一层；按钮标签与动作类型都挤在同一列，没有预览也没有视觉分层。
2. **动作分类不清**：所有 7 种 `ActionType`（`open_card` / `open_workflow` / `send_text` / `send_media` / `open_url` / `copy_text` / `placeholder`）平铺在一个 `Select` 下拉里，运营一眼分不清「打开卡片 / 发文字」是 Telegram 平台能力，「打开工作流」是平台中立能力。
3. **底层设计冗余（list 模式是死代码）**：旧 `action-form.tsx` 用多选 Checkbox 表达 `open_workflow`，但 `quick-config/lib/menu-payload.ts` 每次只产单数 `workflowId`；`internal/channel/capability/open_case.go` 的 `list` 分支、`mode: 'list'` 字段、`workflow_ids: string[]` 数组均无任何生产端产多元素数据，是设计时设想的"主菜单一个按钮聚合多个 case"但实际不可达的死代码。多工作流场景的真正承载点是 `Card.buttons[]`（1 个菜单项 → open_card → 1 张卡片 → N 个按钮，每个按钮 1 个 case），已在完整路径上。

本次重构不破坏既有 API 契约，但整体重做 UI（双视图 + 大纲/卡片库 + 单一 Select + optgroup + 6 种动作各自精准字段），并**彻底清理** list 模式相关死代码。

## What Changes

### UI 与交互
- 左侧 pane 顶部 tab 切换两个视图：「大纲」与「卡片库」，默认在大纲。
- **大纲** = 纯树形结构（菜单项 → 被引用的卡片 → 按钮）；不混"未挂载的卡片"。菜单项下挂载其 `open_card` 引用的卡片，卡片下挂载其按钮。
- **卡片库** = 2 列卡片网格，平铺所有卡片（含未挂载），每张卡显示「name / 正文摘要 / 按钮数 / 引用状态（已引用 ● / 未挂载 ○）」，可点选进入编辑。
- 三个「+ 新建卡片」入口：① 顶部 tab 区右侧（任意视图可见）② 卡片库工具栏 ③ `open_card` 动作参数面板的二次入口。三处都打开同一个对话框（name + 简介），提交后立即在卡片库出现并进入编辑。
- 大纲项旁加 `Badge` 标注能力来源：「工作流 / TG 卡片 / TG 文字 / TG 媒体 / TG 链接 / TG 复制」+「卡片」（区分数据结构层级）。
- 动作类型选择器改为**单一 `Select` + `optgroup` 分组**（工作流能力 / TG 平台能力），当前项左侧 9px 圆点（绿/蓝）+ 右侧 `src-tag` 同步能力来源；**不**用常驻 Tab。
- 6 种动作类型（`open_workflow` / `open_card` / `send_text` / `send_media` / `open_url` / `copy_text`）各自只显所需字段，按 `action.type` 严格 `switch` 早返回；切换动作类型时参数面板实时换内容。
- 右侧编辑面板按 `selectedNode.kind` 渲染：
  - 菜单项（item）→ 动作类型 + 参数面板
  - 卡片（card）→ name / text（卡片正文）/ media（媒体 URL 多行）/ 按钮列表 + 「+ 添加按钮」+ 「删除该卡片」
  - 按钮（btn）→ 按钮标签 + 6 种动作任选（与菜单项共用同一套参数面板）
- `PhoneSimulation` 与选中态联动：
  - 选中菜单项 → 主键盘视图
  - 选中卡片 → 卡片视图（卡片正文 + 按钮列表）
  - 选中按钮 → 卡片视图中对应按钮高亮
  - 草稿未保存到预览时降级展示「当前编辑未保存到预览」
- 必要的 i18n（zh/en）新增/调整：动作分类标题、分组标签、能力来源 Badge、卡片库相关文案、引导文案、提示语。
- 必要时按 shadcn CLI 引入 Tabs / ScrollArea / Sheet 等组件（不重复造轮子），遵循 Pixoma 设计系统 `pixoma-design-system` 与 `shadcn` skill。

### 后端死代码清理（彻底路径 B+）
- **`Action.workflow_ids?: string[]` → `workflow_id: string`**（强类型单数）
- **`Action.mode?: 'list' | 'direct'` → 删除**（list 模式无产品语义）
- **`internal/channel/capability/open_case.go` 删除 list 分支、`case_ids[]` 参数、`backCaseIDs` 辅助函数**
- 同步更新 `internal/...` 多个 test 文件里的 `WorkflowIDs: []string{...}` → `WorkflowID: "10"`
- 同步更新 `web/admin/src/lib/api/channel-menu.ts` 的 `Action` 类型、`features/menu/lib/menu-flow.ts` 的 `validateAction`、`features/quick-config/lib/menu-payload.ts` 的 `addWorkflowMenuEntry` 入参（移除 `mode`）、`features/quick-config/step3-channels.tsx` 的 `WorkflowMenuMode` 类型
- 同步更新 i18n `openModeList` / `openModeDirect` 等文案

### 既有契约保留
- `validateMenuConfig` 关键 UI 契约（`action-form` / `card-picker` / `card-list-panel` 的 `data-testid`）继续保留
- `putMenu` / `createCard` / `updateCard` / `deleteCard` 调用方式不变
- `menu-editor.contract.test.ts` 端到端契约断言继续通过
- 不影响后端 `/api/v1/channels/.../menu` 与 `/cards` 整体契约（仅 `Action` 字段收紧）

### 必要 BREAKING
- **BREAKING（UI）**：编辑器组件的 props / `data-testid` 内部结构不再完全兼容
- **BREAKING（后端）**：`Action.workflow_ids` 字段名与类型变更；`mode` 字段删除；既有数据需迁移（多元素数组降级为取首元素，`mode: 'list'` 数据降级为 `mode: 'direct'` 默认行为或保留为「无 mode 字段」）

## Capabilities

### New Capabilities
- `channel-menu-editor`: 管理端消息平台菜单/卡片编辑器的 UI 行为规范——左 pane 双视图（大纲 / 卡片库）切换、动作按能力来源（工作流 / TG 平台）单一 Select + optgroup 分组、6 种动作类型各自精准字段、卡片库平铺管理、PhoneSimulation 跟随选中层级联动；并彻底清理 `open_workflow` 的 list 模式死代码（`workflow_ids[]` / `mode` / `open_case.list`）。

### Modified Capabilities
- 无。`channel-menu-config`（平台中立菜单模型）与 `channel-menu-interaction`（bot 内交互体验）的 REQUIREMENTS 不变；`quick-config-wizard` 的 `open_workflow` 投放入口（`addWorkflowMenuEntry`）入参变化由本 change 承担，spec 本身不变。

## Impact

- 前端 `web/admin/src/features/menu/`：
  - `menu-card-editor.tsx`：从「三 section 堆叠」改为「左 pane（tab：大纲/卡片库）+ 中 pane（编辑器按 selectedNode.kind 渲染）+ 右 pane（PhoneSimulation）」三栏布局；引入 `OutlinePane` / `CardLibraryPane` / `CardEditorView` / `ButtonEditorView` 组件
  - `action-form.tsx`：把 `Select` 拆为单一 `Select` + `optgroup` 分组；6 种动作参数面板按 `switch (action.type)` 早返回；`open_workflow` 改为单选下拉
  - `card-picker.tsx` / `card-list-panel.tsx`：合并到「卡片库」视图的 `CardLibraryPane`，结构与 `data-testid` 尽量保持稳定
  - `phone-simulation.tsx`：在保留核心算法的前提下，根据 `selectedNode.kind` 切换主键盘 / 卡片视图；草稿未保存时降级
  - i18n：新增/调整 `menu.*` 文案键（中英文同步）
- 前端 `web/admin/src/features/quick-config/`：`step3-channels.tsx` 与 `lib/menu-payload.ts` 移除 `mode` 字段入参；`addWorkflowMenuEntry` 写出的 `Action` 自动不含 `mode` 字段
- 后端 `internal/channel/capability/open_case.go`：删除 list 分支、`case_ids[]` 参数、`backCaseIDs` 辅助
- 后端 `internal/caseadmin/...`、`internal/httpapi/menucards/...` 等 test：单数 `WorkflowID: "..."` 替代 `WorkflowIDs: []string{...}`
- 数据：`Action.workflow_id: string`（强类型单数，替代 `workflow_ids?: string[]`）；`Action.mode` 字段删除
- 依赖/i18n：新增卡片库、动作分类、6 种动作参数面板相关文案
- 设计系统：复用 shadcn 已有 Tabs / ScrollArea / Sheet 等组件，按 `pixoma-design-system-skill` 视觉令牌落地
- 不影响：消息平台管理页本身；quick-config wizard 流程主结构；bot 端 TG 适配器；`channel-menu-config` / `channel-menu-interaction` 既有 spec

```

## docs/openspec/changes/channel-menu-editor-refactor/design.md

- Source: docs/openspec/changes/channel-menu-editor-refactor/design.md
- Lines: 1-140
- SHA256: d6bca3dc71bf7b60b77c3d092041c4177aa625cd3b46f45bba033d4e663712d1

[TRUNCATED]

```md
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

```

Full source: docs/openspec/changes/channel-menu-editor-refactor/design.md

## docs/openspec/changes/channel-menu-editor-refactor/tasks.md

- Source: docs/openspec/changes/channel-menu-editor-refactor/tasks.md
- Lines: 1-99
- SHA256: 7e022cc93bc93ea8b7000cbb1cf5fb2a5512563d709f0b1d92bba95e9745554b

[TRUNCATED]

```md
## 1. 基础与依赖

- [ ] 1.1 盘点 `web/admin/src/components/ui` 已安装的 shadcn 组件，确认是否需要 Tabs / Badge / ScrollArea / Sheet / Dialog
- [ ] 1.2 用 `pnpm dlx shadcn@latest add` 补齐缺失组件；保持与 `pixoma-design-system-skill` 视觉令牌一致
- [ ] 1.3 读取 `web/admin/src/features/menu/menu-editor.contract.test.ts` 与 `action-form.tsx` / `card-picker.tsx` / `card-list-panel.tsx` 的现有 `data-testid`，列出会被复用的护栏
- [ ] 1.4 确认 `internal/channel/capability/open_case.go` 的 list 分支与 `backCaseIDs` 调用点，列出需清理的引用

## 2. 后端数据模型与能力清理（彻底删除 list 模式死代码）

- [ ] 2.1 写一次性数据迁移脚本：把既有 `workflow_ids: []string{"X"}` 改 `workflow_id: "X"`（取首元素）；`mode: 'list'` 数据降级为「无 mode 字段」
- [ ] 2.2 后端 `Action` 类型定义：`workflow_ids?: string[]` → `workflow_id: string`；删除 `mode?: 'list' | 'direct'`
- [ ] 2.3 `internal/channel/capability/open_case.go` 删除 list 分支（`case "list"`）、`case_ids[]` 参数、`backCaseIDs` 辅助函数
- [ ] 2.4 同步更新 `internal/caseadmin/...` / `internal/httpapi/menucards/...` / `internal/channel/capability/open_case_test.go` 等 test 文件里的 `WorkflowIDs: []string{...}` → `WorkflowID: "..."`
- [ ] 2.5 跑 `go test ./...` 全部通过；不破坏既有 contract（开放 admin API 行为不变）

## 3. 前端 API 类型与 quick-config 同步

- [ ] 3.1 `web/admin/src/lib/api/channel-menu.ts` 的 `Action` 类型：`workflow_ids?: string[]` → `workflow_id?: string`；删除 `mode?: 'list' | 'direct'`
- [ ] 3.2 `web/admin/src/features/menu/lib/menu-flow.ts` 的 `validateAction`：`workflow_ids.length === 0` 校验改 `!workflow_id`；删除 `mode` 相关校验
- [ ] 3.3 `web/admin/src/features/quick-config/lib/menu-payload.ts` 的 `addWorkflowMenuEntry` 入参：删除 `mode` 字段；写入时产 `workflow_id: String(entry.workflowId)`
- [ ] 3.4 `web/admin/src/features/quick-config/step3-channels.tsx` 的 `WorkflowMenuMode` 类型删除；UI 上不再有「打开方式」切换
- [ ] 3.5 同步 `web/admin/src/features/menu/lib/menu-flow.test.ts` / `web/admin/src/features/quick-config/lib/menu-payload.test.ts` 等 test

## 4. 左 pane 双视图（大纲 / 卡片库）

- [ ] 4.1 在 `web/admin/src/features/menu/` 新建 `outline-pane.tsx`：消费 `menu.items` / `cards`，按「菜单项 → 被引用的卡片 → 按钮」三层渲染
- [ ] 4.2 大纲项旁加 `Badge`：菜单项/按钮的能力来源（工作流 / TG 卡片 / TG 文字 / TG 媒体 / TG 链接 / TG 复制）；卡片层级标「卡片」
- [ ] 4.3 在 `web/admin/src/features/menu/` 新建 `card-library-pane.tsx`：2 列卡片网格，平铺所有卡片（含未挂载），每张卡显示 name / 正文摘要 / 按钮数 / 引用状态
- [ ] 4.4 左 pane 顶部加 shadcn `Tabs`：「大纲」/「卡片库」；tab 上的数字徽章同步大纲节点数 / 卡片总数
- [ ] 4.5 暴露 `selectedNode: { kind: 'item' | 'card' | 'button'; id: string } | null` 与 `onSelect`；两视图共享同一选中态
- [ ] 4.6 「+ 新建卡片」入口：① 顶部 tab 区右侧按钮 ② 卡片库工具栏 ③ `open_card` 动作参数面板二次入口；三处共用同一个 Dialog

## 5. OutlinePane + CardLibraryPane 共用选中态

- [ ] 5.1 选中态在父组件 `MenuCardEditor` 中用 `selectedNode` 单一来源管理
- [ ] 5.2 大纲选中某菜单项 → 卡片库视图不影响；切到卡片库仍能看到其他卡片
- [ ] 5.3 卡片库选中某卡片 → 切回大纲时大纲里若该卡片被引用，定位到对应节点

## 6. ActionForm 动作分类（单一 Select + optgroup，6 种动作精准字段）

- [ ] 6.1 `action-form.tsx` 改为单一 shadcn `Select`，用 `<optgroup label="工作流能力">` / `<optgroup label="TG 平台能力">` 分组
- [ ] 6.2 当前项左侧 9px 圆点（绿/蓝）+ 右侧 `src-tag` 徽章同步能力来源
- [ ] 6.3 6 种动作类型参数面板按 `switch (action.type)` 早返回：
  - `open_workflow`：菜单项标签 + 工作流单选下拉
  - `open_card`：菜单项标签 + 卡片下拉（从卡片库全部卡片填充）+ 「+ 新建卡片」入口
  - `send_text`：菜单项标签 + 文本 Textarea
  - `send_media`：菜单项标签 + 媒体 URL 多行 Textarea
  - `open_url`：菜单项标签 + URL Input
  - `copy_text`：菜单项标签 + 文本 Textarea
- [ ] 6.4 抽 6 个 `<template>` 常量；切换动作类型时 ac-body 区域从 template 克隆替换
- [ ] 6.5 `open_workflow` 改为工作流单选下拉（不再用多选 Checkbox），写入路径仍产 `workflow_id: singleId` 兼容后端
- [ ] 6.6 移除「打开方式 direct/list」切换控件（list 模式已删除）
- [ ] 6.7 保留 `data-testid='action-form'` / `data-testid='card-picker'`，让 contract test 继续通过

## 7. 右侧编辑面板按 selectedNode.kind 渲染

- [ ] 7.1 选中菜单项（item）→ 渲染 `item-editor`：动作类型 Select + 6 种动作参数面板
- [ ] 7.2 选中卡片（card）→ 渲染 `card-editor`：name / text（卡片正文）/ media（媒体 URL 多行）/ 按钮列表 + 「+ 添加按钮」+ 「删除该卡片」
- [ ] 7.3 选中按钮（btn）→ 渲染 `btn-editor`：按钮标签 + 动作类型 Select + 6 种动作参数面板（与 item-editor 共用）
- [ ] 7.4 菜单项与按钮的 6 种动作参数面板通过共享 `<template>` 复用
- [ ] 7.5 无选中时给出引导文案与「+ 新建菜单项」入口

## 8. MenuCardEditor 整体改造

- [ ] 8.1 重构 `menu-card-editor.tsx` 顶层为「左 pane（双视图）+ 中 pane（编辑器按 selectedNode.kind 渲染）+ 右 pane（PhoneSimulation）」三栏
- [ ] 8.2 用 `selectedNode` 替代旧的 `selectedItemId` / `path: string[]` / `editingButton` 三态
- [ ] 8.3 保留 `validateMenuConfig` 校验、`putMenu` / `createCard` / `updateCard` / `deleteCard` 现有 mutation 行为
- [ ] 8.4 保留 unsaved changes 提示与 `dirtyCount` 计算；保存按钮 disabled 状态
- [ ] 8.5 删除「未挂载的卡片」分组（移到卡片库视图）

## 9. PhoneSimulation 联动

- [ ] 9.1 `phone-simulation.tsx` 接受 `selectedNode` prop
- [ ] 9.2 选中菜单项（item）→ 主键盘视图，按 label 匹配并高亮
- [ ] 9.3 选中卡片（card）→ 卡片视图（卡片正文 + 按钮列表）
- [ ] 9.4 选中按钮（btn）→ 卡片视图内对应按钮高亮
- [ ] 9.5 草稿未保存到预览时降级展示「当前编辑未保存到预览」
- [ ] 9.6 模拟键盘的菜单/卡片数据源与 `menuDraft ?? menuQuery.data` 保持一致，避免脏读

## 10. i18n 文案

```

Full source: docs/openspec/changes/channel-menu-editor-refactor/tasks.md

## docs/openspec/changes/channel-menu-editor-refactor/specs/channel-menu-editor/spec.md

- Source: docs/openspec/changes/channel-menu-editor-refactor/specs/channel-menu-editor/spec.md
- Lines: 1-122
- SHA256: 143fb956b66488804f818387f2291daf40f7b95c5020ab74b1ec05198dca69aa

[TRUNCATED]

```md
## Purpose

管理端消息平台菜单/卡片编辑器的 UI 行为规范：把「动作」按能力来源清晰分成「工作流能力」与「TG 平台能力」两类，把「菜单项 / 卡片 / 卡片按钮」三层用大纲视图统一导航，左 pane 提供「大纲」与「卡片库」双视图，让操作人员能扫读全部动作分布、精准定位当前编辑对象、卡片管理职责分明，并按动作类型只看到必要的参数；同时彻底清理 `open_workflow` 的 list 模式死代码（`workflow_ids[]` 数组、`mode` 字段、`open_case.list` 分支）。

## ADDED Requirements

### Requirement: 左侧双视图切换（大纲 / 卡片库）
编辑器左侧 pane MUST 在顶部提供 tab 切换两个视图：「大纲」与「卡片库」；默认进入「大纲」；tab 上的数字徽章分别显示大纲节点数 / 卡片总数。

#### Scenario: 默认进入大纲
- **WHEN** 操作人员进入消息平台的菜单编辑器
- **THEN** 左侧 pane 默认显示「大纲」tab 内容，tab 上的数字徽章显示当前菜单项数

#### Scenario: 切换到卡片库
- **WHEN** 操作人员点击「卡片库」tab
- **THEN** 左侧 pane 切换到平铺的卡片网格视图，数字徽章更新为当前卡片总数

#### Scenario: 顶部任意视图都能新建卡片
- **WHEN** 操作人员点击 tab 区右侧的「+ 新建卡片」按钮
- **THEN** 弹出新建卡片对话框，无论当前在大纲还是卡片库都可用

### Requirement: 大纲 = 纯树形结构（不混孤儿卡片）
大纲视图 MUST 只显示被引用的树形结构：菜单项下挂载其 `open_card` 引用的卡片，卡片下挂载其按钮；未挂载的卡片 MUST NOT 出现在大纲里；大纲里没有「未挂载的卡片」之类的并列分组。

#### Scenario: 选中菜单项时右侧展示菜单项编辑器
- **WHEN** 操作人员在大纲中点击某个菜单项
- **THEN** 右侧编辑面板只展示菜单项的 label 与动作选择；卡片与按钮的字段不出现

#### Scenario: 选中按钮时右侧展示按钮编辑器
- **WHEN** 操作人员在大纲中点击某个卡片下的按钮
- **THEN** 右侧编辑面板只展示该按钮的 label 与动作选择；菜单项与卡片的其他字段不出现

#### Scenario: 选中卡片时右侧展示卡片编辑器
- **WHEN** 操作人员在大纲中点击某个被引用的卡片
- **THEN** 右侧编辑面板展示卡片编辑（name / text / media / 按钮列表 + 添加按钮 + 删除卡片）

#### Scenario: 大纲项标注能力来源与层级
- **WHEN** 大纲显示任意节点
- **THEN** 菜单项与按钮节点旁出现能力来源 Badge（工作流 / TG 卡片 / TG 文字 / TG 媒体 / TG 链接 / TG 复制）；卡片节点旁出现「卡片」Badge 区分数据结构层级

### Requirement: 卡片库 = 平铺所有卡片
卡片库视图 MUST 2 列网格展示所有卡片（含未挂载的）；每张卡片 MUST 显示「name / 正文摘要 / 按钮数 / 引用状态（已引用 ● / 未挂载 ○）」；点选任一卡片进入卡片编辑视图。

#### Scenario: 查看所有卡片
- **WHEN** 操作人员切到「卡片库」tab
- **THEN** 看到所有卡片的网格；被引用的卡片标「已引用 ●」并显示被哪个菜单项引用；未挂载的卡片标「未挂载 ○」

#### Scenario: 从卡片库进入编辑
- **WHEN** 操作人员在卡片库点击某张卡片
- **THEN** 右侧编辑面板切到卡片编辑视图（name / text / media / 按钮列表）

#### Scenario: 卡片库新建/编辑/删除
- **WHEN** 操作人员在卡片库点「+ 新建卡片」或点击已有卡片
- **THEN** 弹出新建对话框或进入编辑视图；删除卡片时若该卡被菜单项引用则提示并降级引用

### Requirement: 动作按能力来源分组（单一 Select + optgroup）
动作类型选择器 MUST 用单一 `Select`，内部用 `<optgroup label="工作流能力">` / `<optgroup label="TG 平台能力">` 分组；当前项 MUST 同步显示左侧 9px 圆点（绿/蓝）+ 右侧 `src-tag` 徽章（工作流能力 / TG 平台能力）；编辑器 MUST NOT 使用常驻 Tab 切大类。

#### Scenario: 切换大类只展示该类下的动作
- **WHEN** 操作人员切换到「工作流能力」分组
- **THEN** 动作下拉只显示 `open_workflow`；切换到「TG 平台能力」分组时只显示 5 种 TG 动作

#### Scenario: 当前项同步能力来源信号
- **WHEN** 操作人员将动作切到 `open_workflow` 或任一 TG 动作
- **THEN** 选择器左侧圆点颜色与右侧 `src-tag` 徽章随动作类型同步变化（绿=工作流 / 蓝=TG）

#### Scenario: 大类名称不依赖英文枚举
- **WHEN** 编辑器渲染分组标题
- **THEN** 标题使用本地化文案（中文/英文），不暴露后端 `ActionType` 原始字符串

### Requirement: 6 种动作类型各自精准字段
编辑器 MUST 只渲染当前动作类型所需的参数字段，未涉及类型对应的字段 MUST 不可见；具体要求为：`open_workflow` MUST 渲染工作流**单选**下拉与必要提示；`open_card` MUST 渲染卡片选择（含「+ 新建卡片」快捷入口）+ 卡片下拉填充卡片库全部卡片；`send_text` / `copy_text` MUST 渲染文本框；`send_media` MUST 渲染媒体 URL 输入；`open_url` MUST 渲染 URL 输入；切换动作类型时参数面板 MUST 实时换内容（动画/状态可接受同步切换）。

#### Scenario: 切到 open_workflow 只显工作流单选
- **WHEN** 操作人员将动作切到 `open_workflow`
- **THEN** 编辑器只显示工作流单选下拉（每个菜单项/按钮 1 个 case），不出现多选 Checkbox、不出现文本、媒体、URL 等无关字段

#### Scenario: 切到 send_text 只显文本
- **WHEN** 操作人员将动作切到 `send_text`
- **THEN** 编辑器只显示文本输入，不出现工作流选择、卡片选择、URL 等字段

```

Full source: docs/openspec/changes/channel-menu-editor-refactor/specs/channel-menu-editor/spec.md
