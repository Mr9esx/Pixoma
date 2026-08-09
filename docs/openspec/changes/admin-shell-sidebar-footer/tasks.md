## 1. 品牌文案

- [x] 1.1 将 `AppTitle` 主标题改为「Pixoma」，移除「Vite + ShadcnUI」等模板副标题
- [x] 1.2 清理 Logo / 可访问名称中仍残留的「Shadcn-Admin」文案，改为 Pixoma

## 2. 侧栏底栏工具

- [x] 2.1 在 `AppSidebar` 增加 `SidebarFooter`，挂载 `LanguageSwitcher` 与 `ThemeSwitch`
- [x] 2.2 调整 Footer 在侧栏展开/收起态下的布局，确保收起态仍可图标操作（含必要 tooltip）

## 3. 移除内容区顶栏

- [x] 3.1 从 `_app.tsx`（及若仍使用的 `authenticated-layout`）移除壳级顶栏，内容区直接渲染页面
- [x] 3.2 确认语言/主题不再从内容区顶栏导入，避免重复挂载

## 4. 验收

- [x] 4.1 手动确认：侧栏显示 Pixoma、无模板默认文案；内容区无壳级顶栏；底栏可切换语言与主题
- [x] 4.2 手动确认：侧栏收起为图标模式后，仍可切换语言与主题
- [x] 4.3 跑通 `web/admin` 既有相关单测（若有布局/壳子测试则一并更新）
