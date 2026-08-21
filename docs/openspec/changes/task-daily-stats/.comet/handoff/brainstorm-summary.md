# Brainstorm Summary

- Change: task-daily-stats
- Date: 2026-08-21

## 已确认事实

- 范围：A 全量——统计表 + 每日处理任务数柱状图 + 成功率 + 错误码 Top-N + 每节点负载；不含纯前端快赢项（Change B）。
- 口径：每日按 completed_at 归天；processed = succeeded + failed + cancelled。
- 日期范围：类 Grafana，支持起止日期与快捷范围。
- open 阶段已定：三张窄表（task_daily_stats / task_edge_daily_stats / task_error_daily_stats）、orchestrator 终态同步幂等 upsert、STATS_TIMEZONE 默认 Asia/Shanghai、保留 365 天、API 默认近 30 天上限 365 天、前端复用 shadcn ChartContainer + Calendar。

## 候选方案（待确认）

- 方案 1（推荐）：三张窄表 + orchestrator 终态同步 upsert + 一次性 backfill；读 O(天数)，写路径低频率开销可忽略。
- 方案 2：单张宽表 + 后台 job 聚合；写路径零侵入，但引入延迟、额外调度，且维度查询退化为 JSON/宽列扫描。
- 方案 3：不建表，查询时 GROUP BY；实现最快，任务量大时扫描开销不可持续（现 per-edge stats 已暴露该问题）。

## 待确认决策点

- 空白天是否显示为 0 柱（推荐：接口按范围返回每一天，空白天计数为 0，柱状图时间轴连续）。
- 成功率口径：succeeded / (succeeded + failed)（推荐，取消不计入分母）。
- 每节点负载口径：仅终态任务按 edge_id 计数（推荐，避免把排队/运行中的"未处理"算入负载）。

## 已确认决策点（2026-08-21）

- Q1：按范围返回每一天，空白天计数为 0（接口零填充，柱状图时间轴连续）。
- Q2：成功率 = succeeded / (succeeded + failed)；无成功也无失败时返回 null。
- Q3：每节点负载仅统计终态任务（succeeded / failed / cancelled）按 edge_id 计数。

## Spec Patch

- 待确认：若空白天按 0 计数，在 `task-daily-stats` delta spec 的 daily 接口补一条「空白天返回 0」场景。
- 已确认：将回写以下场景——
  - daily 接口：空白天返回 0 计数；
  - daily 汇总：成功率公式 succeeded / (succeeded + failed)，无分母时为空；
  - edges 接口：仅统计终态任务按 edge_id 计数。

## 设计确认（2026-08-21）

用户已确认采用方案 1（三张窄表 + orchestrator 终态同步 upsert + backfill），并确认三个口径（空白天零填充、成功率公式、仅终态任务计每节点负载）。Design Doc 将写入 `docs/superpowers/specs/2026-08-21-task-daily-stats-design.md`，并回写上述 3 处 Spec Patch。

## 测试策略（候选）

- Go：仓储幂等与重复终态、归天时区 00:00/23:59 边界、保留清理、handler 非法范围 400 与空数据 200、写路径三表一致性。
- 前端：parse/format 辅助单测、合同测试（关键 class / i18n / 日期范围交互）、Mock 下端到端切换日期范围。

## Spec Patch

- 待确认：若空白天按 0 计数，在 `task-daily-stats` delta spec 的 daily 接口补一条「空白天返回 0」场景。
