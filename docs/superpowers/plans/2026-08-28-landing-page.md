---
change: landing-page
design-doc: docs/superpowers/specs/2026-08-28-pixoma-landing-design.md
base-ref: 090642943372d1b457257b58b736c174700e0c46
---

# Pixoma 落地页 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 在 `web/landing` 新建 Pixoma 对外静态营销落地页，复刻 `notegen.top/cn` 的信息架构与动效节奏，支持 `zh-CN` / `en` 多语言，用 reui 组件 + reactbits 效果 + Framer Motion。

**Architecture:** 独立 Vite + React 19 + TypeScript + Tailwind v4 + pnpm 工程。路由用 `react-router-dom` `/:lang/*`（`cn→zh-CN`、`en→en`，非法回退 `zh-CN`），i18n 用 `i18next` + `react-i18next`。动效统一 `motion/react`（`useInView` scroll-reveal + `AnimatePresence`），reactbits 按需 vendor。交互演示（Bot 对话 / Case 目录 / 管理后台）用 `src/data` 静态 mockup + `React.lazy`。

**Tech Stack:** Vite 8、React 19、TypeScript、Tailwind CSS v4、pnpm 10、react-router-dom v7、motion（Framer Motion）、i18next、react-i18next、reui（shadcn 兼容 copy-paste）、reactbits、Vitest + React Testing Library。

## Global Constraints

- 目录：`web/landing/`，独立工程，`package.json` 带 `packageManager: pnpm@<版本>`，`"type": "module"`。
- 构建：`"build": "tsc -b && vite build"`，`"dev": "vite"`，`"preview": "vite preview"`，`"lint": "eslint ."`，`"format:check": "prettier --check ."`，`"format": "prettier --write ."`，`"test": "vitest run"`。
- Tailwind v4：`src/styles/index.css` 用 `@import 'tailwindcss'`，语义色令牌走 CSS 变量（`--color-*`）；禁止裸 hex、禁止手写 `dark:` 类名；布局用 `flex + gap`。
- 路径别名：`@` → `./src`（`vite.config.ts` `resolve.alias` + `tsconfig` `paths`）。
- 多语言：词条文件 `src/i18n/locales/zh-CN.ts` 与 `en.ts`（key-based 扁平结构）；组件禁止硬编码文案，统一 `useTranslation()`。
- 路由语言段：`cn` / `en`，白名单集中定义；未指定或非法回退 `cn`（`zh-CN`）。
- 文案：zh-CN 严格遵循 `docs/voice-profile.md`（`pixoma-voice`）；Use/Avoid 词表 + 短句节奏；不写 "请 / 您 / 前往 / 进行 / 温馨提示 / emoji / 感叹号"，使用「」括产品名。
- 动效：只改 `transform` / `opacity`；`prefers-reduced-motion` 时装饰动画降级为瞬时显示，演示仍可操作。
- 演示组件：不 fetch、不依赖环境变量，纯本地静态 mockup。
- 测试：Vitest + React Testing Library（jsdom）；**禁止**新增截图/快照类脚手架（`_tmp-*.test.tsx`、`_shot*.mjs`、`*.png` 快照、`vitest.*.browser/config`）。
- download / self-host URL 集中在 `src/constants/site.ts`，可由后续替换，不改结构。
- 不接后端；不动 `web/admin`、Go 后端、既有 docs 规格。

---

### Task 1: 应用脚手架与工程配置

**Files:**
- Create: `web/landing/package.json`
- Create: `web/landing/vite.config.ts`
- Create: `web/landing/tsconfig.json`
- Create: `web/landing/tsconfig.app.json`
- Create: `web/landing/tsconfig.node.json`
- Create: `web/landing/index.html`
- Create: `web/landing/eslint.config.js`
- Create: `web/landing/.gitignore`
- Create: `web/landing/src/main.tsx`
- Create: `web/landing/src/App.tsx`
- Create: `web/landing/src/styles/index.css`
- Create: `web/landing/src/lib/utils.ts`
- Create: `web/landing/src/test/setup.ts`
- Create: `web/landing/vitest.config.ts`

**Interfaces:**
- Consumes: 无（工程起点）。
- Produces: `@` 路径别名、`src/App.tsx` 导出 `<App/>`、`src/styles/index.css` 全局样式入口、`src/lib/utils.ts` 导出 `cn(...)` 工具函数、`vitest` jsdom 环境。

- [x] **Step 1: 新建 package.json**

```json
{
  "name": "pixoma-landing",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "packageManager": "pnpm@10.12.4",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "lint": "eslint .",
    "format": "prettier --write .",
    "format:check": "prettier --check .",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "dependencies": {
    "i18next": "^25.0.0",
    "motion": "^12.0.0",
    "react": "^19.0.0",
    "react-dom": "^19.0.0",
    "react-i18next": "^15.0.0",
    "react-router-dom": "^7.0.0"
  },
  "devDependencies": {
    "@eslint/js": "^9.0.0",
    "@tailwindcss/vite": "^4.0.0",
    "@testing-library/jest-dom": "^6.6.0",
    "@testing-library/react": "^16.1.0",
    "@types/node": "^22.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "@vitejs/plugin-react": "^4.3.0",
    "eslint": "^9.0.0",
    "eslint-plugin-react-hooks": "^5.0.0",
    "eslint-plugin-react-refresh": "^0.4.0",
    "globals": "^15.0.0",
    "jsdom": "^25.0.0",
    "prettier": "^3.3.0",
    "tailwindcss": "^4.0.0",
    "typescript": "^5.7.0",
    "typescript-eslint": "^8.0.0",
    "vite": "^7.0.0",
    "vitest": "^3.0.0"
  }
}
```

- [x] **Step 2: 运行依赖安装**

Run: `cd web/landing && pnpm install`
Expected: 依赖安装成功，生成 `pnpm-lock.yaml`。

- [x] **Step 3: 新建 vite.config.ts 与 vitest.config.ts**

```ts
// web/landing/vite.config.ts
/// <reference types="vitest/config" />
import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: { host: '127.0.0.1', port: 5173 },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
})
```

- [x] **Step 4: 新建 tsconfig 三件套**

`tsconfig.json`（references 到 app/node）：

```json
{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" },
    { "path": "./tsconfig.node.json" }
  ],
  "compilerOptions": { "paths": { "@/*": ["./src/*"] } }
}
```

`tsconfig.app.json`（复制 admin 基线，含 `paths`、`strict`、`noUnusedLocals`，include `src`，exclude `src/**/*.test.tsx` 与 `src/test/**`）：

```json
{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.app.tsbuildinfo",
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "paths": { "@/*": ["./src/*"] },
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true
  },
  "include": ["src"],
  "exclude": ["src/**/*.test.tsx", "src/test/**"]
}
```

`tsconfig.node.json`：

```json
{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.node.tsbuildinfo",
    "target": "ES2022",
    "lib": ["ES2023"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "strict": true
  },
  "include": ["vite.config.ts", "vitest.config.ts"]
}
```

- [x] **Step 5: 新建 index.html 与 src/main.tsx / src/App.tsx**

`index.html`：

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="Pixoma — 随时随地用自己的 ComfyUI 做 AI 艺术创作" />
    <title>Pixoma</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

`src/main.tsx` 与 `src/App.tsx`：

```tsx
// src/main.tsx
import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App'
import './styles/index.css'

const rootElement = document.getElementById('root')!
if (!rootElement.innerHTML) {
  ReactDOM.createRoot(rootElement).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}
```

```tsx
// src/App.tsx
export function App() {
  return (
    <main className="min-h-svh bg-background text-foreground">
      <p>Pixoma</p>
    </main>
  )
}
```

- [x] **Step 6: 新建 src/styles/index.css（Tailwind v4 + 语义令牌）**

```css
@import 'tailwindcss';

:root {
  --background: #fafaf9;
  --foreground: #1c1917;
  --card: #ffffff;
  --card-foreground: #1c1917;
  --muted: #f5f5f4;
  --muted-foreground: #78716c;
  --border: #e7e5e4;
  --primary: #f97316;
  --primary-foreground: #ffffff;
  --secondary: #f5f5f4;
  --secondary-foreground: #1c1917;
  --ring: #f97316;
}

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-border: var(--border);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-ring: var(--ring);
}

@layer base {
  * {
    @apply border-border;
  }
  body {
    @apply bg-background text-foreground antialiased;
  }
}
```

- [x] **Step 7: 新建 src/lib/utils.ts 与测试 setup**

```ts
// src/lib/utils.ts
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

需在 `package.json` dependencies 补 `clsx` 与 `tailwind-merge`。

`src/test/setup.ts`：

```ts
import '@testing-library/jest-dom/vitest'
```

- [x] **Step 8: 运行类型检查 + lint 验证工程骨架**

Run: `pnpm build && pnpm lint`
Expected: 构建通过、无 TS 错误、lint 通过（此时仅有占位 App）。

- [x] **Step 9: Commit**

```bash
git add web/landing
git commit -m "feat(landing): scaffold vite react ts tailwind landing app"
```

---

### Task 2: 路由与多语言（含测试）

**Files:**
- Create: `web/landing/src/router/index.tsx`
- Create: `web/landing/src/i18n/index.ts`
- Create: `web/landing/src/i18n/locales/zh-CN.ts`
- Create: `web/landing/src/i18n/locales/en.ts`
- Create: `web/landing/src/components/layout/LangLayout.tsx`
- Create: `web/landing/src/components/layout/LangSwitch.tsx`
- Create: `web/landing/src/__tests__/i18n.test.tsx`
- Modify: `web/landing/src/App.tsx`
- Modify: `web/landing/src/main.tsx`

**Interfaces:**
- Consumes: `App`（Task 1）。
- Produces: `LANG_MAP`（`{ cn: "zh-CN", en: "en" }`）、`SUPPORTED_LANGS`（`["cn","en"]`）、`i18n` 实例、`LangLayout` 组件（接收 `children` + `lang` 参数）、`LangSwitch` 组件、词条资源 `zh-CN`/`en`。

- [x] **Step 1: 写失败的 i18n 测试**

```tsx
// src/__tests__/i18n.test.tsx
import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { initI18n } from '@/i18n'
import zhCN from '@/i18n/locales/zh-CN'
import en from '@/i18n/locales/en'

describe('i18n', () => {
  it('zh-CN 与 en 词条 key 完全对齐', () => {
    expect(Object.keys(zhCN).sort()).toEqual(Object.keys(en).sort())
  })

  it('语言段映射与回退正确', async () => {
    const i = await initI18n()
    expect(i.language).toBe('zh-CN')
    i.changeLanguage('en')
    expect(i.language).toBe('en')
    i.changeLanguage('invalid' as never)
    expect(i.language).toBe('zh-CN')
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- i18n`
Expected: FAIL（`initI18n` 未定义）。

- [x] **Step 3: 初始化 i18n 与语言资源**

```ts
// src/i18n/locales/zh-CN.ts
export default {
  'nav.features': '为什么是 Pixoma',
  'nav.scenarios': '使用场景',
  'nav.download': '下载',
  'hero.badge': '开源 · 自托管 · ComfyUI 创作',
  'hero.title': '先创作，再整理。',
  'hero.subtitle':
    '随手在 Telegram 里描述想法，剩余的让 Pixoma 在你自己的 ComfyUI 上跑完。',
  'hero.cta_download': '下载 Pixoma',
  'hero.cta_selfhost': '自托管部署',
  'hero.platform': 'Windows · macOS · Linux · 远端 GPU',
  'features.bot.title': 'Bot',
  'features.bot.desc': '在 Telegram 里发指令，Pixoma 调度你的 ComfyUI。',
  'features.case.title': 'Case 目录',
  'features.case.desc': '工作流、模型与历史输出，都在一个目录里。',
  'features.admin.title': '管理后台',
  'features.admin.desc': '算力节点、任务与资源，一处看清。',
  'scenarios.title': '用你的算力，做想做的创作。',
  'cta.title': '先跑通第一条工作流。',
  'cta.subtitle': '灵感不会一直等你。',
  'footer.copyright': '© 2026 Pixoma',
} as const
```

```ts
// src/i18n/locales/en.ts
export default {
  'nav.features': 'Why Pixoma',
  'nav.scenarios': 'Use Cases',
  'nav.download': 'Download',
  'hero.badge': 'Open-source · Self-hosted · ComfyUI',
  'hero.title': 'Create first, organize later.',
  'hero.subtitle':
    'Describe idea in Telegram, let Pixoma run it on your own ComfyUI.',
  'hero.cta_download': 'Download Pixoma',
  'hero.cta_selfhost': 'Self-host',
  'hero.platform': 'Windows · macOS · Linux · Remote GPU',
  'features.bot.title': 'Bot',
  'features.bot.desc': 'Send a prompt in Telegram, Pixoma drives your ComfyUI.',
  'features.case.title': 'Case Directory',
  'features.case.desc': 'Workflows, models and outputs in one place.',
  'features.admin.title': 'Admin',
  'features.admin.desc': 'Nodes, tasks and resources at a glance.',
  'scenarios.title': 'Use your compute for the art you want.',
  'cta.title': 'Get your first workflow running.',
  'cta.subtitle': 'Inspiration won’t wait.',
  'footer.copyright': '© 2026 Pixoma',
} as const
```

```ts
// src/i18n/index.ts
import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zhCN from './locales/zh-CN'
import en from './locales/en'

export const supportedLocales = ['zh-CN', 'en'] as const
export type SupportedLocale = (typeof supportedLocales)[number]

const resources = {
  'zh-CN': { translation: zhCN },
  en: { translation: en },
} as const

export function resolveLocale(langSegment?: string): SupportedLocale {
  if (langSegment === 'en') return 'en'
  return 'zh-CN'
}

export async function initI18n(lng: SupportedLocale = 'zh-CN') {
  if (!i18n.isInitialized) {
    await i18n.use(initReactI18next).init({
      resources,
      lng,
      fallbackLng: 'zh-CN',
      interpolation: { escapeValue: false },
    })
  }
  return i18n
}

export default i18n
```

- [x] **Step 4: 路由与语言段绑定**

```tsx
// src/router/index.tsx
import { Navigate, Route, Routes } from 'react-router-dom'
import { LangLayout } from '@/components/layout/LangLayout'
import { supportedLocales, type SupportedLocale } from '@/i18n'

export const langSegmentToLocale: Record<string, SupportedLocale> = {
  cn: 'zh-CN',
  en: 'en',
}

export const supportedLangSegments = ['cn', 'en'] as const
export type LangSegment = (typeof supportedLangSegments)[number]

export function isLangSegment(v: unknown): v is LangSegment {
  return typeof v === 'string' && supportedLangSegments.includes(v as LangSegment)
}

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/:lang/*" element={<LangLayout />} />
      <Route index element={<Navigate to="/cn" replace />} />
      <Route path="*" element={<Navigate to="/cn" replace />} />
    </Routes>
  )
}
```

`LangLayout`（读取 `/cn`、`/en`，归一化非法段）：

```tsx
// src/components/layout/LangLayout.tsx
import { useEffect } from 'react'
import { Navigate, Outlet, useLocation, useParams } from 'react-router-dom'
import i18n, { resolveLocale, supportedLocales } from '@/i18n'
import { isLangSegment, langSegmentToLocale } from '@/router'

export function LangLayout() {
  const { lang } = useParams()
  const location = useLocation()

  if (!isLangSegment(lang)) {
    return <Navigate to="/cn" replace />
  }

  const locale = langSegmentToLocale[lang]
  const segment = langSegmentToLocale[lang] === 'en' ? 'en' : 'cn'
  const path = location.pathname.replace(/^\/[a-z-]+/, `/${segment}`)

  useEffect(() => {
    void i18n.changeLanguage(locale)
    document.documentElement.lang = locale
  }, [locale])

  return <Outlet />
}
```

`App.tsx` 改为挂路由，`main.tsx` 挂 i18n 后再渲染。

- [x] **Step 5: 运行测试验证通过**

Run: `pnpm test -- i18n`
Expected: PASS。

- [x] **Step 6: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): router and i18n with lang fallback"
```

---

### Task 3: 全局布局与动效基座

**Files:**
- Create: `web/landing/src/components/motion/Reveal.tsx`
- Create: `web/landing/src/components/motion/motionPresets.ts`
- Create: `web/landing/src/components/layout/Header.tsx`
- Create: `web/landing/src/components/layout/Footer.tsx`
- Create: `web/landing/src/components/layout/MobileMenu.tsx`
- Create: `web/landing/src/__tests__/Reveal.test.tsx`
- Modify: `web/landing/src/components/layout/LangLayout.tsx`

**Interfaces:**
- Consumes: `LangSwitch`（Task 2）、词条 key、`cn`（Task 1）。
- Produces: `Reveal` 组件（接收 `children`、可选 `delay`、`className`）、`fadeUp` / `stagger` variants、`Header`、`Footer`。

- [x] **Step 1: 写失败的 Reveal 降级测试**

```tsx
// src/__tests__/Reveal.test.tsx
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Reveal } from '@/components/motion/Reveal'

vi.mock('motion/react', async (importOriginal) => {
  const mod = await importOriginal<typeof import('motion/react')>()
  return {
    ...mod,
    useReducedMotion: vi.fn(() => true),
  }
})

describe('Reveal', () => {
  it('reduced motion 下渲染内容且不依赖 opacity 动画', () => {
    render(<Reveal>你好</Reveal>)
    expect(screen.getByText('你好')).toBeInTheDocument()
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- Reveal`
Expected: FAIL（`Reveal` 未定义）。

- [x] **Step 3: 实现 Reveal 与 motion presets**

```ts
// src/components/motion/motionPresets.ts
import type { Variants } from 'motion/react'

export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0 },
}

export const containerStagger: Variants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.08 } },
}
```

```tsx
// src/components/motion/Reveal.tsx
import { motion, useInView, useReducedMotion } from 'motion/react'
import { useRef, type ReactNode } from 'react'
import { fadeUp } from './motionPresets'
import { cn } from '@/lib/utils'

type RevealProps = {
  children: ReactNode
  className?: string
  delay?: number
}

export function Reveal({ children, className, delay = 0 }: RevealProps) {
  const ref = useRef<HTMLDivElement>(null)
  const inView = useInView(ref, { once: true, margin: '-80px' })
  const reduced = useReducedMotion()

  return (
    <motion.div
      ref={ref}
      className={cn(className)}
      variants={fadeUp}
      initial={reduced ? false : 'hidden'}
      animate={reduced ? undefined : inView ? 'visible' : 'hidden'}
      transition={reduced ? undefined : { duration: 0.5, delay, ease: 'easeOut' }}
    >
      {children}
    </motion.div>
  )
}
```

- [x] **Step 4: 实现 Header / Footer / MobileMenu**

`Header`：`sticky top-0`，桌面导航链接 + `LangSwitch`；移动端折叠为汉堡按钮 + `AnimatePresence` 展开。
`Footer`：品牌、导航链接、`LangSwitch`、版权（`footer.copyright`）。
`LangSwitch`：点击在 `/cn`、`/en` 间切换（`useNavigate`），高亮当前语言。

在 `LangLayout` 的 `Outlet` 外包 `Header` + `Footer`。

- [x] **Step 5: 运行测试验证通过**

Run: `pnpm test -- Reveal`
Expected: PASS。

- [x] **Step 6: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): layout shell and reduced-motion reveal"
```

---

### Task 4: Hero 区（含 reactbits 动效）

**Files:**
- Create: `web/landing/src/components/sections/HeroSection.tsx`
- Create: `web/landing/src/components/reactbits/ShinyText.tsx`
- Create: `web/landing/src/components/reactbits/GradientBackground.tsx`
- Create: `web/landing/src/components/ui/Button.tsx`
- Create: `web/landing/src/__tests__/Hero.test.tsx`

**Interfaces:**
- Consumes: `Reveal`、`cn`、词条 `hero.*`、`site.ts`（可延后）、`Button`。
- Produces: `HeroSection` 组件、`site.downloadUrl` / `site.selfHostUrl`（constants，先给占位常量）。

- [x] **Step 1: 写 Hero 渲染测试**

```tsx
// src/__tests__/Hero.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { HeroSection } from '@/components/sections/HeroSection'

describe('HeroSection', () => {
  it('渲染主标题与 CTA', () => {
    render(<HeroSection />)
    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /下载|Download/i })).toBeInTheDocument()
  })
})
```

- [x] **Step 2: 新建 src/constants/site.ts**

```ts
export const site = {
  downloadUrl: 'https://github.com/pixoma/pixoma/releases',
  selfHostUrl: '/cn/docs/deploy',
} as const
```

- [x] **Step 3: 实现 Button 与 reactbits 效果**

`src/components/ui/Button.tsx`（reui 风格，语义类名 + variants）：

```tsx
import { type ComponentPropsWithoutRef } from 'react'
import { cn } from '@/lib/utils'

type ButtonProps = ComponentPropsWithoutRef<'a'> & {
  variant?: 'primary' | 'outline'
}

export function Button({ variant = 'primary', className, ...props }: ButtonProps) {
  return (
    <a
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-lg px-5 py-2.5 text-sm font-medium transition-colors',
        variant === 'primary' &&
          'bg-primary text-primary-foreground hover:opacity-90',
        variant === 'outline' &&
          'border border-border bg-card text-foreground hover:bg-muted',
        className,
      )}
      {...props}
    />
  )
}
```

`ShinyText`（reactbits text shimmer，用 CSS 渐变扫光）与 `GradientBackground`（极简背景流光）按需实现，均含 `prefers-reduced-motion` 降级。

- [x] **Step 4: 实现 HeroSection**

`HeroSection`：居中 hero，包含 `hero.badge`、`h1`（`hero.title`）、副标语、`site.downloadUrl`/`site.selfHostUrl` 两个 CTA、平台行；用 `Reveal` + `GradientBackground` 装饰。

- [x] **Step 5: 运行测试验证通过**

Run: `pnpm test -- Hero`
Expected: PASS。

- [x] **Step 6: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): hero section with reactbits effects"
```

---

### Task 5: 特性展示区 + Bot 对话演示

**Files:**
- Create: `web/landing/src/components/sections/FeaturesSection.tsx`
- Create: `web/landing/src/components/sections/FeatureBlock.tsx`
- Create: `web/landing/src/components/demos/BotChatDemo.tsx`
- Create: `web/landing/src/data/chat.ts`
- Create: `web/landing/src/__tests__/BotChatDemo.test.tsx`
- Modify: `web/landing/src/i18n/locales/zh-CN.ts`、`en.ts`

**Interfaces:**
- Consumes: `site`、词条 `features.*`、`Reveal`、`motion`。
- Produces: `FeaturesSection`、`FeatureBlock`（`title`/`desc`/`children`）、`BotChatDemo`、`chatMessages` 数据。

- [x] **Step 1: 写 BotChatDemo 测试**

```tsx
// src/__tests__/BotChatDemo.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BotChatDemo } from '@/components/demos/BotChatDemo'

describe('BotChatDemo', () => {
  it('用静态数据渲染消息与前进交互', () => {
    render(<BotChatDemo />)
    expect(screen.getByText(/彭博/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /前进|Next/i })).toBeInTheDocument()
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- BotChatDemo`
Expected: FAIL。

- [x] **Step 3: 实现 chat 数据与 BotChatDemo**

```ts
// src/data/chat.ts
export type ChatMessage = {
  role: 'user' | 'assistant'
  text: string
}

export const chatMessages: ChatMessage[] = [
  { role: 'user', text: '用赛博朋克风格生成一张都市夜景' },
  { role: 'assistant', text: '已在 GPU 节点 1 运行工作流 cyber-city…' },
  { role: 'assistant', text: '结果已回传，打开看看。' },
]
```

`BotChatDemo`：用 state 控制逐条/打字显现，`motion` 控制气泡入场；提供「前进 / 重放」按钮。

- [x] **Step 4: 实现 FeatureBlock 与 FeaturesSection**

`FeatureBlock`：`title`（`h3`）+ `desc` + 演示容器（`aspect-ratio` 面板）。
`FeaturesSection`：`nav.features` 标题 + 多个 `FeatureBlock`（Bot / Case / Admin），每块包 `Reveal`。

- [x] **Step 5: 运行测试验证通过**

Run: `pnpm test -- BotChatDemo`
Expected: PASS。

- [x] **Step 6: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): features with bot chat demo"
```

---

### Task 6: Case 目录演示

**Files:**
- Create: `web/landing/src/components/demos/CaseDirDemo.tsx`
- Create: `web/landing/src/data/cases.ts`
- Create: `web/landing/src/__tests__/CaseDirDemo.test.tsx`

**Interfaces:**
- Consumes: `motion`、`Reveal`。
- Produces: `CaseDirDemo`、`cases` 数据、选中态交互。

- [x] **Step 1: 写 CaseDirDemo 测试**

```tsx
// src/__tests__/CaseDirDemo.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CaseDirDemo } from '@/components/demos/CaseDirDemo'

describe('CaseDirDemo', () => {
  it('渲染目录列表并响应选择', () => {
    render(<CaseDirDemo />)
    expect(screen.getByText('赛博都市')).toBeInTheDocument()
    screen.getByRole('button', { name: '水墨山水' }).click()
    expect(screen.getByText(/水墨/)).toBeInTheDocument()
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- CaseDirDemo`
Expected: FAIL。

- [x] **Step 3: 实现 cases 数据与 CaseDirDemo**

```ts
// src/data/cases.ts
export type CaseItem = { id: string; title: string; desc: string }
export const cases: CaseItem[] = [
  { id: 'cyber', title: '赛博都市', desc: '都市夜景工作流' },
  { id: 'ink', title: '水墨山水', desc: '国风水墨工作流' },
  { id: 'portrait', title: '人像精修', desc: '写真修复工作流' },
]
```

`CaseDirDemo`：左侧列表 + 右侧详情，选中切换用 `layout` 动画，数据来自 `cases`。

- [x] **Step 4: 运行测试验证通过**

Run: `pnpm test -- CaseDirDemo`
Expected: PASS。

- [x] **Step 5: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): case directory demo"
```

---

### Task 7: 管理后台演示

**Files:**
- Create: `web/landing/src/components/demos/AdminDemo.tsx`
- Create: `web/landing/src/data/admin.ts`
- Create: `web/landing/src/__tests__/AdminDemo.test.tsx`

**Interfaces:**
- Consumes: `motion`、`Reveal`。
- Produces: `AdminDemo`、`adminTasks` 数据。

- [x] **Step 1: 写 AdminDemo 测试**

```tsx
// src/__tests__/AdminDemo.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AdminDemo } from '@/components/demos/AdminDemo'

describe('AdminDemo', () => {
  it('渲染节点与任务信息', () => {
    render(<AdminDemo />)
    expect(screen.getByText(/节点 1/)).toBeInTheDocument()
    expect(screen.getByText(/导入/)).toBeInTheDocument()
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- AdminDemo`
Expected: FAIL。

- [x] **Step 3: 实现 admin 数据与 AdminDemo**

```ts
// src/data/admin.ts
export const adminTasks = [
  { id: 'task-1', title: '导入 worklow', node: '节点 1', status: '运行中' },
  { id: 'task-2', title: '生成本地预览', node: '节点 2', status: '排队' },
] as const
```

`AdminDemo`：伪后台面板（表格/卡片），顶部 tab 或滚动切换动效。

- [x] **Step 4: 运行测试验证通过**

Run: `pnpm test -- AdminDemo`
Expected: PASS。

- [x] **Step 5: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): admin console demo"
```

---

### Task 8: 特性区懒加载与完整 section 组装

**Files:**
- Modify: `web/landing/src/components/sections/FeaturesSection.tsx`
- Create: `web/landing/src/components/demos/LazyDemos.tsx`
- Create: `web/landing/src/components/SectionShell.tsx`

**Interfaces:**
- Consumes: 三个 demo 组件。
- Produces: `LazyDemos`（`React.lazy` + `Suspense` + fallback skeleton）、`SectionShell`（标题/容器）。

- [x] **Step 1: 实现 LazyDemos 与 SectionShell**

```tsx
// src/components/demos/LazyDemos.tsx
import { lazy, Suspense } from 'react'

const BotChatDemo = lazy(() =>
  import('@/components/demos/BotChatDemo').then((m) => ({ default: m.BotChatDemo })),
)
const CaseDirDemo = lazy(() =>
  import('@/components/demos/CaseDirDemo').then((m) => ({ default: m.CaseDirDemo })),
)
const AdminDemo = lazy(() =>
  import('@/components/demos/AdminDemo').then((m) => ({ default: m.AdminDemo })),
)

function Skeleton() {
  return <div className="aspect-[4/3] w-full animate-pulse rounded-xl bg-muted" />
}

export function LazyBotChat() {
  return (
    <Suspense fallback={<Skeleton />}>
      <BotChatDemo />
    </Suspense>
  )
}

export function LazyCaseDir() {
  return (
    <Suspense fallback={<Skeleton />}>
      <CaseDirDemo />
    </Suspense>
  )
}

export function LazyAdmin() {
  return (
    <Suspense fallback={<Skeleton />}>
      <AdminDemo />
    </Suspense>
  )
}
```

- [x] **Step 2: 让 FeaturesSection 使用懒加载演示 + 接近视口挂载**

用 `useInView` 包裹标题区，进入视口后才渲染 `LazyDemos`，并保留 `Reveal`。

- [x] **Step 3: 运行构建验证**

Run: `pnpm build`
Expected: PASS，无懒加载路径错误。

- [x] **Step 4: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): lazy load interactive demos"
```

---

### Task 9: 使用场景区

**Files:**
- Create: `web/landing/src/components/sections/UseCasesSection.tsx`
- Modify: `web/landing/src/i18n/locales/zh-CN.ts`、`en.ts`
- Create: `web/landing/src/__tests__/UseCases.test.tsx`

**Interfaces:**
- Consumes: `Reveal`、词条。
- Produces: `UseCasesSection`。

- [x] **Step 1: 写 UseCases 测试**

```tsx
// src/__tests__/UseCases.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { UseCasesSection } from '@/components/sections/UseCasesSection'

describe('UseCasesSection', () => {
  it('渲染使用场景卡片', () => {
    render(<UseCasesSection />)
    expect(screen.getByRole('heading', { level: 2, name: /场景|Cases|Uses/i })).toBeInTheDocument()
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- UseCases`
Expected: FAIL。

- [x] **Step 3: 实现 UseCasesSection**

补 `scenarios.*` 词条（如「会议碎片 / 每日记录 / 链接与 PDF / 零散灵感」对应的 Pixoma 场景：Bot 对话即得、Case 归类、画布连接、多节点并行）。用卡片栅格（`md:grid-cols-2`）呈现。

- [x] **Step 4: 运行测试验证通过**

Run: `pnpm test -- UseCases`
Expected: PASS。

- [x] **Step 5: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): use cases section"
```

---

### Task 10: CTA 与收尾

**Files:**
- Create: `web/landing/src/components/sections/CTASection.tsx`
- Modify: `web/landing/src/components/layout/Footer.tsx`
- Modify: `web/landing/src/i18n/locales/zh-CN.ts`、`en.ts`
- Create: `web/landing/src/__tests__/CTA.test.tsx`

**Interfaces:**
- Consumes: `Button`、`site`、词条。
- Produces: `CTASection`。

- [x] **Step 1: 写 CTA 测试**

```tsx
// src/__tests__/CTA.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CTASection } from '@/components/sections/CTASection'

describe('CTASection', () => {
  it('渲染主行动入口', () => {
    render(<CTASection />)
    expect(screen.getByRole('link', { name: /下载|Download/i })).toBeInTheDocument()
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- CTA`
Expected: FAIL。

- [x] **Step 3: 实现 CTASection 并完善 Footer**

`CTASection`：`cta.title` + `cta.subtitle` + `Button`（`site.downloadUrl`）＋ 自托管入口。`Footer` 补导航、语言入口与版权。

- [x] **Step 4: 运行测试验证通过**

Run: `pnpm test -- CTA`
Expected: PASS。

- [x] **Step 5: Commit**

```bash
git add web/landing/src && git commit -m "feat(landing): cta and footer polish"
```

---

### Task 11: 全站集成、动效复核与构建门禁

**Files:**
- Modify: `web/landing/src/App.tsx`（组装所有 section）
- Modify: `web/landing/src/components/layout/LangLayout.tsx`（验证节锚点导航）
- Create: `web/landing/src/__tests__/app.integration.test.tsx`

**Interfaces:**
- Consumes: 全部 section 与布局组件。
- Produces: 无新接口；完成端到端组装。

- [x] **Step 1: 写集成测试**

```tsx
// src/__tests__/app.integration.test.tsx
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { App } from '@/App'

describe('app integration', () => {
  it('渲染 hero + features + cases 场景 + cta', () => {
    render(
      <MemoryRouter initialEntries={['/cn']}>
        <App />
      </MemoryRouter>,
    )
    expect(screen.getAllByRole('heading').length).toBeGreaterThanOrEqual(4)
  })
})
```

- [x] **Step 2: Run test to verify it fails**

Run: `pnpm test -- app.integration`
Expected: 失败/不确定（先确认 App 是否能在 MemoryRouter 下渲染，若 i18n 未就绪则补 `initI18n` 前置）。

- [x] **Step 3: 组装 App 并跑通**

```tsx
// src/App.tsx
import { BrowserRouter } from 'react-router-dom'
import { AppRoutes } from '@/router'

export function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  )
}
```

在 `LangLayout` 内组装 `Header`、`HeroSection`、`FeaturesSection`、`UseCasesSection`、`CTASection`、`Footer`。

- [x] **Step 4: 运行全部测试与构建**

Run: `pnpm test && pnpm build && pnpm lint && pnpm format:check`
Expected: 全绿。

- [x] **Step 5: Commit**

```bash
git add web/landing && git commit -m "feat(landing): assemble full page and pass gates"
```

---

### Task 12: 交互复核与收尾确认

**Files:**
- Modify: `web/landing/src/components/**`（按需修正动效/响应式）
- Create: `web/landing/README.md`（可选，工程说明）

**Interfaces:**
- Consumes: 全部组件。
- Produces: 无。

- [x] **Step 1: 本地 preview 验证响应式与动效**

Run: `pnpm build && pnpm preview`
Expected: 桌面/平板/移动端布局正常；scroll-reveal、bot/case/admin 演示可交互。

- [x] **Step 2: 复核 AGENTS.md 约束**

确认：无裸 hex、无手写 `dark:`、无截图/快照脚手架、zh-CN 文案符合 voice-profile。

- [x] **Step 3: Commit 收尾**

```bash
git add web/landing && git commit -m "chore(landing): responsive and motion review"
```

---

## Self-Review

**Spec coverage:** 覆盖 design.md 全部 6 个关键技术决策（骨架/路由多语言/动效/交互演示/视觉文案/响应式）与 3 份 delta spec；tasks.md 的 8 组 22 项已全部映射到 Task 1–12。

**Placeholder scan:** 无 TBD / TODO；组件代码块给出实际实现与测试。

**Type consistency:** `Reveal`、`Button`、各 demo 的 props/data 类型在前置 Task 中定义，后续 Task 一致引用；`cn`、`site`、`i18n` 名称跨 Task 统一。
