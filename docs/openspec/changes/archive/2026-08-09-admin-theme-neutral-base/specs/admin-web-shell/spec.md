## ADDED Requirements

### Requirement: 中性基色主题
系统 MUST 以 shadcn **neutral** 作为管理控制台亮色与暗色共用的基色（`baseColor`），并通过 CSS 语义变量驱动背景、前景、卡片、主色、muted、边框、侧栏等相关 token。暗色模式 MUST 不以蓝灰（slate）为主调；亮色与暗色 MUST 使用同一套基色，避免日夜切换出现「一套灰、一套蓝」的不一致。

#### Scenario: 暗色不发蓝
- **WHEN** 用户将主题切换为暗色（或系统偏好解析为暗色）并打开任意壳页
- **THEN** 页面主背景与主要表面色呈现中性灰，不以明显蓝灰为主调

#### Scenario: 亮暗共用同一基色
- **WHEN** 用户在亮色与暗色之间切换
- **THEN** 两侧均基于 neutral 语义 token，且主题切换能力（含偏好持久化）仍可用

#### Scenario: 基座配置与 token 对齐
- **WHEN** 开发者查看 `web/admin` 的 shadcn 基座配置与主题 CSS
- **THEN** `baseColor` 为 `neutral`，且亮/暗 CSS 变量与该基色一致

#### Scenario: 主表面中性不影响 chart 色相
- **WHEN** 主题已切换为 neutral，且页面使用 chart 语义色（`chart-1`…`chart-5`）
- **THEN** 主背景与主要表面仍为中性灰；chart 色板 MAY 保留色相（不要求无色）
