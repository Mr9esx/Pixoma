# Pixoma 设计系统 — DESIGN.md

> 事实来源：仓库 `web/admin`（shadcn/ui new-york + Tailwind v4 + oklch）。
> 令牌原文：`build/theme.css`；视觉令牌变更史：`docs/superpowers/specs/2026-08-15-admin-visual-tokens-design.md`；
> 筛选规范：`docs/frontend/admin-list-filters.md`；文案规范：`build/source-examples/voice-profile.md`。

## 1. 产品语境（Product Context）与视觉基调

Pixoma 是一套「Telegram Bot + Case 目录 + 对话 Session + Task 运行时 + 多计算节点」的 ComfyUI 调度平台。后台管理控制台面向同组工程师与运维人员：数据密集、操作高频、以列表和向导为主。

视觉基调一句话：**中性灰语义令牌 + 单一陶土色点缀 + 表面无投影 + 动作起头的中文文案**，服务「把高度留给列表、把注意力留给操作」的数据密集管理场景。

三条原则贯穿所有页面：

1. **中性克制** — 纯灰 neutral 基色，亮暗同源。信息靠层级、间距与细线框表达，不用彩色做装饰；图表色相只进图表。
2. **表面扁平** — 卡片、按钮、表单控件用细线框区分层次，不垫投影；只有对话框、下拉、popover 等浮层保留阴影。
3. **列表优先** — Master–Detail 左栏把高度留给列表；筛选只占「搜索 + 一行枚举」的预算，不堆 Label 与全宽控件。

## 2. 视觉基础（Visual Foundations）

### 2.1 取色规则

- 只用 `colors_and_type.css` / `build/theme.css` 的语义令牌取色；禁止新增裸 hex 色值。
- 派生色一律 `color-mix(in oklch, …)`；亮暗成对出现。
- accent（陶土橙）每屏至多出现两次；作小号文字与点缀时改用深档 `--accent-text`，保证 4.5:1 对比。
- chart-1 … chart-5 保留色相，只用于 recharts 数据编码，不进界面装饰。

### 2.2 基色

中性纯灰，禁止 slate/蓝灰。暗色 `--background` 固定 `oklch(0.145 0 0)`，源码不得出现 `slate-*`（有契约测试锁定）。

## 3. 颜色（Color）

| 令牌 | 用途 | 亮色 | 暗色 |
|---|---|---|---|
| `--background` | 页面背景 | `oklch(1 0 0)` | `oklch(0.145 0 0)` |
| `--card / --popover` | 卡片与弹层表面 | `oklch(1 0 0)` | `oklch(0.205 0 0)` |
| `--foreground` | 主文本 | `oklch(0.145 0 0)` | `oklch(0.985 0 0)` |
| `--muted-foreground` | 次要文本、说明 | `oklch(0.556 0 0)` | `oklch(0.708 0 0)` |
| `--border / --input` | 细线边框、输入框描边 | `oklch(0.922 0 0)` | `oklch(1 0 0 / 10%)` |
| `--primary` | 主按钮（黑 / 反白） | `oklch(0.205 0 0)` | `oklch(0.922 0 0)` |
| `--primary-foreground` | 主按钮文字 | `oklch(0.985 0 0)` | `oklch(0.205 0 0)` |
| `--secondary` | 次级表面 | `oklch(0.97 0 0)` | `oklch(0.269 0 0)` |
| `--muted` | 次级表面底色 | `oklch(0.97 0 0)` | `oklch(0.269 0 0)` |
| `--accent` | 悬停 / 选中底色 | `oklch(0.97 0 0)` | `oklch(0.269 0 0)` |
| `--accent-brand` | 陶土橙点缀（快速配置 CTA） | `oklch(0.646 0.222 41.116)` | 同左 |
| `--accent-text` | 陶土橙深档（小号文字 / 点缀） | `oklch(0.55 0.2 40.5)` | `oklch(0.78 0.16 40)` |
| `--accent-brand-foreground` | 陶土 CTA 按钮文字 | `oklch(0.985 0 0)` | 同左 |
| `--destructive` | 危险操作 | `oklch(0.577 0.245 27.325)` | `oklch(0.704 0.191 22.216)` |
| `--destructive-foreground` | 危险按钮文字 | `oklch(0.985 0 0)` | 同左 |
| `--ring` | 聚焦环 | `oklch(0.708 0 0)` | `oklch(0.556 0 0)` |

图表语义色（仅用于图表）：亮色 `chart-1 … 5` = 陶土橙 / 青绿 / 深蓝 / 黄绿 / 琥珀；暗色各成一套，见 `colors_and_type.css`。

## 4. 字体（Typography）

- Display / Body：`'Inter', system-ui, sans-serif` — 标题与正文同族，靠字重（700/600）、字号与字距分层；可经字体开关换 `Manrope` 或系统栈。
- Mono：`ui-monospace, 'SF Mono', Menlo, monospace` — eyebrow、数字、代码。

字号阶梯（设计系统总览页实测值）：

| 层级 | 字号 | 字重 | 行高 |
|---|---|---|---|
| H1 | `clamp(36px, 4.5vw, 48px)` | 700 | 1.3 |
| H2 | `clamp(26px, 3vw, 32px)` | 700 | 1.32 |
| H3 | 20px | 600 | 1.4 |
| Lead | 17px | 400 | 1.75 |
| Body | 15px | 400 | 1.6 |
| Meta / Mono | 12px | 400 | 1.45 |
| Micro / eyebrow | 11px | 600 | 1.4 |

规则：

- 标题禁止换字体族；`--font-heading` 映射为系统无衬线栈，对话框标题用它。
- 数字一律 `tabular-nums`（`--font-mono` + `font-variant-numeric`）。
- 表格数字列等宽右对齐；统计口径标注来源。
- 页面标题最小 36px（1920×1080 幻灯片场景），正文最小 15px。

## 5. 间距与圆角（Spacing & Radius）

- 间距基准 4px：`4 / 8 / 12 / 16 / 24 / 32 / 48 / 64`，对应 `--space-1 … 8`；区块节奏 `--gap-md 16 / --gap-lg 24 / --gap-xl 40 / --gap-2xl 72`。
- 圆角：`sm 6 · md 8 · lg 10 · xl 14 px`，基准 `--radius: 0.625rem`。
- 筛选区块用 `space-y-2`、`px-4 py-3` 即可，避免再套多层 `space-y-3` + 每字段 `space-y-1`。
- 触屏命中区目标 ≥ 44px，不足时补 padding / hit-area。

## 6. 布局与信息架构（Layout & IA）

- 应用壳：侧栏展开 16rem / 图标 3rem / 移动 18rem；variant 支持 `inset · sidebar · floating`，collapsible 支持 `offcanvas · icon · none`，默认 `inset + icon`。语言与主题切换放侧栏底栏。
- 信息架构四组，菜单顺序固定：`01 仪表盘 · 快速配置` → `02 配置管理（工作流 / 渠道 / 调度通道 / 计算节点）` → `03 运营管理（任务 / 会话 / 用户）` → `04 系统管理（设置）`。
- Master–Detail 资源页（Cases / Tasks / Users / Sessions）：左列表、右详情或表单；未选中时右栏为空态；窄屏降级为顺序整页。
- 紧凑筛选：可搜索字段合并为单一搜索框（无 Label，placeholder + aria-label）；封闭枚举用 `FilterSegment`（少于 5 项均分整行，≥5 项横向滑动、尽头箭头淡出）。细则见 `docs/frontend/admin-list-filters.md`。
- 向导流程：初始化向导按「密码 → 数据库 → 对象存储」推进；快速配置为三步向导；每步单一主 CTA，上一步为 secondary。
- Dashboard 分区：「实时状态 / 任务效能 / 业务分析」；实时区不随范围变化，效能与业务区跟随顶部时间范围统一刷新。
- 容器：内容最大 1120px、gutter 28px；920px 断点重排，禁止横向滚动。

## 7. 组件（Components）

复用 shadcn/ui（`components/ui`），不重复造轮子。P0 = 硬性，P1 = 应做。

| 组件 | 规范 | 级别 |
|---|---|---|
| Button | 六种变体：default / destructive / outline / secondary / ghost / link；默认高 36px、圆角 md、聚焦环 3px `ring/50`；同一动作只保留一个主按钮 | P0 |
| Badge | 状态与计数标签：default / secondary / outline / destructive，圆角 md；不用于堆叠装饰 | P1 |
| Card | 表面一律 border 分层、无投影；dialog / alert-dialog / dropdown / popover / sheet / select 弹层保留 `shadow-md` 以上 | P0 |
| Form | react-hook-form + zod 校验；Label 外显、desc 与 label 去重，错误行内提示并定位字段 | P0 |
| Data Table | TanStack Table：hairline 分隔、行 hover 用 `fg-soft`，数字列等宽右对齐；列头排序与分页控件统一走 `data-table` 目录 | P1 |
| Filter Segment | 横向 Segment：<5 项均分整行，≥5 项在边框内滚动并淡出箭头；无可见 Label，用 `aria-label` / `role="group"` | P0 |
| Empty / Error | 每个列表与详情页必须提供空态、错误态与加载骨架；失败直接说「X 失败」 | P0 |
| Chart | recharts + chart-1…5 语义色，数据必须填充编码；单卡片失败隔离 | P1 |

交互状态契约：

- hover / focus / active / disabled 状态齐全，对比度只升不降；文字禁用灰化仅限 disabled。
- hover 移动背景 OKLch L 通道 ±0.06–0.12，或改 border / shadow / 位置；前景色不变浅。
- 聚焦环：`outline: 2px solid var(--ring); outline-offset: 2px`（亮暗两套 ring）。

## 8. 动效（Motion & Interaction）

- 表面控件过渡 ≤ 150ms 的 color / background / border；浮层出现 150–200ms，配合 `shadow-float-md/lg`。
- 骨架屏脉冲 1.6s 透明度动画；`prefers-reduced-motion: reduce` 时全部动画与过渡压缩到 0.01ms。
- 不引入装饰性动效；动画只服务于状态反馈（hover、focus、加载、浮层出现）。

## 9. 文案语气（Voice & Brand）

完整规范见 `build/source-examples/voice-profile.md`，摘要：

- **Peer 口吻**：默认对面会 grep、会读错误、会用 zsh。不用「请 / 您 / 小伙伴」；「用户」只作实体名词。
- **动作起头**：按钮与动作是动词原形：「保存」「下一步」「勾 Mock」；段落 1–3 句，结尾落到可执行动作。
- **三件套格式**：关键概念 **加粗**，产品 / 功能名「」括起，命令 / 路径 / 字段 `` `code` ``。
- **去重原则**：desc 与 placeholder 出现前先扫同一区域 title / label / 按钮，已写过的信息不重复写。
- 二分（本机 / 远程、登录 / 未登录）用对仗；列表用 `:` 起头 + 短项。

禁用词：请 / 烦请 / 前往 / 进行 / 完成 / 实施 / 温馨提示 / 感叹号 / emoji / 卖萌语气词 / 把「用户」当称呼。

## 10. 反模式（Anti-patterns）

- slate/蓝灰基色、任何裸 hex 色值、非 color-mix 派生色。
- 卡片 / 按钮 / 表单控件加 `shadow-sm`（表面投影）；全局 `* { box-shadow: none }`。
- accent 每屏超过两次，或把 chart 色相搬进按钮 / 徽章装饰。
- 筛选区堆「Label + 全宽控件」三层以上；状态枚举用整行 Select + Label。
- 同一动作同屏多个主按钮；每步向导出现两个主 CTA。
- 「请 / 您 / 温馨提示」等文案、感叹号、emoji、把 4 句话塞进一段。
- 数字不用等宽、表格数字列左对齐、统计口径不标来源。
- 缺少 hover / focus / active / disabled 状态，或 hover 后文字变浅。
- 920px 以下横向滚动；触屏命中区小于 44px。

## 11. 验收清单（新页面 10 条）

1. 取色只用语义令牌与 color-mix，不新增裸 hex。
2. 组件优先复用 `components/ui` 与注册表。
3. 表面无投影，浮层才允许 shadow；亮暗成对出现。
4. 筛选走「单一搜索 + FilterSegment」，不堆 Label 与全宽控件。
5. 同一动作只保留一个主 CTA，其余 secondary / ghost / link。
6. 文案走 i18n zh / en，默认中文，语气照 Voice Profile，禁用词不出现。
7. 数字用等宽 tabular，表格数字列右对齐；统计口径标注来源。
8. hover / focus / active / disabled 状态齐全，对比度只升不降，聚焦环可见。
9. 空态、错误态、加载态齐全；失败直接说失败。
10. 920px 断点重排，禁止横向滚动；触屏命中区 ≥ 44px。
