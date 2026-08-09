---
comet_change: admin-theme-neutral-base
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-09-admin-theme-neutral-base
status: final
---

# admin-theme-neutral-base 技术设计

## 1. 目标与边界

把 `web/admin` 的 shadcn 基色从 **slate** 换成 **neutral**，消除暗色「冷蓝屏」，并使亮/暗共用同一套中性灰语义 token。

Canonical 行为见 OpenSpec delta：`docs/openspec/changes/admin-theme-neutral-base/specs/admin-web-shell/spec.md`。高层选型见同 change 的 `design.md`。

**非目标：** 自定义品牌色板、多皮肤、改 ThemeProvider、改布局/字体/圆角体系、单独定制 chart 为无色、等待或依赖尚不可用的 `shadcn migrate base-color`。

## 2. 现状与约束

| 项 | 现状 |
|---|---|
| 配置 | `web/admin/components.json` → `baseColor: "slate"` |
| Token | `web/admin/src/styles/theme.css` `:root` / `.dark` 为 slate OKLCH（暗色背景约 hue 264、chroma 0.042） |
| 切换 | `ThemeProvider` cookie + `html.dark`；本变更不改 |
| Sidebar | 项目约定 `--sidebar: var(--background)` 等映射，非官方独立 sidebar 色值 |
| CLI | 当前 `shadcn migrate --list` 仅有 icons / radix / rtl，**无** base-color |
| 硬编码 | 全仓仅 `show-submitted-data.tsx` 的 `bg-slate-950` |

官方 neutral 暗色主表面为 chroma 0 的真灰，例如：

```css
.dark {
  --background: oklch(0.145 0 0);
  --foreground: oklch(0.985 0 0);
  /* …其余语义 token 对齐官方 neutral 文档脚手架 */
}
```

## 3. 实现设计

### 3.1 更新基座配置

文件：`web/admin/components.json`

- `tailwind.baseColor`: `"slate"` → `"neutral"`
- 其余字段不动

### 3.2 对齐主题 CSS 变量

文件：`web/admin/src/styles/theme.css`

1. 将 `:root` 与 `.dark` 中 **语义色**（background、foreground、card、popover、primary、secondary、muted、accent、destructive、border、input、ring、chart-1…5）替换为 [shadcn theming 文档](https://ui.shadcn.com/docs/theming) 中 **Default Theme CSS（neutral）** 对应值。
2. **保留**：
   - `--radius: 0.625rem`（与现网一致即可）
   - `@theme inline` 中的字体与 `--color-*` 映射块（不必改成官方 radius 公式，除非现有 `@theme` 已依赖旧写法且冲突）
   - 项目约定的 sidebar 变量映射：

```css
--sidebar: var(--background);
--sidebar-foreground: var(--foreground);
--sidebar-primary: var(--primary);
--sidebar-primary-foreground: var(--primary-foreground);
--sidebar-accent: var(--accent);
--sidebar-accent-foreground: var(--accent-foreground);
--sidebar-border: var(--border);
--sidebar-ring: var(--ring);
```

3. **不要**整文件覆盖成官方脚手架（会冲掉 `index.css` 分工与字体主题）。只换色值 + 保持现有结构。

### 3.3 硬编码色

文件：`web/admin/src/lib/show-submitted-data.tsx`

- `bg-slate-950` → `bg-muted` 或 `bg-background`（选对比度足够、暗色代码块可读的一项；优先 `bg-muted`）

实施前再 `rg '\bslate-[0-9]' web/admin` 确认无新增。

### 3.4 不做的事

- 不改 `ThemeProvider`、命令面板主题项、i18n
- 不批量替换 Tailwind 调色板类名
- 不改 emerald 等状态色
- 不把 chart token 洗成灰色

## 4. 测试策略

| 层级 | 内容 |
|---|---|
| 合同测试 | 新增或扩展现有 shell/theme 合同：读 `components.json` 断言 `baseColor === "neutral"`；解析 `theme.css` 暗色 `--background`，断言 chroma≈0（或字符串匹配 `oklch(0.145 0 0)` / 不含 slate 典型 `264` 色相） |
| 回归 | 既有 theme / locale / shell 合同测试保持通过 |
| 手工 | light / dark / system 切换；Dashboard + 任一资源页目视主背景与卡片无蓝灰主调 |

## 5. 风险与回滚

| 风险 | 缓解 |
|---|---|
| 手工 token 与官方文档版本漂移 | 以当前 shadcn 文档 Default Theme CSS（neutral）为单一来源；改完对比 chroma 是否为 0 |
| 局部硬编码残留 | 实施时 ripgrep；目前仅一处 |
| chart 仍带色被误判为「还蓝」 | Spec 已声明主表面与 chart 解耦；验收看主背景/卡片 |

回滚：还原 `components.json` 与 `theme.css`（及那一处 class）即可，无数据迁移。

## 6. Spec Patch（已回写）

delta `specs/admin-web-shell/spec.md` 增加场景「主表面中性不影响 chart 色相」。
