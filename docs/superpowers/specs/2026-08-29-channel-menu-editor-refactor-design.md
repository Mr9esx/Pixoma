---
comet_change: channel-menu-editor-refactor
role: technical-design
canonical_spec: openspec
---

# 消息平台菜单/卡片编辑器重构 · 深度技术设计

## 1. Context

open 阶段已通过 proposal / design / spec / tasks 四个 artifacts 划定本次重构的范围与意图（动机见 `docs/openspec/changes/channel-menu-editor-refactor/proposal.md` Why；高层决策见同目录 `design.md`；需求见 `specs/channel-menu-editor/spec.md`）。本 Design Doc 是对它们的**深度技术细化**，不替代或重写，仅补充实现层细节、边界条件、数据流与测试策略。

事实链（已核对代码）：
- `web/admin/src/features/menu/menu-card-editor.tsx` 现有 472 行，三 section 堆叠 + path 栈，无大纲视图。
- `web/admin/src/features/menu/action-form.tsx` 旧 `open_workflow` 用多选 Checkbox，但 `quick-config/lib/menu-payload.ts` 的 `addWorkflowMenuEntry({ workflowId: number })` 每次只产 1 个 case；`internal/channel/capability/open_case.go` 的 `start` 走单 `case_id`；`list` 分支 / `case_ids[]` / `backCaseIDs` / `mode` 字段均无生产端数据。
- 既有契约：`validateMenuConfig` / `data-testid`（`action-form` / `card-picker` / `card-list-panel`）继续保留；`putMenu` / `createCard` / `updateCard` / `deleteCard` 调用方式不变。

## 2. Goals / Non-Goals

**Goals**
- 三栏布局：左 pane（顶部 tab：大纲 / 卡片库）+ 中 pane（编辑器按 `selectedNode.kind` 渲染）+ 右 pane（PhoneSimulation 按 `selectedNode.kind` 切换）。
- 单一 `Select` + `optgroup` 分组（不强制常驻 Tab）；6 种动作类型各自精准字段。
- 工作流永远单选；彻底删除 list 模式死代码。
- 卡片库平铺管理（含未挂载）；三个「+ 新建卡片」入口。
- PhoneSimulation 按 `selectedNode.kind` 切换主键盘 / 卡片视图 / 按钮高亮。
- 保留既有 contract test 关键 `data-testid` 与 `validateMenuConfig` 行为。

**Non-Goals**
- 不动消息平台管理页、不动 quick-config wizard 主结构、不动 channel-tg 适配器。
- 不引入新第三方 UI 库；只通过 shadcn CLI 添加 shadcn 组件。
- 不动 `channel-menu-config` / `channel-menu-interaction` 既有中立 spec。
- 不重写 i18n 体系；只在 `menu.*` 命名空间新增/调整键。
- 不在本次实现动画库（如 framer-motion）；如需过渡用原生 CSS transition。

## 3. 关键决策的深度细化

### D1 — 三栏布局的栅格与断点

栅格：
- 桌面（≥ 1280px）：`grid-template-columns: 300px 1fr 320px`（左固定 300 / 中弹性 / 右固定 320）。
- 平板（1024–1279px）：`grid-template-columns: 260px 1fr 280px`，左 pane 略窄。
- 窄屏（< 1024px）：本期不响应式；如运营反馈需要再加；此阶段不计入任务。

每个 pane 的内部布局：
- 左 pane：顶部 tab（44px） + 工具栏（48px） + 内容（`flex-1 min-h-0 overflow-auto`）。
- 中 pane：pane-head（46px） + 编辑器主体（`flex-1 min-h-0 overflow-auto`） + 编辑器底栏（48px）。
- 右 pane：pane-head + PhoneSimulation 居中（`flex-1 min-h-0 flex flex-col items-center justify-start`）。

### D2 — `selectedNode` 状态机

类型：

```ts
type SelectedNode =
  | { kind: 'item'; id: string }
  | { kind: 'card'; id: string }
  | { kind: 'btn'; id: string }   // btn id 形如 `cd1-b1`
  | null;
```

存储：放在 `MenuCardEditor` 顶层 `useState<SelectedNode>(...)`。

与 tab 的关系：
- tab 切换只切左 pane 视图（大纲 / 卡片库），**不重置** `selectedNode`。
- 卡片库点选某卡片 → 写 `selectedNode = { kind: 'card', id }` → 中 pane 切到 `card-editor`。
- 切回大纲 tab → `selectedNode` 不变；大纲里若该卡片被引用，DOM 高亮仍能定位。
- 选中节点被删除（卡片删除）→ `selectedNode` 重置为 `null`，中 pane 显示引导。

### D3 — 6 种动作参数面板的 `<template>` 抽象

实现位置：`web/admin/src/features/menu/action-templates.tsx`（新文件），导出 6 个 `<template>` 元素 + 工厂函数。

结构（伪代码）：

```tsx
export const ACTION_TEMPLATES = {
  open_workflow: <Template>...</Template>,
  open_card:     <Template>...</Template>,
  send_text:     <Template>...</Template>,
  send_media:    <Template>...</Template>,
  open_url:      <Template>...</Template>,
  copy_text:     <Template>...</Template>,
};
```

`ActionForm` 用法：

```tsx
function ActionForm({ action, ... }: Props) {
  const tpl = ACTION_TEMPLATES[action.type];
  return (
    <div data-testid="action-form" className="...">
      <ActionTypeSelect value={action.type} onChange={...} />
      {cloneElement(tpl, { /* 注入当前值与 onChange */ })}
    </div>
  );
}
```

`open_card` 模板里的卡片下拉是动态的（从卡片库全部卡片填充），通过 props 注入：

```tsx
<Template
  cardOptions={allCards.map(c => ({ value: c.id, label: c.name }))}
  selectedCardId={action.card_id}
  onPickCard={...}
  onNewCard={openNewCardModal}
/>
```

### D4 — `open_workflow` 写入路径与后端字段收紧的衔接

前端 `ActionForm` 的 `open_workflow` 写入：
- 用户在 shadcn `Select` 单选 1 个 case。
- onChange：调用 `onChange({ ...action, workflow_id: String(caseId) })`（注意：写单数字段 `workflow_id`，不是 `workflow_ids: [...]`）。
- 不再写 `mode` 字段。

后端 `Action` 类型（Go）：

```go
type Action struct {
    Type        string  `json:"type"`
    WorkflowID  string  `json:"workflow_id,omitempty"`
    CardID      string  `json:"card_id,omitempty"`
    Text        string  `json:"text,omitempty"`
    Media       []Media `json:"media,omitempty"`
    URL         string  `json:"url,omitempty"`
}
```

迁移脚本（一次性，发布前跑）：
- 读所有 menu.items[*].action / cards[*].buttons[*].action：
  - 若 `workflow_ids` 非空 → 取 `[0]` 写 `workflow_id`，删除 `workflow_ids`。
  - 若 `mode == "list"` → 删除 `mode`（默认 direct 行为，list 模式无对应运行时分支）。
  - 若 `mode == "direct"` → 删除 `mode`。
- 跑前先备份；跑后用 `validateMenuConfig` 校验全部菜单。

### D5 — 后端 `open_case.go` 清理

清理目标：
- 删除 `case "list":` 分支。
- 删除 `stringSlice` 辅助（或保留为内部工具，但移除 `case_ids` 路径）。
- 删除 `backCaseIDs` 辅助函数。
- `ParamsSchema()` 移除 `case_ids` 字段。
- 保留 `case "start"` / `case "preview"` / `case "text"` / `case "media"` / `case "skip"` / `case "confirm"` / `case "exit"` / `case "check"`。
- `start` 路径的 `case_id` 必填校验保持。

涉及 test：
- `internal/channel/capability/open_case_test.go` 删除 list 相关断言。
- `internal/caseadmin/case_delete_test.go` 改 `WorkflowIDs: []string{"10"}` → `WorkflowID: "10"`。
- `internal/httpapi/menucards/handler_test.go` 同上。

### D6 — 卡片库视图的状态与数据流

数据来源：复用现有 `listCards(channelId)` React Query。

本地状态：
- `cards: Card[]`（来自 React Query 的 `data ?? []`，含未挂载）。
- `selectedNode: SelectedNode` 来自父组件。
- `isModalOpen: boolean`（新建卡片弹框）。

渲染：
- 2 列 grid，每张卡 `<LibraryCard>`：name / 正文摘要（2 行 clamp）/ 按钮数 / 引用状态。
- 引用状态从菜单项派生：`Object.values(items).filter(it => it.action === 'open_card' && it.cardId === card.id).length > 0`。
- 点击卡片 → `setSelectedNode({ kind: 'card', id: card.id })` + 父组件切到 `card-editor`。

新建卡片入口（3 处共用 `openNewCardModal`）：
- 顶部 tab 区右侧：`<Button>+ 新建卡片</Button>`
- 卡片库工具栏：`<Button>+ 新建卡片</Button>`
- `open_card` 动作参数面板里的「+ 新建卡片」二次入口：`<Button onClick={openNewCardModal}>+ 新建卡片</Button>`

Dialog 字段：name（必填） + text（可选）。提交后调用 `createCard(channelId, { id: uid('cd'), name, text, media: [], buttons: [] })`，刷新 React Query，自动选中并切到 `card-editor`。

### D7 — 菜单项与按钮的参数面板复用

`ActionForm` 作为公共组件，被 `item-editor` 与 `btn-editor` 共同渲染。

`item-editor` 结构：
```tsx
<item-editor>
  <ActionTypeSelect />
  <ActionForm action={...} onChange={...} />
</item-editor>
```

`btn-editor` 结构：
```tsx
<btn-editor>
  <Input label="按钮标签" value={...} onChange={...} />
  <ActionTypeSelect />
  <ActionForm action={...} onChange={...} />
</btn-editor>
```

`<ActionForm>` 接受 `data-testid="action-form"`，contract test 不变。

### D8 — PhoneSimulation 联动

`PhoneSimulation` 接受 `selectedNode: SelectedNode | null` 与 `draft: Menu | null`（草稿优先于 server data）。

视图切换：

| selectedNode | 视图 |
|---|---|
| `{ kind: 'item' }` | 主键盘，按 `selectedItem.label` 在键盘列表中匹配并高亮 |
| `{ kind: 'card' }` | 卡片视图：卡片正文 + 按钮列表（无按钮高亮） |
| `{ kind: 'btn' }` | 卡片视图 + 该按钮高亮（需先定位到该按钮所属卡片） |
| `null` | 引导态（"从大纲选择一项开始编辑"） |

数据源：`const displayMenu = menuDraft ?? menuQuery.data ?? emptyMenu(channelId);` 单一来源。

降级逻辑：若 `selectedNode` 引用的节点在 `displayMenu` 中不存在（例如新菜单项还没落库），展示「当前编辑未保存到预览」。

### D9 — `data-testid` 与 contract test 兼容

保留：
- `action-form`（`ActionForm` 根）
- `card-picker`（`open_card` 模板里卡片下拉）
- `card-list-panel`（保留作为兼容锚点，渲染在卡片库但保持 `data-testid`）

新增：
- `outline-pane`（大纲视图根）
- `card-library-pane`（卡片库视图根）
- `editor-pane`（中 pane 根）
- `action-type-select`（动作类型 Select 根）
- `phone-simulation`（右 pane 根；如旧有则复用）

`menu-editor.contract.test.ts` 端到端断言（关键 UI 契约）继续通过：
- 选择动作类型后参数面板正确切换
- `open_workflow` 强制单选
- 卡片被菜单项引用时大纲显示该卡片
- 实时预览与选中态联动

## 4. 数据流图（ASCII）

### 4.1 选中菜单项 → 编辑器 + 预览

```
[大纲: 点击 "开始生成（换脸）"]
  └─ onSelect({ kind: 'item', id: 'mi1' })
      └─ setSelectedNode({ kind: 'item', id: 'mi1' })
          ├─ [左 pane · 大纲]  高亮 mi1
          ├─ [中 pane · item-editor]
          │   └─ 渲染 ActionTypeSelect + ActionForm
          │       └─ 切到 open_workflow → cloneElement(TPL_open_workflow)
          └─ [右 pane · PhoneSimulation]
              └─ 渲染主键盘，按 label 匹配高亮
```

### 4.2 切到 `open_card` 动作 → 卡片下拉填充

```
[ActionTypeSelect: open_workflow → open_card]
  └─ onChange({ type: 'open_card' })
      └─ setAction(newAction)
          └─ [ActionForm 重渲染]
              └─ cloneElement(TPL_open_card, {
                   cardOptions: allCards.map(c => ({ value: c.id, label: c.name })),
                   selectedCardId: action.card_id,
                   onPickCard: ...,
                   onNewCard: openNewCardModal,
                 })
```

### 4.3 新建卡片

```
[任意入口触发 openNewCardModal]
  └─ setIsModalOpen(true)
      └─ <Dialog>
          └─ 提交: createCard(channelId, newCard)
              └─ queryClient.invalidateQueries(['channels', 'menu', channelId, 'cards'])
                  └─ React Query 重新拉取
                      └─ setSelectedNode({ kind: 'card', id: newCard.id })
                          └─ [中 pane · card-editor] 显示新卡片
                          └─ [左 pane · 卡片库] 新卡出现在网格
```

### 4.4 删除卡片（被引用时降级）

```
[卡片库或大纲: 点击卡片"删除"]
  └─ if (引用此卡的菜单项 > 0) {
       confirm: "该卡片被「文生图」菜单项引用，删除将把该菜单项动作降级为 send_text。继续？"
     }
  └─ 确认后:
      1. deleteCard(channelId, cardId)
      2. 引用此卡的菜单项: items[i].action = 'send_text'（或保留其 label）
      3. 失效 React Query
      4. setSelectedNode(null) → 中 pane 引导
```

## 5. 边界条件

- **空数据**：`menu.items = []` → 大纲显示「还没有菜单项」+ 「+ 新建菜单项」入口；卡片库显示所有卡片。
- **草稿为空、server 还在 loading**：用 `menuQuery.isLoading` 渲染 `<LoadingSkeleton>`；`phone-simulation` 同步显示 skeleton。
- **`open_card` 引用了不存在的 cardId**：`validateMenuConfig` 标记 `card_missing` 错误；`card-picker` 视觉标注但保留历史选择（用户可改正或删除卡片）。
- **草稿与 server data 冲突**：以草稿优先（`menuDraft ?? menuQuery.data`）；保存按钮触发 `putMenu(menuDraft)`。
- **删除最后一张卡片**：大纲 `open_card` 菜单项保留但 `card_id = undefined`；触发 `card_missing` 校验错误；提示用户修正。
- **新建卡片未命名**：Dialog 校验 `name.trim() !== ''`；空提交弹 toast。
- **切换 tab 丢失选中**：不会丢失（共享 `selectedNode`）。
- **窄屏（< 1024px）**：本期不响应式；如需后续扩展。

## 6. 错误处理

| 场景 | 处理 |
|---|---|
| `getMenu` / `listCards` 失败 | `<ErrorBanner>` + 重试按钮 |
| `putMenu` / `createCard` / `updateCard` / `deleteCard` 失败 | toast 报错 + 草稿保留 |
| 校验失败 | 错误展示在字段下方 + 编辑器底栏汇总 `saveErrors` |
| API 返回字段与新契约不匹配 | 兜底：旧 `workflow_ids` 数组仍按 `[0]` 处理；`mode` 字段忽略 |
| 迁移脚本异常 | 脚本分批跑；每批 commit；失败回滚该批 |

## 7. 测试策略

### 7.1 Contract test（必须有）

文件：`web/admin/src/features/menu/menu-editor.contract.test.ts`（沿用）

新增断言（建议）：
- 「左 pane 双视图 tab 切换」：tab 计数 + 内容切换
- 「动作按能力来源分组（optgroup）」：6 种动作分别在正确分组
- 「工作流单选 + 删除 list mode」：open_workflow 视图无 Checkbox、无「打开方式」控件
- 「大纲节点能力 Badge」：每个大纲项有正确 Badge 文案

调整（不删）：
- 内部结构断言（指向某个 section 内部的具体 class）允许改写或删除（视为内部重构）
- 端到端断言（用户在编辑器内能完成配置、预览更新、保存）必须保留并通过

### 7.2 单元测试

- `lib/menu-flow.ts::validateAction` 单测：删除 mode 校验、增加 `workflow_id` 非空校验
- `lib/menu-payload.ts::addWorkflowMenuEntry` 单测：入参去 mode；写入时产 `workflow_id` 单数字段
- `ACTION_TEMPLATES` 各 template 渲染快照（可选）

### 7.3 集成测试

- `quick-config/lib/menu-payload.test.ts`：验证与新 `Action` 类型兼容
- 后端 Go test：`internal/channel/capability/open_case_test.go` 删 list 断言；`internal/caseadmin/case_delete_test.go` 改单数字段；`internal/httpapi/menucards/handler_test.go` 同步

### 7.4 视觉/交互

- 跑 demo v8 路径：
  - 切 tab 大纲 ↔ 卡片库
  - 选 6 种动作
  - 选菜单项 / 卡片 / 按钮
  - 新建卡片（3 个入口）
  - 删除卡片（被引用时降级）
  - PhoneSimulation 按 selectedNode.kind 切换

## 8. 迁移计划

阶段 1（数据 + 后端）：
1. 写一次性迁移脚本，备份 → 改 `workflow_ids[]` → `workflow_id`，删 `mode`。
2. 后端 `Action` 类型收紧；`open_case.go` 删 list 分支；同步 test。

阶段 2（前端 API + 库）：
3. `web/admin/src/lib/api/channel-menu.ts` 的 `Action` 类型同步。
4. `lib/menu-flow.ts::validateAction` 同步。
5. `quick-config/lib/menu-payload.ts` 同步；`step3-channels.tsx` 删 `WorkflowMenuMode`。

阶段 3（UI 重构）：
6. 拆 `MenuCardEditor` 为三栏 + 双视图。
7. 实现 `outline-pane.tsx` / `card-library-pane.tsx`。
8. 实现 `action-templates.tsx` + 6 个 `<template>`。
9. 改造 `action-form.tsx` 接入 optgroup + 模板。
10. 改造 `phone-simulation.tsx` 接受 `selectedNode`。
11. i18n 增改；shadcn 组件按需添加。

阶段 4（验证）：
12. 跑 `pnpm test` + `go test` + `pnpm typecheck` + `pnpm lint`。
13. 视觉/交互人工核对 6 种动作 + 卡片库 + 双视图 + 模拟键盘。

回滚：UI 改动集中在 `features/menu/*`；后端收紧可通过迁移脚本反向回填（`workflow_id` → `workflow_ids: [singleId]`）。如遇阻塞，整体 revert 不影响 quick-config 与消息平台管理页。

## 9. 风险评估

- **R1 [D5 后端 BREAKING] 高**：旧 `workflow_ids[]` 数据若漏迁移会导致菜单动作不可用。缓解：迁移脚本必跑 + 备份 + 跑前 dry-run。
- **R2 [D9 contract test 误伤] 中**：内部结构断言需逐条对齐。缓解：先把 test 列全，再实现；分阶段 commit。
- **R3 [D6 卡片库与大纲状态同步] 中**：两个视图都修改 cards，可能竞争。缓解：单一 React Query 数据源，本地只持有 `selectedNode`。
- **R4 [D8 PhoneSimulation 数据漂移] 低**：`menuDraft ?? menuQuery.data` 单一来源。
- **R5 [i18n 翻译遗漏] 低**：新文案先 zh 落地，en 跟齐；PR review 重点过。
- **R6 [shadcn 新组件引入] 低**：按需 `pnpm dlx shadcn@latest add`，遵循 `pixoma-design-system` 视觉令牌。

## 10. Open Questions（保留自 open 阶段 design.md）

- 卡片被删除时若被菜单项引用：本期实现「提示并降级」（更轻量）。后续可评估 strict 模式（拒绝删除，先解除引用）。
- `Card.buttons[]` 上限：本期不设限；`channel-menu-interaction` 已规定「主键盘直达入口数量 MUST 不超过 6 个」，但卡片内按钮上限未规定。后续可作为增强。
- 列表分组（`mode: 'list'`）彻底删除后，admin API 历史日志中 `mode: 'list'` 字段如何展示：默认忽略；UI 不展示。
