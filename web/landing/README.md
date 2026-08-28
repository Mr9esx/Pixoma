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
