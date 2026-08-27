# preview — 审查卡片清单

七个聚焦卡片 + 一个索引，全部引用 `../colors_and_type.css` 语义令牌，品牌卡片直接加载 `../assets/` 真实源文件。

| 卡片 | 文件 | 审查重点 |
|---|---|---|
| 索引 | `index.html` | 总览入口，含各卡片缩略预览 |
| 颜色 | `colors-tokens.html` | 语义令牌亮暗对照、accent 预算、图表色 |
| 排版 | `typography-specimens.html` | 字号阶梯、Inter / Manrope / Mono、tabular |
| 间距 | `spacing-tokens.html` | 4px 刻度、区块节奏、侧栏与命中区 |
| 圆角与阴影 | `radius-shadows.html` | 圆角刻度、表面无投影 vs 浮层阴影 |
| 组件 | `components-library.html` | 按钮 / 徽章 / 表单 / FilterSegment / 表格 / 状态 |
| 品牌资产 | `brand-assets.html` | 真实加载 `assets/logo.png`、favicon 系列、图标清单 |
| 应用界面 | `ui-surfaces.html` | 应用壳、Dashboard、Master–Detail、向导实样 |

建议审查顺序：先看 `ui-surfaces.html`（成品观感），再看 `colors-tokens.html` 与 `components-library.html`（规范细节）。
