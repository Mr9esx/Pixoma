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

---

# Round 2（2026-08-18，图表布局调整后复审）

## 变更

用户要求调整系统监控布局（archive 挂起后回退 build 实施，`verify_failures=1`）：
1. CPU 占用率与内存占用率并排折线图，图高为基准 50%（`h-[100px] w-full min-w-0 sm:h-[120px] lg:h-[140px]`）
2. 显卡占用率与显存占用率并排折线图（每张 GPU 一组），图高 50%
3. 内存占用（字节）与显存占用（字节）分别作为内存占用率图/显存占用率图的辅助序列（右轴字节）
4. I/O 读/写图高 50%

同步更新：delta spec R3 布局要求、Design Doc §5 卡片表与布局说明、tasks.md 3.11–3.14、i18n（monitorMemRate / monitorVramRate）、合同测试（LineChart、50% 高度 class、移除环形图断言）。

## Round 2 验证结果

| 检查项 | 结果 |
|---|---|
| tasks.md 全部勾选（29/29） | PASS |
| 实现符合更新后的 spec R3 布局要求 | PASS（`observation-panel.tsx`：`LineCardShell` + `CHART_HALF_HEIGHT`、`CpuCard`/`MemRateCard` 并排、`GpuLineCards` 每 GPU 两卡并排、`IOCard` 高度 50%；环形图卡已移除） |
| 编译通过 | PASS（`go build ./...` exit 0；`web/admin npm run build` 成功） |
| 测试通过 | PASS（`go test ./...` 全绿；`npx tsc -b` + `npx vitest run` 30 文件/104 测试通过） |
| 无安全回归 | PASS（无新密钥/unsafe；仅前端展示层调整） |
| 代码审查（standard） | build 阶段已审 diff，本轮仅前端布局改动且合同测试锁定关键 class；review-notes.md 已记录 |

## Round 2 结论

无 CRITICAL/WARNING。Ready for archive（待用户归档确认）。

---

# Round 3（2026-08-18，数值可读性与图表内边距复审）

## 变更

用户反馈三点（archive 挂起后回退 build，`verify_failures` 重置）：
1. 图表「已用」显示原始字节大数值 → `formatBytes` 改为自适应二进制单位（B/KiB/MiB/GiB/TiB，1 位小数、≥100 取整），内存占用/显存占用/I/O 读/写序列的 tooltip 与坐标轴 tick 全部走该格式化；百分比序列 tooltip 补 `%`
2. chart 内边距过大 → 左右 margin 12→4px，百分比轴宽 36、字节轴宽 48
3. I/O Y 轴一直为 0 → 根因是旧 `formatBytes` 对 <0.5 MiB 的值四舍五入为「0 MiB」；自适应格式化后小速率正常显示（如 349.5 KiB）

单位说明：贴合实际采用 GiB/MiB（内存/显存为 1024 进制），用户已确认不强制 MB/GB。

## Round 3 验证结果

| 检查项 | 结果 |
|---|---|
| tasks.md 全部勾选（32/32） | PASS |
| 实现符合要求 | PASS（`observation.ts` 自适应 `formatBytes`；`observation-panel.tsx` `chartTooltipFormatter` + margin/轴宽收窄） |
| 编译通过 | PASS（`npm run build` 成功；`go build ./...` 通过） |
| 测试通过 | PASS（`npx vitest run` 30 文件/104 测试；`go test ./...` 全绿） |
| 无安全回归 | PASS（纯展示层改动） |

## Round 3 结论

无 CRITICAL/WARNING。Ready for archive（待用户归档确认）。

---

# Round 4（2026-08-18，I/O 轴阶梯化复审）

## 变更

用户反馈 I/O Y 轴小数零碎难看 → 新增 `ioAxisTicks`（observation.ts 纯逻辑，含 4 条单测）：按显示单位（B/KiB/MiB/GiB，取 max/unit ≥ 4 的最大单位）以 1/2/5×10ⁿ 步进生成整齐 tick，Y 轴只显示整单位（如 0/200/400/600 KiB）；tooltip 仍显示精确值。

## Round 4 验证结果

| 检查项 | 结果 |
|---|---|
| tasks.md 全部勾选（33/33） | PASS |
| 实现符合要求 | PASS（`ioAxisTicks` 单测覆盖 KiB/MiB/B/空数据） |
| 编译通过 | PASS（`npm run build` 成功） |
| 测试通过 | PASS（`npx vitest run` 30 文件/108 测试；`go test ./...` 全绿） |
| 无安全回归 | PASS（纯展示层） |

## Round 4 结论

无 CRITICAL/WARNING。Ready for archive（待用户归档确认）。
