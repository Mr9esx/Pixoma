---
comet_change: task-daily-stats
role: technical-design
canonical_spec: openspec
---

# task-daily-stats 深度技术设计

## 1. 目标与范围

为管理端 Dashboard 提供全量、按天聚合的任务统计：每日处理任务数柱状图（支持类 Grafana 的起止日期与快捷范围）、成功率、错误码 Top-N、每节点负载。统计以 `completed_at` 归天，processed = succeeded + failed + cancelled。解决两个现状问题：Dashboard 只能基于 `limit=200` 列表样本前端聚合（数字偏低、缺天），以及按实例全表扫描统计的性能不可持续。

范围边界：不做小时粒度、不做用户/会话维度、不把现有每节点 stats 端点切到新表、不做实时推送。归天时区默认 Asia/Shanghai，统计保留 365 天，均可配置。

## 2. 数据模型

三张窄表，维度各自独立，查询走主键唯一索引：

```sql
task_daily_stats (
  stat_date         DATE      PRIMARY KEY,
  processed_count   INTEGER   NOT NULL DEFAULT 0,
  succeeded_count   INTEGER   NOT NULL DEFAULT 0,
  failed_count      INTEGER   NOT NULL DEFAULT 0,
  cancelled_count   INTEGER   NOT NULL DEFAULT 0,
  total_duration_ms BIGINT    NOT NULL DEFAULT 0,
  updated_at        TIMESTAMP NOT NULL
)

task_edge_daily_stats (
  stat_date       DATE      NOT NULL,
  edge_id         VARCHAR(64) NOT NULL,
  processed_count INTEGER   NOT NULL DEFAULT 0,
  updated_at      TIMESTAMP NOT NULL,
  PRIMARY KEY (stat_date, edge_id)
)

task_error_daily_stats (
  stat_date   DATE         NOT NULL,
  error_code  VARCHAR(128) NOT NULL,
  count       INTEGER      NOT NULL DEFAULT 0,
  updated_at  TIMESTAMP    NOT NULL,
  PRIMARY KEY (stat_date, error_code)
)
```

- 平均耗时读时计算：`avg_duration_ms = total_duration_ms / processed_count`（processed 为 0 时返回 null）。
- DB 只存 DATE 与 UTC 时间戳，归天时区由应用层负责（§3），避免存储层时区歧义。
- 三表由 AutoMigrate 创建，无存量数据迁移。

## 3. 归天时区与日期计算

- `STATS_TIMEZONE` 环境变量，默认 `Asia/Shanghai`；解析失败回退默认并输出告警日志。
- 归天：`DateOf(t) = t.In(loc)` 的 `YYYY-MM-DD`。
- 接口日期参数：`2006-01-02` 格式，按同一时区解释为当天 00:00（含）至次日 00:00（不含）。
- 边界单测：23:59:59.999 归当天；00:00:00 归新一天；跨时区输入按配置时区归天。

## 4. 写路径

接入点：`internal/runtime/application/orchestrator/service.go` 的 `applyStatus`（succeeded / failed / cancelled 分支）与 `RequestCancel`。

规则：

- 应用状态迁移前记录 `old := t.Status`；迁移成功且 `old != new` 且 `new` 为终态时才触发统计。
- 任务行保存后调用 `AddTerminal(ctx, edgeID, errorCode, completedAt, durationMs)`：
  - `task_daily_stats`：processed +1、对应状态 +1、total_duration_ms += durationMs；
  - `task_edge_daily_stats`：edgeID 非空时 processed +1；
  - `task_error_daily_stats`：errorCode 非空时 count +1。
- 更新用 GORM `clause.OnConflict{DoUpdates: ...}` 原子增量，多任务并发写同一天由 DB 行锁保证不丢计数。
- 事务边界：orchestrator 当前无跨仓储事务包装，采用「任务行保存后独立 upsert」。崩溃窗口（任务已保存、统计未写）靠幂等 upsert 与可重跑 backfill 兜底；对账/重复事件被 `old != new` 守卫挡住，不会双计。

## 5. 读路径 API

路由：`internal/httpapi/stats` Handler，admin-api 挂载 `/api/v1/stats`。

### GET /api/v1/stats/tasks/daily?from&to

- 缺省 from = 今天 - 29 天，to = 今天；格式非法、`from > to`、跨度 > 365 天 → 400。
- 查询 `task_daily_stats` 按 stat_date 升序；内存中按 [from, to] 逐日零填充，空白天计数为 0（时间轴连续）。
- summary 求和：processed / succeeded / failed / cancelled；`success_rate = succeeded / (succeeded + failed)`，分母为 0 时返回 null。

```json
{
  "range": { "from": "2026-08-01", "to": "2026-08-21" },
  "days": [
    { "date": "2026-08-01", "processed": 0, "succeeded": 0, "failed": 0, "cancelled": 0, "avg_duration_ms": null }
  ],
  "summary": { "processed": 0, "succeeded": 0, "failed": 0, "cancelled": 0, "success_rate": null }
}
```

### GET /api/v1/stats/tasks/errors?from&to&limit=10

- limit 默认 10、上限 100、`0 < limit <= 100`；返回 error_code 非空的条目按 count 降序。

```json
{ "items": [ { "error_code": "timeout", "count": 5 } ] }
```

### GET /api/v1/stats/tasks/edges?from&to

- 仅统计终态任务（succeeded / failed / cancelled）按 edge_id 计数，count 降序，并返回 total。

```json
{ "items": [ { "edge_id": "gpu-1", "count": 42 } ], "total": 87 }
```

## 6. 保留与 backfill

- `TASK_STATS_RETENTION` 默认 365 天；`Prune(ctx, before)` 删除三表 `stat_date < before` 的行；在 `AddTerminal` 成功后顺带执行（低频幂等），backfill 结束后也执行一次。
- backfill：独立 cmd（如 `apps/admin-api/cmd/backfill-task-stats`），直接按任务表终态行以 `GROUP BY stat_date` 重算三个维度的**绝对值**，`ON CONFLICT DO UPDATE` 覆盖；任务表是权威源，重复执行结果一致。建议上线建表后立即执行一次。

## 7. 前端

- 组件：Dashboard 任务区改造为统计区块（改造现有 `TasksCard` 或新增 `task-stats-section.tsx`）；Edges / Cases 卡不动。
- 日期范围：`from` / `to` 本地 state（YYYY-MM-DD），默认近 30 天；快捷预设按钮（近 7 / 30 / 90 天）；基于 react-day-picker 9 的 `Calendar`（mode="range"）置于 `Popover`，禁止反向选择。
- 查询：TanStack Query，key 含 from/to（`queryKeys.stats.daily` 等）；可选 `refetchInterval: 60000`。
- 每日柱状图：shadcn `ChartContainer` + recharts `BarChart`，X 轴为连续日期，单系列 processed，tooltip 展示当日各状态明细；卡片/配色沿用现有 Sprint health 图表体系。
- 统计卡：成功率（null 显示「—」）、错误码 Top-N（列表）、每节点负载（横向条形或列表）。
- 失败隔离：任一统计接口失败仅对应卡片进入 ErrorBanner 错误态；空数据渲染空态。
- i18n：zh/en 新增每日任务标题、快捷预设、成功率、错误码 Top-N、每节点负载等文案。
- 合同测试：锁定图表/日历关键 class、i18n 文案、范围交互；parse / format 辅助（零填充、success_rate null、日期格式化）单测。

## 8. 风险与缓解

| 风险 | 缓解 |
| --- | --- |
| 任务保存后、统计 upsert 前崩溃，当日计数缺失 | upsert 幂等 + backfill 可修复；对账重放被 old≠new 守卫挡住；一致性窗口为秒级 |
| 时区误配导致跨天错位 | `STATS_TIMEZONE` 配置化 + 00:00/23:59 边界单测 |
| 重复事件双计 | 仅非终态 → 终态迁移触发；OnConflict 原子增量 |
| 统计表增长 | 保留 365 天 + 写入后 prune；行数约 365 × 维度基数 |
| 大跨度零填充内存开销 | 上限 365 天，365 行内存可忽略 |
| backfill 与实时增量竞态 | backfill 以任务表为权威源重算绝对值，覆盖安全；建议上线时执行 |
| 前端反向日期范围 | 组件禁止反向选择 + API 400 兜底 |

## 9. 测试策略

Go：

- stats 仓储：AddTerminal 幂等、重复终态不双计、时区边界、prune、daily 零填充、errors / edges 聚合；
- orchestrator 写路径：三种终态写对三表、非终态不写、edgeID / errorCode 为空时跳过对应维度；
- handler：from/to 默认值与解析、跨度上限、400、空数据 200、summary 成功率 null；
- backfill：绝对值覆盖、重复执行幂等。

前端：

- 单测：日期范围工具（预设、零填充、格式化）、stats parse（success_rate null 等）；
- 合同测试：BarChart / Calendar 关键 class、i18n 文案、范围交互、错误隔离；
- Mock 端到端：切换日期范围后查询参数与柱状图/统计卡正确刷新。

## 10. 迁移与回滚

1. 后端先合并：AutoMigrate 三表 → orchestrator 写路径 → stats API → 执行 backfill；空表期接口返回零填充数据，不影响既有功能。
2. 前端后合并：Dashboard 任务区切到统计接口；任一接口失败仅对应卡片降级。
3. 回滚：前端回退为样本版任务卡；后端停止写统计表即可，不影响调度、列表与任务功能。
