# Provenance — 证据映射与生成记录

## 源项目

- 源项目 id：`9d4ece8d-9cf2-411b-8589-7e499a46e3fe`
- 源项目名：整理这个项目的设计系统、设计语言、以
- 源仓库：`/Users/mr9esx/Documents/Pixoma`
- 本设计系统 id：`user:design-system`
- 生成日期：2026-08-26

## 证据映射

| 本包文件 | 源文件 | 处理 |
|---|---|---|
| `build/theme.css` | `web/admin/src/styles/theme.css` | 原样复制（事实来源） |
| `build/source-examples/filter-segment.tsx` | `web/admin/src/components/filters/filter-segment.tsx` | 原样复制（P0 组件实现） |
| `build/source-examples/list-panel.tsx` | `web/admin/src/features/cases/list-panel.tsx` | 原样复制（Master–Detail 范例） |
| `build/source-examples/voice-profile.md` | `docs/voice-profile.md` | 原样复制（文案规范） |
| `build/icons/icon-*.tsx` | `web/admin/src/assets/brand-icons/*.tsx`、`assets/custom/*.tsx` | 原样复制（运行时图标） |
| `assets/logo.png` | 仓库根 `logo.png`（1024×1024） | 原样复制 |
| `assets/logo-admin.png` | `web/admin/public/images/logo.png`（512×512） | 原样复制 |
| `assets/apple-touch-icon.png` | `web/admin/public/images/apple-touch-icon.png` | 原样复制 |
| `assets/favicon.png` / `favicon-light.png` / `favicon.ico` | `web/admin/public/images/` 同名文件 | 原样复制（`favicon_light` 改名 `favicon-light`） |
| `pixoma-design-system.html` | 源项目交付的规范总览页 | 保留原样 |
| `pixoma-design-tokens.css` | 源项目交付的参考令牌 | 保留原样 |
| `brand-spec.md` | 源项目交付的品牌摘要 | 保留原样 |
| `context/source-context.md` | 源项目交接上下文 | 保留原样 |

## 设计决策记录

- `colors_and_type.css` 以 `theme.css` 语义令牌为唯一色源，补充品牌点缀（`--accent-brand` / `--accent-text`）、图表色、间距、圆角、阴影契约与字号阶梯；派生色全部 `color-mix(in oklch, …)`。
- 阴影契约：`--shadow-surface: none`；浮层 `--shadow-float-md/lg`，暗色改用黑色混合。
- 视觉规则取自 `2026-08-15-admin-visual-tokens-design.md`（表面无投影、浮层保留）、`docs/frontend/admin-list-filters.md`（紧凑筛选）与 `docs/voice-profile.md`（文案）。
- 数据统计（34 基础组件 / 16 业务模块 / 亮暗 × 中英）来自源项目规范总览页，非虚构。

## 已知限制

- Inter / Manrope 字体文件未在源仓库提供（应用经 Google Fonts 加载），`fonts/` 仅含托管说明与自托管接入方案，见 `fonts/README.md`。
- `build/icons/` 为 React TSX 源文件；运行时可被 lucide-react 组件替代，本包保留实现证据。
