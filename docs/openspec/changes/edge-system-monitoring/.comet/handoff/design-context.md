# Comet Design Handoff

- Change: edge-system-monitoring
- Phase: design
- Mode: compact
- Context hash: 5182cd588ca1df28d4a51d71502d42ec7ba0af49c3961a5af9cbea923dc1a008

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## docs/openspec/changes/edge-system-monitoring/proposal.md

- Source: docs/openspec/changes/edge-system-monitoring/proposal.md
- Lines: 1-32
- SHA256: 8279371780953c6d59784862ce12ad9b49f1fc604bc97420f38fe8a34c1c5ebe

```md
## Why

节点详情页的「系统」观测目前由控制面反向连接 Edge 本机 ComfyUI 的 `/system_stats` 取数。Edge 部署在内网时控制面连不上，页面长期显示「连接被拒绝」。需要改成 Edge 在自己心跳（presence 上报）时主动携带实时系统指标，控制面记录这些信息，页面从服务端数据画图，彻底摆脱对 Edge 入站连接的依赖。

## What Changes

- 节点详情页「系统」节文案改为「系统监控」。
- Edge-Agent 新增实时指标采集：CPU 占用率、内存占用（已用字节）、内存占用率、GPU 占用率、GPU 显存占用（已用字节）、GPU 显存占用率、磁盘 I/O 信息（读/写速率）。采集结果随 presence 心跳上报到 `/agent/v1/presence`。
- 控制面持久化指标：新增 `edge_metrics` 记录每次上报的指标快照，并提供管理端查询接口 `GET /api/v1/edges/{id}/metrics`。
- Admin 节点详情页「系统监控」节改用 MHTML（Shadcnblocks Admin Kit）里的 chart 组件与样式：同一套 shadcn `ChartContainer` 组件（recharts + Tailwind v4，`data-slot="chart"` 与同款 class 串），卡片、图例、渐变、配色与参考页一致，不只是抄布局。
- 管理端系统节不再直连 Edge 取数；「连接被拒绝」类错误不再出现在该节。

## Capabilities

### New Capabilities
- `edge-metrics`: Edge 心跳上报实时系统指标、控制面持久化并提供查询 API、Admin 节点详情页以参考页同款 chart 组件展示系统监控图表。

### Modified Capabilities
<!-- 本次为新增能力，不修改既有 spec 级行为。 -->

## 范围与非目标

范围：Edge-Agent 指标采集与心跳上报、控制面 `edge_metrics` 持久化与查询接口、节点详情页「系统监控」图表（CPU / 内存 / GPU / I/O）。不修改「队列」与「最近任务」节、不修改右列静态机器规格语义。

非目标：不做实时推送（WebSocket/SSE）；不引入 NVML / ROCm（GPU 占用率用 `nvidia-smi`，无 NVIDIA 时字段留空）；不改调度、健康检查与任务执行路径；不做多节点聚合或全局监控页。

## Impact

- `apps/edge-agent`：新增 `internal/metrics` 采集器（CPU/内存/GPU/磁盘 I/O），presence 载荷扩展 `metrics` 字段；新增 `gopsutil` 依赖（GPU 占用率用 `nvidia-smi` 查询，不引入 NVML）。
- 控制面：`internal/platform` 新增指标领域与 `edge_metrics` 持久化（AutoMigrate）；`internal/httpapi/agent` 的 presence 处理接收并落库指标；`internal/httpapi/edges` 新增 metrics 查询端点。
- Admin 前端：新增 `web/admin/src/components/ui/chart.tsx`（shadcn chart 组件）；`features/edges` 的系统监控图表、i18n 文案、API 类型与合同测试。
- 文档：`docs/architecture/data-model.md`、`runtime.md` 等同步 `edge_metrics` 与上报路径。

```

## docs/openspec/changes/edge-system-monitoring/design.md

- Source: docs/openspec/changes/edge-system-monitoring/design.md
- Lines: 1-97
- SHA256: e9963e9cb3599401e05265f3b36c69ca331ff14ac946387ae8620b8f6c9bce56

[TRUNCATED]

```md
## Context

现状（参见 proposal.md - Why）：节点详情页「系统」节由控制面代理 `GET /{id}/system` 直连 Edge 本机 ComfyUI，内网 Edge 下必然失败并显示「连接被拒绝」。Edge-Agent 已有 5 秒一次的 presence 心跳（`/agent/v1/presence`），并已携带静态硬件规格（ghw + Comfy `system_stats` 的型号/容量），但没有任何实时指标。管理前端目前没有 chart 组件（`components/ui/chart.tsx` 缺失），`recharts@3.8.1` 与 Tailwind v4 已就绪；参考页 MHTML（Shadcnblocks Admin Kit dashboard-3）的图表即 recharts + shadcn `ChartContainer`（`data-slot="chart"`）体系。

## Goals / Non-Goals

**Goals:**
- Edge 侧采集 CPU/内存/GPU/磁盘 I/O 实时指标，随 presence 心跳推送给控制面，控制面落库并提供管理端查询。
- 节点详情页「系统监控」用与 MHTML 相同的 chart 组件与 class（`data-slot="chart"`、`[&_.recharts-*]` 修饰、`aspect-video`、图例、渐变、`var(--primary)`/`color-mix` 配色）展示指标，不再直连 Edge。
- 沿用现有测试契约方式（Vitest 合同测试锁 class、Go 单测锁 handler/持久化）。

**Non-Goals:**
- 不引入 NVML / ROCm；GPU 占用率用 `nvidia-smi` 查询，非 NVIDIA 或不可用时字段留空。
- 不做实时推送（WebSocket/SSE）；沿用 presence 心跳 + 管理端轮询。
- 不改静态机器规格（右列 CPU/内存/显卡）语义，不改调度与健康检查。
- 不做多节点聚合或全局监控页；只改节点详情页系统节。

## Decisions

### 1. Edge 指标采集：gopsutil + nvidia-smi（不引入 NVML）

新增 `apps/edge-agent/internal/metrics` 包：
- CPU：`gopsutil/cpu.Percent(0, false)` 取整机占用率。
- 内存：`gopsutil/mem.VirtualMemory()` 取 `Used`、`Total`、`UsedPercent`。
- 磁盘 I/O：`gopsutil/disk.IOCounters()` 汇总各设备 `read_bytes` / `write_bytes`，与上次采样做差值除以间隔得到字节/秒速率；首次采样无差值时 I/O 字段留空，后续每拍都有。
- GPU：解析 `nvidia-smi --query-gpu=index,name,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits`；每个 GPU 得到名称、占用率、显存已用/总量，占用率与显存占用率由数据直接得出。`nvidia-smi` 缺失/执行失败时 GPU 字段留空。
- Mock：`COMFY_MOCK=true` 时合成一张假 GPU 与稳定波动的 CPU/内存/显存数值，保证开发与演示环境页面有图（与现有 Comfy mock 同思路，真实主机仍走真实采集）。

备选：`go-nvml` 库直读 NVIDIA 驱动。放弃原因是 compute-nodes 设计已明确「不上 NVML」，且 `nvidia-smi` 随 NVIDIA 驱动自带、无新增二进制依赖；gopsutil 相对手读 `/proc` 跨平台且与既有 ghw 风格一致。

### 2. 采样节奏：默认 30 秒采样，随 presence 心跳捎带

presence 仍 5 秒一拍；Reporter 内部记录上次采样时间，距上次超过 `METRICS_INTERVAL`（默认 30s，环境变量可配）才重新采集，并把新快照放进本次 presence 载荷。这样「心跳时上报」成立，又避免每 5 秒跑一次 `nvidia-smi`。载荷新增 `metrics` 字段（`edge.Metrics`），与现有 `hardware` 字段并列；响应不变。

### 3. 数据模型：`edge_metrics` 表，整快照 JSON 一列

```go
// internal/platform/edge 新增类型
type GPUMetric struct {
    Name              string   `json:"name"`
    UsagePercent      *float64 `json:"usage_percent,omitempty"`
    VRAMUsedBytes     uint64   `json:"vram_used_bytes,omitempty"`
    VRAMTotalBytes    uint64   `json:"vram_total_bytes,omitempty"`
    VRAMUsagePercent  *float64 `json:"vram_usage_percent,omitempty"`
}

type Metrics struct {
    CPUUsagePercent     float64     `json:"cpu_usage_percent"`
    MemUsedBytes        uint64      `json:"mem_used_bytes"`
    MemTotalBytes       uint64      `json:"mem_total_bytes"`
    MemUsagePercent     float64     `json:"mem_usage_percent"`
    GPUs                []GPUMetric `json:"gpus,omitempty"`
    DiskReadBytesPerSec *float64    `json:"disk_read_bytes_per_sec,omitempty"`
    DiskWriteBytesPerSec *float64   `json:"disk_write_bytes_per_sec,omitempty"`
    CollectedAt         time.Time   `json:"collected_at"`
}
```

`edge_metrics` 行：自增 `id`、`edge_id`（索引）、`metrics_json`（上述完整快照）、`collected_at`（索引）。整快照一列与现有 `edges.hardware_json` 模式一致，字段演进不用改表结构。查询按 `collected_at` 升序取窗口内数据。

保留策略：每次 `Append` 后顺带删除 `collected_at < now - retention` 的旧行；保留窗口默认 24 小时，控制面环境变量 `METRICS_RETENTION` 可调。管理端查询默认返回最近 1 小时、最多 720 个点（约 6 小时 @30s），`window` 参数可选 `1h|6h|24h`，超出上限截尾保留最近点。

### 4. 控制面接入点

- Agent `presence` handler：请求体增加 `metrics *edge.Metrics`；鉴权通过后若非空则写入 `MetricsRepository.Append`。鉴权失败不落库（spec 已覆盖）。
- Admin `edges` handler：新增 `GET /{id}/metrics`，返回 `{ latest, series }`；Edge 不存在返回 404，无数据返回空序列。
- 装配：`apps/admin-api/cmd/admin-api/main.go` 与测试用 `appboot` 中把 `MetricsRow` 加入 AutoMigrate，并把 `MetricsRepository` 注入 agent 与 admin handler。

### 5. 前端：补齐 shadcn chart 组件 + 同款卡片

- 新增 `web/admin/src/components/ui/chart.tsx`（shadcn/ui 现行 Tailwind v4 + `data-slot` 版：`ChartContainer`、`ChartTooltip`、`ChartTooltipContent`、`ChartLegend`、`ChartLegendContent`、`ChartStyle`），与参考 MHTML 渲染出的 `data-slot="chart"` 与 class 串同源。
- `features/edges/observation-panel.tsx` 的 `SystemSection` 替换为「系统监控」图表卡片组，卡片结构逐条对照 dashboard-3：
  - CPU 占用率：面积图卡，抄「Total Revenue」卡（`bg-card flex min-w-0 flex-1 flex-col gap-4 rounded-xl border p-4 sm:gap-6 sm:p-6`；顶部大数字 + 小标签 + 图例点 `size-2.5 rounded-full sm:size-3`；图高 `h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]`；`--color-cpu: var(--primary)`；渐变 0.3→0.05；Area strokeWidth 2）。
  - 内存：环形图卡，抄「Sales by Category」卡（`bg-card flex flex-1 flex-col gap-4 rounded-xl border p-4 sm:p-5`；图 `relative size-[100px] shrink-0 sm:size-[120px]`；中心叠「已用 %」+ 小标签；图例行 `size-2 rounded-full sm:size-2.5` + `tabular-nums`；配色 `var(--primary)` / `color-mix(in oklch, var(--primary) 75%, var(--background))`）。
  - GPU 占用率与显存：每张 GPU 各一张环形图卡（占用率环形中心为百分比，显存环形中心为已用字节），无 GPU 数据整组隐藏。
  - I/O：双系列面积图卡，抄「Total Revenue」的 This Year/Prev Year 双系列（读/写两条 Area，`--color-read: var(--primary)`、`--color-write: color-mix(... 75% ...)`，各自渐变），y 轴字节/秒。
  - 空数据：无历史时渲染空态，不渲染无数据图表。
- `detail-panel.tsx` 去掉 `getEdgeSystem`，改 `getEdgeMetrics` + `refetchInterval: 15000`；`queryKeys.edges.metrics`；i18n `observationSystem` → `系统监控` / `System Monitoring`，hint 更新为「CPU、内存、GPU、I/O」。
- 合同测试：在 `edges.contract.test.ts` / 新增测试中锁定上述关键 class 字符串与 zh.json 文案；Go 侧为 metrics 持久化、presence 落库、admin metrics 端点补单测。


```

Full source: docs/openspec/changes/edge-system-monitoring/design.md

## docs/openspec/changes/edge-system-monitoring/tasks.md

- Source: docs/openspec/changes/edge-system-monitoring/tasks.md
- Lines: 1-36
- SHA256: e7dd4a1421fe1d59f2e4cdffaa8198cb7b36fd21f3374d4fa8db48169993c06e

```md
## 1. Edge 指标采集

- [ ] 1.1 在 `internal/platform/edge` 新增 `Metrics` / `GPUMetric` 领域类型（JSON 标签、可选字段用指针/omitempty）
- [ ] 1.2 新增 `apps/edge-agent/internal/metrics` 包：CPU 占用率与内存占用/总量/占用率采集（gopsutil）
- [ ] 1.3 磁盘 I/O 读/写速率采集（gopsutil `disk.IOCounters` 差值除间隔，首拍留空）
- [ ] 1.4 GPU 指标采集：解析 `nvidia-smi`（占用率、显存已用/总量、占用率），不可用时留空；`COMFY_MOCK=true` 时合成假 GPU 数据
- [ ] 1.5 metrics 采集器单测：mock 模式、部分指标不可用、首拍 I/O 空、数值边界
- [ ] 1.6 `presence.Reporter` 按 `METRICS_INTERVAL`（默认 30s）采样，随 presence 载荷携带 `metrics`
- [ ] 1.7 `pull.Client.ReportPresence` 请求体支持 `metrics` 字段并补测试

## 2. 控制面持久化与 API

- [ ] 2.1 新增 `MetricsRow` GORM 模型（`edge_metrics` 表：id/edge_id/metrics_json/collected_at），接入 AutoMigrate 装配
- [ ] 2.2 新增 `MetricsRepository` 接口与 GORM 实现：`Append`（写入并清理超保留窗口旧行）、`ListSince`、`Prune`；保留窗口默认 24h 可配置
- [ ] 2.3 agent `presence` handler 接收 `metrics` 并落库；鉴权失败不落库；补 handler 测试
- [ ] 2.4 admin `GET /api/v1/edges/{id}/metrics`：`window` 解析（1h/6h/24h）、未知 Edge 404、无数据空序列、返回 `latest` + `series`
- [ ] 2.5 admin-api 装配：把 `MetricsRepository` 注入 agent 与 edges handler，`METRICS_RETENTION` 环境变量接线

## 3. Admin 前端系统监控

- [ ] 3.1 新增 `web/admin/src/components/ui/chart.tsx`（shadcn 现行 Tailwind v4 + `data-slot` 版 ChartContainer / ChartTooltip / ChartLegend / ChartStyle）
- [ ] 3.2 `lib/api/types.ts`、`lib/api/edges.ts`、`query-keys.ts` 增加 metrics 类型与 `getEdgeMetrics(id, window)`
- [ ] 3.3 `observation.ts` 新增 `parseMetrics`：解析 latest/series、可选 GPU 与 I/O 字段
- [ ] 3.4 `observation-panel.tsx` 系统节替换为「系统监控」图表卡组：CPU 占用率面积图卡（抄 Total Revenue 卡 class 与渐变）
- [ ] 3.5 内存环形图卡（抄 Sales by Category 卡：中心已用 %、图例已用/剩余、`var(--primary)`/`color-mix` 配色）
- [ ] 3.6 GPU 占用率与显存环形图卡（每 GPU 一张；无 GPU 数据整组隐藏）
- [ ] 3.7 I/O 双系列面积图卡（读/写两条 Area，`--color-read`/`--color-write` 与各自渐变）
- [ ] 3.8 `detail-panel.tsx` 去掉 `getEdgeSystem`，改 `getEdgeMetrics` + `refetchInterval: 15000`；无历史数据渲染空态
- [ ] 3.9 i18n：`observationSystem` 改为「系统监控」/「System Monitoring」，hint 改为「CPU、内存、GPU、I/O」
- [ ] 3.10 合同测试锁定 chart 关键 class 与文案；补 `parseMetrics` 与空态测试

## 4. 文档与验证

- [ ] 4.1 同步 `docs/architecture/data-model.md`、`runtime.md`：`edge_metrics` 表与指标上报路径
- [ ] 4.2 根 `README.md` 补充 `METRICS_INTERVAL` / `METRICS_RETENTION` 环境变量
- [ ] 4.3 全量验证：`go build ./...` + `go test ./...`、前端 `tsc -b` + `vitest`，Mock 模式端到端确认详情页系统监控出图

```

## docs/openspec/changes/edge-system-monitoring/specs/edge-metrics/spec.md

- Source: docs/openspec/changes/edge-system-monitoring/specs/edge-metrics/spec.md
- Lines: 1-57
- SHA256: 501a246675e015833d7a56bfa1eaa4fa3dc1b7335236acbfceb4fc296bc36d58

```md
## Purpose

让节点详情页在不依赖控制面直连 Edge 的前提下展示实时系统指标：由 Edge 心跳主动上报、控制面持久化记录、管理端以参考页同款 chart 组件展示图表。

## ADDED Requirements

### Requirement: Edge 心跳上报系统指标

Edge-Agent MUST 在每次 presence 心跳上报时携带最近一次采集的系统指标快照。快照 MUST 包含以下字段：CPU 占用率（百分比）、内存已用字节、内存总量字节、内存占用率（百分比）、GPU 列表（每张 GPU 的名称、占用率百分比、显存已用字节、显存总量字节、显存占用率百分比）、磁盘 I/O 读速率与写速率（字节/秒）。任一单项指标采集失败或当前不可用时，MUST 允许该字段为空或以明确标记呈现，且 MUST NOT 阻止其余字段的正常上报。控制面收到快照后 MUST 将其记录到该 Edge 名下。

#### Scenario: 内网 Edge 正常上报
- **WHEN** 一台部署在内网的 Edge 按心跳周期完成一次系统指标采集并上报
- **THEN** 控制面成功接收并记录该快照，快照包含 CPU 占用率、内存占用与占用率、GPU 占用率与显存占用/占用率、I/O 读/写速率字段

#### Scenario: 部分指标不可用
- **WHEN** Edge 所在机器没有可查询的 GPU 占用率来源（例如非 NVIDIA 主机）
- **THEN** 该 Edge 仍上报 CPU、内存与 I/O 等可用指标，GPU 字段为空或带不可用标记，上报不被整体拒绝

#### Scenario: 心跳鉴权失败
- **WHEN** Edge 携带非法或缺失的 agent token 上报指标
- **THEN** 控制面拒绝该上报且不记录指标

### Requirement: 控制面持久化并查询指标

系统 MUST 为每个 Edge 持久化每次上报的系统指标快照及采集时间，并提供管理端查询接口 `GET /api/v1/edges/{id}/metrics`。该接口 MUST 返回指定时间窗口内按时间升序排列的指标序列与最新快照；查询不存在的 Edge 时 MUST 返回未找到错误；Edge 存在但尚无任何指标时 MUST 返回空序列。系统 MUST 定期清理超出保留窗口的历史数据，保留窗口默认为 24 小时且可通过配置调整。

#### Scenario: 查询存在指标
- **WHEN** 管理端查询一台已多次上报指标的 Edge 的 metrics 接口
- **THEN** 响应包含按时间升序的指标序列，每项含 CPU、内存、GPU 与 I/O 字段及采集时间

#### Scenario: 查询暂无指标的 Edge
- **WHEN** 管理端查询一台已登记但从未上报过指标的 Edge
- **THEN** 响应为 200 且指标序列为空，页面可呈现空态

#### Scenario: 查询不存在的 Edge
- **WHEN** 管理端查询一个不存在的 Edge ID
- **THEN** 响应为未找到错误

#### Scenario: 过期数据被清理
- **WHEN** 系统写入新指标且存在超出保留窗口的旧记录
- **THEN** 旧记录被清理，查询结果只包含窗口内的数据

### Requirement: Admin 节点详情页系统监控视图

节点详情页原有「系统」节 MUST 改名为「系统监控」。该节 MUST 从服务端指标接口取数并展示图表，MUST NOT 依赖控制面直连 Edge 或 Edge 本机 ComfyUI 获取系统数据；因此当 Edge 位于内网、控制面无法入站连接时，该节 MUST 仍能显示已上报的最新指标与历史图表，不得显示「连接被拒绝」等直连错误。展示 MUST 覆盖 CPU 占用率、内存占用与占用率、GPU 占用率、GPU 显存占用与占用率、磁盘 I/O 读/写速率。图表 MUST 使用参考 MHTML（Shadcnblocks Admin Kit dashboard-3）中同一套 chart 组件与样式体系，包括 `data-slot="chart"` 容器、`[&_.recharts-*]` 样式修饰、`aspect-video`、卡片图例、渐变填充与 `var(--primary)` / `color-mix` 配色，不得仅复制布局。页面 MUST 周期性刷新指标数据。

#### Scenario: 内网 Edge 展示监控图表
- **WHEN** 管理员打开一台已上报指标的内网 Edge 的详情页
- **THEN** 「系统监控」节显示 CPU、内存、GPU 与 I/O 图表，数据来自服务端记录，页面不出现直连错误

#### Scenario: 无历史数据
- **WHEN** 管理员打开一台尚未上报指标的 Edge 的详情页
- **THEN** 「系统监控」节显示空态且不渲染无数据的图表

#### Scenario: 无 GPU 数据
- **WHEN** Edge 上报的 GPU 字段为空
- **THEN** 系统监控节隐藏或降级 GPU 相关图表，CPU、内存与 I/O 图表正常显示

```
