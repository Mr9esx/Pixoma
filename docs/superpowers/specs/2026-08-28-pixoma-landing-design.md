---
comet_change: landing-page
role: technical-design
canonical_spec: openspec
---

# Pixoma 落地页技术设计

> 本体是 `comet-change: landing-page` 的深度技术细化；高层目标与方案框架见 `docs/openspec/changes/landing-page/design.md`，不重复需求。

## 上下文

在 `web/landing` 新建独立的静态营销落地页，复刻 `notegen.top/cn` 的信息架构与动效节奏，面向终端用户介绍 Pixoma（随时随地用自己的 ComfyUI 做 AI 艺术创作）。纯前端、不接后端；演示区用静态 mockup。仓库已有 `web/admin`（Vite 8 + React 19 + TS + Tailwind v4 + pnpm）可作为工程习惯与 `motion` 运行时参照。

## 目标 / 非目标

**目标：** 独立可构建、可静态部署；完整信息架构与多语言；reui 组件 + reactbits 动效 + Framer Motion；遵循仓库约定（无截图脚手架、文案走 pixoma-voice）。

**非目标：** 不接后端/登录/真实数据；不改 `web/admin`、Go 后端与既有 specs；不做按语言 SEO 预渲染（后续增强）；不接部署编排。

## 架构总览

```text
index.html ──> src/main.tsx ──> <I18nextProvider>
                                   │
                             <App> <BrowserRouter>
                                   │
                        <LangLayout>(/:lang/*)
                                   │
        ┌────────────┬─────────────┼──────────────┬─────────────┐
     <Header>    <HeroSection>  <Features>     <UseCases>    <CTA + Footer>
                                │
                    <FeatureSection x 5~6>
                                │
                  <BotChatDemo> <CaseDirDemo> <AdminDemo>  (React.lazy)
```

数据流为「单向、只读」：所有内容来自 i18n 词条与 `src/data/*` 静态常量，组件只做展示与本地交互状态（打字推进、切换、滚动），不产生网络请求。

## 目录结构

```text
web/landing/
├── package.json
├── vite.config.ts
├── tsconfig.json / tsconfig.app.json / tsconfig.node.json
├── index.html
├── eslint.config.js
├── .gitignore
└── src/
    ├── main.tsx
    ├── App.tsx
    ├── router/index.tsx          # /:lang/* 路由 + 语言白名单/回退
    ├── i18n/
    │   ├── index.ts              # i18next 初始化 + changeLanguage 驱动
    │   └── locales/{zh-CN,en}.ts # key-based 词条
    ├── components/
    │   ├── motion/               # Reveal、AnimatePresence 封装
    │   ├── reui/                 # vendor 的 reui 组件（标注来源）
    │   ├── reactbits/            # vendor 的 reactbits 效果
    │   ├── sections/             # Hero、Features、UseCases、CTA
    │   ├── demos/                # BotChatDemo、CaseDirDemo、AdminDemo
    │   ├── layout/               # Header、Footer、LangSwitch
    │   └── ui/                   # 通用小件（Button、SectionTitle…）
    ├── constants/site.ts         # 下载/自托管 URL 集中配置
    ├── data/                     # 静态 mockup 数据
    ├── lib/{i18n.ts, motion.ts, utils.ts}
    └── styles/index.css          # Tailwind v4 + 语义色令牌
```

## 关键技术决策

### 1. 应用骨架

独立 Vite 工程，`package.json` 带 `packageManager: pnpm`，脚本 `dev/build/preview/lint/format`。构建用 `tsc -b` + `vite build`，产出 `dist/` 静态文件。复用 `@vitejs/plugin-react` + `@tailwindcss/vite`。

选择独立工程（而非并入 admin）原因：避免与后台路由/登录/依赖耦合，且可单独静态部署。

### 2. 路由与多语言

用 `react-router-dom` v7。路由 `/:lang/*`；语言段到 locale 的映射集中在一处：

```ts
// router/index.tsx
const LANG_MAP: Record<string, string> = { cn: "zh-CN", en: "en" };
const SUPPORTED = new Set(["cn", "en"]);
// 未指定/非法 段 → 回退 "cn"（zh-CN）
```

`i18next` 的 `lng` 由路由段决定；`<LangLayout>` 在路由切换时调用 `i18n.changeLanguage(locale)`，并对 `document.documentElement.lang` 与 `<title>`/描述同步。语言切换器导航到 `/cn`、`/en`，无需整页 reload。

词条为 key-based 扁平资源：`zh-CN.ts`、`en.ts`，组件统一 `useTranslation()` 读取，禁止硬编码文案。新增语言仅需新增资源文件并加入 `SUPPORTED`，不改组件。

### 3. 动效系统

- `<Reveal>`：基于 `motion` `useInView`（`once: true`）+ `variants`（默认 fade + 少量 y 位移）。`useReducedMotion()` 为真时跳过位移动画，仅保留瞬时显示。
- `<AnimatePresence mode="wait">`：用于演示切换、导航菜单展开/收起。
- reactbits 效果（`TextShimmer`、`GradientBackground`、`Particles`…）按需 vendor 进 `components/reactbits/`，只在用到的场景引入；无 reactbits 时用 `motion` 原生实现兜底。
- 统一进度：所有动效仅改 `transform`/`opacity`，避免布局抖动；演示组件 `React.lazy` + `Suspense`，滚动接近视口才挂载（`useInView` 包装或手动 IntersectionObserver）。

### 4. 交互式演示（静态 mockup）

- `BotChatDemo`：本地消息数组，用 state 控制逐条/打字出现，`motion` 控制气泡入场；提供「前进/重放」交互。
- `CaseDirDemo`：左侧列表 + 右侧详情，切换用 `layout` 动画；数据来自 `data/cases.ts`。
- `AdminDemo`：伪后台面板（表格/任务卡片），滚动或 tab 切换动效；数据来自 `data/admin.ts`。
- 所有演示不 fetch、不依赖环境变量，能离线渲染。

### 5. 视觉与文案

- Tailwind v4，`styles/index.css` 定义语义色令牌（CSS 变量，如 `--color-primary` 等），组件只用 `bg-primary/ text-muted-foreground` 这类语义类，不写裸 hex；形状/布局遵循 reui/shadcn 风格（`flex + gap`、内置 variants）。
- zh-CN 文案严格按 `pixoma-voice`（`docs/voice-profile.md`）Use/Avoid 词表与节奏；en 文案镜像调性。文案作为词条，评审时过一遍 `pixoma-design-system` 的验收清单（仅用于文案，不把后台视觉强套到落地页）。

### 6. 响应式

- 断点：mobile-first，栅格在 `md`/`lg` 折叠为单列；Hero 与特性区用 `clamp()` 字号。
- Header：桌面显示导航 + 语言切换；移动端折叠为汉堡菜单（`AnimatePresence` 展开）。
- 图片/演示组件按容器 `aspect-ratio` 缩放，避免横向溢出。

## 测试策略

采用「Vitest + React Testing Library（jsdom）」组件/单元测试 + 构建门禁，不引入截图/快照脚手架。

覆盖点：
- 路由映射与回退：非法/缺失语言段 → zh-CN；`/cn`、`/en` 渲染对应语言。
- i18n：`zh-CN`/`en` 词条 key 对齐完整；切换后显示对应文案。
- `<Reveal>`：`useReducedMotion` 为真时渲染不透明且无 transform 依赖。
- 演示组件：用静态数据渲染、离线可显示；交互（推进/切换）状态更新。
- `site.ts` 常量存在且为可配置字符串。

构建门禁：`pnpm build`（tsc -b + vite build）无 TS 错误，`pnpm lint` + `prettier` 通过。

## 边界条件与错误处理

- 语言段非法：回退默认语言并在导航层级做归一化，避免 404/空白。
- `Suspense`/懒加载未就绪：用一致高度的 skeleton，避免布局跳动。
- 动效被系统禁用：`Reveal` 瞬时显示，演示仍可操作。
- 多语言资源缺 key：开发期通过 `i18next` fallback 与 lint 检查暴露缺词条。
- 无后端：演示数据只读本地，不 catch 网络错误（无网络请求）。

## 风险 / 权衡

- [vendor 组件维护成本] → 只引入用到的组件，目录标注来源版本，便于回同步。
- [复杂动效低端设备掉帧] → 懒加载、transform/opacity 专属、reduced-motion 降级。
- [大 bundle 影响首屏] → 演示组件懒加载 + 按需引入 reactbits。
- [SPA `/cn` `/en` SEO] → 本次以信息与动效为主，后续 `vite-ssg` 预渲染增强。
- [下载/自托管链接未定] → `site.ts` 集中配置，后补不改结构。

## 迁移 / 部署

- 全新应用，无数据迁移。交付后 `pnpm build` 产出静态文件，静态托管侧配置 SPA fallback 到 `index.html`。
- 回滚：仅新增 `web/landing` 目录，移除即回滚；不影响现有功能。

## 边界

与 `comet-change: landing-page` 的 delta specs 一一对应：`landing-page`（结构/响应式/构建）、`landing-i18n`（语言/路由/回退/可扩展）、`landing-interactive-demos`（静态数据/交互动效/scroll-reveal/减弱动态）。
