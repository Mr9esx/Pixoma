# ui_kits/app — Pixoma 应用界面套件

从 Pixoma `web/admin`（shadcn/ui new-york + Tailwind v4）提炼的可复用界面套件。所有页面引用 `../../colors_and_type.css` 语义令牌，形状照源项目组件。

## Structure

| 文件 | 内容 |
|---|---|
| `index.html` | 套件索引：壳 + 统计卡 + 引用方式说明 |
| `components.html` | 组件库：按钮六变体与状态、徽章、表单、FilterSegment、数据表、状态与浮层 dialog |
| `surfaces.html` | 应用表面：Dashboard（统计 + 图表 + 任务表）、Master–Detail 紧凑筛选、三步向导 |
| `components/button.css` | 按钮六变体与 hover / focus / disabled 状态 |
| `components/filter-segment.css` | 枚举筛选：<5 均分、≥5 横向滚动、箭头淡出 |
| `components/data-table.css` | 数据表：hairline 分隔、行 hover、数字列右对齐 |

## Usage

新页面复制任意文件骨架，把 `<link rel="stylesheet" href="../../colors_and_type.css">` 换成项目内相对路径，并视需要引入 `components/*.css`；只使用语义令牌，禁止裸 hex。每个表面都是一次组件组合：入口 `index.html` 相当于 App shell，侧栏相当于 Sidebar，卡片与预览区相当于 PreviewCard 的组合链。

## Source

基于（based on）源仓库 `web/admin` 的 source 证据：组件对应 `src/components/ui/*` 与 `src/components/filters/filter-segment.tsx`；表面对应 `src/features/dashboard`、`src/features/cases`（list-panel）与初始化向导。layout / colors / typography / tokens 的完整规范见 `../../DESIGN.md` 与 `../../colors_and_type.css`；实现源码证据见 `../../build/source-examples/`。
