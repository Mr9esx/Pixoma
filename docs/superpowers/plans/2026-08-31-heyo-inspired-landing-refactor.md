---
change: heyo-inspired-landing-refactor
design-doc: docs/superpowers/specs/2026-08-31-heyo-inspired-landing-refactor-design.md
base-ref: d18317afcc0322f1a77abd3bed49982b530438c1
---

# Heyo 风格 Landing Page 重构实施计划

> **给实施代理：** 必须使用 `subagent-driven-development`（推荐）或 `executing-plans` skill，按任务逐项实施。步骤使用 checkbox（`- [ ]`）跟踪。

**目标：** 在 `web/landing` 内以原创的 Pixoma 双语内容重建八段式落地页，保留三组交互演示、下载与自托管入口，并满足静态部署、响应式、键盘操作和减少动态效果要求。

**架构：** `LangLayout` 只负责语言同步、非法路径重定向和八个顶层区域的顺序；导航、适用人群和 FAQ 使用稳定 ID 配置，所有可见文案由 i18n 提供。Hero 静态加载一份独立的 `BotChatDemo`，功能区直接复用 Bot 模块并继续动态加载 Case 和后台演示；FAQ 使用 shadcn Accordion，其他区块保持纯本地数据，不引入后端请求或页面级状态。

**技术栈：** React 19、TypeScript 6、React Router 7、i18next、Tailwind CSS v4、Motion、shadcn/ui、Vitest、Testing Library、Vite 8、pnpm 10。

## 全局约束

- 需求事实源是 `docs/openspec/changes/heyo-inspired-landing-refactor/specs/landing-page-experience/spec.md`；任务边界是 `docs/openspec/changes/heyo-inspired-landing-refactor/tasks.md`。
- 页面顺序固定为 Header、Hero、Feature Showcase、Cross Device、Audience、FAQ、Final CTA、Footer；不增加工具集成区、价格区或对应导航。
- 保留 Pixoma 品牌、`/cn` 与 `/en` 路由、非法语言回到 `/cn`、下载和自托管入口。
- 所有新增可见文案只进入 `src/i18n/locales/zh-CN.ts` 与 `src/i18n/locales/en.ts`；中文遵循 `docs/voice-profile.md`，不出现「请 / 您 / 前往 / 进行 / 完成 / 实施」、感叹号、emoji 或排除项解释文案。
- 颜色只在 `src/styles/index.css` 的语义变量中定义；TSX 只使用语义类，不写裸色值、`slate-*` 或手工 `dark:` 分支。采用设计确认的黑白中性色，不沿用当前橙色主点缀。
- 卡片、按钮和演示表面不使用常驻投影；交互控件具备 hover、focus-visible、active、disabled 状态，触控目标不小于 44×44 CSS 像素。
- 320px、平板和桌面视口均不得出现页面级横向滚动；区块设置 `scroll-margin-top`，避免 sticky Header 遮挡标题。
- `Reveal` 只动画 opacity 与纵向 transform；`prefers-reduced-motion: reduce` 下内容立即可见，循环装饰和非必要过渡关闭或压缩到接近零。
- Hero 直接导入 `BotChatDemo`；功能区的 `LazyBotChat` 复用已加载模块，`LazyCaseDir`、`LazyAdmin` 保留动态 import 和稳定尺寸占位，四个演示实例不共享局部状态。
- 外部目标只读取 `src/constants/site.ts`，组件不得复制 URL；新内容不使用网络图片、视频、远程数据或后端 API。
- 使用 `pnpm dlx shadcn@latest`；先预览 Accordion 变更，禁止 `--overwrite`。初始化必须保留 `@/*` alias、Vite、Tailwind v4 和 `src/styles/index.css`。
- 不新增截图、PNG 快照、浏览器测试配置或临时视觉测试文件。
- 本 change 不改变模块边界、数据存储、请求链路或外部系统连接，不更新 `docs/architecture/`。

---

## 文件结构与职责

**新增：**

- `web/landing/components.json`：shadcn CLI 项目配置，指向现有 alias、工具类和全局样式入口。
- `web/landing/src/components/ui/accordion.tsx`：CLI 生成的可访问 Accordion 源码。
- `web/landing/src/constants/navigation.ts`：Header、MobileMenu、Footer 共用的只读锚点配置。
- `web/landing/src/constants/content.ts`：`audienceIds` 与 `faqIds` 稳定 key 列表。
- `web/landing/src/components/sections/FeatureShowcase.tsx`：单个“说明 + 演示”布局和稳定演示外壳。
- `web/landing/src/components/sections/FeatureShowcaseSection.tsx`：Bot、Case、后台三组演示的固定顺序。
- `web/landing/src/components/sections/CrossDeviceSection.tsx`：Telegram 手机发起与 Pixoma 桌面管理关系。
- `web/landing/src/components/sections/AudienceSection.tsx`：三类适用人群卡片。
- `web/landing/src/components/sections/FaqSection.tsx`：双语 FAQ 与 Accordion 组合。
- `web/landing/src/__tests__/navigation.test.tsx`：三处导航有效锚点与移动菜单闭合。
- `web/landing/src/__tests__/FeatureShowcase.test.tsx`：三组演示顺序、惰性加载与两个 Bot 实例状态隔离。
- `web/landing/src/__tests__/ContentSections.test.tsx`：跨设备、适用人群、FAQ 双语内容。
- `web/landing/src/__tests__/Faq.test.tsx`：FAQ 鼠标与键盘行为、ARIA 关联、焦点保持。

**修改：**

- `web/landing/package.json`、`web/landing/pnpm-lock.yaml`：只纳入 shadcn Accordion 实际需要的依赖。
- `web/landing/src/components/layout/LangLayout.tsx`：仅组装八个顶层区域并保留语言副作用。
- `web/landing/src/components/layout/Header.tsx`、`MobileMenu.tsx`、`Footer.tsx`：消费共享导航，补齐响应式与可访问状态。
- `web/landing/src/components/sections/HeroSection.tsx`：双栏首屏、两级 CTA、平台 meta 与完整 Bot 演示。
- `web/landing/src/components/sections/CTASection.tsx`：最终行动区复用 `site` 目标。
- `web/landing/src/components/SectionShell.tsx`：统一区块标题、容器、垂直节奏和锚点偏移。
- `web/landing/src/components/reui/Button.tsx`：补齐语义 variants、44px 命中区和可见焦点。
- `web/landing/src/components/demos/LazyDemos.tsx`：复用 Bot 模块，并为 Case、后台动态 import 提供稳定占位。
- `web/landing/src/components/demos/BotChatDemo.tsx`、`CaseDirDemo.tsx`、`AdminDemo.tsx`：只做语义类、焦点和减少动态效果适配，不改 mock 数据语义与公开接口。
- `web/landing/src/components/motion/Reveal.tsx`、`motionPresets.ts`：锁定纵向轻量动效与 reduced-motion 直出。
- `web/landing/src/styles/index.css`：黑白语义令牌、排版、圆角、焦点、区块节奏和 reduced-motion 规则。
- `web/landing/src/i18n/locales/zh-CN.ts`、`en.ts`：完整且对等的新导航、区块、FAQ、CTA 文案。
- `web/landing/src/__tests__/app.integration.test.tsx`、`i18n.test.tsx`、`Hero.test.tsx`、`CTA.test.tsx`、`Reveal.test.tsx`：覆盖 canonical spec 的页面级行为。

**删除：**

- `web/landing/src/components/sections/FeaturesSection.tsx`
- `web/landing/src/components/sections/FeatureBlock.tsx`
- `web/landing/src/components/sections/UseCasesSection.tsx`
- `web/landing/src/__tests__/UseCases.test.tsx`
- `web/landing/src/components/reactbits/ShinyText.tsx`
- `web/landing/src/components/reactbits/GradientBackground.tsx`

---

### 任务 1：锁定基线、稳定配置与双语契约

**文件：**

- 新增：`web/landing/src/constants/navigation.ts`
- 新增：`web/landing/src/constants/content.ts`
- 修改：`web/landing/src/__tests__/i18n.test.tsx`
- 修改：`web/landing/src/i18n/locales/zh-CN.ts`
- 修改：`web/landing/src/i18n/locales/en.ts`

**接口：**

- 产出：`navigationItems: readonly { id: "features" | "devices" | "audience" | "faq" | "download"; labelKey: string; href: string }[]`
- 产出：`audienceIds: readonly ["creator", "team", "multiNode"]`
- 产出：`faqIds: readonly ["install", "comfyOwnership", "telegram", "selfHost", "platforms"]`
- 后续任务只用上述数组生成导航与文案 key，不在组件内复制 ID。

- [x] **步骤 1：记录基线结果**

运行：

```bash
cd web/landing
pnpm test
pnpm lint
pnpm build
```

预期：三条命令均 PASS；若 base ref 已有失败，把命令、失败测试/规则和完整错误摘要记入实施会话，不在本 change 顺手修复。

- [x] **步骤 2：写失败的稳定 key 与双语完整性测试**

在 `src/__tests__/i18n.test.tsx` 增加：

```tsx
import { audienceIds, faqIds } from "@/constants/content";
import { navigationItems } from "@/constants/navigation";

it("新增导航与内容 key 在两种语言中存在且非空", () => {
  const keys = [
    ...navigationItems.map((item) => item.labelKey),
    "nav.menu.open",
    "nav.menu.close",
    "nav.mobile",
    "crossDevice.title",
    "crossDevice.mobile.title",
    "crossDevice.mobile.desc",
    "crossDevice.desktop.title",
    "crossDevice.desktop.desc",
    "audience.title",
    ...audienceIds.flatMap((id) => [
      `audience.${id}.title`,
      `audience.${id}.benefit`,
      `audience.${id}.usage`,
    ]),
    "faq.title",
    ...faqIds.flatMap((id) => [`faq.${id}.question`, `faq.${id}.answer`]),
    "cta.title",
    "cta.subtitle",
  ] as const;

  const locales: ReadonlyArray<Record<string, string>> = [zhCN, en];
  for (const key of keys) {
    for (const locale of locales) {
      expect(locale[key]).toBeTruthy();
    }
  }
});

it("导航配置只指向确认存在的五个区块", () => {
  expect(navigationItems.map(({ id, href }) => [id, href])).toEqual([
    ["features", "#features"],
    ["devices", "#devices"],
    ["audience", "#audience"],
    ["faq", "#faq"],
    ["download", "#download"],
  ]);
});
```

- [x] **步骤 3：运行定向测试，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/i18n.test.tsx
```

预期：FAIL，原因是 `constants/content`、`constants/navigation` 或新增 i18n key 尚不存在。

- [x] **步骤 4：实现只读配置与对等词条**

`src/constants/navigation.ts`：

```ts
export const navigationItems = [
  { id: "features", labelKey: "nav.features", href: "#features" },
  { id: "devices", labelKey: "nav.devices", href: "#devices" },
  { id: "audience", labelKey: "nav.audience", href: "#audience" },
  { id: "faq", labelKey: "nav.faq", href: "#faq" },
  { id: "download", labelKey: "nav.download", href: "#download" },
] as const;

export type NavigationItem = (typeof navigationItems)[number];
```

`src/constants/content.ts`：

```ts
export const audienceIds = ["creator", "team", "multiNode"] as const;
export const faqIds = [
  "install",
  "comfyOwnership",
  "telegram",
  "selfHost",
  "platforms",
] as const;
```

在 `zh-CN.ts` 中加入或替换以下词条：

```ts
"nav.features": "功能",
"nav.devices": "跨设备",
"nav.audience": "适用人群",
"nav.faq": "常见问题",
"nav.download": "下载",
"nav.menu.open": "打开菜单",
"nav.menu.close": "关闭菜单",
"nav.mobile": "移动导航",
"crossDevice.title": "手机发起，桌面管理",
"crossDevice.mobile.title": "Telegram / 手机",
"crossDevice.mobile.desc": "描述想法，发起 ComfyUI 创作。",
"crossDevice.desktop.title": "Pixoma / 桌面",
"crossDevice.desktop.desc": "管理 Case、任务和计算节点。",
"audience.title": "适合这些创作方式",
"audience.creator.title": "个人创作者",
"audience.creator.benefit": "随时发起自己的 ComfyUI 工作流。",
"audience.creator.usage": "在 Telegram 描述想法，由自己的算力生成结果。",
"audience.team.title": "小团队",
"audience.team.benefit": "用 Case 组织任务和创作结果。",
"audience.team.usage": "围绕同一组工作流整理过程与产物。",
"audience.multiNode.title": "多节点使用者",
"audience.multiNode.benefit": "统一查看多台 GPU 和不同工作流。",
"audience.multiNode.usage": "在后台管理计算节点与任务状态。",
"faq.title": "常见问题",
"faq.install.question": "怎么安装 Pixoma？",
"faq.install.answer": "下载对应平台版本，或按部署文档自托管。",
"faq.comfyOwnership.question": "ComfyUI 跑在哪里？",
"faq.comfyOwnership.answer": "工作流运行在你连接的 ComfyUI 和计算节点上。",
"faq.telegram.question": "一定要使用 Telegram 吗？",
"faq.telegram.answer": "Telegram 用来随时发起创作，Case、任务和节点在桌面页面管理。",
"faq.selfHost.question": "可以自托管吗？",
"faq.selfHost.answer": "可以。服务和创作数据由你自己的环境管理。",
"faq.platforms.question": "支持哪些平台？",
"faq.platforms.answer": "支持 Windows、macOS、Linux 和远程 GPU。",
```

在 `en.ts` 中加入或替换同一组 key：

```ts
"nav.features": "Features",
"nav.devices": "Cross-device",
"nav.audience": "Who it’s for",
"nav.faq": "FAQ",
"nav.download": "Download",
"nav.menu.open": "Open menu",
"nav.menu.close": "Close menu",
"nav.mobile": "Mobile navigation",
"crossDevice.title": "Start on mobile, manage on desktop",
"crossDevice.mobile.title": "Telegram / Mobile",
"crossDevice.mobile.desc": "Describe an idea and start a ComfyUI creation.",
"crossDevice.desktop.title": "Pixoma / Desktop",
"crossDevice.desktop.desc": "Manage Cases, tasks, and compute nodes.",
"audience.title": "Built for these workflows",
"audience.creator.title": "Independent creators",
"audience.creator.benefit": "Start your own ComfyUI workflows from anywhere.",
"audience.creator.usage": "Describe an idea in Telegram and generate it on your own compute.",
"audience.team.title": "Small teams",
"audience.team.benefit": "Organize tasks and results with Cases.",
"audience.team.usage": "Keep workflows, progress, and outputs together.",
"audience.multiNode.title": "Multi-node users",
"audience.multiNode.benefit": "See multiple GPUs and workflows in one place.",
"audience.multiNode.usage": "Manage compute nodes and task status in the admin.",
"faq.title": "Frequently asked questions",
"faq.install.question": "How do I install Pixoma?",
"faq.install.answer": "Download the build for your platform or follow the deployment guide to self-host.",
"faq.comfyOwnership.question": "Where does ComfyUI run?",
"faq.comfyOwnership.answer": "Workflows run on the ComfyUI instances and compute nodes you connect.",
"faq.telegram.question": "Is Telegram required?",
"faq.telegram.answer": "Telegram starts creations on the go. Cases, tasks, and nodes are managed on desktop.",
"faq.selfHost.question": "Can I self-host Pixoma?",
"faq.selfHost.answer": "Yes. Your environment owns the service and creation data.",
"faq.platforms.question": "Which platforms are supported?",
"faq.platforms.answer": "Pixoma supports Windows, macOS, Linux, and remote GPUs.",
```

- [x] **步骤 5：运行定向测试，确认 GREEN**

运行：

```bash
pnpm test -- src/__tests__/i18n.test.tsx
```

预期：PASS，所有新增 key 在 `zh-CN` 与 `en` 中均非空且 key 集合完全相同。

- [x] **步骤 6：提交本任务**

```bash
git add web/landing/src/constants/navigation.ts web/landing/src/constants/content.ts web/landing/src/i18n/locales/zh-CN.ts web/landing/src/i18n/locales/en.ts web/landing/src/__tests__/i18n.test.tsx
git commit -m "test(landing): lock navigation and bilingual content contracts"
```

### 任务 2：建立语义视觉基础与可访问 Accordion

**文件：**

- 新增：`web/landing/components.json`
- 新增：`web/landing/src/components/ui/accordion.tsx`
- 修改：`web/landing/package.json`
- 修改：`web/landing/pnpm-lock.yaml`
- 修改：`web/landing/src/styles/index.css`
- 修改：`web/landing/src/components/reui/Button.tsx`
- 修改：`web/landing/src/components/SectionShell.tsx`

**接口：**

- 产出：`Button` 继续接受锚点属性与 `variant?: "primary" | "outline"`。
- 产出：`SectionShell({ id, title, children })` 为所有内容区提供统一容器、H2 和 `scroll-mt-20`。
- 产出：`Accordion`、`AccordionItem`、`AccordionTrigger`、`AccordionContent` 从 `@/components/ui/accordion` 导出。

- [x] **步骤 1：增加视觉契约测试**

在 `src/__tests__/app.integration.test.tsx` 增加源码级契约，读取全局 CSS 并阻断旧橙色和裸色散落：

```tsx
import fs from "node:fs";
import path from "node:path";

it("全局样式使用中性语义令牌并提供 reduced-motion 降级", () => {
  const css = fs.readFileSync(
    path.resolve(import.meta.dirname, "../styles/index.css"),
    "utf8",
  );
  expect(css).toContain("--primary: oklch(0.205 0 0)");
  expect(css).toContain("--ring:");
  expect(css).toContain("@media (prefers-reduced-motion: reduce)");
  expect(css).not.toContain("#f97316");
});
```

- [x] **步骤 2：运行视觉契约测试，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/app.integration.test.tsx
```

预期：FAIL，当前 `--primary` 仍是橙色 hex，且没有全局 reduced-motion 规则。

- [x] **步骤 3：检查并预览 shadcn 变更**

运行：

```bash
pnpm dlx shadcn@latest info --json
pnpm dlx shadcn@latest init --help
pnpm dlx shadcn@latest view @shadcn/accordion
```

预期：CLI 识别 Vite、Tailwind v4、`@/*` alias；若 `info` 因缺少 `components.json` 退出，继续初始化，但不得接受覆盖 `src/styles/index.css` 的操作。

- [x] **步骤 4：初始化 shadcn 配置并预览 Accordion**

运行：

```bash
pnpm dlx shadcn@latest init
pnpm dlx shadcn@latest info --json
pnpm dlx shadcn@latest add accordion --dry-run
pnpm dlx shadcn@latest add accordion --diff
```

初始化时选择 Vite、Tailwind v4、现有 `src/styles/index.css`、`@/components`、`@/components/ui`、`@/lib/utils`，保留当前 alias；检查预览不删除既有变量、不改路由和业务组件。

- [x] **步骤 5：添加 Accordion 并审查生成源码**

运行：

```bash
pnpm dlx shadcn@latest add accordion
```

预期：新增 `components.json` 和 `src/components/ui/accordion.tsx`，更新依赖与锁文件；不使用 `--overwrite`。确认 trigger 是原生可聚焦按钮，内容 ID 与 `aria-controls` 由 primitive 关联。

- [x] **步骤 6：实现中性令牌、统一容器和按钮状态**

`src/styles/index.css` 的变量改为 oklch 中性色，并添加：

```css
:root {
  --background: oklch(1 0 0);
  --foreground: oklch(0.145 0 0);
  --card: oklch(1 0 0);
  --card-foreground: oklch(0.145 0 0);
  --muted: oklch(0.97 0 0);
  --muted-foreground: oklch(0.556 0 0);
  --border: oklch(0.922 0 0);
  --primary: oklch(0.205 0 0);
  --primary-foreground: oklch(0.985 0 0);
  --secondary: oklch(0.97 0 0);
  --secondary-foreground: oklch(0.205 0 0);
  --ring: oklch(0.708 0 0);
  --radius: 0.625rem;
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    scroll-behavior: auto !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

`Button` 使用 `min-h-11`、语义 variants、`focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring`，不靠降低文字对比度表达 hover。`SectionShell` 使用统一 `max-w-6xl`、`scroll-mt-20`、流式 H2 和一致区块间距。

- [x] **步骤 7：运行测试与静态检查，确认 GREEN**

运行：

```bash
pnpm test -- src/__tests__/app.integration.test.tsx
pnpm lint
```

预期：PASS；生成的 Accordion 和新增配置无 lint 错误。

- [x] **步骤 8：提交本任务**

```bash
git add web/landing/components.json web/landing/package.json web/landing/pnpm-lock.yaml web/landing/src/components/ui/accordion.tsx web/landing/src/styles/index.css web/landing/src/components/reui/Button.tsx web/landing/src/components/SectionShell.tsx web/landing/src/__tests__/app.integration.test.tsx
git commit -m "feat(landing): establish neutral semantic UI foundation"
```

### 任务 3：重组八段骨架、共享导航与首屏

**文件：**

- 新增：`web/landing/src/__tests__/navigation.test.tsx`
- 修改：`web/landing/src/__tests__/Hero.test.tsx`
- 修改：`web/landing/src/components/layout/Header.tsx`
- 修改：`web/landing/src/components/layout/MobileMenu.tsx`
- 修改：`web/landing/src/components/layout/Footer.tsx`
- 修改：`web/landing/src/components/sections/HeroSection.tsx`

**接口：**

- 消费：`navigationItems`、`site.downloadUrl`、`site.selfHostUrl`、`BotChatDemo`。
- 产出：`MobileMenu({ open, onClose })` 保持现有公开 props；每个锚点先调用 `onClose`。
- 产出：Header、MobileMenu、Footer 只从 `navigationItems` 渲染五个有效锚点；Hero 产出稳定的 `#hero`。

- [x] **步骤 1：写共享导航、排除项和移动闭合失败测试**

`navigation.test.tsx`：

```tsx
function NavigationFixture() {
  return (
    <>
      <Header />
      <main>
        {navigationItems.map((item) => (
          <section key={item.id} id={item.id} />
        ))}
      </main>
      <Footer />
    </>
  );
}

it("桌面与页尾导航只指向存在的五个区块", () => {
  render(
    <MemoryRouter initialEntries={["/cn"]}>
      <NavigationFixture />
    </MemoryRouter>,
  );
  for (const item of navigationItems) {
    const links = screen.getAllByRole("link", { name: zhCN[item.labelKey] });
    expect(links).toHaveLength(2);
    expect(links.every((link) => link.getAttribute("href") === item.href)).toBe(
      true,
    );
    expect(document.querySelector(item.href)).toBeInTheDocument();
  }
  expect(
    screen.queryByRole("link", { name: /tools|pricing|价格|工具集成/i }),
  ).not.toBeInTheDocument();
});

it("选择移动导航后关闭菜单", () => {
  render(
    <MemoryRouter initialEntries={["/cn"]}>
      <Header />
    </MemoryRouter>,
  );
  fireEvent.click(screen.getByRole("button", { name: zhCN["nav.menu.open"] }));
  fireEvent.click(screen.getByRole("link", { name: zhCN["nav.faq"] }));
  expect(
    screen.queryByRole("navigation", { name: zhCN["nav.mobile"] }),
  ).not.toBeInTheDocument();
});
```

- [x] **步骤 2：写 Hero 目标与完整演示失败测试**

`Hero.test.tsx`：

```tsx
it("首屏显示 Pixoma 价值、配置目标和完整 Bot 演示", () => {
  render(<HeroSection />);
  expect(screen.getByText("Pixoma")).toBeInTheDocument();
  expect(screen.getByText(/ComfyUI/i)).toBeInTheDocument();
  expect(screen.getByRole("link", { name: /下载|Download/i })).toHaveAttribute(
    "href",
    site.downloadUrl,
  );
  expect(
    screen.getByRole("link", { name: /自托管|Self-host/i }),
  ).toHaveAttribute("href", site.selfHostUrl);
  expect(
    screen.getByRole("button", { name: /前进|Next/i }),
  ).toBeInTheDocument();
});
```

- [x] **步骤 3：运行两个测试文件，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/navigation.test.tsx src/__tests__/Hero.test.tsx
```

预期：FAIL，当前没有共享五项导航、菜单可访问名称和 Hero Bot 演示。

- [x] **步骤 4：实现共享导航**

Header、MobileMenu、Footer 均 `navigationItems.map(...)`。Header 保持 sticky；窄屏同时显示品牌、语言切换和菜单按钮。菜单按钮使用 i18n 的打开/关闭 label、`aria-expanded={open}` 与 `aria-controls="mobile-navigation"`；移动链接的 `onClick={onClose}` 保持浏览器原生锚点定位。

- [x] **步骤 5：实现双栏 Hero**

`HeroSection` 静态导入 `BotChatDemo`，宽屏使用“两栏”，窄屏按“说明、CTA、演示”顺序堆叠：

```tsx
<section id="hero" data-testid="hero" className="overflow-hidden">
  <div className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-16 sm:px-6 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:py-24">
    <div className="flex min-w-0 flex-col items-start gap-6">
      <p className="text-sm font-medium text-muted-foreground">Pixoma</p>
      <h1 className="max-w-2xl text-balance text-4xl font-semibold tracking-tight sm:text-5xl lg:text-6xl">
        {t("hero.title")}
      </h1>
      <p className="max-w-xl text-base leading-7 text-muted-foreground sm:text-lg">
        {t("hero.subtitle")}
      </p>
      <div className="flex flex-col gap-3 sm:flex-row">
        <Button href={site.downloadUrl}>{t("hero.cta_download")}</Button>
        <Button href={site.selfHostUrl} variant="outline">
          {t("hero.cta_selfhost")}
        </Button>
      </div>
      <p className="text-sm text-muted-foreground">{t("hero.platform")}</p>
    </div>
    <div className="min-h-80 min-w-0 overflow-hidden rounded-xl border border-border bg-card">
      <BotChatDemo />
    </div>
  </div>
</section>
```

不渲染第三个 CTA，不使用 ShinyText 或彩色渐变。

- [x] **步骤 6：运行定向测试，确认 GREEN**

运行：

```bash
pnpm test -- src/__tests__/navigation.test.tsx src/__tests__/Hero.test.tsx
```

预期：PASS；排除项、有效锚点、菜单闭合和 Hero 目标均被锁定。

- [x] **步骤 7：提交本任务**

```bash
git add web/landing/src/components/layout/Header.tsx web/landing/src/components/layout/MobileMenu.tsx web/landing/src/components/layout/Footer.tsx web/landing/src/components/sections/HeroSection.tsx web/landing/src/__tests__/navigation.test.tsx web/landing/src/__tests__/Hero.test.tsx
git commit -m "feat(landing): rebuild page shell navigation and hero"
```

### 任务 4：重排三组功能演示并保持加载与状态边界

**文件：**

- 新增：`web/landing/src/components/sections/FeatureShowcase.tsx`
- 新增：`web/landing/src/components/sections/FeatureShowcaseSection.tsx`
- 新增：`web/landing/src/__tests__/FeatureShowcase.test.tsx`
- 修改：`web/landing/src/components/demos/LazyDemos.tsx`
- 修改：`web/landing/src/components/demos/BotChatDemo.tsx`
- 修改：`web/landing/src/components/demos/CaseDirDemo.tsx`
- 修改：`web/landing/src/components/demos/AdminDemo.tsx`
- 删除：`web/landing/src/components/sections/FeaturesSection.tsx`
- 删除：`web/landing/src/components/sections/FeatureBlock.tsx`

**接口：**

- 产出：`FeatureShowcase({ title, description, reverse?, children })`，DOM 始终先说明后演示，只在宽屏用 CSS order 反转视觉顺序。
- 消费：`LazyBotChat`、`LazyCaseDir`、`LazyAdmin` 的公开函数名不变。
- 保证：Hero Bot 与功能区 Bot 是独立 React 实例；Case 与后台 mock 数据和操作不变。

- [x] **步骤 1：写三组顺序和两份 Bot 独立状态失败测试**

`FeatureShowcase.test.tsx`：

```tsx
it("按 Bot、Case、后台顺序渲染三组说明与演示", async () => {
  render(<FeatureShowcaseSection />);
  const headings = screen.getAllByRole("heading", { level: 3 });
  expect(headings.map((heading) => heading.textContent)).toEqual([
    zhCN["features.bot.title"],
    zhCN["features.case.title"],
    zhCN["features.admin.title"],
  ]);
  expect(await screen.findByText("都市夜景工作流")).toBeInTheDocument();
  expect(await screen.findByText(/节点 1/)).toBeInTheDocument();
});

it("首屏 Bot 与功能区 Bot 的前进状态互不影响", async () => {
  render(
    <>
      <HeroSection />
      <FeatureShowcaseSection />
    </>,
  );
  const nextButtons = await screen.findAllByRole("button", {
    name: /前进|Next/i,
  });
  fireEvent.click(nextButtons[0]);
  expect(screen.getAllByText(/赛博朋克/)).toHaveLength(2);
  expect(screen.getAllByText(/GPU 节点 1/)).toHaveLength(1);
});
```

- [x] **步骤 2：写模块复用、动态 import 与稳定占位契约**

```tsx
it("功能区演示继续由 LazyDemos 动态加载", () => {
  const source = fs.readFileSync(
    path.resolve(import.meta.dirname, "../components/demos/LazyDemos.tsx"),
    "utf8",
  );
  expect(source).toContain("lazy(() =>");
  expect(source).toContain(
    'import { BotChatDemo } from "@/components/demos/BotChatDemo"',
  );
  expect(source).not.toContain('import("@/components/demos/BotChatDemo")');
  expect(source).toContain('import("@/components/demos/CaseDirDemo")');
  expect(source).toContain('import("@/components/demos/AdminDemo")');
  expect(source).toContain("min-h-");
});
```

- [x] **步骤 3：运行功能测试，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/FeatureShowcase.test.tsx src/__tests__/BotChatDemo.test.tsx src/__tests__/CaseDirDemo.test.tsx src/__tests__/AdminDemo.test.tsx
```

预期：新测试 FAIL；三个既有演示测试继续 PASS。

- [x] **步骤 4：实现统一 Showcase 与稳定占位**

`FeatureShowcase` 使用：

```tsx
<article className="grid items-center gap-8 lg:grid-cols-2 lg:gap-12">
  <div className={cn("flex min-w-0 flex-col gap-4", reverse && "lg:order-2")}>
    <h3 className="text-2xl font-semibold tracking-tight">{title}</h3>
    <p className="max-w-lg leading-7 text-muted-foreground">{description}</p>
  </div>
  <div
    className={cn(
      "min-h-80 min-w-0 overflow-hidden rounded-xl border border-border bg-card",
      reverse && "lg:order-1",
    )}
  >
    {children}
  </div>
</article>
```

`FeatureShowcaseSection` 固定 Bot、Case、后台顺序，第二项 `reverse`。三项都从 `LazyDemos` 引用；Bot 直接复用首屏模块，Case 与后台占位使用相同 `min-h-80` 和 `aria-hidden`，不得用网络资源。动态组件若抛错只影响该演示容器，不能包住整个页面。

- [x] **步骤 5：只适配演示语义类和输入状态**

为 Bot、Case、后台演示补 `focus-visible`、44px 按钮命中区和 reduced-motion 判断；Case 动画只保留 opacity/y，不保留横向 x。保留 `chatMessages`、`cases`、`adminTasks` 数据、按钮名称、选择行为和组件签名。

- [x] **步骤 6：运行演示回归，确认 GREEN**

运行：

```bash
pnpm test -- src/__tests__/FeatureShowcase.test.tsx src/__tests__/BotChatDemo.test.tsx src/__tests__/CaseDirDemo.test.tsx src/__tests__/AdminDemo.test.tsx
```

预期：PASS；两个 Bot 状态互不影响，Case 与后台行为无回归。

- [x] **步骤 7：提交本任务**

```bash
git add web/landing/src/components/sections/FeatureShowcase.tsx web/landing/src/components/sections/FeatureShowcaseSection.tsx web/landing/src/components/demos web/landing/src/__tests__/FeatureShowcase.test.tsx
git rm web/landing/src/components/sections/FeaturesSection.tsx web/landing/src/components/sections/FeatureBlock.tsx
git commit -m "feat(landing): present interactive demos as feature showcases"
```

### 任务 5：增加跨设备与适用人群区

**文件：**

- 新增：`web/landing/src/components/sections/CrossDeviceSection.tsx`
- 新增：`web/landing/src/components/sections/AudienceSection.tsx`
- 新增：`web/landing/src/__tests__/ContentSections.test.tsx`
- 删除：`web/landing/src/components/sections/UseCasesSection.tsx`
- 删除：`web/landing/src/__tests__/UseCases.test.tsx`

**接口：**

- 消费：`audienceIds` 与 `crossDevice.*` / `audience.*` i18n key。
- 产出：`#devices` 与 `#audience` 两个稳定 section ID。
- 保证：内容只描述 Telegram 移动端入口和 Pixoma 桌面网页，不出现原生移动客户端、虚构统计、客户 logo、价格或未实现能力。

- [x] **步骤 1：写双语事实范围失败测试**

`ContentSections.test.tsx`：

```tsx
it.each([
  ["zh-CN", /Telegram/, /Case/, /计算节点/],
  ["en", /Telegram/, /Case/, /compute nodes/i],
])(
  "跨设备区在 %s 下说明手机发起和桌面管理",
  async (locale, telegram, cases, nodes) => {
    await i18n.changeLanguage(locale);
    render(<CrossDeviceSection />);
    expect(screen.getByText(telegram)).toBeInTheDocument();
    expect(screen.getByText(cases)).toBeInTheDocument();
    expect(screen.getByText(nodes)).toBeInTheDocument();
    expect(
      screen.queryByText(/原生客户端|native mobile app/i),
    ).not.toBeInTheDocument();
  },
);

it("适用人群区渲染三类独立标题、收益和用法", () => {
  render(<AudienceSection />);
  const zh: Record<string, string> = zhCN;
  for (const id of audienceIds) {
    expect(
      screen.getByRole("heading", { name: zh[`audience.${id}.title`] }),
    ).toBeInTheDocument();
    expect(screen.getByText(zh[`audience.${id}.benefit`])).toBeInTheDocument();
    expect(screen.getByText(zh[`audience.${id}.usage`])).toBeInTheDocument();
  }
});
```

- [x] **步骤 2：运行内容测试，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/ContentSections.test.tsx
```

预期：FAIL，两个新区块尚不存在。

- [x] **步骤 3：实现 Cross Device**

用 `SectionShell id="devices"` 包裹两块同级表面；第一块从 `crossDevice.mobile.*` 读取 Telegram / 手机文案，第二块从 `crossDevice.desktop.*` 读取 Pixoma / 桌面管理文案。用 CSS 边框、留白和箭头关系表达流程，不使用远程素材，不写“客户端”承诺。

- [x] **步骤 4：实现 Audience**

用 `SectionShell id="audience"` 和 `audienceIds.map` 渲染三张卡片：

```tsx
<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
  {audienceIds.map((id) => (
    <article
      key={id}
      className="flex flex-col gap-4 rounded-xl border border-border bg-card p-6"
    >
      <h3 className="text-xl font-semibold">{t(`audience.${id}.title`)}</h3>
      <p className="text-foreground">{t(`audience.${id}.benefit`)}</p>
      <p className="text-sm leading-6 text-muted-foreground">
        {t(`audience.${id}.usage`)}
      </p>
    </article>
  ))}
</div>
```

- [x] **步骤 5：删除旧 Use Cases 模型并运行 GREEN**

运行：

```bash
pnpm test -- src/__tests__/ContentSections.test.tsx src/__tests__/app.integration.test.tsx
```

预期：PASS；旧 `#scenarios` 区和测试已移除，页面只保留 canonical spec 的两块内容。

- [x] **步骤 6：提交本任务**

```bash
git add web/landing/src/components/sections/CrossDeviceSection.tsx web/landing/src/components/sections/AudienceSection.tsx web/landing/src/__tests__/ContentSections.test.tsx
git rm web/landing/src/components/sections/UseCasesSection.tsx web/landing/src/__tests__/UseCases.test.tsx
git commit -m "feat(landing): explain cross-device use and audiences"
```

### 任务 6：实现 FAQ、最终 CTA 与页尾

**文件：**

- 新增：`web/landing/src/components/sections/FaqSection.tsx`
- 新增：`web/landing/src/__tests__/Faq.test.tsx`
- 修改：`web/landing/package.json`
- 修改：`web/landing/pnpm-lock.yaml`
- 修改：`web/landing/src/components/layout/LangLayout.tsx`
- 修改：`web/landing/src/components/sections/CTASection.tsx`
- 修改：`web/landing/src/components/layout/Footer.tsx`
- 修改：`web/landing/src/__tests__/app.integration.test.tsx`
- 修改：`web/landing/src/__tests__/CTA.test.tsx`

**接口：**

- 消费：shadcn `Accordion` 系列、`faqIds`、`navigationItems`、`site`。
- 产出：FAQ 使用 `type="single" collapsible`，默认 `defaultValue="install"`；允许全部收起。
- 产出：八个 landmark 可由 `header`、`main > section[id]`、`footer` 和稳定测试 ID 查询。
- 保证：Hero 与 Final CTA 都从同一 `site` 配置取值；Footer 只含品牌、共享导航、语言切换与版权。

- [x] **步骤 1：安装键盘交互测试依赖**

```bash
pnpm add -D @testing-library/user-event
```

预期：只更新 `package.json` 与 `pnpm-lock.yaml`，测试可以使用真实的 Tab、Enter 和 Space 交互序列。

- [x] **步骤 2：写 FAQ 鼠标、键盘、ARIA 与焦点失败测试**

`Faq.test.tsx`：

```tsx
import userEvent from "@testing-library/user-event";

it("点击同一标题可展开和收起并保持焦点", async () => {
  const user = userEvent.setup();
  render(<FaqSection />);
  const trigger = screen.getByRole("button", {
    name: zhCN["faq.install.question"],
  });
  await user.click(trigger);
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  await user.click(trigger);
  expect(trigger).toHaveAttribute("aria-expanded", "true");
  expect(trigger).toHaveAttribute("aria-controls");
  expect(
    document.getElementById(trigger.getAttribute("aria-controls")!),
  ).toBeInTheDocument();
  expect(trigger).toHaveFocus();
});

it.each(["{Enter}", " "])("%s 激活 FAQ 得到与鼠标一致的结果", async (key) => {
  const user = userEvent.setup();
  render(<FaqSection />);
  const trigger = screen.getByRole("button", {
    name: zhCN["faq.telegram.question"],
  });
  await user.click(trigger);
  await user.click(trigger);
  await user.keyboard(key);
  expect(trigger).toHaveAttribute("aria-expanded", "true");
  expect(trigger).toHaveFocus();
});
```

- [x] **步骤 3：写两处 CTA 配置目标失败测试**

`CTA.test.tsx`：

```tsx
it.each([
  ["HeroSection", <HeroSection />],
  ["CTASection", <CTASection />],
])("%s 使用 site 中的下载和自托管地址", (_name, section) => {
  render(section);
  expect(screen.getByRole("link", { name: /下载|Download/i })).toHaveAttribute(
    "href",
    site.downloadUrl,
  );
  expect(
    screen.getByRole("link", { name: /自托管|Self-host/i }),
  ).toHaveAttribute("href", site.selfHostUrl);
});
```

- [x] **步骤 4：写八段顺序、排除项和非法语言失败测试**

在 `app.integration.test.tsx` 增加：

```tsx
it("按确认顺序渲染八个顶层区域且不出现排除区", () => {
  render(
    <MemoryRouter initialEntries={["/cn"]}>
      <App />
    </MemoryRouter>,
  );
  const ids = [
    "site-header",
    "hero",
    "features",
    "devices",
    "audience",
    "faq",
    "download",
    "site-footer",
  ];
  const nodes = ids.map((id) => screen.getByTestId(id));
  nodes.slice(1).forEach((node, index) => {
    expect(
      nodes[index].compareDocumentPosition(node) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });
  expect(
    screen.queryByText(
      /Works with your favorite tools|Affordable for everyone/i,
    ),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("link", { name: /tools|pricing|价格|工具集成/i }),
  ).not.toBeInTheDocument();
});

it("非法语言路径重定向到中文", async () => {
  render(
    <MemoryRouter initialEntries={["/invalid"]}>
      <App />
    </MemoryRouter>,
  );
  expect(await screen.findByRole("heading", { level: 1 })).toHaveTextContent(
    zhCN["hero.title"],
  );
});
```

- [x] **步骤 5：运行 FAQ、CTA 与页面集成测试，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/Faq.test.tsx src/__tests__/CTA.test.tsx src/__tests__/app.integration.test.tsx
```

预期：FAQ 与八段页面测试 FAIL；CTA 双目标测试 PASS，并在后续重构中持续保护真实地址。

- [x] **步骤 6：实现 FAQ**

```tsx
<SectionShell id="faq" title={t("faq.title")}>
  <Accordion type="single" collapsible defaultValue="install">
    {faqIds.map((id) => (
      <AccordionItem key={id} value={id}>
        <AccordionTrigger>{t(`faq.${id}.question`)}</AccordionTrigger>
        <AccordionContent>{t(`faq.${id}.answer`)}</AccordionContent>
      </AccordionItem>
    ))}
  </Accordion>
</SectionShell>
```

保留 primitive 自带标题按钮、`aria-expanded`、`aria-controls`、Enter/Space 和焦点行为；只用语义类调整边框、焦点和 reduced-motion 过渡。

- [x] **步骤 7：组装八段页面并收口 CTA 与 Footer**

`LangLayout` 保留既有语言副作用与非法路径重定向，把页面组装为：

```tsx
return (
  <div className="flex min-h-svh flex-col bg-background text-foreground">
    <Header />
    <main className="flex-1">
      <HeroSection />
      <FeatureShowcaseSection />
      <CrossDeviceSection />
      <AudienceSection />
      <FaqSection />
      <CTASection />
    </main>
    <Footer />
  </div>
);
```

CTA 使用 `id="download"`、统一 SectionShell 节奏、下载 primary 和自托管 outline。Footer 从 `navigationItems` 渲染全部五项锚点，并保留 `LangSwitch` 与版权；不增加“为什么没有价格/工具”类说明。

- [x] **步骤 8：运行定向测试，确认 GREEN**

运行：

```bash
pnpm test -- src/__tests__/Faq.test.tsx src/__tests__/CTA.test.tsx src/__tests__/navigation.test.tsx src/__tests__/app.integration.test.tsx
```

预期：PASS；八段顺序和非法语言回退成立；FAQ 可通过鼠标与键盘展开/收起，状态和内容关联正确，收起后焦点不丢；两处 CTA 都使用真实配置目标。

- [x] **步骤 9：提交本任务**

```bash
git add web/landing/package.json web/landing/pnpm-lock.yaml web/landing/src/components/layout/LangLayout.tsx web/landing/src/components/layout/Footer.tsx web/landing/src/components/sections/FaqSection.tsx web/landing/src/components/sections/CTASection.tsx web/landing/src/__tests__/Faq.test.tsx web/landing/src/__tests__/CTA.test.tsx web/landing/src/__tests__/app.integration.test.tsx
git commit -m "feat(landing): add accessible FAQ and final actions"
```

### 任务 7：收口响应式、键盘与减少动态效果

**文件：**

- 修改：`web/landing/src/__tests__/Reveal.test.tsx`
- 修改：`web/landing/src/__tests__/app.integration.test.tsx`
- 修改：`web/landing/src/components/motion/Reveal.tsx`
- 修改：`web/landing/src/components/motion/motionPresets.ts`
- 修改：`web/landing/src/components/layout/Header.tsx`
- 修改：`web/landing/src/components/layout/MobileMenu.tsx`
- 修改：`web/landing/src/components/sections/*.tsx`
- 修改：`web/landing/src/styles/index.css`
- 删除：`web/landing/src/components/reactbits/ShinyText.tsx`
- 删除：`web/landing/src/components/reactbits/GradientBackground.tsx`

**接口：**

- 保持：`Reveal({ children, className?, delay? })`。
- 保证：reduced motion 时 `initial={false}` 且不等待 `useInView`；默认只使用 opacity/y。
- 保证：所有交互控件可聚焦、可激活、焦点可见，最小命中区 44px。

- [x] **步骤 1：强化 reduced-motion 失败测试**

`Reveal.test.tsx`：

```tsx
it("reduced motion 下立即显示且不传入移动过渡", () => {
  render(<Reveal delay={0.4}>立即可见</Reveal>);
  const content = screen.getByText("立即可见").parentElement;
  expect(content).toBeVisible();
  expect(content).not.toHaveStyle("opacity: 0");
  expect(content).not.toHaveStyle("transform: translateY(16px)");
});
```

同时在 `app.integration.test.tsx` 增加源码契约：

```tsx
it("页面组件不引入裸颜色、手工暗色或横向最小宽度", () => {
  const sources = fs
    .readdirSync(path.resolve(import.meta.dirname, "../components"), {
      recursive: true,
    })
    .filter((name) => typeof name === "string" && name.endsWith(".tsx"))
    .map((name) =>
      fs.readFileSync(
        path.resolve(import.meta.dirname, "../components", name),
        "utf8",
      ),
    )
    .join("\n");
  expect(sources).not.toMatch(/#[0-9a-f]{3,8}\b/i);
  expect(sources).not.toMatch(/\bdark:/);
  expect(sources).not.toMatch(/\bmin-w-\[[1-9]/);
});
```

- [x] **步骤 2：运行动效与契约测试，确认 RED**

运行：

```bash
pnpm test -- src/__tests__/Reveal.test.tsx src/__tests__/app.integration.test.tsx
```

预期：当前内容断言 PASS；新增源码契约先因遗留装饰组件或样式失败。测试继续使用现有 `useReducedMotion: true` mock，不等待 IntersectionObserver。

- [x] **步骤 3：实现动效降级与清理循环装饰**

`Reveal` 保留：

```tsx
const inView = useInView(ref, { once: true, margin: "-80px" });
const reduced = useReducedMotion();

<motion.div
  ref={ref}
  className={cn(className)}
  variants={fadeUp}
  initial={reduced ? false : "hidden"}
  animate={reduced ? "visible" : inView ? "visible" : "hidden"}
  transition={
    reduced ? { duration: 0 } : { duration: 0.5, delay, ease: "easeOut" }
  }
>
  {children}
</motion.div>;
```

`fadeUp` 只含 `{ opacity, y }`。删除所有横向 x 入场；Hero 已不引用 `ShinyText` 与 `GradientBackground`，用 `git rm` 删除两文件，并移除 `.animate-shiny` 与对应 keyframes。

- [x] **步骤 4：逐断点检查布局和输入方式**

运行开发服务器：

```bash
pnpm dev
```

人工检查 `/cn` 与 `/en` 的 320px、768px、920px、1440px：

- 320px：标题、CTA、三组演示、三张 Audience 卡、FAQ 不被裁切，页面无横向滚动。
- 768px / 920px：Audience 从一列到两列再到三列，Showcase DOM 顺序仍是说明在演示前。
- 1440px：Hero 双栏，Showcase 奇偶交替，最大宽度和正文行宽稳定。
- 键盘：依次到达 Header 导航、语言切换、菜单、Hero CTA、演示按钮、FAQ、Final CTA、Footer；每个控件都有可见焦点。
- 触控：菜单、语言切换、演示按钮、Accordion trigger 和 CTA 命中区至少 44×44 CSS 像素。
- reduced motion：系统偏好打开后内容无需滚入视口即可出现，菜单/FAQ 仍可操作，无循环动效。

- [x] **步骤 5：完成 CSS 与语义收口**

修正只限于语义类、`overflow-hidden`、`min-w-0`、grid/flex 断点和焦点规则；不使用固定大 `min-width` 撑开演示。检查 H1 仅一个、各区 H2、Showcase/Audience H3，Header/main/footer landmark 正确。

- [x] **步骤 6：运行定向与全量测试，确认 GREEN**

运行：

```bash
pnpm test -- src/__tests__/Reveal.test.tsx src/__tests__/app.integration.test.tsx src/__tests__/navigation.test.tsx src/__tests__/Faq.test.tsx
pnpm test
```

预期：PASS；内容顺序、操作结果在默认与 reduced-motion 下相同，既有演示测试无回归。

- [x] **步骤 7：提交本任务**

```bash
git add web/landing/src
git commit -m "fix(landing): finish responsive accessible motion behavior"
```

### 任务 8：全量验证与范围审计

**文件：**

- 检查：`web/landing/**`
- 检查：`docs/openspec/changes/heyo-inspired-landing-refactor/tasks.md`
- 检查：`docs/openspec/changes/heyo-inspired-landing-refactor/specs/landing-page-experience/spec.md`

**接口：**

- 不新增运行时接口。
- 产出：test、lint、format、production build 和最终 diff 的完整验证证据。

- [x] **步骤 1：运行全量自动验证**

```bash
cd web/landing
pnpm test
pnpm lint
pnpm format:check
pnpm build
```

预期：四条命令全部 PASS；Vite 产出静态 `dist`，页面渲染与交互不依赖 Pixoma 后端 API。

- [x] **步骤 2：检查静态产物和惰性分包**

```bash
rg -n "api/|fetch\\(|axios|WebSocket" src
rg -n "BotChatDemo|CaseDirDemo|AdminDemo" dist/assets
```

预期：新增区块无后端请求；构建产物包含首屏共享 Bot 模块与功能区动态演示 chunk，Case 和后台未被强制并入首屏入口。若 Vite 压缩改名导致第二条无文本命中，以 `vite build` 的 chunk 列表和 `LazyDemos.tsx` 动态 import 契约测试为证。

- [x] **步骤 3：扫描禁止项和临时文件**

```bash
rg -n "Works with your favorite tools|Affordable for everyone|pricing|#tools|#pricing" src
rg -n "#[0-9A-Fa-f]{3,8}\\b|slate-|dark:|shadow-(sm|md|lg|xl|2xl)" src/components
rg -n "请|您|前往|进行|完成|实施|！|!" src/i18n/locales/zh-CN.ts
git status --short
```

预期：前三条无命中；`git status` 不包含 `*.png`、截图测试、浏览器配置、临时视觉测试或 change 外文件。Accordion 若作为浮层 primitive 自带 shadow，只允许在浮层，不允许落到卡片、按钮或演示表面。

- [x] **步骤 4：逐项映射 tasks.md 与 canonical spec**

核对：

```text
1.1 -> 任务 1 步骤 1：test、lint、build 基线
1.2 -> 任务 6 步骤 4：八段顺序与排除项测试
1.3 -> 任务 1 步骤 2：新增 key 双语对等测试
1.4 -> 任务 3 步骤 1：桌面、移动、页尾有效锚点与菜单闭合
1.5 -> 任务 6 步骤 2、6：FAQ 鼠标、键盘、ARIA、焦点
1.6 -> 任务 3 步骤 2、任务 6 步骤 3、任务 7 步骤 1：CTA 与 reduced-motion
2.1 -> 任务 2 步骤 6：语义颜色、字体、圆角、边框、背景、焦点与响应式基础
2.2 -> 任务 2 步骤 6：SectionShell 与 Button 共用视觉
2.3 -> 任务 7 步骤 3：Reveal 仅 opacity/y 且 reduced-motion 直出
2.4 -> 任务 2 步骤 3–5：shadcn 检查、预览、Accordion 添加
3.1 -> 任务 6 步骤 4、7：八段测试与 LangLayout 固定组装
3.2 -> 任务 1 步骤 4、任务 3 步骤 4：共享 navigationItems
3.3 -> 任务 3 步骤 1、4：Header/MobileMenu 响应式、44px 与闭合
3.4 -> 任务 3 步骤 2、5：Hero 价值、双 CTA、完整 Bot 演示
4.1 -> 任务 4 步骤 1、4：Bot、Case、后台三段交替布局
4.2 -> 任务 4 步骤 2、4、5：Bot 模块复用、Case/后台动态 import、稳定占位与原行为
4.3 -> 任务 4 步骤 4、任务 7 步骤 4–5：窄屏文案先于演示且无横溢
4.4 -> 任务 4 步骤 3、6：既有三组演示回归
5.1 -> 任务 5 步骤 1、3：Telegram 手机发起与桌面管理
5.2 -> 任务 5 步骤 1、4：三类人群、收益和使用方式
5.3 -> 任务 6 步骤 2、6、8：可访问 Accordion
5.4 -> 任务 6 步骤 3、7、8：Final CTA 与 Footer
5.5 -> 任务 1 步骤 2、4、5 与任务 8 步骤 3：双语文案和中文规则
6.1 -> 任务 7 步骤 4–5：320px、平板、桌面与横向溢出
6.2 -> 任务 7 步骤 4–5：标题、landmark、语义、焦点、44px
6.3 -> 任务 7 步骤 1–3、6：默认与 reduced-motion 路径
6.4 -> 任务 8 步骤 4：canonical spec 逐条证据
7.1 -> 任务 8 步骤 1：全量 test
7.2 -> 任务 8 步骤 1：lint 与 format:check
7.3 -> 任务 8 步骤 1–2：production build、静态运行时与分包
7.4 -> 任务 8 步骤 3、5：最终 diff 与禁止脚手架审计
```

确认 canonical spec 的每个 Requirement 至少有一个自动测试或明确人工检查证据；尤其检查排除区、原创 Pixoma 叙事、三组演示、有效锚点、非法语言、320px、键盘、reduced motion、静态构建。

- [x] **步骤 5：检查最终 diff 只在范围内**

```bash
git diff --check
git diff --stat d18317afcc0322f1a77abd3bed49982b530438c1 -- web/landing docs/openspec/changes/heyo-inspired-landing-refactor
git diff --name-only d18317afcc0322f1a77abd3bed49982b530438c1
```

预期：`git diff --check` 无输出；业务改动只位于 `web/landing`，change 文档只在需要勾选任务或记录验证证据时变化；不触及 admin、API、数据库、部署拓扑或 `docs/architecture/`。

- [x] **步骤 6：提交验证收口**

仅当格式化或验证修复产生文件变化时提交：

```bash
git add web/landing docs/openspec/changes/heyo-inspired-landing-refactor
git commit -m "chore(landing): close verification and scope checks"
```

若无文件变化，不创建空提交。

---

## 自审结果

- **Spec 覆盖：** 八段顺序、排除区、Pixoma 原创叙事、三组演示、按需加载、跨设备、三类人群、FAQ 三种输入、有效导航、双 CTA、移动菜单闭合、双语与非法语言、320px/键盘/44px、reduced motion、静态构建均已映射到任务和验证。
- **占位扫描：** 计划没有未定义实现项；所有新增文件、稳定 ID、公开接口、测试目标和命令已写明。
- **类型一致性：** `navigationItems`、`audienceIds`、`faqIds`、`FeatureShowcase`、`MobileMenu`、`Reveal`、`LazyDemos` 的名称和签名在前后任务一致。
- **边界确认：** 不修改后端、后台、数据模型、部署拓扑或架构文档；不新增工具集成区、价格区、远程媒体和截图测试脚手架。

## 实施交接

计划保存在 `docs/superpowers/plans/2026-08-31-heyo-inspired-landing-refactor.md`。

执行方式：

1. **Subagent-Driven（推荐）**：每个任务派发新的子代理，两阶段审查后进入下一任务。
2. **Inline Execution**：当前会话使用 `executing-plans`，分批实施并在检查点审查。
