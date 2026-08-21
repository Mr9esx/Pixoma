# Comet Design Handoff

- Change: task-daily-stats
- Phase: design
- Mode: compact
- Context hash: 0b29135fc7e54a4fb68486292da9aa114700e66d32a16e26d4a0bceaf8ca7ae6

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/task-daily-stats/proposal.md

- Source: docs/openspec/changes/task-daily-stats/proposal.md
- Lines: 1-28
- SHA256: 86abb2c9508e4eefb99f33c13ef03cf33f3e4911b88868c45459eb64b6f40349

```md
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

```

## docs/openspec/changes/task-daily-stats/design.md

- Source: docs/openspec/changes/task-daily-stats/design.md
- Lines: 1-76
- SHA256: 9220a1fde244f13c84f1370f791b17f7e184700f9c3cd6bab5d25ea31e34e732

```md
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

```

## docs/openspec/changes/task-daily-stats/tasks.md

- Source: docs/openspec/changes/task-daily-stats/tasks.md
- Lines: 1-41
- SHA256: 500b6de5adbcd2ab070fefbad61edf54f9b38a662f2d47ea0d3c82e052f5f993

```md
## 1. 数据层：统计表与仓储

- [ ] 1.1 新增 `task_daily_stats` / `task_edge_daily_stats` / `task_error_daily_stats` GORM 模型（含唯一索引），接入 AutoMigrate
- [ ] 1.2 新增 `StatsRepository` 接口与 GORM 实现：`AddTerminal`（按 completed_at 归天，OnConflict 增量 upsert 三表）、`ListDaily`、`ListErrors`、`ListEdges`、`Prune`
- [ ] 1.3 归天时区：解析 `STATS_TIMEZONE`（默认 Asia/Shanghai）并用于 date 计算与查询；单测锁定 00:00 / 23:59 边界
- [ ] 1.4 仓储单测：增量幂等、重复终态不重复计数、空表、保留清理、edge / error 维度聚合

## 2. 写路径：orchestrator 终态统计

- [ ] 2.1 orchestrator `applyStatus` / `RequestCancel` 在任务由非终态转入终态（succeeded / failed / cancelled）后调用 `AddTerminal`（旧≠新守卫，任务保存后执行）
- [ ] 2.2 admin-api 装配注入 `StatsRepository`；测试用 appboot 同步接线
- [ ] 2.3 写路径测试：三种终态各写对三张表；重复/对账事件不重复计数

## 3. API：管理端统计接口

- [ ] 3.1 `GET /api/v1/stats/tasks/daily`：from/to 解析（默认近 30 天、上限 365 天、from>to 或格式非法返回 400）、无数据空序列、summary 成功率
- [ ] 3.2 `GET /api/v1/stats/tasks/errors`：from/to/limit，按 count 降序 Top-N
- [ ] 3.3 `GET /api/v1/stats/tasks/edges`：from/to，按 count 降序并返回 total
- [ ] 3.4 admin-api 挂载 `/api/v1/stats` 路由并接线；handler 单测（非法范围 400、空数据 200、日期边界）

## 4. Backfill 与保留

- [ ] 4.1 一次性 backfill 命令：从任务表按 completed_at 归天重算三表绝对值并覆盖（可重复执行）
- [ ] 4.2 `TASK_STATS_RETENTION`（默认 365 天）接线：写入后清理过期行
- [ ] 4.3 README 与架构文档补充 `STATS_TIMEZONE` / `TASK_STATS_RETENTION` 环境变量与统计表

## 5. 前端 Dashboard 任务统计

- [ ] 5.1 `lib/api` 新增 stats client（daily / errors / edges）、类型与 query-keys
- [ ] 5.2 日期范围状态与快捷预设（近 7 / 30 / 90 天），基于现有 `Calendar` 的 range 选择器（类 Grafana 交互）
- [ ] 5.3 每日处理任务数柱状图卡：shadcn `ChartContainer` + recharts `BarChart`，沿用现有卡片/配色体系
- [ ] 5.4 统计卡：成功率、错误码 Top-N、每节点负载；空数据降级为空态
- [ ] 5.5 `TasksCard` 替换为任务统计区块（Edges / Cases 卡不动）；接口失败仅对应卡片进入错误态
- [ ] 5.6 i18n zh/en：每日任务标题、快捷预设、成功率等文案
- [ ] 5.7 合同测试：关键 class、日期范围交互、i18n；parse / format 辅助单测

## 6. 验证与文档

- [ ] 6.1 全量验证：`go build ./...` + `go test ./...`；前端 `tsc -b` + `vitest`
- [ ] 6.2 端到端确认：Mock 数据下 Dashboard 切换日期范围后柱状图与统计卡正确刷新
- [ ] 6.3 同步 `docs/architecture/data-model.md`、`runtime.md` 与根 `README.md`

```

## docs/openspec/changes/task-daily-stats/specs/admin-resource-pages/spec.md

- Source: docs/openspec/changes/task-daily-stats/specs/admin-resource-pages/spec.md
- Lines: 1-20
- SHA256: 41c936be49d39eee705a1ec8fa219e493ce2d4f07eee0386e25601d7c1fa98f6

```md
## MODIFIED Requirements

### Requirement: Dashboard 中等总览
控制台 MUST 提供 Dashboard 页：数字卡片与简单状态/占比分布；任务相关统计 MUST 来自专用统计接口（`/api/v1/stats/tasks/*`），为全量按天聚合，不再受列表接口 limit 样本限制；实例与 Case 汇总仍可来自既有列表类接口。

#### Scenario: Dashboard 展示聚合信息
- **WHEN** 用户打开 Dashboard 且 admin-api 可用
- **THEN** 页面展示实例与 Case 汇总卡片，以及任务统计区块（每日处理任务数柱状图、成功率、错误码 Top-N、每节点负载），网络请求指向 admin-api

#### Scenario: Dashboard 卡片失败隔离
- **WHEN** 某一汇总依赖的 API 请求失败
- **THEN** 仅对应卡片或区块进入错误/空态，其它区块仍可展示

#### Scenario: 任务统计不受样本限制
- **WHEN** 所选日期范围内实际任务数超过 200
- **THEN** 每日柱状图与统计卡展示真实全量计数，而非列表样本计数

#### Scenario: 日期范围选择
- **WHEN** 运维选择起止日期或快捷范围（如近 7 / 30 / 90 天）
- **THEN** 每日处理任务数柱状图按所选范围重新查询并展示

```

## docs/openspec/changes/task-daily-stats/specs/task-daily-stats/spec.md

- Source: docs/openspec/changes/task-daily-stats/specs/task-daily-stats/spec.md
- Lines: 1-73
- SHA256: dfeb1f83a7bca46f293684c746e6247748cc8c92bea3b35feddc84fd8a3709c6

```md
## Purpose

为管理端 Dashboard 提供全量、按天聚合的任务统计能力：每日已处理任务数、成功率、错误码分布与每节点负载，数据来自专门的统计表与统计接口，不受列表接口样本数量限制。

## ADDED Requirements

### Requirement: 任务按天统计持久化
系统 MUST 在任务进入终态（succeeded / failed / cancelled）时，按完成时间（completed_at）归天更新当日统计；同一任务的重复终态处理 MUST 幂等，不得重复累加计数。

#### Scenario: 任务完成写入当日统计
- **WHEN** 任务进入 succeeded / failed / cancelled 终态
- **THEN** 对应日期（completed_at 所在天）的 processed 计数与对应状态计数均增加，且 processed = succeeded + failed + cancelled

#### Scenario: 重复终态处理不重复计数
- **WHEN** 同一任务的终态被重复应用（如重试、对账）
- **THEN** 该日统计计数不重复累加

#### Scenario: 取消计入已处理
- **WHEN** 任务被取消并进入 cancelled 终态
- **THEN** 对应日期 processed 与 cancelled 计数均增加

### Requirement: 每日统计查询接口
系统 MUST 提供 `GET /api/v1/stats/tasks/daily` 管理接口，支持 `from` / `to` 日期范围参数，返回范围内逐日 processed / succeeded / failed / cancelled 与平均耗时，并附范围汇总（成功率等）。

#### Scenario: 按范围返回每日数据
- **WHEN** 客户端请求 daily 接口并携带合法的 from / to 日期范围
- **THEN** 返回范围内逐日统计，按日期升序排列，并附范围成功率汇总

#### Scenario: 无数据返回空
- **WHEN** 请求范围内没有任何终态任务
- **THEN** 返回空序列与零值汇总，HTTP 200 而非错误

#### Scenario: 非法日期范围被拒绝
- **WHEN** from 晚于 to，或日期格式非法
- **THEN** 接口返回 400 与明确错误信息

#### Scenario: 空白天返回零计数
- **WHEN** 请求范围内某一天没有任何终态任务
- **THEN** 该天仍出现在逐日序列中，计数为 0，时间轴连续

#### Scenario: 成功率口径
- **WHEN** 计算范围汇总中的成功率
- **THEN** success_rate 等于 succeeded / (succeeded + failed)，且成功与失败均为 0 时返回空值

### Requirement: 错误码与每节点负载统计
系统 MUST 提供错误码 Top-N 与每节点已处理任务数统计，供 Dashboard 展示。

#### Scenario: 错误码 Top-N
- **WHEN** 客户端请求错误码统计接口并携带 from / to 与 limit
- **THEN** 返回按错误码聚合计数降序的前 N 条（无错误码的任务不占位）

#### Scenario: 每节点负载
- **WHEN** 客户端请求每节点统计接口并携带 from / to
- **THEN** 返回范围内每个节点的已处理任务数与占比，按计数降序

#### Scenario: 仅统计终态任务
- **WHEN** 范围内同时存在排队、运行中的任务与终态任务
- **THEN** 每节点负载仅按终态任务（succeeded / failed / cancelled）的 edge_id 计数，非终态任务不计入

#### Scenario: 空数据降级
- **WHEN** 范围内无失败任务或无派发节点数据
- **THEN** 对应统计返回空序列，前端展示空态而非报错

### Requirement: 统计口径
每日统计 MUST 以 completed_at 归天，processed 定义为 succeeded + failed + cancelled；历史任务 MUST 支持通过一次性 backfill 补齐统计表。

#### Scenario: 历史数据补齐
- **WHEN** 统计表刚上线或存在历史终态任务
- **THEN** 执行 backfill 后，历史每天均能按上述口径查出统计

#### Scenario: 口径一致
- **WHEN** 同一时间范围内分别查询 daily 汇总与逐日数据
- **THEN** 汇总计数等于各日计数之和

```
