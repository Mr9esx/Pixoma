# Brainstorm Summary

- Change: landing-page
- Date: 2026-08-28

## 确认的技术方案

- 在 `web/landing` 新建独立 Vite 8 + React 19 + TypeScript + Tailwind v4 工程（与 `web/admin` 同源工程习惯），独立静态部署。
- 结构完整复刻 `notegen.top/cn`：Hero → 特性展示区（含交互式 App UI 演示）→ 使用场景 → CTA。
- 路由：`react-router-dom`（`/:lang/*`），语言段 `cn→zh-CN`、`en→en`，未指定/非法回退 `zh-CN`。
- 多语言：`i18next + react-i18next`，key-based 词条（`zh-CN` / `en`），切换语言即导航到 `/cn`、`/en`，可分享。
- 动效：`motion/react`（Framer Motion）：`<Reveal>`（`useInView` once + fade/slide）+ `AnimatePresence`；所有动画尊重 `prefers-reduced-motion`。
- 组件与效果：reui 以 copy-paste/vendor 引入 `src/components/reui/`；reactbits 按需 vendor 到 `src/components/reactbits/`（只取用到的效果）。
- 交互式演示：`BotChatDemo`（对话打字/推进）、`CaseDirDemo`（案例切换 layout 动画）、`AdminDemo`（伪后台面板滚动/切换），数据来自 `src/data/*.ts` 静态常量，零网络请求，`React.lazy` 懒加载。
- 视觉与文案：Tailwind v4 语义令牌、不写裸色值；zh-CN 文案遵循 `pixoma-voice`（`docs/voice-profile.md`），en 对齐同调。
- 可配 URL：`src/constants/site.ts` 集中下载/自托管链接，后补不改结构。

## 关键取舍与风险

- vendor 组件（reui/reactbits）复制代码维护成本 → 只引入用到的组件并标注来源。
- 大量动效低端设备掉帧 → 懒加载演示组件、动画只用 transform/opacity、减弱动态降级。
- SPA `/cn` `/en` 逐语言 SEO 不理想 → 本次以信息 + 动效复刻为主，后续可加按语言静态预渲染（`vite-ssg`）。
- 下载/自托管链接待定 → `site.ts` 集中可配，后补不改结构。

## 测试策略

- Vitest + React Testing Library（jsdom）组件/单元测试：语言回退与切换、词条完整性、`Reveal` 减弱动态降级、演示组件用静态数据渲染、`site.ts` 常量。
- 构建门禁：`tsc -b` + `vite build` + `eslint` + `prettier` 全绿。
- 不引入截图/快照脚手架（AGENTS.md）；动画按行为/存在性断言，不比对像素。

## Spec Patch

- 无（现有 3 份 delta spec 已覆盖全部需求，无需回写）。
