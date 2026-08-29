# Brainstorm Summary

- Change: channel-menu-editor-refactor
- Date: 2026-08-29

## 确认的技术方案

三栏布局（左 pane 双视图 + 中 pane 编辑器 + 右 pane PhoneSimulation）；左 pane 顶部 tab 切换「大纲」/「卡片库」共享同一 `selectedNode`；大纲 = 纯树形（菜单项 → 被引用的卡片 → 按钮，不混孤儿卡片）；卡片库 = 2 列卡片网格显示 name/正文摘要/按钮数/引用状态；三个「+ 新建卡片」入口（顶部 tab 区 / 卡片库 / `open_card` 动作参数面板）；动作类型选择器 = 单一 shadcn `Select` + `optgroup` 分组（工作流 / TG 平台），不用常驻 Tab；6 种动作类型（`open_workflow` / `open_card` / `send_text` / `send_media` / `open_url` / `copy_text`）各自精准字段，按 `switch (action.type)` 早返回，切换时整个 `ac-body` 从 `<template>` 克隆替换；`open_workflow` 永远单选；`PhoneSimulation` 按 `selectedNode.kind` 切换主键盘 / 卡片视图 / 按钮高亮；保留 `validateMenuConfig` 与 `menu-editor.contract.test.ts` 关键 `data-testid`（`action-form` / `card-picker` / `card-list-panel`）。

## 关键取舍与风险

- **D5 BREAKING 风险**：后端 `Action.workflow_ids: string[]` 改 `workflow_id: string`；`Action.mode` 字段删除；既有数据通过一次性迁移脚本（多元素数组取首元素、`mode: 'list'` 数据降级为「无 mode 字段」）。
- **D1 窄屏挤压**：用 `min-w-0 + flex-1 + overflow-auto`；大纲/卡片库内部 `ScrollArea`。
- **D8 PhoneSimulation 数据源同步**：`menuDraft ?? menuQuery.data` 单一来源，避免脏读；草稿未保存时降级展示「当前编辑未保存到预览」。
- **D9 contract test 兼容**：保留 `action-form` / `card-picker` / `card-list-panel` 三个 `data-testid`；新增 `outline-pane` / `card-library-pane` / `editor-pane` / `action-type-select` 等用于新结构断言。
- **D7 菜单项/按钮共用参数面板**：通过 `<template>` 共享；后续若按钮需要定制字段（如「长按」），可扩展。

## 测试策略

- TDD 模式：先补 contract test 断言「左 pane 双视图 tab 切换」「动作 optgroup 分组」「工作流单选 + 删除 list mode」「大纲节点能力 Badge」
- `pnpm test src/features/menu` + `pnpm test src/features/quick-config` + `go test ./...` 全绿
- `pnpm typecheck` + `pnpm lint` 无新增告警
- 视觉/交互人工核对：双 pane 三栏布局、tab 切换、6 种动作参数面板切换、卡片库平铺、卡片新增/编辑/删除、模拟键盘按 `selectedNode.kind` 切换、未保存降级、unsaved 提示；对照 `pixoma-design-system-skill` 验收清单核对视觉令牌使用

## Spec Patch

无。v8 的所有需求已经在 `specs/channel-menu-editor/spec.md` 反映，无需回写 delta spec。
