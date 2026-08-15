# Comet Design Handoff

- Change: admin-shell-sidebar-footer
- Phase: design
- Mode: compact
- Context hash: f9e34b593181fc44185cabf35e24b0914dfbedf9eda76e597cc3d87f5a4825e7

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/admin-shell-sidebar-footer/proposal.md

- Source: docs/openspec/changes/admin-shell-sidebar-footer/proposal.md
- Lines: 1-28
- SHA256: 00399dc0bf35925cea8647a6baa48cb8bee8661542f9add314385d48edf89bc7

```md
## Why

管理控制台壳子仍带 shadcn-admin 模板痕迹：侧栏标题是「Shadcn-Admin / Vite + ShadcnUI」，内容区上方另有一条仅放语言/主题的顶栏，占空间且品牌不清晰。需要把壳子收成 Pixoma 后台的样子：品牌进侧栏，工具进侧栏底栏，内容区干净铺开。

## What Changes

- 移除内容区上方的 Header（语言/主题所在顶栏）
- 将语言切换、主题切换迁到侧栏 **Footer 底栏**（方案 A）
- 侧栏品牌文案改为 **Pixoma**，去掉模板默认副标题与「Shadcn-Admin」文案
- 侧栏收起为图标模式时，底栏控件以图标形式仍可操作（tooltip/等价可发现交互）
- 不改业务资源页、不改菜单条目与顺序、不重做主题体系

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `admin-web-shell`: 布局从「侧栏 + 内容顶栏」调整为「侧栏（品牌 + 菜单 + 底栏工具）+ 无顶栏内容区」；品牌标识为 Pixoma；语言/主题入口位于侧栏底栏

## Impact

- 代码：`web/admin` 壳子相关（如 `_app.tsx` / 布局、`AppTitle`、`AppSidebar`、语言与主题组件挂载点）；可能触及侧栏 Footer / 收起态样式
- API / 后端：无
- 依赖：无新增运行时依赖
- 并行 change：与 `tg-menu-config` 均可能改 `admin-web-shell` 侧栏；本 change 只动品牌与顶栏/底栏工具，不改菜单项集合；合并时注意侧栏文件冲突

```

## docs/openspec/changes/admin-shell-sidebar-footer/design.md

- Source: docs/openspec/changes/admin-shell-sidebar-footer/design.md
- Lines: 1-56
- SHA256: 34efa7c0372768a1dc27a5ef58079fd04c7e681b294d842e542493c4460d16f5

```md
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

```

## docs/openspec/changes/admin-shell-sidebar-footer/tasks.md

- Source: docs/openspec/changes/admin-shell-sidebar-footer/tasks.md
- Lines: 1-20
- SHA256: 77543369db08ba2b8cea04ee6c0beaf628cc43252b66d96d79fad870374c9411

```md
## 1. 品牌文案

- [ ] 1.1 将 `AppTitle` 主标题改为「Pixoma」，移除「Vite + ShadcnUI」等模板副标题
- [ ] 1.2 清理 Logo / 可访问名称中仍残留的「Shadcn-Admin」文案，改为 Pixoma

## 2. 侧栏底栏工具

- [ ] 2.1 在 `AppSidebar` 增加 `SidebarFooter`，挂载 `LanguageSwitcher` 与 `ThemeSwitch`
- [ ] 2.2 调整 Footer 在侧栏展开/收起态下的布局，确保收起态仍可图标操作（含必要 tooltip）

## 3. 移除内容区顶栏

- [ ] 3.1 从 `_app.tsx`（及若仍使用的 `authenticated-layout`）移除壳级顶栏，内容区直接渲染页面
- [ ] 3.2 确认语言/主题不再从内容区顶栏导入，避免重复挂载

## 4. 验收

- [ ] 4.1 手动确认：侧栏显示 Pixoma、无模板默认文案；内容区无壳级顶栏；底栏可切换语言与主题
- [ ] 4.2 手动确认：侧栏收起为图标模式后，仍可切换语言与主题
- [ ] 4.3 跑通 `web/admin` 既有相关单测（若有布局/壳子测试则一并更新）

```

## docs/openspec/changes/admin-shell-sidebar-footer/specs/admin-web-shell/spec.md

- Source: docs/openspec/changes/admin-shell-sidebar-footer/specs/admin-web-shell/spec.md
- Lines: 1-25
- SHA256: c1da1e50ce0c7a098cf7d19afcfba05520883fc220a692666f55af09763fa8f2

```md
## MODIFIED Requirements

### Requirement: 布局与基础主题
系统 MUST 提供含侧栏的管理布局：侧栏包含品牌区、导航菜单与底栏工具区；内容区 MUST NOT 再提供仅承载语言/主题切换的顶栏。系统 MUST 继续支持基座自带的基础主题切换能力（若基座提供明暗主题）。语言切换与主题切换控件 MUST 放置在侧栏底栏；当侧栏收起为图标模式时，上述控件 MUST 仍可以图标形式被发现并操作。

#### Scenario: 布局稳定包裹页面
- **WHEN** 用户在资源页之间切换
- **THEN** 侧栏布局保持，内容区切换为目标页，且内容区上方不出现仅含语言/主题的顶栏

#### Scenario: 侧栏底栏可切换语言与主题
- **WHEN** 用户打开控制台并查看侧栏底栏
- **THEN** 可见语言切换与主题切换入口，且操作后界面语言或主题相应变化

#### Scenario: 侧栏收起后仍可操作底栏工具
- **WHEN** 侧栏处于收起（图标）模式
- **THEN** 用户仍可通过底栏对应图标入口切换语言或主题

## ADDED Requirements

### Requirement: 侧栏品牌为 Pixoma
侧栏品牌区 MUST 展示项目名称「Pixoma」，且 MUST NOT 展示 shadcn-admin 模板默认文案（如「Shadcn-Admin」「Vite + ShadcnUI」）。

#### Scenario: 侧栏可见 Pixoma
- **WHEN** 用户打开管理控制台
- **THEN** 侧栏顶部品牌区显示「Pixoma」，且不显示上述模板默认主/副标题文案

```
