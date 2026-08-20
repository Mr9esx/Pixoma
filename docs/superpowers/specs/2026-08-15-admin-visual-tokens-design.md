---
role: technical-design
status: proposed
---

# admin-visual-tokens 技术设计

## 1. 目标与边界

只改「鼓不鼓」：`web/admin` 贴在页面上的盒子改成细线框、无表面投影。配色保持现有 **neutral** 纯灰，不改回 slate。

日常观感：卡片、内容画布、按钮、输入框不再垫高；颜色、左右分栏、点哪里都和现在一样。下拉菜单和对话框仍有阴影，否则会糊在页面上。

**非目标：**

- 不改 `theme.css` 语义色、`components.json` 的 `baseColor: "neutral"`、圆角、字体
- 不改写「中性基色主题」旧约定（暗色仍不许以 slate 冷蓝为主调）
- 不换成 Next.js，不搬付费 Admin Kit 源码
- 不改 Master–Detail、菜单、路由、i18n、ThemeProvider
- 不把 inset 内容区改成齐平 `sidebar` variant（只去掉这块画布的阴影，圆角与边距保留）

## 2. 现状

表面投影来自组件 class，不是色板：

| 位置 | 现状 |
|---|---|
| `components/ui/card.tsx` | `shadow-sm` + `border` |
| `SidebarInset`（inset 变体） | `shadow-sm` + `rounded-xl` |
| `button` 若干变体、多数表单控件 | `shadow-xs` |
| 对话框 / 下拉 / popover / sheet | `shadow-md` / `shadow-lg`（浮层，保留） |

色值已是官方 neutral，合同测试锁在 `theme-neutral.contract.test.ts`。本变更不碰这些色值。

## 3. 实现设计

### 3.1 表面无投影、浮层保留

「表面」= 贴在页面上的盒子；「浮层」= 盖在页面上的层。

| 改 | 文件 | 动作 |
|---|---|---|
| 卡片 | `components/ui/card.tsx` | `shadow-sm` → `shadow-none`，保留 `border` |
| 内容画布 | `components/ui/sidebar.tsx` `SidebarInset` | 去掉 inset 的 `shadow-sm`，保留 `m-2` / `rounded-xl` |
| 按钮 | `components/ui/button.tsx` default / destructive / outline / secondary | 去掉 `shadow-xs` |
| 表单控件 | `input` / `textarea` / `select` trigger / `checkbox` / `switch` / `radio-group` / `input-otp` / `calendar` | 去掉 `shadow-xs` |
| 筛选箭头 | `components/filters/filter-segment.tsx` | 左右箭头去掉 `shadow-sm`，保留 `border` |

**保留投影：** `dialog`、`alert-dialog`、`dropdown-menu`、`popover`、`select` 弹出层、`sheet`、批量操作条、加载条。

不要全局 `* { box-shadow: none }`。不要改 `theme.css`。

### 3.2 合同测试与规范

1. **保留** `theme-neutral.contract.test.ts`：`baseColor === "neutral"`、暗色 `--background` 为 `oklch(0.145 0 0)`、源码无硬编码 `slate-*`。
2. **新增**表面阴影抽样（可并入该文件或旁建 `theme-surface.contract.test.ts`）：
   - `card.tsx` 不含 `shadow-sm`
   - `SidebarInset` 不含 `shadow-sm`
   - `button.tsx` 的 default 变体不含 `shadow-xs`
3. OpenSpec `admin-web-shell`：基色条款不动；可补一条「表面无 drop shadow、浮层可保留」。不改 `docs/architecture/`。

### 3.3 不做的事

- 不改 ThemeProvider、命令面板、locale
- 不把 Master–Detail 改成整页大表
- 不引入多皮肤 / 自定义品牌主色
- 不把 chart token 洗成灰色

## 4. 测试策略

| 层级 | 内容 |
|---|---|
| 合同 | 既有 neutral 色值断言 + 3.2 阴影抽样 |
| 回归 | 既有 shell / locale / 资源页测试保持通过 |
| 手工 | 亮色与暗色：Dashboard、任一资源左右分栏、打开下拉或对话框。表面无投影；浮层仍有层次；颜色仍是现在的纯灰 |

## 5. 风险与回滚

| 风险 | 缓解 |
|---|---|
| 漏改某一控件仍有 `shadow-xs` | 实施时检索 `web/admin/src/components` 的表面 `shadow-sm` / `shadow-xs`，按 3.1 表处理 |
| 后续 `shadcn add` 把卡片阴影加回来 | 合同测试锁卡片/按钮；新增 UI 不要把 `shadow-sm` 加回表面 |

回滚：还原上述 class 与新增测试。无数据迁移，不涉及 `theme.css`。
