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
