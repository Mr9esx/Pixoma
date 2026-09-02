---
change: puck-channel-menu-editor
design-doc: docs/superpowers/specs/2026-09-02-puck-channel-menu-editor-design.md
base-ref: 32070fd5e458273744b442b50fc1aca59f4851a7
---

# Puck 渠道菜单编辑器 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用嵌套 `MenuTree` 替换 Menu+Card 分表，Bot 编译后执行；管理端 Puck 整页手机画布编辑，详情只读手机。

**Architecture:** `channel_main_menus.DocJSON` 存整棵树；`open_card` 内嵌卡片。HTTP 只 GET/PUT `/menu`，`/cards` 返回 410。Admin `puckData ↔ MenuTree`，路由 `/channels/$id/menu`。

**Tech Stack:** Go + GORM，Vite admin，`@measured/puck`，shadcn，vitest，`go test`。

## Global Constraints

- 产物与提交说明用 zh-CN 可读标题；界面文案只留字段/按钮/校验。
- 根键盘按钮 1–6，列数 1–6，open_card 链深度 ≤8，树内 Button.ID 唯一。
- 不兼容旧 Menu JSON / 不做卡片库 / 不 vendor shadcn-builder。
- TDD：每个任务先失败测试再实现。
- 架构文档与表/API 同步：`docs/architecture/data-model.md` 等。

## File map

- `internal/menucard/domain/`：`MenuTree`、`ValidateTree`、`DefaultMenuTree`、`Compile`、placements 遍历
- `internal/menucard/infrastructure/persistence/`：只读写树；去掉卡片 CRUD
- `internal/httpapi/menucards/`：PUT/GET 树；cards → 410
- `internal/channel/tg/`：用 Compile 结果
- `web/admin/src/lib/api/channel-menu.ts`：MenuTree 类型
- `web/admin/src/features/menu/`：Puck config、映射、`MenuPhone`、整页编辑、详情只读
- `web/admin/src/routes/_app/channels/$id.menu.tsx`（或等价）
- `web/admin/src/features/quick-config/lib/menu-payload.ts`

---

### Task 1: MenuTree 校验与默认树

**Files:** `internal/menucard/domain/tree.go`（新）、`tree_test.go`（新）；逐步停用平铺 `Menu`+`CardID`

**Produces:** `type MenuTree struct { ID string; Columns int; Items []Button }`；`ValidateTree(t MenuTree, workflowExists func(string) bool) error`；`DefaultMenuTree(channelID string) MenuTree`

- [ ] 在 `tree_test.go` 写失败用例：根 0/7 个按钮、columns 0/7、open_card 无 Card、深度 9、重复 ID、open_workflow 回调返回 false、非法 URL
- [ ] `go test ./internal/menucard/domain/ -count=1` 确认失败
- [ ] 实现 `ValidateTree` / `DefaultMenuTree`（图片 open_workflow `default-image` + 帮助 send_text）
- [ ] 测试通过后提交：`feat(menucard): add nested MenuTree validation`

### Task 2: Compile 与 placements

**Consumes:** `MenuTree`  
**Produces:** `type CompiledMenu struct { Rows [][]CompiledButton }`；`type CompiledButton struct { ID, Label string; Action ButtonAction }`；`Compile(t MenuTree) CompiledMenu`（Action.Card 仅用于发送，adapter 按 ID 查 `CardByOpenerID map[string]Card`）；`WalkWorkflowPlacements(t MenuTree) []Placement`

- [ ] 测试：2 列 3 按钮折成 2+1 行；open_card 进入 map；两层卡片 DFS placements 路径 labels
- [ ] 实现 Compile + Walk
- [ ] `go test ./internal/menucard/domain/` 通过并提交

### Task 3: GET/PUT 树与旧 JSON

**Modify:** `internal/httpapi/menucards/handler.go`、`handler_test.go`、persistence

- [ ] 测试：PUT 合法树 GET 一致；无行 GET 默认树且不写库；旧 `card_id` JSON GET 当无配置
- [ ] Put 调 `ValidateTree`；unmarshal 失败当无配置
- [ ] 提交

### Task 4: 废卡片 API

- [ ] 测试 GET/POST `/cards` → 410，不插入 `channel_cards`
- [ ] handler 去掉卡片写；repository 删卡片方法；渠道/Case 删除只处理树
- [ ] 提交

### Task 5: Bot 下钻

**Modify:** `internal/channel/tg/`、`menu_runtime_test.go`、livedemo seed

- [ ] 测试点根按钮发送内嵌卡片；点卡片按钮再发下一层
- [ ] adapter 用 Compile + 节点 id callback
- [ ] seed 改默认树；提交

### Task 6: 架构文档

- [ ] 改 `docs/architecture/data-model.md`（及仍写分表的 bounded-contexts/runtime）：单 JSON 树，无 cards 管理路径
- [ ] 提交 docs

### Task 7: puckData 映射

**Create:** `web/admin/src/features/menu/menu-tree.ts`、`puck-map.ts`、`puck-map.test.ts`

**Produces:** `toPuckData(tree)` / `fromPuckData(data): MenuTree` 往返含两层卡片

- [ ] 失败测试往返；实现；`pnpm --filter pixoma-admin test` 相关文件；加 `@measured/puck`；提交

### Task 8: Puck config + 整页 + 详情只读

**Create:** puck config 组件、`MenuPhone`、`routes/_app/channels/$id.menu.tsx`  
**Modify:** `channel-detail-panel.tsx`、删地图/弹层/卡片 API 调用

- [ ] 契约测试：详情无 capability map 有只读手机；整页有调色板；config 无 form-input；非法动作不能 PUT
- [ ] 实现整页 Puck（DropZone+grid，不行再 dnd-kit）、保存 PUT、未保存 `useBlocker`
- [ ] 浏览器或契约验证后提交

### Task 9: quick-config

- [ ] `addWorkflowMenuEntry` 满 6 失败；append open_workflow
- [ ] 改 commit 写树；提交

### Task 10: 回归

- [ ] `go test` menucard / channel/tg / httpapi/menucards；admin menu + quick-config vitest
- [ ] 勾选 `docs/openspec/changes/puck-channel-menu-editor/tasks.md` 对应项

## Spec coverage

- puck 编辑器：Task 7–8
- 详情只读/整页：Task 8
- 嵌套树/API/410：Task 1–4
- Bot 卡片链：Task 5
- 主键盘 ≤6：Task 1 + 9
- 架构：Task 6
