## 1. 脚手架与壳

- [x] 1.1 在 `web/admin` 基于 shadcn-admin 初始化可运行项目并写清安装/启动文档
- [x] 1.2 移除或旁路模板鉴权挡板，使无登录可进入布局
- [x] 1.3 配置 admin-api 基址环境变量与 API 客户端封装

## 2. 菜单、i18n 与 Dashboard

- [x] 2.1 实现集中式菜单配置（Dashboard → 实例 → Case → Task → User → Session）
- [x] 2.2 侧栏渲染菜单并与 TanStack Router 路由绑定；默认路由为 Dashboard
- [ ] 2.3 清理无关 demo 菜单项与鉴权挡板，保留管理布局与主题能力
- [x] 2.4 中英 i18n（默认中文）+ 语言切换
- [ ] 2.5 Dashboard 中等总览（卡片 + 简单分布，list API 前端聚合）

## 3. 资源页面

- [ ] 3.1 实例管理页：列表/创建或编辑/观测入口，对接 foundation API
- [ ] 3.2 Case 管理页：列表/编辑/创建/禁用，对接 resources API
- [ ] 3.3 User、Session 列表与详情页（只读为主）
- [ ] 3.4 Task 列表/详情与取消操作
- [ ] 3.5 统一空态、加载态与错误提示

## 4. 联调验收

- [ ] 4.1 与 admin-api 联调：菜单可达全部资源页并完成基础管理操作
- [ ] 4.2 确认请求仅指向 admin-api，文档说明联调步骤
