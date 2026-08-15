---
comet_change: admin-shell-sidebar-footer
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-15-admin-shell-sidebar-footer
status: final
---

# admin-shell-sidebar-footer 技术设计

## 1. 目标与边界

把 `web/admin` 壳子从「模板顶栏 + Shadcn-Admin 文案」收成 Pixoma 后台壳：内容区无壳级顶栏；语言/主题在侧栏底栏；品牌为 Pixoma。

Canonical 行为见 OpenSpec delta：`docs/openspec/changes/admin-shell-sidebar-footer/specs/admin-web-shell/spec.md`。高层选型见同 change 的 `design.md`（方案 A）。

**非目标：** 改菜单项集合/顺序；重做 Logo/主题体系；新增 E2E；为手机单独保留内容顶栏。

## 2. 组件与数据流

```
_app.tsx (AppLayout)
  SidebarProvider
    AppSidebar
      SidebarHeader → AppTitle（Pixoma）
      SidebarContent → MENU_ITEMS（不动）
      SidebarFooter → LanguageSwitcher + ThemeSwitch
    SidebarInset
      Outlet（页面内容，无壳级 <header>）
```

- 唯一挂载语言/主题的壳路径：`AppSidebar` Footer。
- `authenticated-layout.tsx` 当前已无顶栏、也未挂语言/主题；本 change **不以它为主路径**，避免双布局漂移；若后续路由改用该布局，需同样走 Footer，不在此扩张范围。

## 3. 实现设计

### 3.1 移除内容区顶栏

文件：`web/admin/src/routes/_app.tsx`

- 删除 `SidebarInset` 内壳级 `<header>`（含右对齐的 `LanguageSwitcher` / `ThemeSwitch`）。
- 删除仅服务于该顶栏的 import。
- 内容区保持 `flex-1` + padding；页面从顶部直接开始。

### 3.2 侧栏品牌

文件：`web/admin/src/components/layout/app-title.tsx`（及 `assets/logo.tsx` 若含可访问名）

- 主标题文案改为 `Pixoma`。
- 移除副标题行（原「Vite + ShadcnUI」）。
- Logo / `<title>` / sr-only 中「Shadcn-Admin」改为 Pixoma。
- 保留现有折叠触发（`ToggleSidebar`）行为。

### 3.3 侧栏底栏工具

文件：`web/admin/src/components/layout/app-sidebar.tsx`

- 引入 `SidebarFooter`，在菜单 `SidebarContent` 之后渲染。
- Footer 内横向放置 `LanguageSwitcher` 与 `ThemeSwitch`（间距用现有 gap/padding token）。
- **不**改 `MENU_ITEMS` map，降低与 `tg-menu-config` 合并冲突。

### 3.4 LanguageSwitcher：图标 + 下拉

文件：`web/admin/src/components/layout/language-switcher.tsx`

现状：两个 `size='sm'` 文字按钮（中/EN），侧栏 `collapsible=icon` 时放不下。

**采用形态（已确认）：** 与 `ThemeSwitch` 对齐——

- Trigger：`Button` `variant='ghost'` `size='icon'`，图标用 `Globe`（或等价 lucide），`sr-only` 标明语言切换。
- Content：`DropdownMenu` 两项——中文 / English（文案继续走 i18n key）；当前项可带 `Check`。
- 行为不变：`i18n.changeLanguage` + `setStoredLocale`。
- 展开与收起共用同一套 UI，不做「展开双钮 / 收起另一套」分支。

`ThemeSwitch` 本身已是 icon trigger，Footer 收起态可直接复用；必要时外层加 `group-data-[collapsible=icon]:…` 居中。

### 3.5 并行 change 冲突面

| 文件 | 本 change | `tg-menu-config` 可能 |
|---|---|---|
| `app-sidebar.tsx` | Footer + import | 菜单项/顺序 |
| `app-title.tsx` | 文案 | 通常不动 |
| `_app.tsx` | 删顶栏 | 通常不动 |
| `language-switcher.tsx` | 下拉改版 | 通常不动 |

合并策略：先合菜单 diff，再合 Footer/Title；冲突时保留对方菜单块。

## 4. 边界条件

| 场景 | 期望 |
|---|---|
| 桌面展开侧栏 | 顶无壳栏；底栏可切语言/主题；品牌 Pixoma |
| 桌面收起（icon） | Footer 两图标仍可点；语言下拉可选 zh/en |
| 移动端抽屉 | 打开侧栏后底栏可用；无内容区壳顶栏 |
| 刷新 | 语言偏好仍按现有 storage 生效 |

## 5. 测试策略

- **手动（必做）：** 4.1–4.2 场景对照 OpenSpec scenarios。
- **单测：** 若有断言「Shadcn-Admin」或顶栏 `language-switcher` 位置的测试，改为 Pixoma / Footer 可达；`data-testid='language-switcher'` 可保留在新根节点上。
- **不做：** 新 E2E、视觉回归套件。

## 6. 风险

- 收起态下拉被侧栏 overflow 裁剪 → 确认 `DropdownMenu` portal 到 body（与 ThemeSwitch 一致，`modal={false}` 可沿用）。
- 双布局文件漂移 → 主路径只维护 `_app.tsx` + `AppSidebar`。

## 7. 任务映射

对应 `tasks.md`：1.x 品牌 → §3.2；2.x Footer → §3.3–3.4；3.x 去顶栏 → §3.1；4.x 验收 → §5。
