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

- [ ] 10.1 zh 资源新增/调整：`menu.tabOutline` / `menu.tabLibrary` / `menu.outlineTitle` / `menu.outlineSearch` / `menu.outlineEmptyHint` / `menu.libraryTitle` / `menu.librarySearch` / `menu.libraryPinUsed` / `menu.libraryPinFree` / `menu.libraryBtnCount` / `menu.actionGroupWorkflow` / `menu.actionGroupTg` / `menu.badgeWorkflow` / `menu.badgeTgCard` / `menu.badgeTgText` / `menu.badgeTgMedia` / `menu.badgeTgUrl` / `menu.badgeTgCopy` / `menu.badgeCard` / `menu.editorEmptyHint` / `menu.workflowSingleHint` / `menu.cardName` / `menu.cardText` / `menu.cardMedia` / `menu.newCardTitle` / `menu.newCardName` / `menu.newCardText` / `menu.newCardConfirm` / `menu.deleteCard` / `menu.previewNotSaved` 等
- [ ] 10.2 en 资源同步新增/调整同名键
- [ ] 10.3 移除 `menu.openModeList` / `menu.openModeDirect` / `menu.openModeListAdvanced` 等 list 模式相关文案
- [ ] 10.4 在 `MenuCardEditor` / `OutlinePane` / `CardLibraryPane` / `ActionForm` / `PhoneSimulation` 引用新键，避免暴露英文 `ActionType`

## 11. Contract Test 兼容

- [ ] 11.1 跑 `pnpm test src/features/menu`，识别 `menu-editor.contract.test.ts` 中因结构调整误伤的断言
- [ ] 11.2 仅调整内部结构断言（不修改端到端 UI 契约断言）；如需重写则记录到本 tasks 后续条目
- [ ] 11.3 补 1-2 条针对「左 pane 双视图 tab 切换」与「动作按能力来源分组（optgroup）」与「工作流单选 + 删除 list 模式」与「大纲节点能力分类 Badge」的轻量断言

## 12. 验证

- [ ] 12.1 跑 `pnpm typecheck`，确保无新增类型错误
- [ ] 12.2 跑 `pnpm test src/features/menu` + `pnpm test src/features/quick-config`，contract test 与新增断言全绿
- [ ] 12.3 跑 `go test ./...`，所有 Go test 全绿
- [ ] 12.4 跑 `pnpm lint`，无新增告警
- [ ] 12.5 视觉/交互人工核对：双 pane 三栏布局、tab 切换、6 种动作参数面板切换、卡片库平铺、卡片新增/编辑/删除、模拟键盘按 selectedNode.kind 切换、未保存降级、unsaved 提示；对照 `pixoma-design-system-skill` 验收清单核对视觉令牌使用
