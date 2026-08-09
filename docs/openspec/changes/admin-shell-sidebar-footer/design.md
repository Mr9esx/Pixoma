## Context

当前壳子在 `web/admin/src/routes/_app.tsx` 的 `SidebarInset` 内有一条 `header`（语言 + 主题右对齐）；侧栏 `AppTitle` 仍写死「Shadcn-Admin / Vite + ShadcnUI」。基座已提供 `SidebarHeader` / `SidebarFooter` 与收起态。动机与范围见 `proposal.md`；行为契约见 `specs/admin-web-shell/spec.md`。用户已选定 **方案 A：侧栏底栏工具**。

## Goals / Non-Goals

**Goals:**

- 内容区去掉壳级顶栏，页面从内容区顶部直接开始
- 语言/主题迁入侧栏 Footer，展开与收起均可操作
- 品牌文案改为 Pixoma，清掉模板默认标题/副标题

**Non-Goals:**

- 不改菜单项集合与顺序（含并行 change 可能新增的 TG Menu）
- 不重绘 Logo SVG、不做新设计系统
- 不为手机单独保留内容区顶栏（侧栏 sheet + 现有折叠触发足够）
- 不改业务页内部布局

## Decisions

### 1. 顶栏删除点：`_app.tsx`（及等价 authenticated layout）

- **选择**：直接移除 `SidebarInset` 内壳级 `<header>`；`LanguageSwitcher` / `ThemeSwitch` 改由侧栏挂载。
- **备选**：保留空顶栏只放 `SidebarTrigger` — 否决，桌面侧栏已有折叠控件，移动端触发在 `AppTitle`。
- **备选**：仅 CSS 隐藏顶栏 — 否决，DOM/无障碍仍会残留。

### 2. 工具挂载：`SidebarFooter` + 紧凑控件组

- **选择**：在 `AppSidebar` 增加 `SidebarFooter`，内放语言与主题控件（横向或图标组）；收起态依赖现有 sidebar collapsed 样式，必要时包一层可点图标 + tooltip。
- **备选**：做成菜单项（方案 C）或品牌下工具条（方案 B）— 用户已否决。

### 3. 品牌：只改文案，少动结构

- **选择**：`AppTitle` 主标题改为 `Pixoma`，去掉副标题行；`Logo` 的 `<title>` / 可访问名称若仍写 Shadcn-Admin 则一并改为 Pixoma。
- **备选**：引入图片 Logo — 本期不做。

### 4. 与并行 change 的边界

- **选择**：本 change 只碰品牌区与 Footer/顶栏；`MENU_ITEMS` 与菜单渲染逻辑尽量不动，降低与 `tg-menu-config` 的合并冲突面。
- 若两边同改 `app-sidebar.tsx`，优先保留对方菜单 diff，本侧只加 Footer 与 Title。

## Risks / Trade-offs

- **[Risk] 侧栏收起后语言/主题难发现** → Footer 用图标 + tooltip；手动验一次 collapsed 态。
- **[Risk] 与 `tg-menu-config` 同改侧栏文件冲突** → 任务拆小、少动菜单 map；合并时先合菜单再合 Footer。
- **[Trade-off] 手机无内容顶栏** → 依赖侧栏抽屉；接受以换取内容区更干净。

## Migration Plan

- 纯前端布局变更，无数据迁移；发布即生效。
- 回滚：恢复 `_app.tsx` 顶栏与原 `AppTitle` 文案即可。

## Open Questions

（无；方案 A 已确认。）
