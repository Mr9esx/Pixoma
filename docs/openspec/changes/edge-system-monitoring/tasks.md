## 1. Edge 指标采集

- [x] 1.1 在 `internal/platform/edge` 新增 `Metrics` / `GPUMetric` 领域类型（JSON 标签、可选字段用指针/omitempty）
- [x] 1.2 新增 `apps/edge-agent/internal/metrics` 包：CPU 占用率与内存占用/总量/占用率采集（gopsutil）
- [x] 1.3 磁盘 I/O 读/写速率采集（gopsutil `disk.IOCounters` 差值除间隔，首拍留空）
- [x] 1.4 GPU 指标采集：解析 `nvidia-smi`（占用率、显存已用/总量、占用率），不可用时留空；`COMFY_MOCK=true` 时合成假 GPU 数据
- [x] 1.5 metrics 采集器单测：mock 模式、部分指标不可用、首拍 I/O 空、数值边界
- [x] 1.6 `presence.Reporter` 按 `METRICS_INTERVAL`（默认 30s）采样，随 presence 载荷携带 `metrics`
- [x] 1.7 `pull.Client.ReportPresence` 请求体支持 `metrics` 字段并补测试

## 2. 控制面持久化与 API

- [x] 2.1 新增 `MetricsRow` GORM 模型（`edge_metrics` 表：id/edge_id/metrics_json/collected_at），接入 AutoMigrate 装配
- [x] 2.2 新增 `MetricsRepository` 接口与 GORM 实现：`Append`（写入并清理超保留窗口旧行）、`ListSince`、`Prune`；保留窗口默认 24h 可配置
- [x] 2.3 agent `presence` handler 接收 `metrics` 并落库；鉴权失败不落库；补 handler 测试
- [x] 2.4 admin `GET /api/v1/edges/{id}/metrics`：`window` 解析（1h/6h/24h）、未知 Edge 404、无数据空序列、返回 `latest` + `series`
- [x] 2.5 admin-api 装配：把 `MetricsRepository` 注入 agent 与 edges handler，`METRICS_RETENTION` 环境变量接线

## 3. Admin 前端系统监控

- [x] 3.1 新增 `web/admin/src/components/ui/chart.tsx`（shadcn 现行 Tailwind v4 + `data-slot` 版 ChartContainer / ChartTooltip / ChartLegend / ChartStyle）
- [x] 3.2 `lib/api/types.ts`、`lib/api/edges.ts`、`query-keys.ts` 增加 metrics 类型与 `getEdgeMetrics(id, window)`
- [x] 3.3 `observation.ts` 新增 `parseMetrics`：解析 latest/series、可选 GPU 与 I/O 字段
- [x] 3.4 `observation-panel.tsx` 系统节替换为「系统监控」图表卡组：CPU 占用率面积图卡（抄 Total Revenue 卡 class 与渐变）
- [x] 3.5 内存环形图卡（抄 Sales by Category 卡：中心已用 %、图例已用/剩余、`var(--primary)`/`color-mix` 配色）
- [x] 3.6 GPU 占用率与显存环形图卡（每 GPU 一张；无 GPU 数据整组隐藏）
- [x] 3.7 I/O 双系列面积图卡（读/写两条 Area，`--color-read`/`--color-write` 与各自渐变）
- [x] 3.8 `detail-panel.tsx` 去掉 `getEdgeSystem`，改 `getEdgeMetrics` + `refetchInterval: 15000`；无历史数据渲染空态
- [x] 3.9 i18n：`observationSystem` 改为「系统监控」/「System Monitoring」，hint 改为「CPU、内存、GPU、I/O」
- [x] 3.10 合同测试锁定 chart 关键 class 与文案；补 `parseMetrics` 与空态测试
- [x] 3.11 CPU 占用率与内存占用率并排折线图，图高改为基准 50%（`h-[100px] w-full min-w-0 sm:h-[120px] lg:h-[140px]`）；内存占用字节作为内存占用率图的辅助序列（右轴字节）
- [x] 3.12 显卡占用率与显存占用率并排折线图（每张 GPU 一组），图高 50%；显存占用字节作为显存占用率图的辅助序列（右轴字节）；移除环形图卡
- [x] 3.13 I/O 读/写图高度改为 50%
- [x] 3.14 更新 i18n 标签（内存占用率/显存占用率）与合同测试（LineChart、50% 高度 class、无环形图 class）

## 4. 文档与验证

- [x] 4.1 同步 `docs/architecture/data-model.md`、`runtime.md`：`edge_metrics` 表与指标上报路径
- [x] 4.2 根 `README.md` 补充 `METRICS_INTERVAL` / `METRICS_RETENTION` 环境变量
- [x] 4.3 全量验证：`go build ./...` + `go test ./...`、前端 `tsc -b` + `vitest`，Mock 模式端到端确认详情页系统监控出图
