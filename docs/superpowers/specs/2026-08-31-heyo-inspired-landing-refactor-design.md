---
comet_change: heyo-inspired-landing-refactor
role: technical-design
canonical_spec: openspec
---

# Heyo 风格 Landing Page 重构技术设计

## 1. 设计目标

在现有 `web/landing` 内重建页面结构与视觉节奏。页面保留 Pixoma 品牌、双语路由、下载与自托管入口，以及 Bot、Case 目录、管理后台三组交互演示；参考 Heyo 的信息密度、区块节奏、产品演示组织和响应式布局，不复制其品牌、文案、素材或代码。

需求事实源为：

- `docs/openspec/changes/heyo-inspired-landing-refactor/specs/landing-page-experience/spec.md`
- `docs/openspec/changes/heyo-inspired-landing-refactor/proposal.md`

本设计不增加工具集成区或价格区，也不改变后台、API、数据模型与部署拓扑。

## 2. 当前实现约束

`web/landing` 是独立的 Vite 8 + React 19 + TypeScript + Tailwind CSS v4 SPA：

- `LangLayout` 负责语言同步并直接组装页面区块。
- `/cn`、`/en` 由 React Router 与 i18next 驱动。
- `BotChatDemo`、`CaseDirDemo`、`AdminDemo` 使用本地数据。
- `LazyDemos.tsx` 通过 `React.lazy` 与 `Suspense` 延迟加载三组演示。
- `Reveal` 已使用 Motion 的 `useInView` 与 `useReducedMotion`。
- `web/landing` 没有 `components.json`，现有按钮是本地 reui 风格组件。

重构应沿用这些工程边界，不把 landing 并入 `web/admin`。

## 3. 目标组件结构

```text
AppRoutes
└── LangLayout
    ├── Header
    │   ├── DesktopNavigation
    │   ├── LangSwitch
    │   └── MobileMenu
    ├── main
    │   ├── HeroSection
    │   │   ├── HeroCopy
    │   │   ├── HeroActions
    │   │   └── BotChatDemo
    │   ├── FeatureShowcaseSection
    │   │   ├── FeatureShowcase(Bot, LazyBotChat)
    │   │   ├── FeatureShowcase(Case, LazyCaseDir)
    │   │   └── FeatureShowcase(Admin, LazyAdmin)
    │   ├── CrossDeviceSection
    │   ├── AudienceSection
    │   ├── FaqSection
    │   └── CTASection
    └── Footer
```

`LangLayout` 只保留语言副作用、非法语言重定向与顶层顺序。区块内部状态不提升到页面根节点，避免一个演示更新导致整页重渲染。

## 4. 文件边界

### 4.1 页面与区块

- `components/layout/LangLayout.tsx`：顶层区块顺序。
- `components/layout/Header.tsx`：桌面导航、语言切换、移动菜单开关。
- `components/layout/MobileMenu.tsx`：窄屏导航；选中锚点后闭合。
- `components/layout/Footer.tsx`：品牌、共享导航、语言切换与版权。
- `components/sections/HeroSection.tsx`：价值主张、CTA 与首屏 Bot 演示。
- `components/sections/FeatureShowcaseSection.tsx`：三组功能演示的顺序与布局。
- `components/sections/FeatureShowcase.tsx`：单个“文案 + 演示”结构。
- `components/sections/CrossDeviceSection.tsx`：移动端发起、桌面端管理。
- `components/sections/AudienceSection.tsx`：三类适用人群。
- `components/sections/FaqSection.tsx`：双语问答数据与 Accordion 组合。
- `components/sections/CTASection.tsx`：最终下载与自托管入口。

旧 `FeaturesSection.tsx`、`FeatureBlock.tsx`、`UseCasesSection.tsx` 在新结构稳定后删除或改名迁移，不保留两套并行页面模型。

### 4.2 配置与数据

新增 `constants/navigation.ts`，导出只读导航项：

```text
features  -> #features
devices   -> #devices
audience  -> #audience
faq       -> #faq
download  -> #download
```

每项只保存稳定 ID 与 i18n key。Header、MobileMenu、Footer 从同一数组渲染；工具集成与价格不进入该配置。

适用人群与 FAQ 使用稳定 key 列表，不在组件内保存中英文对象：

```text
audienceIds = creator | team | multiNode
faqIds = install | comfyOwnership | telegram | selfHost | platforms
```

组件用这些 ID 拼接 i18n key。这样两种语言共享结构，测试可以枚举相同 ID 检查词条完整性。

## 5. 页面区块设计

### 5.1 Header

- 默认位于页面顶部，滚动时保持可见。
- 宽屏显示品牌、共享导航、语言切换。
- 窄屏显示品牌、语言切换和菜单按钮；菜单使用已有结构重构，不引入第二套路由。
- 锚点链接直接指向当前页面区块，不改语言路径。
- 区块设置 `scroll-margin-top`，避免 sticky Header 遮住标题。
- 菜单项激活后先关闭菜单，再交给浏览器处理锚点定位。

### 5.2 Hero

首屏采用宽屏双栏、窄屏单栏：

```text
宽屏: 文案与 CTA | 完整 BotChatDemo
窄屏: 文案与 CTA
      完整 BotChatDemo
```

Hero 直接静态导入 `BotChatDemo`，确保首屏出现完整交互，不经过视口惰性门槛。功能区仍渲染 `LazyBotChat`，两个实例各自持有 `visible` 状态，互不联动。

首屏只保留一个视觉主 CTA。下载是 primary，自托管是 outline。平台信息作为短 meta 文本，不做第三个按钮。

### 5.3 Feature Showcase

三组演示顺序固定为 Bot、Case、后台：

- 宽屏奇数项“文案左、演示右”，偶数项反转。
- 窄屏全部“文案上、演示下”，保持 DOM 阅读顺序一致。
- 每组使用统一演示外壳，控制边框、圆角、最小高度和 overflow。
- Bot 功能区继续使用 `LazyBotChat` 接口，但直接复用首屏已加载的 `BotChatDemo` 模块；Case 与后台先由 `IntersectionObserver` 判断接近视口，再挂载原惰性组件。
- 占位内容必须具有稳定宽高，避免演示加载时页面跳动。
- 每组演示使用局部错误边界；动态 chunk 加载失败时只替换当前演示容器，页面其余区域继续显示。

演示内部只做语义类适配，不改变 mock 数据、按钮行为或公开接口。

### 5.4 Cross Device

该区只呈现已存在的产品关系：

```text
Telegram / 手机  -> 描述想法并发起创作
Pixoma / 桌面端  -> 管理 Case、任务与计算节点
```

视觉可组合手机与桌面框架，但框架内容来自现有 Bot、Case、后台界面语言，不宣称 Pixoma 提供原生移动客户端。

### 5.5 Audience

使用三张同级卡片：

- 个人创作者：通过 Telegram 使用自己的 ComfyUI。
- 小团队：共享 Case、任务与创作结果的组织方式。
- 多节点使用者：管理多台 GPU 与不同工作流。

卡片不包含虚构统计、客户 logo、价格或未实现能力。宽屏三列，中屏与手机逐步收为两列和单列。

### 5.6 FAQ

实现阶段使用项目包管理器运行 shadcn CLI：

1. 在 `web/landing` 检查 CLI 对 Vite、Tailwind v4、alias 与样式入口的识别结果。
2. 预览初始化与 Accordion 会修改的文件。
3. 初始化 `components.json` 时保留当前 alias 与 `src/styles/index.css`。
4. 添加 Accordion 后检查生成源码、依赖与 import。
5. 将 CLI 生成源码收窄到 `@radix-ui/react-accordion` 与现有 `lucide-react`；`shadcn` CLI 和 Radix 聚合包不进入生产依赖。
6. 将视觉样式接到 landing 语义变量，不覆盖现有页面基础样式。

Accordion 支持标题按钮、`aria-expanded`、内容关联、Enter/Space 操作和可见焦点。默认只展开一项并允许全部收起，减少长页面占用；切换语言后保留组件结构，但不要求保留已展开项。

### 5.7 CTA 与 Footer

CTA 复用 Hero 的行动入口配置，下载为 primary，自托管为 outline。Footer 只包含品牌、共享导航、语言切换与版权，不补充排除区说明。

## 6. 视觉系统

### 6.1 配色

采用已确认的黑白中性色，不使用 Heyo 色板，也不保留当前橙色主点缀。`styles/index.css` 定义 landing 语义变量：

- `--background` / `--foreground`
- `--card` / `--card-foreground`
- `--muted` / `--muted-foreground`
- `--border`
- `--primary` / `--primary-foreground`
- `--secondary` / `--secondary-foreground`
- `--ring`

实际色值只出现在变量定义中，组件使用 Tailwind 语义类。派生表面用 `color-mix(in oklch, …)`，不在 TSX 中写裸色值或手工暗色分支。

### 6.2 排版与节奏

- H1 使用流式字号，桌面突出但不把 CTA 推出首屏。
- H2 在所有区块共享同一字号、字重与最大行宽。
- 正文限制行宽，避免宽屏长行。
- 区块使用统一垂直间距；Hero、Feature、CTA 可按职责覆盖，但不各自发明一套尺度。
- 容器以现有 `max-w-6xl` 为基准；演示区可在不引起横向滚动的前提下略宽。

### 6.3 表面、背景与窗口

页面不再使用通篇白底细边框。背景系统由 CSS 类提供，组件不写裸色值：

- `.surface-dots`：Hero / Footer 点阵，`24×24px`，透明度约 `0.14`
- `.surface-grid`：跨设备区淡网格，`32×32px`
- `.surface-glow`：对角柔光伪元素，直径约 `36–40rem`
- `.product-frame`：浅灰外衬 + 内层圆角窗口 + `--shadow-window`
- `.product-stack-plate`：Hero 错位背板
- `.cta-island`：黑底大圆角行动岛
- `.faq-card`：FAQ 独立卡片

功能区改为三列嵌套卡，标题居中；跨设备用桌面窗叠手机框；Audience 为嵌套卡；CTA 黑底点阵。卡片可用窗口阴影，按钮和 FAQ 仍保持可见焦点与 44px 命中区。
- 320px 宽度下不依赖 `min-width` 撑开页面。

## 7. 动效与性能

### 7.1 动效

`Reveal` 保留 `useInView({ once: true })`，只动画 opacity 与纵向 transform。交替功能区不根据左右方向增加横向大位移，避免手机和减少动态效果场景产生方向干扰。

命中 `prefers-reduced-motion: reduce` 时：

- Reveal 初始状态直接可见。
- 去掉入场 delay 与 transform。
- 关闭 ShinyText、渐变背景等循环装饰；若这些旧组件不再符合黑白中性方向则直接移除。
- Accordion 立即切换或把过渡压缩到接近零。
- 锚点不强制平滑滚动。

### 7.2 加载

- Hero 的 Bot 演示进入首屏初始包。
- 功能区 Bot 直接复用首屏已加载模块，避免同一模块同时静态和动态导入产生无效分包；Case 与后台保留动态 import，并在距离视口 240px 时才挂载 lazy 组件。
- 新区块不引入网络图片、视频或远程数据。
- 不增加大型图标包；简单装饰优先使用 CSS 与现有资产。

首屏与功能区的 Bot 演示共用同一个静态 JS 模块，页面创建两个独立 React 实例；Case 与后台仍按需生成独立 chunk。这是已接受的交互与性能取舍。

## 8. i18n 与文案

所有新增可见文案进入：

- `i18n/locales/zh-CN.ts`
- `i18n/locales/en.ts`

组件不内联中文或英文。测试从稳定 key 列表检查两种资源：

- key 集合一致；
- 值非空；
- 页面渲染不出现原始 key；
- 新区块在两种语言下都有标题、正文与操作文案。

中文按 `docs/voice-profile.md` 编写：短句、动作起头、无感叹号、无卖萌语气，不用解释排除项。英文保持相同事实范围，不逐字翻译中文句式。

## 9. 状态与数据流

页面没有远程数据流：

```text
URL lang segment
  -> LangLayout 校验
  -> i18next language
  -> 各区块读取文案

navigation config
  -> Header
  -> MobileMenu
  -> Footer

local demo data
  -> Hero BotChatDemo state
  -> Feature BotChatDemo state
  -> CaseDirDemo state
  -> AdminDemo render
```

移动菜单的 `open` 继续由 Header 局部管理。FAQ 展开状态由 Accordion 管理。演示状态只留在各自实例内，不创建页面级 context 或状态库。

## 10. 测试设计

### 10.1 RED：先锁行为

- 页面集成测试：按 DOM 顺序断言八个区域；断言工具集成、价格文案和锚点不存在。
- 导航测试：桌面、移动、页尾链接全部命中存在的 ID；移动选择后菜单闭合。
- i18n 测试：新增 key 在中英文资源中完整且非空。
- FAQ 测试：点击、Enter、Space 展开/收起；检查 `aria-expanded` 与内容关联；收起后焦点不丢。
- CTA 测试：Hero 与 Final CTA 均使用 `site` 配置中的下载、自托管地址。
- 动效测试：mock 减少动态效果偏好，内容无需等待视口事件即可出现。
- 演示测试：Hero Bot 与功能区 Bot 状态互不影响；Case 和后台原测试继续通过。

### 10.2 GREEN：最小实现

按“共享配置与测试工具 → 顶层骨架 → Hero → 功能区 → 新区块 → FAQ → 样式”的顺序让测试逐批通过。每批只实现对应行为，不同时清理无关代码。

### 10.3 收口

在 `web/landing` 运行：

- `pnpm test`
- `pnpm lint`
- `pnpm format:check`
- `pnpm build`

不新增截图、PNG 快照、浏览器配置或临时视觉测试文件。人工浏览只用于最终桌面与手机观感核对。

## 11. 边界与失败处理

- 非法语言路径继续重定向 `/cn`。
- i18n key 缺失由测试阻断，不在 UI 增加解释文案。
- 动态演示接近视口前与加载期间显示稳定尺寸的中性占位；局部错误边界保证加载失败不会让页面其余区块消失。
- 外部下载或自托管地址只从 `constants/site` 读取，不在新组件复制 URL。
- 外部参考站后续新增、删除或改版不自动进入本 change。
- shadcn 初始化若会覆盖现有样式，停止覆盖操作，改为保留现有样式入口的最小手工合并；不得使用 `--overwrite`。

## 12. 实施顺序与回滚

实施顺序：

1. 跑基线并写行为测试。
2. 建立导航配置和语义样式基础。
3. 重组 Header、Hero 与顶层布局。
4. 重排三组演示。
5. 增加跨设备、适用人群与 FAQ。
6. 补齐双语、响应式、无障碍和动效降级。
7. 跑全量验证并清理旧区块。

本次没有数据迁移。回滚只需恢复 `web/landing` 和本 change 相关文件，不影响后台、API、数据库或部署拓扑。由于未改变模块边界、数据存储、请求链路和外部系统连接，无需更新 `docs/architecture/`。
