## Why

管理后台暗色模式当前使用 shadcn **slate** 基色，OKLCH 色相约 265° 且带明显饱和度，大面积背景/卡片/muted 叠在一起会显「冷蓝屏」。需要换成真中性灰（**neutral**），让日夜主题都干净、不发蓝，符合专业后台观感。

## What Changes

- 将 `web/admin` 的 shadcn `baseColor` 从 `slate` 切换为 `neutral`
- 同步更新亮色（`:root`）与暗色（`.dark`）CSS 语义 token（background、foreground、card、primary、muted、sidebar 等）
- 审计并清理会与新基色打架的硬编码 `slate-*` 工具类（若存在且影响观感）
- 保持现有明暗主题切换、cookie 持久化与命令面板主题命令行为不变

## Capabilities

### New Capabilities

- （无）

### Modified Capabilities

- `admin-web-shell`: 「布局与基础主题」从未指定基色，收紧为亮/暗共用 **neutral** 中性基色；暗色 MUST 不以蓝灰（slate）为主调

## Impact

- 代码：`web/admin/src/styles/theme.css`、`web/admin/components.json`；可能波及少量硬编码色类的组件
- API / 后端：无
- 依赖：可选用 shadcn `migrate base-color` CLI，或以官方 neutral token 手工对齐
- 用户可见：亮暗模式整体灰阶观感变化；功能行为不变
