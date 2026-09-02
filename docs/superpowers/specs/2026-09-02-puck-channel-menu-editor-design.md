---
comet_change: puck-channel-menu-editor
role: technical-design
canonical_spec: openspec
---

# Puck 渠道菜单编辑器 — 技术设计

Canonical spec：`docs/openspec/changes/puck-channel-menu-editor/specs/`。高层决策见同 change 的 `design.md`。本文细化实现、边界和测试。

## 1. 现状锚点

- 表：`channel_main_menus`（每渠道一份 `DocJSON`）、`channel_cards`（卡片另表）。
- API：`GET/PUT /api/v1/channels/{id}/menu` + 全套 `/cards`。
- 管理端：`MenuCardEditor` 嵌在 `channel-detail-panel`：只读 `MenuCapabilityMap` + `MenuEditorModal`。
- Bot：`internal/channel/tg` 读 `Menu`/`Card`，callback 用菜单项/卡片按钮 id。

## 2. 领域模型

Go 用嵌套 struct 作唯一文档（JSON 入库）。建议形状：

```go
type MenuTree struct {
    ID      string
    Columns int
    Items   []Button
}

type Button struct {
    ID     string
    Label  string
    Action ButtonAction
}

type ButtonAction struct {
    Type       string // open_card | open_workflow | list_tasks | send_text | send_media | open_url | copy_text
    WorkflowID string
    Text       string
    Media      []Media
    URL        string
    Card       *Card // 仅 open_card，必须非 nil
}

type Card struct {
    Text    string
    Media   []Media
    Buttons []Button
}
```

规则（保存与编译共用）：

- `Columns` ∈ [1, 6]；`len(Items)` ∈ [1, 6]（空根拒绝；缺文档由 GET 填默认树，不把空树写库）。
- 每个 `Button.ID` 在整棵树唯一；前端生成 `btn-`/`kbd-` 前缀 uuid，后端拒绝冲突。
- `open_card` 必须带 `Card`；禁止 `card_id` 外键。
- 从根到任意卡片的 `open_card` 边数 ≤ 8。
- `open_workflow` 的 Case 必须存在（handler 查 case 目录后再 `Put`）。
- 切换动作类型时丢掉他类字段（前端映射时清；后端 Validate 忽略未知键）。

**默认树**：与今日 `DefaultMenu` 同语义（图片工作流按钮 + 帮助发文字），改写成 `MenuTree`，无独立卡片。

**Placements**：DFS 收集 `open_workflow`，路径 = 从根按钮 label 链；`Kind` 改为 `keyboard` | `card_button`。Case 删除仍调用「从树摘掉该 workflow」：摘掉后若根 `Items` 为空则改存默认树，禁止留下空根。

## 3. 持久化与 HTTP

- **继续用** `channel_main_menus.DocJSON` 存整棵 `MenuTree`。
- **停止写** `channel_cards`。`/cards*` 返回 **410**（body 短错误即可），测试断言不插入 `channel_cards`。
- GET：无行或 JSON 不能 unmarshal 成带 `items` 的新树 → 返回默认树（`ID=channelID`），**不**把默认树隐式 PUT。
- PUT：Validate + Case 存在性 → 整行覆盖。旧平铺 `Menu`（有 `items[].action.card_id`、无内嵌 `card`）视为不能 unmarshal 新树。
- Repository 端口收敛为树：`GetTree` / `PutTree` / `WorkflowPlacements` / `RemoveWorkflowReferences`；删 List/Create/Update/Delete Card。

渠道删除：只删 `channel_main_menus` 该行（卡片表不再作为管理数据源；迁移不做，残留 `channel_cards` 行可忽略）。

## 4. 编译与 Bot

`Compile(tree) → Runtime`：

- Keyboard：按 `Columns` 把根 `Items` 切成行，文案 = `Label`，callback/payload 键 = `Button.ID`。
- 卡片索引：`map[buttonID]Card`，仅 `open_card` 节点。
- 点击：查当前节点动作；`open_card` 发该 `Card`（媒体+正文+inline 按钮，按钮 callback = 子 `Button.ID`）；其它类型走现有 capability。

Adapter 不再 `GetCard(card_id)`。Session 若仍记「当前卡片」，改为记「当前按钮 id」（打开这张卡的那个节点）。

## 5. 管理端结构

### 路由

- 详情：`/channels/$id` 菜单区 = `<MenuPhonePreview tree={saved} readOnly />` +「编辑菜单」。
- 整页：新建 TanStack Router 子路由，例如 `/channels/$id/menu`。顶栏：渠道名、返回、保存。返回走 `useBlocker` 类未保存确认。

### 映射

`puckData ↔ MenuTree` 放纯函数（可单测，无 React）。Puck `content` 用自定义 type：`menu-button`、`menu-card`、`card-button`。Root props 持 `columns`。嵌套：`menu-button` 在 `action.type===open_card` 时含一个 `menu-card` slot；`menu-card` 含 `card-button` 列表。

入库 **只** PUT `MenuTree`。Load：GET 树 → `toPuckData`。

### 画布

1. 优先：Puck `DropZone` + CSS grid 做键盘；卡片视图切换用选中节点 kind（root/button 看键盘，card 看该卡）。
2. 若 DropZone 无法稳定成 N 列键盘：手机框仍是 Puck root，键盘格子用已有 `@dnd-kit`，Puck 只负责选中与右侧 fields。

Puck overrides：侧栏、字段换成 shadcn；禁止出码 Dialog、模板、主题商店。调色板白名单上述三积木，动作为 button fields 的 discriminated union，不是第四种积木。

只读手机与编辑画布 **共用** `MenuPhone` 展示组件（`readOnly` 关掉拖拽）。删除 `MenuCapabilityMap`、`MenuEditorModal`、`MenuWritableMap`。

### quick-config

`addWorkflowMenuEntry(tree, {workflowId, label})`：若 `len(items)>=6` 抛错；否则 append `open_workflow` 按钮。commit 走 PUT 树。

## 6. 错误与边界

| 情况 | 行为 |
|---|---|
| 根 0 个按钮 PUT | 400，库中仍为上一份 |
| 根 7 个按钮 | 400 |
| 深度 9 的 open_card | 400 |
| 两按钮同一 ID | 400 |
| 旧 /cards | 410，无写 |
| 旧 JSON GET | 当无配置，返回默认树 |
| 打开编辑时 GET 失败 | 整页错误条，不进空 Puck 可保存状态 |
| 并发两次 PUT | 最后写赢（与现菜单一致，不加乐观锁） |

## 7. 测试策略

- **Go**：Validate 表；Compile 折行与卡片索引；handler GET 默认 / PUT 圆整 / 410 cards；placements DFS；RemoveWorkflow 后空根→默认树；TG 测试点键盘出卡片、卡片再下钻。
- **前端**：映射往返（含两层卡片）；config 不含 input/select 积木；详情无 capability map；整页保存校验；quick-config 满 6 失败。
- **浏览器**：详情只读 → `/menu` 拖一个打开卡片 → 保存 → 回详情手机一致；非法 URL 保存失败。

## 8. 实现顺序

与 `tasks.md` 一致：domain → HTTP → bot → Puck 整页 → quick-config → 架构文档（`data-model.md` 表说明改为单 JSON 树，去掉 cards 管理路径）。

## 9. Spec Patch

已写入 `specs/puck-channel-menu-editor/spec.md`：详情只读手机、编辑整页。
