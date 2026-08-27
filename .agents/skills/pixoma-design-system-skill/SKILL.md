---
name: pixoma-design-system
description: 使用 Pixoma 设计系统生成后台管理界面。命中「Pixoma」「后台」「Admin」「管理控制台」等上下文，或要求沿用 Pixoma 视觉语言时使用。核心：中性灰语义令牌 + 单一陶土色点缀 + 表面无投影 + 动作起头中文文案；组件复用 shadcn/ui，筛选走 FilterSegment。
user-invocable: true
---

# Pixoma Design System — Skill

## What's inside

本包包含可复用的设计系统资产：语义令牌（tokens，`colors_and_type.css`）、品牌资产（assets/）、字体说明（fonts/）、聚焦审查卡片（preview/）、应用界面套件（ui_kits/app）与源组件实现（build/source-examples/）。

**Source context:** 来源是本地仓库 `/Users/mr9esx/Documents/Pixoma` 的 `web/admin`（source evidence，见 `context/source-context.md`）。令牌原文 `build/theme.css`；视觉令牌变更史、筛选规范与文案规范均在 `context/provenance.md` 中映射到具体路径。

**When to use this skill:** 生成或修改 Pixoma 后台页面、原型（prototypes）、界面（interfaces）或相关设计交付物时使用。若任务与 Pixoma 无关，跳过。本 skill 适用于生产级（production）后台界面与设计系统衍生页面的构建（build）。

**How to use:**

1. 先读 `README.md` 与 `DESIGN.md`，锁定视觉原则与验收清单。
2. 页面引入 `colors_and_type.css`，只使用其中的语义令牌；禁止新增裸 hex，派生色一律 `color-mix(in oklch, …)`。
3. 组件形状照 `ui_kits/app/components.html` 与 `ui_kits/app/components/*.css`；布局照 `ui_kits/app/surfaces.html`。
4. 文案照 `build/source-examples/voice-profile.md`；交付前过 `DESIGN.md` 第 11 节 10 条验收，并检查 `preview/` 卡片是否仍可审查。

**Design-system highlights:**

- 颜色（colors）：neutral 纯灰基色 + 单一陶土橙 accent；亮暗成对，禁止 slate。
- 排版（typography）：Inter 同族分层，数字等宽 tabular。
- 间距（spacing）与圆角（radius）：4px 基准刻度，`sm 6 / md 8 / lg 10 / xl 14`。
- 阴影（shadows）：表面无投影，浮层才允许 shadow。
- 布局（layout）：应用壳 + Master–Detail + 紧凑筛选；交互（interaction）状态齐全，对比度只升不降。

## 硬性约束

- 表面（卡片 / 按钮 / 表单控件）无投影；只有 dialog / dropdown / popover / sheet / select 弹层可保留 shadow。
- accent（陶土橙）每屏至多两次；chart-1…5 只进图表。
- 同一动作只保留一个主 CTA，其余 secondary / ghost / link。
- 筛选只占「单一搜索框 + 一行 FilterSegment」预算；<5 项均分整行，≥5 项横向滑动。
- 暗色 `--background` 必须为 `oklch(0.145 0 0)`，不得引入 slate 冷蓝。
- 数字用等宽 tabular；920px 断点重排，禁止横向滚动；触屏命中区 ≥ 44px。
- 所有用户可见文案用简体中文（品牌名与技术标识除外）。

## 反模式（立即修正）

- 卡片加 `shadow-sm`、页面出现裸 hex、accent 当按钮底色、筛选堆 Label + 全宽 Select、同屏双主 CTA。
- 文案出现「请 / 您 / 温馨提示 / 前往 / 进行 / 完成」、感叹号、emoji。
- hover 后文字变浅、缺失聚焦环、缺失空态 / 错误态 / 加载态。

## 校验工具

```bash
"$OD_NODE_BIN" "$OD_BIN" tools connectors design-system-package-audit --path . --fail-on-warnings
```
