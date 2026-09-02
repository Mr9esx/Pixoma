## Context

现行实现已是 `Menu`（`columns` + `items[].action`）加独立 `Card` 表（`channel_menus` / `channel_cards`），管理端是只读能力地图 + 弹层可写地图，不是可视化画布。规格里仍残留更早的「capability_id 中立树」。Bot 在 `internal/channel/tg` 消费 domain Action。Admin 已有 `@dnd-kit`、`@xyflow/react`、shadcn；**不**引入 shadcn-builder（Next.js + Convex + Clerk 的表单导出器）。详见 `proposal.md` 的 Why / 非目标。

## Goals / Non-Goals

**Goals:**

- 管理端嵌入 Puck，用自定义组件 + 手机 root 画布编辑一棵树。
- 后端只存这份树，编译成 Bot 已理解的键盘/发消息/callback 形态（可新 IR，不必兼容旧 Card id）。
- 删掉 cards 集合 API 与卡片库 UI。

**Non-Goals:**

- 不把 Puck JSON 原样交给 Telegram。
- 不做多渠道平台的第二套画布（本期按 Telegram 手机）。
- 不归档、不修改 `channel-menu-editor-refactor` 目录。

## Decisions

### 1. 编辑内核用 Puck，不用 shadcn-builder / Craft.js / 纯 xyflow

- **选择**：`@measured/puck` 嵌在 Vite admin 里。调色板注册我们的组件：`MenuKeyboard`（root）、`MenuButton`、`MenuCard`、`CardButton`。动作是按钮 props，不是独立积木。
- **理由**：要的是「自己的 React 组件 + JSON 文档 + 现成三栏编辑壳」。shadcn-builder 是表单字段和出码；Craft.js 要自绘全部壳；xyflow 是图不是手机界面。
- **备选**：自研 dnd-kit 三栏——工作量更大，本期不采用。

### 2. 画布 root 做成手机，不用 Puck 默认整页

- **选择**：自定义 `root` render：固定宽度手机框，主键盘在下、卡片在上（进入卡片节点时切换）。列数是键盘 props（1–6，与交互规格主键盘上限一致）。
- **理由**：运营要对着用户视线编。Puck 默认网页堆叠块会编出不像 TG 的布局。
- **备选**：网页画布 + 旁路手机预览——已否决。

### 3. 存储一份「规范树」，Puck data 是投影

- **选择**：持久化领域模型为嵌套 JSON（Go struct 可序列化），不是 Puck 的 `content` 数组原样入库。前端 load/save 时做 `puckData ↔ MenuTree` 映射。校验、Case 存在性、placements 都走领域树。
- **理由**：Bot、quick-config、placements 不应依赖 Puck 版本字段；换编辑器不必迁库。
- **备选**：只存 Puck JSON——拒绝，编辑器实现细节会泄漏到 Go。

规范树（示意）：

```
MenuTree
  columns
  items[]: Button
    id, label
    action: { type, workflow_id | text | media | url | card? }
    card?: Card          // 仅 type=open_card
      text, media[]
      buttons[]: Button  // 可再带 card
```

编译：根 `items` → ReplyKeyboard（按 `columns` 折行）；`open_card` → 发 `card` 为消息 + inline keyboard；其余动作沿用现有 capability 行为。节点 id 用于 callback，不再用独立 `card_id` 外键。

### 4. 一张表、废卡片表

- **选择**：`channel_menus`（或等价）存整份树 JSON；删除 `channel_cards` 的管理路径。上线不迁移：读到无法解析的旧行则视为无配置，走默认树。
- **理由**：用户选择 wipe；树内嵌卡片后第二张表没有产品语义。
- **备选**：双写过渡——不做。

### 5. 主键盘最多 6 个直达按钮

- **选择**：与 `channel-menu-interaction` 一致，根层 `items.length > 6` 拒绝保存。更多入口只放卡片按钮。列数 1–6。
- **理由**：避免和现行规格打架；现编辑器曾允许 8 列，本期收口。
- **备选**：跟现 UI 放宽到 8——不采用，除非以后改交互规格。

### 6. 嵌套深度上限

- **选择**：打开卡片链最大深度 8（根键盘为 0）。超限拒绝保存。
- **理由**：树-only 允许再挂卡片，无限嵌套会撑爆消息路径和编辑器。
- **备选**：不限制——不采用。

### 数据流

```
运营拖拽 Puck
    → puckData
    → toMenuTree() + Validate（含 Case 存在、深度、根≤6）
    → PUT /menu 存 JSON
Bot 读树
    → Compile(tree) → Keyboard + 按 node id 取卡片
    → TG Adapter 发送
```

### quick-config

`addWorkflowMenuEntry` 改为：在树根 `items` 追加一个 `open_workflow` 按钮（受 6 上限约束，满则失败并让运营去编辑器）。不再写 `Card`、不再 `putMenu` 旧 Menu 形状。

## Risks / Trade-offs

- [Puck 默认 UI 与 Pixoma 设计系统不一致] → 用 Puck overrides 换成 shadcn 控件与设计令牌，画布外壳走现有 Dialog/页面。
- [Puck 画布难做成「真·键盘网格」] → `MenuKeyboard` 用 CSS grid + 我们的 DropZone；不行则 Puck 只管选中/属性，网格拖拽用已有 `@dnd-kit` 嵌在 root 内。
- [wipe 导致演示/现网菜单空白] → 默认树覆盖空配置；livedemo seed 改写成新树。
- [旧前端/脚本仍打 `/cards`] → 接口明确失败；admin 去掉调用。
- [深度嵌套 UX] → 画布进入子卡片时提供面包屑回到键盘。

## Migration Plan

1. 先落地领域树 + GET/PUT `/menu` + 编译器 + 测试；旧 cards 路由改为不可用。
2. 再换 admin 编辑器与 quick-config。
3. 部署后旧 JSON 不解析即默认树；无 rollback 到旧 Card 模型（用户接受作废）。
4. 更新 `docs/architecture/data-model.md`（及仍写 Menu/Card 分表的 bounded-contexts / runtime 段落）。

## Open Questions

- Puck 版本选定以当时 npm 稳定版为准，不锁进规格。
- 只读能力地图：用同一套组件只读渲染，或保存后编译预览；实现时选一种，不影响契约。
