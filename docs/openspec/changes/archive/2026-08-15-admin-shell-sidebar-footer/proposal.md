## Why

管理控制台壳子仍带 shadcn-admin 模板痕迹：侧栏标题是「Shadcn-Admin / Vite + ShadcnUI」，内容区上方另有一条仅放语言/主题的顶栏，占空间且品牌不清晰。需要把壳子收成 Pixoma 后台的样子：品牌进侧栏，工具进侧栏底栏，内容区干净铺开。

## What Changes

- 移除内容区上方的 Header（语言/主题所在顶栏）
- 将语言切换、主题切换迁到侧栏 **Footer 底栏**（方案 A）
- 侧栏品牌文案改为 **Pixoma**，去掉模板默认副标题与「Shadcn-Admin」文案
- 侧栏收起为图标模式时，底栏控件以图标形式仍可操作（tooltip/等价可发现交互）
- 不改业务资源页、不改菜单条目与顺序、不重做主题体系

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `admin-web-shell`: 布局从「侧栏 + 内容顶栏」调整为「侧栏（品牌 + 菜单 + 底栏工具）+ 无顶栏内容区」；品牌标识为 Pixoma；语言/主题入口位于侧栏底栏

## Impact

- 代码：`web/admin` 壳子相关（如 `_app.tsx` / 布局、`AppTitle`、`AppSidebar`、语言与主题组件挂载点）；可能触及侧栏 Footer / 收起态样式
- API / 后端：无
- 依赖：无新增运行时依赖
- 并行 change：与 `tg-menu-config` 均可能改 `admin-web-shell` 侧栏；本 change 只动品牌与顶栏/底栏工具，不改菜单项集合；合并时注意侧栏文件冲突
