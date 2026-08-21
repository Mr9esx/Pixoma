## Why

管理端 Dashboard 目前只能基于 `limit=200` 的列表接口样本做前端聚合（页面注明「基于当前拉取样本」），无法给出真实的全量任务统计。「每日处理任务数」这类按天聚合在样本口径下必然偏低甚至缺天；同时现有按实例统计任务的方式（`ListByInstance` 全表扫描后在内存中计数）随任务量增长性能不可持续。需要引入专门的任务统计表与统计接口，为 Dashboard 提供全量、按天、可查询时间范围的准确数据。

## What Changes

- 新增 `task_daily_stats` 统计表：按 `completed_at` 归天记录每日已处理任务数（成功/失败/取消）、总耗时与平均耗时；任务进入终态时在 orchestrator 写路径内幂等 upsert；提供一次性历史数据 backfill。
- 新增管理端统计接口（admin-api）：
  - `GET /api/v1/stats/tasks/daily?from=YYYY-MM-DD&to=YYYY-MM-DD`：按天返回 processed / succeeded / failed / cancelled 与平均耗时，并附时间范围汇总（成功率等）。
  - `GET /api/v1/stats/tasks/errors?from&to&limit`：错误码 Top-N。
  - `GET /api/v1/stats/tasks/edges?from&to`：每节点处理任务数（负载分布）。
- Dashboard 任务区改造：新增「每日处理任务数」柱状图，支持类 Grafana 的日期范围选择（起止日期 + 快捷范围）；新增任务统计卡（成功率、错误码 Top-N、每节点负载）。数据来自统计接口，不再受 200 样本限制。
- 保留「卡片失败隔离」行为：任一统计接口失败仅对应卡片进入错误/空态，其它区块不受影响。

## Capabilities

### New Capabilities
- `task-daily-stats`: 任务按天统计的持久化聚合、管理端统计接口，以及 Dashboard 任务统计图表（每日柱状图 + 日期范围选择 + 成功率 / 错误码 / 每节点负载）。

### Modified Capabilities
- `admin-resource-pages`: Dashboard 中等总览需求变更——任务统计改为使用专用统计接口（不再仅靠列表接口前端聚合），并新增每日处理任务数柱状图与日期范围选择。

## Impact

- 数据层：新增 `task_daily_stats` 表（AutoMigrate）与统计仓储；orchestrator 任务终态写路径增加幂等 upsert；一次性 backfill 命令/任务。
- API：`internal/httpapi` 新增 `/api/v1/stats/tasks/*` 路由与 handler；admin-api 装配注入统计仓储。
- 前端：`web/admin` Dashboard 任务区（柱状图、日期范围选择器、统计卡）、API client / 类型、query-keys、i18n、合同测试。
- 文档：`docs/architecture/data-model.md`、`runtime.md` 与根 `README.md` 同步统计表与接口。
