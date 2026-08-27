# Pixoma Design System

从 Pixoma 仓库 `web/admin`（shadcn/ui new-york + Tailwind v4 + oklch）提炼的 OpenDesign 设计系统包。一句话总结：**中性灰语义令牌 + 单一陶土色点缀 + 表面无投影 + 动作起头的中文文案**，面向数据密集管理控制台。

## Product Overview

Pixoma 是 ComfyUI 艺术创作调度平台（admin platform）：Telegram Bot + Case 目录 + 对话 Session + Task 运行时 + 多计算节点。后台管理控制台面向工程师与运维，主要提供（provides）仪表盘、快速配置、Master–Detail 资源页、三步初始化向导与部署命令管理；包括（includes）亮暗两套 neutral 主题与中英双语 i18n，默认中文。设计目标是把高度留给列表、把注意力留给操作。

## 快速开始

1. 读 [DESIGN.md](DESIGN.md) 锁定视觉原则与验收清单。
2. 页面引入 [colors_and_type.css](colors_and_type.css)，只用其中的语义令牌；派生色一律 `color-mix(in oklch, …)`。
3. 组件形状照 [ui_kits/app/components.html](ui_kits/app/components.html) 与 `ui_kits/app/components/*.css`，布局照 [ui_kits/app/surfaces.html](ui_kits/app/surfaces.html)。
4. 文案照 [build/source-examples/voice-profile.md](build/source-examples/voice-profile.md)，禁用词不出现。
5. 交付前过一遍 DESIGN.md 第 11 节 10 条验收。

## Package Contents

```text
.
├── DESIGN.md                  # 设计系统总规范（context / color / typography / spacing / layout / components / motion / voice / anti-patterns）
├── README.md                  # 本文件：包说明与索引
├── SKILL.md                   # Codex 可用 skill：接入与执行规则
├── colors_and_type.css        # 语义令牌 + 字体 + 间距 / 圆角 / 阴影契约
├── brand-spec.md              # 源项目品牌摘要（保留）
├── pixoma-design-tokens.css   # 源项目参考令牌（保留）
├── pixoma-design-system.html  # 源项目规范总览页（保留）
├── context/                   # source-context.md + provenance.md（证据映射）
├── assets/                    # 品牌资产：logo / 后台 logo / favicon 系列（preserved source-backed）
├── build/                     # theme.css 原文 + build/icons/*.tsx + build/source-examples/*
├── fonts/                     # 字体托管说明（Google Fonts，仓库无字体文件）
├── preview/                   # 聚焦审查卡片（清单见 Preview Manifest）
└── ui_kits/app/               # 应用界面套件（index + components + surfaces + components/*.css）
```

## Source Context

来源（source）是本地仓库 `/Users/mr9esx/Documents/Pixoma`（evidence 见 `context/source-context.md` 与 `context/provenance.md`，local folder）。令牌原文 `web/admin/src/styles/theme.css`；视觉令牌变更史 `docs/superpowers/specs/2026-08-15-admin-visual-tokens-design.md`；筛选规范 `docs/frontend/admin-list-filters.md`；文案规范 `docs/voice-profile.md`。`build/` 与 `assets/` 下的文件为原样复制的源证据，勿以本包覆盖仓库。

## Review Workflow

1. 先开 `preview/index.html`（review hub），逐张 inspect 七个聚焦卡片。
2. 再开 `ui_kits/app/surfaces.html` 与 `ui_kits/app/components.html` 检查成品表面与组件。
3. 改动后重跑审计（open gate）：

```bash
"$OD_NODE_BIN" "$OD_BIN" tools connectors design-system-package-audit --path . --fail-on-warnings
```

## Preview Manifest

- preview/index.html
- preview/colors-tokens.html
- preview/typography-specimens.html
- preview/spacing-tokens.html
- preview/radius-shadows.html
- preview/components-library.html
- preview/brand-assets.html
- preview/ui-surfaces.html

建议审查顺序：先 `preview/ui-surfaces.html`（成品观感），再 `preview/colors-tokens.html` 与 `preview/components-library.html`（规范细节）。

## 与源仓库的关系

- `build/theme.css`、`build/source-examples/*`、`build/icons/*`、`assets/*` 是从源仓库原样复制的证据。
- 规范可扩展但不可删除来源条款（neutral 基色、表面无投影、FilterSegment 预算、Voice Profile 禁用词）。
- 变更视觉令牌需同时更新 `colors_and_type.css` 与 `build/theme.css` 的对照注释。
