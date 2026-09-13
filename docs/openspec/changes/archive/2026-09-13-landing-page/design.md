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
