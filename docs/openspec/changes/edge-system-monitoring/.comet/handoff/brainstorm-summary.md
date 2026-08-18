# Brainstorm Summary

- Change: edge-system-monitoring
- Date: 2026-08-18

## 确认的技术方案

（2026-08-18 已由用户确认设计方案）基于已确认的 OpenSpec 产物，深度技术设计如下：

- Edge 采集：新增 `apps/edge-agent/internal/metrics`，gopsutil 采 CPU 占用率、内存占用/总量/占用率、磁盘 I/O 读/写速率（差值法，首拍留空）；`nvidia-smi` 采 GPU 占用率与显存占用/总量/占用率（不可用留空，`COMFY_MOCK=true` 合成假 GPU）。

已确认（2026-08-18，用户选择方案 2）：GPU 占用率**只做 NVIDIA**（`nvidia-smi`）；AMD / Intel 不单独接采集，字段留空；不引入 NVML / ROCm。非 NVIDIA 主机仅 CPU/内存/I/O 有数据，页面隐藏 GPU 卡。
- 上报：presence 心跳仍 5s，`METRICS_INTERVAL`（默认 30s）才采样，新快照随 presence 载荷 `metrics` 字段上报；鉴权失败不落库。
- 落库：`edge_metrics` 表（自增 id、edge_id、metrics_json 整快照、collected_at），Append 时顺带清理超 `METRICS_RETENTION`（默认 24h）旧行。
- API：`GET /api/v1/edges/{id}/metrics?window=1h|6h|24h`（默认 1h，最多 720 点），返回 `{latest, series}`；未知 Edge 404，无数据空序列。
- 前端：新增 shadcn `components/ui/chart.tsx`（data-slot 版）；系统节改「系统监控」图表卡组——CPU 面积图卡（抄 Total Revenue）、内存环形图卡（抄 Sales by Category）、GPU 占用/显存环形图卡（每 GPU 一张）、I/O 双系列面积图卡；`getEdgeMetrics` + 15s 轮询；i18n 文案与合同测试锁 class。

## 关键取舍与风险

- 取舍：nvidia-smi 而非 NVML（沿用 compute-nodes「不上 NVML」约束，无新增驱动依赖）；整快照 JSON 一列而非分列（与 hardware_json 一致，字段演进免迁移）；单 goroutine 随心跳采样而非独立采样器（无并发复杂度，nvidia-smi 30s 一次开销可接受）。
- 风险：nvidia-smi 缺失 → GPU 卡隐藏；首拍无 I/O 差值 → 该字段留空；表增长 → 写入时清理 + 查询限窗口点数；多实例并发写 → 幂等删除无事务依赖。

## 测试策略

- Go：metrics 采集器单测（mock/部分不可用/首拍 I/O）；presence handler 落库与鉴权拒绝；`MetricsRepository` Append/ListSince/Prune；admin metrics 端点（window、404、空序列）。
- 前端：`parseMetrics` 单测；合同测试锁定 chart class 与「系统监控」文案；空态/无 GPU 降级测试。
- 端到端：Mock 模式起服务，详情页系统监控出图。

## Spec Patch

无（open 阶段 delta spec 已覆盖需求与验收场景）。
