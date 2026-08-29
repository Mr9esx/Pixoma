## Why

现有 `/` 仪表盘是纵向排列的信息页，顶部单独展示标题与说明，且各分区（实时状态 / 任务效能 / 业务分析）彼此独立、缺乏主次与"当前需要关注"的聚合入口。运维打开首页时先看到的是描述文字而非数据，无法一眼得知集群健康、任务负载与待处理异常，操作效率与信息密度都不足。

本次将把该页改造成"工作台"：移除顶部独立标题/说明，换成一条全宽的欢迎卡（含当前用户昵称与快捷入口），其下按"左侧数据大盘 / 右侧关注信息"双栏组织，让健康状态、负载热点、任务趋势与待办异常在同一视图中即可浏览与跳转。

## What Changes

- 移除 `/` 页顶部独立的 `title` 与 `desc`（`dashboard.title` / `dashboard.fullAccuracyNote` 区块）。
- 新增全宽欢迎卡，基于 `@kokonutui/mouse-effect-card`：展示当前登录用户昵称（`/setup/me` 的 `nickname || username`）与快捷入口（新建工作流、添加/管理节点）。
- 页面主体改为左右双栏布局：左栏为数据大盘，右栏为"需要关注的信息"。
- 数据大盘新增/重排以下区块：
  - 全宽 `kibo-ui/contribution-graph` 展示每天执行任务量。
  - 三个卡片：节点情况（总 / 生效中）、平均负载、负载最高节点 Top5。
  - "工作流使用热度 Top" 与 "任务耗时" 左右两列。
  - "任务状态分布" 与 "错误 Top5" 左右两列。
- 右栏"需要关注的信息"接入真实数据：离线节点 / Comfy 未运行节点、未启用的工作流、近段时间失败任务 Top。
- 保留全局时间范围选择（近 7 / 30 / 90 天 + 自定义起止日期），contribution-graph 与统计卡片随范围联动刷新；节点在线/未启用等"实时"信息不受时间范围影响。
- 去掉旧的"集群实时负载图"（RealtimeStatusSection 中的 fleet bar chart 与"每节点任务量"卡），由"平均负载 / 负载最高节点 Top5"卡片替代，避免重复。

> 说明：原 `Dashboard 中等总览` 需求中的"分区展示实时与区间数据"与"全局时间范围"等展示性要求被新工作台视图取代；数据请求仍全部来自 `/api/v1/stats/*`、`/edges`、`/cases` 等 admin-api，不允许直连数据库或引入前端 mock 作为验收路径。

## Capabilities

### New Capabilities
- `admin-dashboard-workbench`: 后台工作台页的信息架构与展示需求，包括欢迎卡、数据大盘（任务热度图/节点情况/平均负载/负载 Top5/工作流热度 Top/任务耗时/状态分布/错误 Top5）与右侧关注信息区，以及全局时间范围联动与 admin-api 数据来源约束。

### Modified Capabilities
- `admin-resource-pages`: 原 `Dashboard 中等总览` 需求中的"数字卡片与简单状态/占比分布"及"分区展示实时与区间数据"将被新工作台视图覆盖；保留数据非样本全量口径与 admin-api 通信约束。

## Impact

- **前端**：`web/admin/src/features/dashboard/*`（新增 `workbench-*` 分区组件，调整/移除 `realtime-status-section`、`task-stats-section`、`case-analysis-section` 的部分卡片）、`web/admin/src/routes/_app/index.tsx`（渲染入口）、`web/admin/src/lib/i18n/locales/{zh,en}.json`（新增工作台文案）。
- **组件**：新增 `@kokonutui/mouse-effect-card` 与 `kibo-ui/contribution-graph` 注册表组件（写入 `web/admin/src/components/` 对应目录），遵循项目 shadcn 语义令牌与设计体系。
- **数据**：复用现有 `/api/v1/stats/tasks/daily`、`/stats/tasks/errors`、`/stats/tasks/edges`、`/stats/cases/top`、`/stats/fleet`、`/edges`、`/edges/presence`、`/cases`、`/setup/me`；无后端聚合逻辑变更。
- **测试**：调整 `dashboard` 相关契约/组件测试以匹配新布局；不新增截图类或浏览器截图测试脚手架。
- **行为兼容**：`/` 路由与菜单不变；不引入破坏性 API 变更（无 `**BREAKING**`）。
