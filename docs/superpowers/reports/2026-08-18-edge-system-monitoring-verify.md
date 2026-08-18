# Verification Report: edge-system-monitoring

- Date: 2026-08-18
- 语言: zh-CN
- 范围: `e3e0870cd9741dd665a3ce5dbe24bf1b59ea290a..HEAD`（实现区间）
- verify_mode: full（任务数 25 > 3，变更文件 74 > 8）

## Summary

| 维度 | 状态 |
|---|---|
| Completeness | 25/25 任务完成；3 条需求（10 个场景）实现 |
| Correctness | 3/3 需求实现，场景均有实现或测试覆盖 |
| Coherence | 设计决策基本一致；1 处偏差已记录（Implementation Divergence §8） |

## Completeness

- tasks.md：25/25 `[x]`（`openspec instructions apply` 返回 `state: all_done`）
- Superpowers plan：73/73 步骤勾选
- delta spec：`specs/edge-metrics/spec.md` 3 条需求全部落地

## Correctness

### Requirement 1：Edge 心跳上报系统指标

- 实现：`apps/edge-agent/internal/metrics/collect.go`（CPU/内存/磁盘 I/O/gpu nvidia-smi/mock）、`presence/reporter.go`（按 `METRICS_INTERVAL` 采样携带 `metrics`）、`pull/client.go`（载荷）
- 场景覆盖：
  - 内网 Edge 正常上报 → Mock 端到端实测（每 5s 上报，CPU/内存/Mock GPU/磁盘 I/O 字段齐全）
  - 部分指标不可用 → `TestSample_NvidiaUnavailableKeepsOtherFields`
  - 心跳鉴权失败 → `TestAgent_PresenceUnauthorizedDoesNotStoreMetrics`

### Requirement 2：控制面持久化并查询指标

- 实现：`internal/platform/edge/persistence/gorm_metrics.go`（Append/ListSince/过期清理）、`internal/httpapi/agent/handler.go`（落库）、`internal/httpapi/edges/handler.go`（`GET /{id}/metrics`）
- 场景覆盖：`TestMetricsRepository_AppendAndList`、`TestMetricsRepository_PrunesOldRows`、`TestHandler_MetricsEndpoint`（含 404）、`TestHandler_MetricsEmptySeries`、`TestAgent_PresenceStoresMetrics`

### Requirement 3：Admin 节点详情页系统监控视图

- 实现：`web/admin/src/components/ui/chart.tsx`（shadcn chart）、`observation-panel.tsx`（CPU 面积图/内存环形图/GPU 环形图/I/O 双系列面积图）、`detail-panel.tsx`（`getEdgeMetrics` + 15s 轮询）、i18n（系统监控 / System Monitoring）
- 场景覆盖：合同测试锁定 chart class 与文案；`parseMetrics` 空态/缺省测试；无 GPU 降级（`GpuCards` 返回 null）；内网场景由端到端验证（控制面不连 Edge 也可从服务端数据出图）

## Coherence

- 决策符合：GPU 只做 NVIDIA（nvidia-smi，无 NVML/ROCm）；`edge_metrics` 整快照 JSON 一列；presence 5s + `METRICS_INTERVAL` 默认 30s 下限 5s；保留窗口默认 24h；`GET /metrics` 默认 1h、最多 720 点；admin-api 与 pixoma 双装配
- 偏差记录：CPU 采样从 `interval>0`（gopsutil 会 Sleep）改为 `interval=0` 非阻塞，见 Design Doc §8 Implementation Divergence，回归测试 `TestSample_CPUPercentMustNotSleep`

## Issues

### CRITICAL

无。

### WARNING

无。

### SUGGESTION（已接受，来自 build review notes）

1. CPU 采集失败上报 0 与真实 0% 不可区分（可接受）
2. 磁盘 I/O 采集失败时基线不更新、下一拍差值偏大（罕见，可接受）
3. `GET /metrics` 非法 window 的 400 未单测（代码路径简单，可接受）

## 证据

- `go build ./... && go test ./...`：exit 0，无失败输出（含 `TestSample_CPUPercentMustNotSleep`、nvidia-smi CSV 解析测试）
- `web/admin`：`npx tsc -b` 通过；`npx vitest run` 30 文件 / 104 测试通过；`npm run build` 成功
- Mock 端到端：临时 DATA_DIR 起 pixoma + edge-agent（`METRICS_INTERVAL=5s`），向导初始化后 `GET /api/v1/edges/local/metrics` 返回 CPU/内存/Mock GPU/磁盘 I/O 系列，验证控制面不依赖入站连接 Edge
- 构建证据已记录：`comet state record-check build`（npm run build、go build ./...）

## 工作区说明

工作树存在大量未提交改动，归属为用户并行开发（pagination、channel-platform-refactor 新 change、bot/sessions/setup 等），与本次 change 的提交区间不重叠；本次验证以提交区间为准，未触碰用户改动。`docs/openspec/changes/edge-system-monitoring/.comet/*` 的 dirty 为本流程运行时状态，归档时统一提交。

## Assessment

无 CRITICAL/WARNING，3 条需求与 10 个场景全部覆盖，Ready for archive。
