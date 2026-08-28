## 1. 应用脚手架

- [x] 1.1 新建 `web/landing` 目录与 `package.json`（`type: module`、`pnpm`、`packageManager`），与 admin 工程习惯一致
- [x] 1.2 配置 `vite.config.ts`（`@vitejs/plugin-react` + `@tailwindcss/vite`）与 `tsconfig*.json`、`index.html`、`src/main.tsx`
- [x] 1.3 建立 Tailwind v4 基线：`src/index.css` 引入 Tailwind，并定义基础字体/颜色语义令牌（遵循 reui/shadcn 风格，不写裸 hex）
- [x] 1.4 安装依赖：`react-router-dom`、`motion`、`i18next`、`react-i18next`、`tailwindcss`，以及所需 devDeps（`typescript`、`@types/react*`、`eslint`、`prettier` 等）
- [x] 1.5 按需 vendor reui 组件到 `src/components/reui/`（标注来源），不重复造轮子

## 2. 路由与多语言

- [x] 2.1 建立 `react-router-dom` 路由：`/:lang/*`，`lang` 限定 `zh-CN` / `en`，未指定或非法回退 `zh-CN`
- [x] 2.2 初始化 `i18next` + `react-i18next`，提供 `zh-CN` / `en` 的 key-based 词条 resources
- [x] 2.3 实现导航/页头的语言切换，切换时导航到对应 `/cn`、`/en` 路径，且无需整页 reload

## 3. 全局布局与动效基座

- [x] 3.1 实现顶层布局（导航、主内容、页脚）与响应式容器/断点
- [x] 3.2 封装基于 `motion`（Framer Motion）的 `useInView` scroll-reveal 入场组件，并处理 `prefers-reduced-motion` 降级
- [x] 3.3 在页头/页脚提供语言切换入口，并接入主题/色彩语义关系

## 4. Hero 区

- [x] 4.1 实现 Hero 组件：主标语 + 副标语 + 主行动入口（下载/自托管），文案走 i18n
- [x] 4.2 为 Hero 添加入场动画与 reactbits 装饰性效果（如文本/背景特效），支持减弱动态降级

## 5. 特性展示区（含交互式 App UI 演示）

- [x] 5.1 实现特性区通用组件（对齐参考页：标题 + 描述 + 演示容器）
- [x] 5.2 实现 Telegram Bot 对话演示组件（静态 mockup 数据 + 打字/交互动效）
- [x] 5.3 实现 Case 目录 / 画布演示组件（静态 mockup + 切换/布局动效）
- [x] 5.4 实现管理后台演示组件（静态 mockup + 滚动/切换动效）
- [x] 5.5 为各特性区补齐 `zh-CN` / `en` 文案，确保无硬编码文案

## 6. 使用场景区

- [x] 6.1 实现使用场景卡片区（多场景 + 图示 + i18n 文案）

## 7. CTA 与收尾

- [x] 7.1 实现 CTA 区（主行动入口，`src/constants/site.ts` 集中可配的下载/自托管 URL）
- [x] 7.2 完善页脚（导航、版权、语言切换）

## 8. 构建与验证

- [x] 8.1 `pnpm build`（`tsc -b` + `vite build`）通过且无 TS 错误，产物为静态可托管文件
- [x] 8.2 `pnpm dev` 或 preview 验证各 section 渲染与动效，覆盖桌面/平板/移动端响应式
- [x] 8.3 运行 `lint`/`format:check` 与仓库约定对齐，确认无截图类测试脚手架
