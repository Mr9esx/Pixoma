# 消息平台菜单/卡片编辑器重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans（本项目默认 build_mode）配合 superpowers:test-driven-development 逐任务实施。步骤用 `- [ ]` 勾选跟踪。

**Goal:** 把 `web/admin/src/features/menu` 编辑器重做为三栏布局（左 pane 双视图：大纲 / 卡片库 + 中 pane 编辑器按 selectedNode.kind 渲染 + 右 pane PhoneSimulation 联动），动作类型用单一 Select + optgroup 分组，6 种动作各自精准字段，工作流永远单选；并彻底清理 list 模式死代码（后端 `Action.workflow_ids[]` 改 `workflow_id` 单数、删 `mode` 字段、删 `open_case.list` 分支）。

**Architecture:** 父组件 `MenuCardEditor` 拆为三栏；左 pane 顶部 shadcn `Tabs` 切 `OutlinePane`（树形结构）/ `CardLibraryPane`（2 列卡片网格）；中 pane 按 `selectedNode.kind` 渲染 `ItemEditor` / `CardEditor` / `ButtonEditor`，三视图共享 `selectedNode` 单一来源；6 种动作参数面板抽到 `action-templates.tsx` 的 `<template>` 常量。PhoneSimulation 按 `selectedNode.kind` 切换。后端 `internal/channel/capability/open_case.go` 删 list 分支 / `case_ids[]` / `backCaseIDs`；`Action` 类型收紧；多个 internal test 改单数字段；一次性数据迁移脚本（多元素数组取首元素、`mode: 'list'` 数据降级为「无 mode」）。

**Tech Stack:** React/TS · TanStack Query · shadcn/ui (Tabs / Badge / ScrollArea / Sheet / Dialog) · vitest（contract test 用文件断言）· Go（后端收紧 + 一次性迁移）· Pixoma 设计系统（语义令牌色 / `flex + gap` / 内置 variants）。

---
change: channel-menu-editor-refactor
design-doc: docs/superpowers/specs/2026-08-29-channel-menu-editor-refactor-design.md
base-ref: 7632b5c0ddd0531a52b508da6531755169935a0a
---

## Global Constraints

- 全部代码在 `web/admin/src/`（前端）与 `internal/...`（后端），TypeScript strict 与 Go 1.22+。
- 前端优先复用 `web/admin/src/components/ui/` 已安装的 shadcn 组件，缺什么就 `pnpm dlx shadcn@latest add`。
- 文案必须成对写入 `web/admin/src/lib/i18n/locales/zh.json` 与 `en.json`，命名空间 `menu.*`。
- 设计系统：使用语义令牌色（`--workflow` / `--tg` / `--muted` / `--line-soft` 等），不写裸 hex；遵循 `pixoma-design-system` 视觉令牌与验收清单。
- 既有契约保留：`data-testid`（`action-form` / `card-picker` / `card-list-panel`）、`validateMenuConfig` 行为；`putMenu` / `createCard` / `updateCard` / `deleteCard` 调用方式不变。
- 不提交 git（本项目 guard 管理）。
- 工作区隔离：`current`（当前分支 feat-init 直接工作）；TDD 模式：`tdd`；代码审查：`standard`。

---

### Task 1: 一次性数据迁移脚本（后端）

**Files:**
- Create: `scripts/migrate-menu-action/main.go`（一次性脚本，可独立运行）
- Test: `scripts/migrate-menu-action/main_test.go`（dry-run 断言）

**Interfaces:**
- Consumes: 数据库中既有 menu.items[*].action / cards[*].buttons[*].action JSON。
- Produces:
  - 多元素 `workflow_ids: ["X", ...]` → `workflow_id: "X"`（取首元素），删除 `workflow_ids`。
  - `mode: "list"` 或 `mode: "direct"` → 删除整个 `mode` 字段。
  - 迁移前自动备份到 `backups/menu-action-<timestamp>.json`。

- [ ] **Step 1.1: 写失败测试**
`main_test.go` 写：
```go
func TestMigrateAction(t *testing.T) {
    cases := []struct{
        name string
        in, want map[string]any
    }{
        {"workflow_ids 多元素取首", map[string]any{"type":"open_workflow","workflow_ids":[]any{"10","20"}}, map[string]any{"type":"open_workflow","workflow_id":"10"}},
        {"workflow_ids 单元素取首", map[string]any{"type":"open_workflow","workflow_ids":[]any{"10"}}, map[string]any{"type":"open_workflow","workflow_id":"10"}},
        {"mode list 字段删除", map[string]any{"type":"open_workflow","workflow_id":"10","mode":"list"}, map[string]any{"type":"open_workflow","workflow_id":"10"}},
        {"mode direct 字段删除", map[string]any{"type":"open_workflow","workflow_id":"10","mode":"direct"}, map[string]any{"type":"open_workflow","workflow_id":"10"}},
        {"无变化透传", map[string]any{"type":"send_text","text":"hi"}, map[string]any{"type":"send_text","text":"hi"}},
    }
    for _, c := range cases { ... }
}
```
- [ ] **Step 1.2: 验证测试失败**（脚本还不存在）—— `go test ./scripts/migrate-menu-action/...` FAIL
- [ ] **Step 1.3: 实现脚本**：`migrateAction(map) → (map, bool, error)`；`main()` 读 DB → 备份 → 遍历迁移 → 写回
- [ ] **Step 1.4: 验证测试通过**
- [ ] **Step 1.5: 跑 dry-run 模式** 在 staging 环境跑一次，确认无报错

---

### Task 2: 后端 `Action` 类型收紧

**Files:**
- Modify: `internal/mcdomain/action.go`（或等价类型定义文件）
- Test: `internal/mcdomain/action_test.go`（新增或修改）

**Interfaces:**
- Produces:
  ```go
  type Action struct {
      Type       string  `json:"type"`
      WorkflowID string  `json:"workflow_id,omitempty"`
      CardID     string  `json:"card_id,omitempty"`
      Text       string  `json:"text,omitempty"`
      Media      []Media `json:"media,omitempty"`
      URL        string  `json:"url,omitempty"`
  }
  ```
  `WorkflowIDs []string` 字段删除；`Mode string` 字段删除。

- [ ] **Step 2.1: 写失败测试**——`action_test.go` 增加 `TestActionMarshal` 断言新 schema：序列化含 `workflow_id` 不含 `workflow_ids` / `mode`。
- [ ] **Step 2.2: 验证测试失败**
- [ ] **Step 2.3: 修改类型定义**——删 `WorkflowIDs` / `Mode`；加 `WorkflowID`。
- [ ] **Step 2.4: 验证测试通过**
- [ ] **Step 2.5: 跑 `go test ./internal/mcdomain/...` 全绿**

---

### Task 3: 后端 `open_case.go` 清理 list 分支

**Files:**
- Modify: `internal/channel/capability/open_case.go`
- Test: Modify `internal/channel/capability/open_case_test.go`

**Interfaces:**
- 删 `case "list":` 分支；删 `case_ids[]` 参数；删 `backCaseIDs` 辅助；`ParamsSchema()` 移除 `case_ids`。

- [ ] **Step 3.1: 写失败测试**—— `open_case_test.go` 增加 `TestOpenCaseListBranchRemoved`：调用 `Invoke(ctx, ..., params: {"step":"list"})` 期望返回错误 `open_case: unknown step "list"`。
- [ ] **Step 3.2: 验证测试失败**（旧代码仍接受 list）
- [ ] **Step 3.3: 删除 list 分支 + 辅助 + ParamsSchema 字段**
- [ ] **Step 3.4: 验证测试通过**
- [ ] **Step 3.5: 跑 `go test ./internal/channel/...` 全绿**

---

### Task 4: 后端 test 同步（多文件改单数字段）

**Files:**
- Modify: `internal/caseadmin/case_delete_test.go`
- Modify: `internal/httpapi/menucards/handler_test.go`
- (其它 grep `WorkflowIDs` 出现处)

- [ ] **Step 4.1: 跑 `go test ./...` 看所有 FAIL**（数据模型收紧后 test 失败）
- [ ] **Step 4.2: 用 `rg "WorkflowIDs: \[\]string\{" internal/` 找出所有点，逐一改 `WorkflowID: "..."`**
- [ ] **Step 4.3: 跑 `go test ./...` 全绿**

---

### Task 5: 前端 `Action` 类型同步

**Files:**
- Modify: `web/admin/src/lib/api/channel-menu.ts`
- Test: `web/admin/src/lib/api/channel-menu.test.ts`（断言新 schema）

- [ ] **Step 5.1: 写失败测试**——`channel-menu.test.ts` 断言 `Action` 类型序列化：`{type: 'open_workflow', workflow_id: '10'}` 序列化后字段正确。
- [ ] **Step 5.2: 验证测试失败**
- [ ] **Step 5.3: 修改类型**——`workflow_ids?: string[]` → `workflow_id?: string`；删 `mode?` 字段。
- [ ] **Step 5.4: 验证测试通过**
- [ ] **Step 5.5: 跑 `pnpm test src/lib/api/` 全绿**

---

### Task 6: `validateAction` 同步

**Files:**
- Modify: `web/admin/src/features/menu/lib/menu-flow.ts`
- Modify: `web/admin/src/features/menu/lib/menu-flow.test.ts`

- [ ] **Step 6.1: 写失败测试**——`menu-flow.test.ts` 增加 `TestValidateActionNoMode`：用 `{type:'open_workflow', workflow_id:'10', mode:'list'}` 不应报错。
- [ ] **Step 6.2: 验证测试失败**
- [ ] **Step 6.3: 修改校验**——`workflow_ids.length === 0` 改 `!workflow_id`；删 mode 相关校验。
- [ ] **Step 6.4: 验证测试通过**

---

### Task 7: `addWorkflowMenuEntry` 同步

**Files:**
- Modify: `web/admin/src/features/quick-config/lib/menu-payload.ts`
- Modify: `web/admin/src/features/quick-config/lib/menu-payload.test.ts`
- Modify: `web/admin/src/features/quick-config/step3-channels.tsx`（删 `WorkflowMenuMode`）

- [ ] **Step 7.1: 写失败测试**——`menu-payload.test.ts` 改入参 `{label, workflowId}` 不再含 `mode`；写入产 `workflow_id: '12'` 不含 `mode`。
- [ ] **Step 7.2: 验证测试失败**
- [ ] **Step 7.3: 修改**——`addWorkflowMenuEntry` 入参删 `mode`；写入产 `workflow_id` 单数字段。
- [ ] **Step 7.4: 验证测试通过**
- [ ] **Step 7.5: `step3-channels.tsx` 删 `WorkflowMenuMode` 类型 + 删「打开方式 direct/list」Select**

---

### Task 8: shadcn 组件盘点与补齐

**Files:**
- Modify: `web/admin/src/components/ui/`（按需补齐）

- [ ] **Step 8.1: 盘点 `web/admin/src/components/ui/` 已安装组件**——确认是否有 Tabs / Badge / ScrollArea / Sheet / Dialog
- [ ] **Step 8.2: 用 `pnpm dlx shadcn@latest add <missing>` 补齐缺失组件**（按需）
- [ ] **Step 8.3: 验证组件被正确引入**——`pnpm typecheck` 无 TS 错误

---

### Task 9: `action-templates.tsx`（6 种动作参数面板抽象）

**Files:**
- Create: `web/admin/src/features/menu/action-templates.tsx`
- Test: `web/admin/src/features/menu/action-templates.test.tsx`（快照或渲染断言）

- [ ] **Step 9.1: 写失败测试**——6 个 template 各渲染一次，断言关键字段存在（如 `open_workflow` 含工作流 Select、`open_card` 含卡片 Select 等）。
- [ ] **Step 9.2: 验证测试失败**
- [ ] **Step 9.3: 实现 6 个 template**：每个是 React 元素，接收 props（value / onChange / options）
- [ ] **Step 9.4: 验证测试通过**

---

### Task 10: `OutlinePane` 大纲视图

**Files:**
- Create: `web/admin/src/features/menu/outline-pane.tsx`
- Test: `web/admin/src/features/menu/outline-pane.test.tsx`

- [ ] **Step 10.1: 写失败测试**——传入 mock items + cards，断言渲染三层结构（菜单项 / 被引用的卡片 / 按钮），且未挂载卡片不出现。
- [ ] **Step 10.2: 验证测试失败**
- [ ] **Step 10.3: 实现 `OutlinePane`**：消费 `menu.items` / `cards`，递归渲染三层 + Badge + onSelect
- [ ] **Step 10.4: 验证测试通过**

---

### Task 11: `CardLibraryPane` 卡片库视图

**Files:**
- Create: `web/admin/src/features/menu/card-library-pane.tsx`
- Test: `web/admin/src/features/menu/card-library-pane.test.tsx`

- [ ] **Step 11.1: 写失败测试**——传入 mock cards + items，断言渲染 2 列卡片网格；未挂载卡片显示「未挂载 ○」；被引用的显示「已引用 ●」+ 引用方菜单项名。
- [ ] **Step 11.2: 验证测试失败**
- [ ] **Step 11.3: 实现 `CardLibraryPane`**
- [ ] **Step 11.4: 验证测试通过**

---

### Task 12: 新建卡片 Dialog（3 个入口共用）

**Files:**
- Create: `web/admin/src/features/menu/new-card-dialog.tsx`
- Test: `web/admin/src/features/menu/new-card-dialog.test.tsx`

- [ ] **Step 12.1: 写失败测试**——Dialog 打开时显示 name + text 输入；空 name 提交报错；有效 name 提交回调 `onConfirm({name, text})`。
- [ ] **Step 12.2: 验证测试失败**
- [ ] **Step 12.3: 实现 Dialog**（基于 shadcn Dialog）
- [ ] **Step 12.4: 验证测试通过**

---

### Task 13: `ActionForm` 改造（单一 Select + optgroup + 6 种动作参数面板）

**Files:**
- Modify: `web/admin/src/features/menu/action-form.tsx`
- Test: Modify `web/admin/src/features/menu/menu-editor.contract.test.ts`

- [ ] **Step 13.1: 写失败 contract test**——断言：
  - 单一 Select 含 `<optgroup label="工作流能力">` 与 `<optgroup label="TG 平台能力">`
  - 切到 `open_workflow` 时渲染工作流单选下拉
  - 切到 `open_card` 时渲染卡片下拉 + 「+ 新建卡片」按钮
  - 切到 `send_text` / `send_media` / `open_url` / `copy_text` 时各自精准字段
  - 没有任何「打开方式」切换控件
- [ ] **Step 13.2: 验证 contract test 失败**
- [ ] **Step 13.3: 改造 ActionForm**：单一 Select + optgroup；`switch (action.type)` 早返回；`open_workflow` 改单选；`open_card` 用 `ACTION_TEMPLATES.open_card` 注入 cardOptions
- [ ] **Step 13.4: 验证 contract test 通过**
- [ ] **Step 13.5: 保留 `data-testid='action-form'` / `'card-picker'`**

---

### Task 14: `CardEditor` 卡片编辑视图

**Files:**
- Create: `web/admin/src/features/menu/card-editor.tsx`
- Test: `web/admin/src/features/menu/card-editor.test.tsx`

- [ ] **Step 14.1: 写失败测试**——传入 card prop，断言渲染 name / text / media 输入 + 按钮列表 + 「+ 添加按钮」+ 「删除该卡片」。
- [ ] **Step 14.2: 验证测试失败**
- [ ] **Step 14.3: 实现 CardEditor**
- [ ] **Step 14.4: 验证测试通过**

---

### Task 15: `ButtonEditor` 按钮编辑视图

**Files:**
- Create: `web/admin/src/features/menu/button-editor.tsx`
- Test: `web/admin/src/features/menu/button-editor.test.tsx`

- [ ] **Step 15.1: 写失败测试**——渲染按钮标签输入 + 动作类型 Select + 参数面板（共用 ActionForm）
- [ ] **Step 15.2: 验证测试失败**
- [ ] **Step 15.3: 实现 ButtonEditor**
- [ ] **Step 15.4: 验证测试通过**

---

### Task 16: `PhoneSimulation` 联动 selectedNode

**Files:**
- Modify: `web/admin/src/features/menu/phone-simulation.tsx`
- Test: Modify `web/admin/src/features/menu/menu-editor.contract.test.ts`

- [ ] **Step 16.1: 写失败 contract test**——断言：
  - 选中菜单项 → 主键盘视图 + label 高亮
  - 选中卡片 → 卡片视图（卡片正文 + 按钮列表）
  - 选中按钮 → 卡片视图内按钮高亮
  - 草稿未保存时降级展示「当前编辑未保存到预览」
- [ ] **Step 16.2: 验证 contract test 失败**
- [ ] **Step 16.3: 改造 PhoneSimulation**——接受 `selectedNode` prop + `menuDraft ?? menuQuery.data` 单一数据源
- [ ] **Step 16.4: 验证 contract test 通过**

---

### Task 17: `MenuCardEditor` 顶层改造（三栏 + 双视图 + selectedNode 状态机）

**Files:**
- Modify: `web/admin/src/features/menu/menu-card-editor.tsx`

- [ ] **Step 17.1: 保留 contract test 既有断言**——跑 `pnpm test src/features/menu/menu-editor.contract.test.ts`，识别失败点
- [ ] **Step 17.2: 改造 MenuCardEditor**：
  - 三栏布局（grid 300/1fr/320）
  - 顶层 `selectedNode: SelectedNode` 状态
  - 左 pane：顶部 shadcn `Tabs`「大纲」/「卡片库」+ 三个「+ 新建卡片」入口
  - 中 pane：按 `selectedNode.kind` 渲染 ItemEditor / CardEditor / ButtonEditor / 引导态
  - 右 pane：PhoneSimulation 接受 selectedNode
  - 保留 `validateMenuConfig` / `putMenu` / `createCard` / `updateCard` / `deleteCard` 调用
  - 保留 `dirtyCount` 与 unsaved 提示
- [ ] **Step 17.3: 验证 contract test 通过**

---

### Task 18: i18n 文案增改

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`

- [ ] **Step 18.1: 列新增/调整键清单**（按 Design Doc 第 10 节）
  - 新增：tabOutline / tabLibrary / outlineTitle / outlineSearch / outlineEmptyHint / libraryTitle / librarySearch / libraryPinUsed / libraryPinFree / libraryBtnCount / actionGroupWorkflow / actionGroupTg / badgeWorkflow / badgeTgCard / badgeTgText / badgeTgMedia / badgeTgUrl / badgeTgCopy / badgeCard / editorEmptyHint / workflowSingleHint / cardName / cardText / cardMedia / newCardTitle / newCardName / newCardText / newCardConfirm / deleteCard / previewNotSaved / outlineCardPinned / outlineCardFree
  - 删除：openModeList / openModeDirect / openModeListAdvanced
- [ ] **Step 18.2: zh 资源先填**——按 demo v8 文案落
- [ ] **Step 18.3: en 资源同步**
- [ ] **Step 18.4: 跑 `pnpm test` 检查 i18n 类型**（如有 i18n 类型断言）

---

### Task 19: Contract test 调整

**Files:**
- Modify: `web/admin/src/features/menu/menu-editor.contract.test.ts`

- [ ] **Step 19.1: 跑 contract test**，列出所有失败项
- [ ] **Step 19.2: 区分端到端断言（必须保留）与内部结构断言（可调整）**
- [ ] **Step 19.3: 调整内部结构断言**——把指向旧 section 内部 class 的断言改写为指向新结构（`data-testid='outline-pane'` / `'card-library-pane'` / `'editor-pane'` / `'action-type-select'`）
- [ ] **Step 19.4: 端到端断言全绿**

---

### Task 20: 验证

- [ ] **Step 20.1: `pnpm typecheck`** —— 无 TS 错误
- [ ] **Step 20.2: `pnpm test src/features/menu` + `pnpm test src/features/quick-config` + `pnpm test src/lib/api`** —— 全绿
- [ ] **Step 20.3: `go test ./...`** —— 全绿（含迁移脚本 dry-run）
- [ ] **Step 20.4: `pnpm lint`** —— 无新增告警
- [ ] **Step 20.5: 视觉/交互人工核对**：
  - 切 tab 大纲 ↔ 卡片库
  - 选 6 种动作
  - 选菜单项 / 卡片 / 按钮
  - 新建卡片（3 个入口）
  - 删除卡片（被引用时降级）
  - PhoneSimulation 按 selectedNode.kind 切换
  - 对照 `pixoma-design-system-skill` 验收清单核对视觉令牌
