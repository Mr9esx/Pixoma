# Pixoma Landing

Pixoma 对外静态营销落地页。复刻 `notegen.top/cn` 的结构：Hero → 特性展示区（含交互式 App UI 演示）→ 使用场景 → CTA；支持 `zh-CN` / `en` 多语言。

## 技术栈

- Vite 8 + React 19 + TypeScript + Tailwind CSS v4 + pnpm
- `react-router-dom`（`/:lang/*` 承载语言，`cn→zh-CN`、`en→en`，非法回退 `zh-CN`）
- `i18next` + `react-i18next`（key-based 词条，位于 `src/i18n/locales/`）
- `motion`（Framer Motion）：scroll-reveal、入场动画、`prefers-reduced-motion` 降级
- reui 风格原语（`src/components/reui/`）+ reactbits 式效果（`src/components/reactbits/`）+ 惰性拆分的交互演示

## 运行

```bash
pnpm install
pnpm dev        # http://127.0.0.1:5173
pnpm test       # Vitest
pnpm build      # tsc -b && vite build，产物 dist/
pnpm preview    # 静态预览
```

## 约定

- 组件只用语义类名 / `flex + gap`，不写裸 hex、不手写 `dark:`；颜色令牌在 `src/styles/index.css`。
- zh-CN 文案遵循 `docs/voice-profile.md`（短句、动作起头、无「请/您/前往/进行/完成/感叹号/emoji」）。
- 演示区为静态 mockup（`src/data/*`），零网络请求；懒加载（`React.lazy`）。
- 下载 / 自托管入口集中在 `src/constants/site.ts`。

## 代码审查记录

`review_mode: standard` 下对 `web/landing` 整段改动（base `0906429` → head）做了轻量审查。结果：

- **无 Critical / Important 问题**；测试验证真实行为（非 mock），边界（语言回退、`prefers-reduced-motion` 降级、离线演示）已覆盖。
- **接受的少量偏差**：
  - `components/reui/Button.tsx` 是按 reui 约定重写的原语，非逐字拷贝上游源码（reui 本就是 copy-paste + 定制，此做法符合其哲学）。
  - 工程依赖对齐 `web/admin`（Vite 8 / React 19.2 / vitest 4 / motion 13），而非计划中的旧版本号。
  - `ShinyText` / `GradientBackground` 为 reactbits 风格的简化实现，未引入完整 reactbits 依赖。
