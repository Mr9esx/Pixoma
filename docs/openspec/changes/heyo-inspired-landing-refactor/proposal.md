## Why

当前 `web/landing` 已能展示 Pixoma 的基础价值与交互演示，但页面节奏、内容层次和转化路径较弱。参考 Heyo 落地页的结构重做视觉与信息架构，可以在不改变 Pixoma 产品定位的前提下，让访客更快理解产品、找到适合自己的使用方式，并进入下载或自托管入口。

## What Changes

- 以 Heyo 当前落地页为视觉和布局参考，结构化重建 `web/landing`，保留 Pixoma 品牌、产品叙事与现有中英文能力。
- 页面按导航、首屏、功能展示、跨设备使用、适用人群、FAQ、最终 CTA、页尾的顺序呈现。
- 保留 Bot、Case 目录、管理后台三组现有交互演示，并按新页面节奏重新编排。
- 新增 Pixoma 语境下的跨设备、适用人群与 FAQ 内容；所有面向访客的文案继续覆盖 `zh-CN` 与 `en`。
- 重做 landing 专用视觉变量、排版、容器、卡片、按钮、分区背景、进入视口动效和响应式布局；系统要求减少动态效果时关闭非必要动画。
- 更新导航锚点、移动菜单、FAQ 键盘交互、CTA、双语与页面区块顺序测试。
- 明确移除参考页中的「Works with your favorite tools」工具集成区和「Affordable for everyone!」价格区，也不增加对应导航入口。
- 不照搬 Heyo 品牌、文案、产品素材或产品能力，不改变下载与自托管目标地址。

## Capabilities

### New Capabilities

- `landing-page-experience`: Pixoma 对外落地页的新信息架构、交互演示、跨设备与适用人群内容、FAQ、CTA、双语、无障碍和响应式行为。

### Modified Capabilities

- 无。主规格目录中尚无 landing capability；本 change 以新 capability 记录重构后的完整行为契约。

## Impact

- 主要影响 `web/landing/src` 中的页面布局、区块组件、landing 专用 UI 组件、i18n 词条、样式与测试。
- 可能按项目约定为 landing 补充必要的 shadcn/ui 源码组件；沿用现有 React 19、Vite 8、Tailwind CSS v4、Motion、React Router 与 i18next 工程栈。
- 不涉及 `web/admin`、Go 服务、数据库、API、ComfyUI 执行链路、部署拓扑或架构边界。
- 参考站点可能继续改版；实现与验收以本 change 记录的页面顺序、视觉特征和行为要求为准。
