# Admin 表面层级规范

适用：`web/admin` 的页面容器、面板、表格、卡片内部区块，以及后续同类页面。

目标：**底色只表达层级**。同一个组件在不同位置承担不同层级时，靠调用点决定底色，不靠改组件默认值。

## 原则（必须遵守）

1. **底色按层级选，不按组件类型选**
   决定底色的唯一问题是：这个容器是页面的主体内容，还是主体内部的一块？是主体就用内容表面色，是内部的一块就用次级表面色或者凹陷色。组件叫什么名字不参与判断。

2. **两个主题下含义一致**
   `--background` 在亮色与暗色下都比 `--card` 暗，所以同一种搭配在两个主题里都读作「抬起」与「凹陷」。新增令牌时 MUST 维持这个方向，不要让某个主题反过来。

3. **页面主体用内容表面色**
   页面的主体内容 MUST 用 `--card`：`Card` 组件、Master–Detail 面板、页面级表格、Studio 工作台。页面级表格用 `DataTable` 的 `surface='card'`。

4. **主体内部的一块用次级表面色或者凹陷色**
   卡片内部的说明条、侧栏、分组块用 `--muted` / `--secondary`；代码块、JSON 预览、输入框、只读文本块用 `--background`。嵌在卡片里的 `DataTable` 保持默认的 `surface='page'`。

5. **浮层单独一层**
   dialog / dropdown / popover / select / command MUST 用 `--popover` 并配 `shadow-md` 以上。浮层是唯一允许投影的层级。

6. **禁止裸 hex 与手写 `dark:`**
   底色 MUST 使用语义令牌；派生色一律 `color-mix(in oklch, …)`。禁止用 `slate-*`，契约测试会拦截。

7. **亮色下的层级靠边框承担**
   `--card` 与 `--background` 在亮色下只差 4/255，层次几乎全部由那圈 `border` 表达。因此有边框的面板 MUST 保留 `border`；去掉边框等于抹掉层级。

## 参考实现

- 圆角：`docs/frontend/admin-radius-rules.md`
- 令牌：`web/admin/src/styles/theme.css`
- 表格：`DataTable` 的 `surface` prop（`web/admin/src/components/data-table/data-table.tsx`）
- 面板：`MasterDetailShell`（`web/admin/src/components/master-detail/master-detail-shell.tsx`）
- 校验：`web/admin/src/styles/theme-neutral.contract.test.ts`、`web/admin/src/components/master-detail/master-detail.contract.test.ts`
