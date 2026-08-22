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

## 7. 数据层与接口扩展（范围扩展：分区仪表盘）

- [x] 7.1 `task_daily_stats` 增加 total_queue_ms / total_exec_ms 列；`task_edge_daily_stats` 增加 succeeded_count / failed_count；新增 `task_case_daily_stats`（stat_date + case_id + count + total_duration_ms）模型与 AutoMigrate
- [x] 7.2 `AddTerminalInput` 增加 CaseID / QueueDurationMS / ExecDurationMS；GORM `AddTerminal` 写入新列与新表（排队=started−created、执行=completed−started，负值按 0）；单测
- [x] 7.3 backfill 重算排队/执行耗时与每节点成功失败、Case 维度
- [x] 7.4 `MetricsRepository.LatestAll(ctx, since)`（每 edge 最新快照）与 `GET /api/v1/stats/fleet`（在线数、平均 CPU/内存/GPU、VRAM、最热节点、每节点利用率）；单测
- [x] 7.5 `GET /api/v1/stats/tasks/daily` 返回 avg_queue_ms / avg_exec_ms；`/tasks/edges` 返回每节点 succeeded/failed/success_rate；新增 `GET /api/v1/stats/cases/top`（count + avg_duration_ms）；handler 单测
- [x] 7.6 adminhost / main 接线：stats Handler 注入 metrics 仓储；AutoMigrate 新增表

## 8. 前端分区仪表盘

- [x] 8.1 全局时间范围组件（近 7 / 30 / 90 天 + Calendar range），从任务区提取为共享状态
- [x] 8.2 Dashboard 重构三区：实时状态（节点启用、在线/Comfy、算力池、实时负载摘要）、任务效能（每日双轴、排队/执行堆叠、状态 donut、错误码 donut、每节点任务量+成功率）、业务分析（Case 散点、Case 热度 Top）
- [x] 8.3 fleet API client、类型与集群负载分组条形图（每节点 CPU/内存/GPU + VRAM + 最热节点）
- [x] 8.4 每日柱状图叠加成功率折线（双轴）；排队 vs 执行耗时堆叠柱
- [x] 8.5 错误码占比 donut、任务状态分布 donut（区间内终态）
- [x] 8.6 Case 耗时散点（recharts ScatterChart，点=Case）与 Case 热度 Top
- [x] 8.7 i18n 更新（去样本提示改全量口径、分区标题、实时徽标）；合同测试更新

## 9. 扩展验证与文档

- [x] 9.1 全量验证：`go build ./...` + `go test ./...`；前端 `tsc -b` + `vitest`
- [x] 9.2 端到端：fleet / cases/top / daily 双轴与堆叠数据冒烟确认
- [x] 9.3 架构文档补充新表列、fleet 与 cases/top 端点
