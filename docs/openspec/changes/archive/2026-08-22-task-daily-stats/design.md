## Context

现状参见 proposal.md - Why：Dashboard 任务区基于 `limit=200` 的列表样本前端聚合，无法给出真实每日统计；现有按实例统计（`ListByInstance` 全表扫描后内存计数）随任务量增长不可持续。任务终态写路径集中在 `internal/runtime/application/orchestrator`（`applyStatus` 内的 `MarkSucceeded` / `MarkFailed` / `MarkCancelled`），任务持久化走 `TaskRepository`（GORM）。`edge_metrics` 已建立「采集 → 持久化 → 管理端查询 → 前端 chart」的链路，前端已有 shadcn `chart.tsx` 组件、`Calendar` / `Popover`（react-day-picker 9）可用于日期范围选择。

## Goals / Non-Goals

**Goals:**
- 以 `task_daily_stats` 系列统计表提供全量、按天（completed_at 归天）的任务统计，读路径 O(天数)，不再扫描任务全表。
- 任务进入终态时在 orchestrator 写路径内幂等更新统计，保证当日计数实时准确。
- 提供 `GET /api/v1/stats/tasks/{daily,errors,edges}` 接口，支持 from/to 日期范围。
- Dashboard 任务区：每日处理任务数柱状图（支持类 Grafana 的起止日期 + 快捷范围选择）、成功率、错误码 Top-N、每节点负载。

**Non-Goals:**
- 不把现有每节点 stats 端点切到统计表（可作后续优化，避免本次范围膨胀）。
- 不做实时推送；沿用 TanStack Query 轮询/手动刷新。
- 不做小时粒度；表结构按天，将来可平行扩展。
- 不做用户/会话维度统计。

## Decisions

### 1. 统计表结构：三个窄表，按维度拆

```
task_daily_stats        (stat_date PK, processed/succeeded/failed/cancelled, total_duration_ms, updated_at)
task_edge_daily_stats   (stat_date + edge_id PK, processed_count, updated_at)
task_error_daily_stats  (stat_date + error_code PK, count, updated_at)
```

全局每日统计、每节点负载、错误码 Top-N 各自独立成表，查询直接走唯一索引，语义清晰。备选：单表加 JSON 维度列，或全局表派生各维度——前者查询/索引别扭且类型弱，后者无法表达按节点/错误码聚合，均放弃。

### 2. 更新时机：orchestrator 终态写路径同步 upsert

任务状态从非终态转入终态（succeeded / failed / cancelled）时，在任务行保存后执行统计 upsert（`clause.OnConflict` 增量更新）。只有「旧状态 ≠ 新状态且新状态为终态」才更新，天然挡住对账/重复事件导致的重复计数；幂等 upsert 使崩溃重放安全。

备选：后台定时任务扫描增量任务刷表——写路径零侵入但引入延迟与额外调度；终态频率低，同步 upsert 代价可忽略，故不采用。

### 3. 归天时区：可配置，默认 Asia/Shanghai

`completed_at` 按配置时区（环境变量 `STATS_TIMEZONE`，默认 `Asia/Shanghai`）归天；接口的 from/to 同按该时区解释。备选：纯 UTC——实现最简但中国运营视角下早晚高峰会错天，故不采用。

### 4. API 设计

挂载于 `/api/v1/stats`：

- `GET /api/v1/stats/tasks/daily?from=YYYY-MM-DD&to=YYYY-MM-DD`
  - 默认近 30 天（含今天），最远 365 天；`from > to` 或格式非法返回 400。
  - 返回 `{ range, days: [{date, processed, succeeded, failed, cancelled, avg_duration_ms}], summary: {processed, succeeded, failed, cancelled, success_rate} }`，days 升序，无数据返回空数组与零值汇总。
- `GET /api/v1/stats/tasks/errors?from&to&limit=10` → `{ items: [{error_code, count}] }`，按 count 降序。
- `GET /api/v1/stats/tasks/edges?from&to` → `{ items: [{edge_id, count}], total }`，按 count 降序。

### 5. 保留与 backfill

统计表默认保留 365 天（`TASK_STATS_RETENTION` 可配），与 `edge_metrics` 的 24h 不同——运营趋势需要长窗口。上线时提供一次性 backfill（独立 cmd 或启动参数），直接从任务表按天重算**绝对值** upsert 覆盖；任务表是权威源，覆盖不会丢失实时增量。

### 6. 前端：复用现有 chart 与 Calendar

- 每日柱状图用 shadcn `ChartContainer` + recharts `BarChart`，沿用 edge-system-monitoring 的卡片/图例/配色体系。
- 日期范围选择：`Calendar`（range 模式）+ 快捷预设按钮（近 7 / 30 / 90 天），类 Grafana 的「预设 + 精确起止」交互；范围存入 TanStack Query key。
- `TasksCard` 改造为统计区块：柱状图卡 + 统计卡（成功率、错误码 Top-N、每节点负载）；Edges / Cases 卡不动；接口失败仅对应卡片进错误/空态。

## Risks / Trade-offs

- 时区口径错误导致跨天错位 → 时区配置化 + 单测锁定归天边界（23:59 / 00:00）。
- 终态写入后统计更新前崩溃 → 幂等 upsert + backfill 可修复；对账重放被「旧≠新」守卫挡住。
- 统计表保留清理与实时写入并发 → 清理幂等，按 `stat_date < cutoff` 删除，无事务依赖。
- 前端日期选择交互复杂度 → 复用既有 `Calendar` + 预设按钮，合同测试锁定关键 class 与文案。

## Migration Plan

1. 后端先合并：AutoMigrate 建三张统计表；orchestrator 接入写路径；新增 stats 路由；执行一次 backfill。空表期查询返回空数据，不影响既有功能。
2. 前端后合并：Dashboard 任务区切到统计接口；失败时回退为空态展示，不阻塞其它卡片。
3. 回滚：前端回退为样本版任务卡即可；后端停止写统计表不影响调度与列表功能。

## Open Questions

无阻塞项。快捷预设默认（7/30/90 天）与统计保留期（365 天）为可配置默认值，可在最终审视时调整。
