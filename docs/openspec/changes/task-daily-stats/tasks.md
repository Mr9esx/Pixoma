## 1. 数据层：统计表与仓储

- [x] 1.1 新增 `task_daily_stats` / `task_edge_daily_stats` / `task_error_daily_stats` GORM 模型（含唯一索引），接入 AutoMigrate
- [x] 1.2 新增 `StatsRepository` 接口与 GORM 实现：`AddTerminal`（按 completed_at 归天，OnConflict 增量 upsert 三表）、`ListDaily`、`ListErrors`、`ListEdges`、`Prune`
- [x] 1.3 归天时区：解析 `STATS_TIMEZONE`（默认 Asia/Shanghai）并用于 date 计算与查询；单测锁定 00:00 / 23:59 边界
- [x] 1.4 仓储单测：增量幂等、重复终态不重复计数、空表、保留清理、edge / error 维度聚合

## 2. 写路径：orchestrator 终态统计

- [x] 2.1 orchestrator `applyStatus` / `RequestCancel` 在任务由非终态转入终态（succeeded / failed / cancelled）后调用 `AddTerminal`（旧≠新守卫，任务保存后执行）
- [x] 2.2 admin-api 装配注入 `StatsRepository`；测试用 appboot 同步接线
- [x] 2.3 写路径测试：三种终态各写对三张表；重复/对账事件不重复计数

## 3. API：管理端统计接口

- [x] 3.1 `GET /api/v1/stats/tasks/daily`：from/to 解析（默认近 30 天、上限 365 天、from>to 或格式非法返回 400）、无数据空序列、summary 成功率
- [x] 3.2 `GET /api/v1/stats/tasks/errors`：from/to/limit，按 count 降序 Top-N
- [x] 3.3 `GET /api/v1/stats/tasks/edges`：from/to，按 count 降序并返回 total
- [x] 3.4 admin-api 挂载 `/api/v1/stats` 路由并接线；handler 单测（非法范围 400、空数据 200、日期边界）

## 4. Backfill 与保留

- [x] 4.1 一次性 backfill 命令：从任务表按 completed_at 归天重算三表绝对值并覆盖（可重复执行）
- [x] 4.2 `TASK_STATS_RETENTION`（默认 365 天）接线：写入后清理过期行
- [x] 4.3 README 与架构文档补充 `STATS_TIMEZONE` / `TASK_STATS_RETENTION` 环境变量与统计表

## 5. 前端 Dashboard 任务统计

- [x] 5.1 `lib/api` 新增 stats client（daily / errors / edges）、类型与 query-keys
- [x] 5.2 日期范围状态与快捷预设（近 7 / 30 / 90 天），基于现有 `Calendar` 的 range 选择器（类 Grafana 交互）
- [x] 5.3 每日处理任务数柱状图卡：shadcn `ChartContainer` + recharts `BarChart`，沿用现有卡片/配色体系
- [x] 5.4 统计卡：成功率、错误码 Top-N、每节点负载；空数据降级为空态
- [x] 5.5 `TasksCard` 替换为任务统计区块（Edges / Cases 卡不动）；接口失败仅对应卡片进入错误态
- [x] 5.6 i18n zh/en：每日任务标题、快捷预设、成功率等文案
- [x] 5.7 合同测试：关键 class、日期范围交互、i18n；parse / format 辅助单测

## 6. 验证与文档

- [x] 6.1 全量验证：`go build ./...` + `go test ./...`；前端 `tsc -b` + `vitest`
- [x] 6.2 端到端确认：Mock 数据下 Dashboard 切换日期范围后柱状图与统计卡正确刷新
- [x] 6.3 同步 `docs/architecture/data-model.md`、`runtime.md` 与根 `README.md`
