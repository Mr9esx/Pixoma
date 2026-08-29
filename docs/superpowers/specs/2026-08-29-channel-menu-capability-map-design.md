---
comet_change: channel-menu-editor-refactor
role: technical-design
prototype: docs/superpowers/prototypes/2026-08-29-channel-menu-capability-map.html
---

# 渠道详情 · 菜单能力地图

## 1. 目标

渠道详情的菜单区块回答三句话，再让人改配置：

1. 这个 bot 底下有哪些键
2. 点某一键之后对方会看到什么 / 会开始什么
3. 若弹出的是卡片消息：正文、图、上面的按钮分别是什么

详情页是**只读能力地图**。改配置走唯一主按钮「改菜单」，打开既有 `MenuEditorModal`。地图上不出现动作类型下拉、新建卡片、改列数。

交互与视觉以已确认原型为准：`docs/superpowers/prototypes/2026-08-29-channel-menu-capability-map.html`。

## 2. 范围

**做**

- 替换 `MenuCardEditor` 主视图（当前：大纲 tab + 手机预览）为能力地图
- 顶栏数字、主键盘格子、点下去路径、未挂上口袋、空 / 加载 / 失败 / 断引用
- 动作结果用人话写出；工作流显示 Case 名，不显示裸 id
- i18n `zh` / `en`；详情区块标题改为「菜单」，去掉说明性 hint

**不做**

- 不改菜单/卡片 API 与 `Action` 六种类型
- 不改 `MenuEditorModal` 内部编辑交互（大纲 / 卡片库 / 表单仍只在弹层）
- 不把 PhoneSimulation 留在详情主视图
- 不画系统自动加的「‹ 返回」
- 不改 bot 运行时、不改架构文档

## 3. 信息架构

菜单是两层，不是文件夹树。

| 层 | 对方在 Telegram 里遇到什么 | 数据 |
|---|---|---|
| 主键盘 | 输入框上方常驻键，按 `columns`（1–8）分行 | `Menu.items[]`，每项 `{label, action}` |
| 卡片 | 机器人发出的一条：媒体 + 正文 + 贴在消息上的按钮 | `Card`；可被多个键/按钮引用；也可写好但未挂上 |

主键盘的键与卡片按钮共用同一套动作：

| `action.type` | 地图人话 | 点下去必须摊开的内容 |
|---|---|---|
| `open_card` | 打开「卡片名」 | 媒体缩略图、正文、卡片按钮 |
| `open_workflow` | 开始「工作流名」 | Case 名称 |
| `send_text` | 发这段话 | 那段文本 |
| `send_media` | 发图 | 媒体缩略图（`image` →「图」，`video` →「视频」，`animation` →「动画」） |
| `open_url` | 打开链接 | URL 原文 |
| `copy_text` | 复制这段字 | 那段文本 |

卡片缺失 → 人话「卡片不存在」。工作流 id 在 Case 列表里找不到 → 「工作流不存在」。键仍留在主键盘上，不藏。

未挂上：从主键盘 `items` 出发，沿所有 `open_card`（含卡片按钮上的）能走到的卡片为已挂；其余卡片进底栏口袋。口袋没有条目时整块不渲染。

## 4. 版面

区块仍在 `channel-menu-section`。一张细线框卡片，无投影。陶土 accent 本屏不用。

**顶栏：** 左「菜单」+ 三个等宽数字（键 / 卡片 / 工作流入口）→ 右主按钮「改菜单」。工作流入口 = 主键盘与全部卡片按钮上、互不重复的 `open_workflow` 个数（断引用的也计入）。

**两栏（≥ 920px）：** 左主键盘，右点下去。`< 920px` 上下叠：顶栏 → 主键盘 → 点下去 → 未挂上。不横滑。

**主键盘：** `grid-template-columns: repeat(columns, minmax(0, 1fr))`。每键两行：`label` + 人话结果。`min-height: 44px`。选中：边框用 `--foreground` + `--muted` 底。断引用键加 destructive 描边，人话用 destructive 色。

**点下去：** 跟着当前路径走。未选任何键时右栏只留栏名「点下去」，不写引导句。打开卡片时按「一条消息」画（管理用 `name` 用 12px muted 放在消息框上方）。打开工作流 / 发文字 / 发媒体 / 链接 / 复制：不装成消息框，栏名 + 内容。系统「‹ 返回」不画。

**路径：** 面包屑 `主键盘 / 键名 / …`（未挂上起点为 `未挂上`）。点中间节回退。点卡片上的按钮把该按钮的动作推进路径。

**未挂上：** 虚线芯片，点了在右栏看内容，不把它画进主键盘。

## 5. 状态

| 状态 | 表现 |
|---|---|
| 加载 | 既有 `LoadingSkeleton`，不闪空地图 |
| 读取失败 | `菜单读取失败`（`ErrorBanner`） |
| 空菜单 | 数字全 0；主键盘一句「还没有键」；点下去空白；无未挂上；「改菜单」仍在 |
| 断引用 | 键/按钮留着，人话与点下去用「卡片不存在」或「工作流不存在」 |
| 「改菜单」 | 打开 `MenuEditorModal`；保存成功后关闭弹层并刷新地图，选中路径清空 |

## 6. 文案

| 位置 | 中文 | 备注 |
|---|---|---|
| 区块标题 | 菜单 | 去掉 `channels.tabMenuHint` |
| 主按钮 | 改菜单 | 已有 `menu.editMenu` |
| 栏名 | 主键盘 / 点下去 | |
| 空键盘 | 还没有键 | |
| 口袋标题 | 未挂上 · N | N 等宽 |
| 失败 | 菜单读取失败 | |
| 人话 | 打开「…」/ 开始「…」/ 发这段话 / 发图 / 打开链接 / 复制这段字 | |
| 断 | 卡片不存在 / 工作流不存在 | |
| 点下去栏名 | 开始工作流 / 发这段话 / 复制这段字 / 发图 / 打开链接 | 与人话对齐，不解释 |

禁用词与说明性 hint 不出现。数字 `tabular-nums`。

## 7. 组件与数据

宿主仍是 `MenuCardEditor`（`channel-detail-panel` 的 `channel-menu-section` 不改挂载点）：拉 `getMenu` / `listCards` / `listCases`，渲染地图，挂 `MenuEditorModal`。

新组件 `web/admin/src/features/menu/menu-capability-map.tsx`：纯展示 + 路径 state。纯函数进 `menu-flow.ts`（单测）：

- `actionOutcomeLabel(action, cards, workflows)` → 人话
- `actionIsBroken(action, cards, workflows)` → bool
- `reachableCardIds(menu, cards)` → 从主键盘沿 `open_card` 走到的 id（走访集合去重，卡片互指不循环）
- `orphanCards(menu, cards)` → 未挂上
- `workflowEntryCount(menu, cards)` → 顶栏第三个数

路径 state：

```ts
type MapTrailStep = {
  kind: 'item' | 'orphan' | 'btn'
  id: string
  label: string
  action: Action
}
```

`kind: 'btn'` 时 `id` 为卡片按钮 id。点主键盘键则 trail 重置为该项；点未挂上芯片则起点 `kind: 'orphan'`。

媒体缩略图：有 URL 用 `<img>`；无 URL 或加载失败用 muted 色块 + 「图」/「视频」。不引入新图库。

## 8. 测试

- `menu-flow`：人话、断引用、可达/未挂上、循环引用不死循环、工作流去重计数
- 地图：点键展开卡片；点卡片按钮推进路径；面包屑回退；空菜单；未挂上不进主键盘
- 契约：`data-testid='menu-card-editor'`、`data-testid='edit-menu'` 保留；主视图**不再**要求 `tab-outline` / `tab-library` / `preview-pane`（这两 tab 只在弹层）
- `channel-layout.contract.test.ts` 的 `#channel-menu-section` 挂载点不变

## 9. 设计系统

- 语义令牌 + `color-mix`；表面无投影；主 CTA 仅「改菜单」
- 触屏键 ≥ 44px；聚焦环可见；920px 重排
- 亮暗成对；hover 后文字不变浅
