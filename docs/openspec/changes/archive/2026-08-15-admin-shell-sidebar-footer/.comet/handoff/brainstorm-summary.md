# Brainstorm Summary

- Change: admin-shell-sidebar-footer
- Date: 2026-08-09

## 确认的技术方案

- 壳级顶栏从 `routes/_app.tsx` 删除；内容区只渲染页面
- 语言/主题挂到 `AppSidebar` 的 `SidebarFooter`（方案 A）
- `AppTitle` 主标题改为 Pixoma，去掉模板副标题；Logo 无障碍名同步为 Pixoma
- `LanguageSwitcher` 改为与 `ThemeSwitch` 一致的「图标触发 + 下拉」形态，展开/收起共用一套
- `MENU_ITEMS` 与菜单渲染尽量不动，降低与 `tg-menu-config` 冲突
- `authenticated-layout.tsx` 本无顶栏，仅在出现漂移时对齐，非本 change 主路径

## 关键取舍与风险

- 收起态双文字钮不可用 → 语言控件统一改下拉（已确认）
- 与并行 change 同改侧栏文件 → 改动面收在 Title/Footer
- 手机无内容顶栏 → 依赖侧栏抽屉（已接受）

## 测试策略

- 手动验收：展开/收起/移动端抽屉下语言与主题可切；侧栏仅显示 Pixoma、无模板文案；内容区无壳级顶栏
- 更新既有相关单测；不新增 E2E

## Spec Patch

无
