## 1. 锁定现状与行为测试

- [x] 1.1 在 `web/landing` 运行现有 test、lint、build，记录与本 change 无关的基线失败
- [x] 1.2 扩展页面集成测试，先锁定八个顶层区域的顺序，并断言工具集成区、价格区及对应导航不存在
- [x] 1.3 扩展 i18n 测试，先覆盖新增导航、跨设备、适用人群、FAQ 与最终 CTA 的中英文 key 对等
- [x] 1.4 为桌面导航和移动菜单补失败测试，覆盖有效锚点、选择后闭合与不存在排除区入口
- [x] 1.5 为 FAQ 补失败测试，覆盖鼠标与键盘展开/收起、状态属性、内容关联和焦点保持
- [x] 1.6 为下载、自托管入口与减少动态效果补失败测试，锁定现有目标地址和动效降级行为

## 2. 建立 landing 视觉基础

- [x] 2.1 整理 `web/landing/src/styles/index.css`，建立 landing 专用语义颜色、字体、圆角、边框、背景、焦点环和响应式变量
- [x] 2.2 重构共用区块容器、短标签、标题组和行动按钮，使页面统一消费语义类并保留完整交互状态
- [x] 2.3 统一 `Reveal` 等动效边界，只动画 opacity/transform，并在 `prefers-reduced-motion: reduce` 下立即显示内容
- [x] 2.4 核对 `web/landing` 已有 UI 组件与 shadcn 项目配置；缺少 Accordion 时用 pnpm shadcn CLI 添加并检查生成源码

## 3. 重组页面骨架与导航

- [x] 3.1 将顶层布局重组为 Header、Hero、Feature Showcase、Cross Device、Audience、FAQ、Final CTA、Footer 的固定顺序
- [x] 3.2 建立共享导航配置，让桌面导航、移动菜单和页尾复用实际存在的区块 ID 与 i18n key
- [x] 3.3 重构 Header 与 Mobile Menu，使宽屏和窄屏导航符合新结构、触控目标不小于 44×44 CSS 像素
- [x] 3.4 重构 Hero，呈现 Pixoma 价值主张、下载与自托管主次 CTA，以及不依赖后端的产品演示表面

## 4. 重排功能演示

- [x] 4.1 将功能区改为三列嵌套卡，分别组合 Bot、Case 目录、管理后台说明与现有交互演示
- [x] 4.2 让功能区 Bot 复用首屏模块，Case 与后台接近视口后再惰性加载；保留原有操作行为并适配新容器
- [x] 4.3 让三段功能演示在窄屏按说明在上、演示在下重排，消除页面级横向溢出
- [x] 4.4 运行现有演示测试并修正重排引入的回归，不改 mock 数据语义和演示交互步骤

## 5. 增加内容区块

- [x] 5.1 实现 Cross Device 区，明确 Telegram 移动端发起创作与桌面端管理能力，不宣称原生移动客户端
- [x] 5.2 实现 Audience 区，覆盖个人创作者、小团队、多计算节点使用者及各自的真实收益
- [x] 5.3 使用可访问 Accordion 实现 FAQ 区，并让鼠标、触控、键盘操作通过已写测试
- [x] 5.4 重构最终 CTA 与 Footer，沿用下载、自托管目标并只保留实际存在的区块入口
- [x] 5.5 补齐 `zh-CN` 与 `en` 文案，按 Pixoma Voice 检查中文禁用词、信息去重和中英文覆盖范围

## 6. 响应式与无障碍收口

- [x] 6.1 检查 320px 手机、平板和桌面布局，修正标题换行、卡片重排、演示尺寸与页面级横向溢出
- [x] 6.2 检查标题层级、landmark、链接与按钮语义、焦点顺序、可见焦点和触控命中区
- [x] 6.3 检查默认动效和减少动态效果两种路径，确保关闭动效后内容顺序与操作结果不变
- [x] 6.4 逐项对照 `landing-page-experience` spec，确认排除区块、双语、导航、FAQ、CTA 和静态运行时约束均有实现与测试证据

## 7. 全量验证

- [x] 7.1 运行 `web/landing` 全量 test，修复本 change 引入的失败
- [x] 7.2 运行 `web/landing` lint 与 format check，修复本 change 引入的问题
- [x] 7.3 运行 `web/landing` production build，确认 TypeScript 与 Vite 构建成功且产物不依赖后端 API
- [x] 7.4 检查最终 diff，只保留 landing 与本 change 范围内的改动，并确认未新增截图或浏览器测试脚手架
