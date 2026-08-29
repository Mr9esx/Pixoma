# Comet Design Handoff

- Change: landing-page
- Phase: design
- Mode: compact
- Context hash: fc106fb758e30dddb11329d31e6b64564a67948d6087cedabffd5e1545517dd1

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/landing-page/proposal.md

- Source: docs/openspec/changes/landing-page/proposal.md
- Lines: 1-31
- SHA256: 87060714caf30b78dcf0a4972484a21f14d34e72968029950b1e5c9a342705f0

```md
## Why

Pixoma 当前只有面向内部运营的管理后台（`web/admin`）与 Go 控制面，缺少一个面向终端用户的对外产品落地页。需要一个能清楚传递「随时随地用自己的 ComfyUI 做 AI 艺术创作」这一价值主张、且具备现代动效与多语言能力的官网，来承接下载/自托管和后续曝光转化。

## What Changes

- 在 `web/landing` 新建一个独立的静态落地页应用（Vite + React 19 + TypeScript + Tailwind v4 + pnpm），与 `web/admin` 保持相同的工程习惯。
- 页面结构完整复刻 `notegen.top/cn`：首屏 Hero → 多个特性展示区（含交互式 App UI 演示动效）→ 使用场景 → CTA。
- 组件优先复用/引入 reui（基于 shadcn 模式的 copy-paste / vendor 组件）；动画使用 Framer Motion（`motion` 包）；装饰性/交互动效使用 reactbits。
- 多语言支持：先提供 `zh-CN` 与 `en` 两种语言，按 `/cn`、`/en` 路由切换，i18n 层设计为可扩展（后续可加语言）。
- 落地页为纯前端静态营销页，不接后端、不登录、不拉取真实数据；App UI 演示使用静态 mockup 数据渲染。
- 遵循仓库 AGENTS.md 约定：不新增任何截图类测试脚手架，不使用截图把视觉发给模型。

## Capabilities

### New Capabilities

- `landing-page`: Pixoma 对外落地页的整体结构、内容区块（Hero、特性区、使用场景、CTA）与响应式渲染。
- `landing-i18n`: 落地页多语言能力，`zh-CN` / `en` 内容组织与路由切换，词条结构可扩展。
- `landing-interactive-demos`: 落地页中交互式 App UI 演示（Telegram Bot 对话、Case 目录、管理后台等）与 Framer Motion / reactbits 动效。

### Modified Capabilities

- 无（本次不改变任何既有 spec 的能力要求）。

## Impact

- 新增应用：`web/landing/`（Vite 工程、`src/`、`package.json`、`vite.config.ts`、Tailwind 配置）。
- 新增依赖：`reui`、`reactbits`、`motion`（Framer Motion）、i18n 相关（如 `react-i18next`），落地页用 Tailwind v4。
- 不涉及：`web/admin`、Go 后端、现有 OpenSpec specs、部署编排。
- 文档/静态产物：构建后可静态部署；本次只保证 `web/landing` 可独立构建与预览。

```

## docs/openspec/changes/landing-page/design.md

- Source: docs/openspec/changes/landing-page/design.md
- Lines: 1-73
- SHA256: f71ec563d2184e19a2e3d8667fb372f4fb864302aca11d399c585134ae37c0f3

```md
## Context

需要在 `web/landing` 新建一个独立的静态营销落地页，面向终端用户传递 Pixoma 价值主张，并完整复刻 `notegen.top/cn` 的信息结构与动效节奏。仓库已有 `web/admin`（Vite 8 + React 19 + TypeScript + Tailwind v4 + pnpm，shadcn 组件）可作为工程习惯参考；`web/admin` 已使用 `motion`（Framer Motion 的继任包），因此在落地页可沿用同一动画运行时。

落地页是纯前端静态页：不接后端、不登录、不拉取真实数据；演示区使用静态 mockup 数据。

## Goals / Non-Goals

**Goals:**
- 产出独立、可构建、可静态部署的 `web/landing` 应用。
- 完整复刻参考页结构：Hero → 特性展示区（含交互式 App UI 演示动效）→ 使用场景 → CTA。
- 多语言（`zh-CN` / `en`），按 `/cn`、`/en` 路由组织，i18n 词条可扩展。
- 组件优先引入 reui，动效用 reactbits，动画用 Framer Motion；遵循仓库 AGENTS.md（无截图脚手架）。

**Non-Goals:**
- 不实现真实后端对接、登录、数据拉取。
- 不动 `web/admin`、Go 后端、既有 OpenSpec specs。
- 本次不接部署编排；只保证 `web/landing` 能独立构建、预览、静态托管。

## Decisions

### 1. 应用骨架：Vite + React 19 + TS + Tailwind v4，独立于 `web/admin`

落地页作为独立 Vite 工程放在 `web/landing`，复用与 admin 一致的 `@vitejs/plugin-react`、`@tailwindcss/vite`、`tailwindcss`、`tsc -b` 构建。独立工程避免与 admin 的依赖/路由/状态耦合，也便于单独静态部署。

- 备选：并入 `web/admin` 作为新路由 → 会带上后台依赖与登录隔离，且与「对外营销页」定位冲突，不采用。

### 2. 路由：`react-router-dom`，以 URL 路径段承载语言

使用 `react-router-dom`（v7），路由形态为 `/:lang/*`，`lang` 限定在 `zh-CN` / `en`；未指定或非法时回退 `zh-CN`。落地页不需要 TanStack Router 的加载/校验能力，选用更轻、生态通用的 react-router。

- 备选：`@tanstack/react-router`（与 admin 一致）→ 能力过剩、包更大，不采用。

### 3. 多语言：`react-i18next` + `i18next`，key-based 词条目录

面向用户的文案全部走 `i18next` resources（`zh-CN`、`en`）。语言从路由段读取，切换语言即导航到对应 `/cn`、`/en` 路径；组件不写死文案，新增语言只需补词条。

- 备选：自研 `useCopy(locale)` 键值查找 → 缺少插值/复数等能力，扩展性差，不采用。

### 4. 组件：reui 以 copy-paste / vendor 方式引入

reui 是 shadcn 风格的「复制组件进项目」库，不需要作为运行时 npm 依赖。按需把用到的组件拷贝到 `web/landing/src/components/`，并遵循语义类名、内置 variants、`flex + gap` 约定。

- 备选：引入 `@keenthemes/ktui` npm 包 → 引入面大且定制成本高，不采用。

### 5. 动效：`motion`（Framer Motion）+ 按需 vendor 的 reactbits 效果

动画统一用 `motion/react`（与 admin 同源），用于 scroll-reveal、入场/离场、数字滚动等；reactbits 按其文档以组件形式拷贝进项目，用于文本动画、背景特效等装饰性效果。统一封装 `motion` 的 `useInView` / `AnimatePresence` 与 `prefers-reduced-motion` 处理（通过 `useReducedMotion`），保证弱化动态时降级。

- 备选：直接用原生 CSS keyframes 或 GSAP → 与「动画库 Framer Motion」的明确要求不符，不采用。

### 6. 演示区：静态 mockup 组件，本地数据，不接后端

把 Telegram Bot 对话、Case 目录、管理后台等工作台界面做成可交互的静态演示组件，数据来自本地 TS 常量，不发起任何网络请求。交互（输入框、切换、滚动）由 React 状态 + motion 驱动。

### 7. 构建与静态托管：SPA + 客户端路由，静态托管需回退到入口

构建输出为静态文件，直接可托管。因使用客户端路由承载 `/cn`、`/en`，静态托管侧需要 SPA fallback 到 `index.html`。SEO 上的逐语言静态预渲染（按语言拆分 HTML）作为后续增强，不在本次实现。

## Risks / Trade-offs

- [SPA 客户端路由导致逐语言 SEO 不理想] → Mitigation：本次以信息与动效复刻为主；如后续需要 SEO，再引入按语言静态预渲染（`vite-ssg` / 每语言 `lang` 前缀构建）。
- [vendor 组件（reui / reactbits）带来复制代码维护成本] → Mitigation：只引入实际使用到的组件，并在组件目录处标注来源；遵循 shadcn/reui 风格以利后续同步。
- [大量动效在低端设备可能掉帧] → Mitigation：统一 `prefers-reduced-motion` 降级；滚动动画尽量用 transform/opacity，减少布局抖动；必要时做惰性加载演示组件。

## Migration Plan

- 本次为全新应用，无数据迁移。交付后通过 `make` 或独立 `pnpm` 脚本运行 `dev`/`build` 验证构建即可。
- 回滚：本次只在 `web/landing` 新增文件，不影响现有功能；回滚即移除该目录。

## Open Questions

- 是否需要为落地页接入真实下载/自托管入口（如 release 下载链接、部署文档锚点）？该信息可后补，不影响本次页面结构。

```

## docs/openspec/changes/landing-page/tasks.md

- Source: docs/openspec/changes/landing-page/tasks.md
- Lines: 1-47
- SHA256: 1a2e8929215df0d3ae150061ccc52863b9716d87240003f97ca5e50ca3b1b454

```md
## 1. 应用脚手架

- [ ] 1.1 新建 `web/landing` 目录与 `package.json`（`type: module`、`pnpm`、`packageManager`），与 admin 工程习惯一致
- [ ] 1.2 配置 `vite.config.ts`（`@vitejs/plugin-react` + `@tailwindcss/vite`）与 `tsconfig*.json`、`index.html`、`src/main.tsx`
- [ ] 1.3 建立 Tailwind v4 基线：`src/index.css` 引入 Tailwind，并定义基础字体/颜色语义令牌（遵循 reui/shadcn 风格，不写裸 hex）
- [ ] 1.4 安装依赖：`react-router-dom`、`motion`、`i18next`、`react-i18next`、`tailwindcss`，以及所需 devDeps（`typescript`、`@types/react*`、`eslint`、`prettier` 等）
- [ ] 1.5 按需 vendor reui 组件到 `src/components/reui/`（标注来源），不重复造轮子

## 2. 路由与多语言

- [ ] 2.1 建立 `react-router-dom` 路由：`/:lang/*`，`lang` 限定 `zh-CN` / `en`，未指定或非法回退 `zh-CN`
- [ ] 2.2 初始化 `i18next` + `react-i18next`，提供 `zh-CN` / `en` 的 key-based 词条 resources
- [ ] 2.3 实现导航/页头的语言切换，切换时导航到对应 `/cn`、`/en` 路径，且无需整页 reload

## 3. 全局布局与动效基座

- [ ] 3.1 实现顶层布局（导航、主内容、页脚）与响应式容器/断点
- [ ] 3.2 封装基于 `motion`（Framer Motion）的 `useInView` scroll-reveal 入场组件，并处理 `prefers-reduced-motion` 降级
- [ ] 3.3 在页头/页脚提供语言切换入口，并接入主题/色彩语义关系

## 4. Hero 区

- [ ] 4.1 实现 Hero 组件：主标语 + 副标语 + 主行动入口（下载/自托管），文案走 i18n
- [ ] 4.2 为 Hero 添加入场动画与 reactbits 装饰性效果（如文本/背景特效），支持减弱动态降级

## 5. 特性展示区（含交互式 App UI 演示）

- [ ] 5.1 实现特性区通用组件（对齐参考页：标题 + 描述 + 演示容器）
- [ ] 5.2 实现 Telegram Bot 对话演示组件（静态 mockup 数据 + 打字/交互动效）
- [ ] 5.3 实现 Case 目录 / 画布演示组件（静态 mockup + 切换/布局动效）
- [ ] 5.4 实现管理后台演示组件（静态 mockup + 滚动/切换动效）
- [ ] 5.5 为各特性区补齐 `zh-CN` / `en` 文案，确保无硬编码文案

## 6. 使用场景区

- [ ] 6.1 实现使用场景卡片区（多场景 + 图示 + i18n 文案）

## 7. CTA 与收尾

- [ ] 7.1 实现 CTA 区（主行动入口，`src/constants/site.ts` 集中可配的下载/自托管 URL）
- [ ] 7.2 完善页脚（导航、版权、语言切换）

## 8. 构建与验证

- [ ] 8.1 `pnpm build`（`tsc -b` + `vite build`）通过且无 TS 错误，产物为静态可托管文件
- [ ] 8.2 `pnpm dev` 或 preview 验证各 section 渲染与动效，覆盖桌面/平板/移动端响应式
- [ ] 8.3 运行 `lint`/`format:check` 与仓库约定对齐，确认无截图类测试脚手架

```

## docs/openspec/changes/landing-page/specs/landing-i18n/spec.md

- Source: docs/openspec/changes/landing-page/specs/landing-i18n/spec.md
- Lines: 1-33
- SHA256: 64ebc31bff4475b6876995c334af572fdf98657fc1763831c5a3e9dabfd8dd46

```md
## Purpose

落地页多语言能力，按语言路由组织面向用户的中英文内容，并让新增语言时只扩展词条而无须改动页面结构。

## ADDED Requirements

### Requirement: 支持 zh-CN 与 en 两种语言
The landing page SHALL provide localized copy for `zh-CN` and `en` locales.

#### Scenario: 切换语言显示对应文案
- **WHEN** 用户在 `zh-CN` 与 `en` 之间切换
- **THEN** 页面上所有面向用户的文案随语言切换变化，且无缺失或错位的硬编码串

### Requirement: 基于路由的语言切换
The locale selection SHALL be reflected in the URL path (e.g. `/cn`, `/en`), so a given locale is shareable and bookmarkable.

#### Scenario: 语言体现在 URL 且可分享
- **WHEN** 用户访问 `/zh-CN` 或 `/en` 对应路径
- **THEN** 页面渲染对应语言内容，且该 URL 可被直接分享打开

### Requirement: 默认语言与未知语言回退
The landing page SHALL default to `zh-CN` when no locale is specified, and SHALL fall back to `zh-CN` when an unsupported or invalid locale is requested.

#### Scenario: 未指定语言时使用中文
- **WHEN** 用户访问不带语言前缀的路径或请求一个不支持的语言
- **THEN** 页面以 `zh-CN` 渲染，而非报错或空白

### Requirement: 词条可扩展
The i18n layer SHALL organize strings as a key-based catalogue so that adding a new language requires only adding new entries, without modifying page components.

#### Scenario: 新增语言只加词条
- **WHEN** 开发者需要新增一种语言
- **THEN** 只需补充对应语言词条，无需改动渲染组件

```

## docs/openspec/changes/landing-page/specs/landing-interactive-demos/spec.md

- Source: docs/openspec/changes/landing-page/specs/landing-interactive-demos/spec.md
- Lines: 1-33
- SHA256: 6e08772e1ce6a5e5b624b2a407da76c2db59c9c1c7de9c4976ba86dca1732b83

```md
## Purpose

落地页中的交互式 App UI 演示与动效，用静态 mockup 数据渲染 Pixoma 工作台界面，并配合 Framer Motion 与 reactbits 提供流畅、非阻塞的浏览体验。

## ADDED Requirements

### Requirement: 演示使用静态 mockup 数据
The interactive app UI demos SHALL render from static mockup data and SHALL NOT depend on a live backend, login, or real data fetch.

#### Scenario: 离线即可渲染演示
- **WHEN** 用户在无后端环境下打开落地页并浏览演示区
- **THEN** 演示区正常渲染，无网络请求或后端依赖，也不出现数据加载错误

### Requirement: 演示区具备交互动效
The demo surfaces SHALL support interactive behaviors (e.g. typing-like bot 对话、可切换 Case、可滚动内容) driven by Framer Motion and/or reactbits effects.

#### Scenario: 交互演示可操作且带动效
- **WHEN** 用户点击或滚动到演示区并触发交互
- **THEN** 演示产生相应动画/反馈，且不导致页面卡顿或布局跳变

### Requirement: Scroll-reveal 入场动画
Sections and demo surfaces SHALL reveal or animate in response to scroll position via a scroll-triggered mechanism, while remaining performant.

#### Scenario: 滚动触发入场动画
- **WHEN** 用户滚动使某个区块进入视口
- **THEN** 该区块触发入场动画（如淡入 / 位移动效），且多次往返滚动不累积冲突

### Requirement: 动效尊重减弱动态偏好
Animation behaviors SHALL respect the user's reduced-motion preference by disabling or simplifying non-critical animations when requested.

#### Scenario: 减弱动态时简化动画
- **WHEN** 用户开启了系统级「减弱动态效果」
- **THEN** 页面仍完整可读可操作，但跳过或弱化装饰性动画

```

## docs/openspec/changes/landing-page/specs/landing-page/spec.md

- Source: docs/openspec/changes/landing-page/specs/landing-page/spec.md
- Lines: 1-40
- SHA256: 002194c4ef16c11f68c9781015459369bb917d411a8e4bff766df7f630fab256

```md
## Purpose

Pixoma 对外产品落地页，通过完整的信息架构与内容区块向终端用户传递「随时随地用自己的 ComfyUI 做 AI 艺术创作」的价值主张，并引导下载/自托管。

## ADDED Requirements

### Requirement: 落地页渲染完整信息架构
The landing page SHALL render the top-level information architecture in the following order: a hero section, multiple feature showcase sections, a use-case section, and a call-to-action (CTA) section.

#### Scenario: 打开落地页看到完整区块
- **WHEN** 用户打开落地页首页
- **THEN** 页面按顺序渲染 Hero、特性展示区、使用场景与 CTA 四个区块，且各区块内容可见

### Requirement: Hero 展示价值主张与主行动入口
The hero section SHALL display a primary headline conveying the product value proposition and at least one primary CTA that links to the download or self-hosting entry point.

#### Scenario: Hero 主标语与 CTA 可见
- **WHEN** 用户浏览首屏
- **THEN** 可见主标语（价值主张）与至少一个主 CTA 按钮，且 CTA 指向下载或自托管入口

### Requirement: 特性展示区包含交互式 App UI 演示
Each feature showcase section SHALL pair its descriptive copy with an interactive app UI demo that renders the corresponding Pixoma workbench surface (e.g. Telegram Bot 对话、Case 目录、管理后台).

#### Scenario: 特性区展示对应界面演示
- **WHEN** 用户滚动到一个特性展示区
- **THEN** 该区展示特性说明，并渲染对应的交互式 App UI 演示，而不只是静态截图

### Requirement: 落地页响应式布局
The landing page SHALL adapt its layout across desktop, tablet, and mobile viewport widths so that all sections remain usable and readable without horizontal overflow.

#### Scenario: 移动端布局可用
- **WHEN** 用户在移动端视口打开落地页
- **THEN** 页面内容自适应换行、无横向溢出，CTA 与导航可操作性正常

### Requirement: 产出可静态部署的构建结果
The landing page SHALL produce a static, deployable build output via its build pipeline, with no TypeScript errors and no runtime dependencies on a backend.

#### Scenario: 构建通过并产出静态文件
- **WHEN** 对 `web/landing` 运行构建命令
- **THEN** 构建成功（无 TS 错误），并产出可直接托管的静态文件

```
