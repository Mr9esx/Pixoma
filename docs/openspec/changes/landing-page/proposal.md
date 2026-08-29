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
