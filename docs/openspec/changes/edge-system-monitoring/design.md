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

## Risks / Trade-offs

- `nvidia-smi` 缺失 → GPU 图表空白 → 字段留空 + mock 模式有假数据；页面降级隐藏 GPU 卡，不报错。
- 磁盘 I/O 速率依赖两次采样差值 → 首次/重启后首拍无 I/O 数据 → 首拍该字段留空，后续正常。
- `edge_metrics` 随高频上报增长 → 每次 Append 顺带清理过期行，查询限窗口与点数；数据量级（30s 一拍 ≈ 2880 行/天/Edge）对 SQLite/MySQL/Postgres 均可接受。
- 多实例控制面并发写同一表 → 清理与写入均幂等，无事务依赖。
- 前端 chart 组件与 recharts v3 版本匹配 → 使用与参考页同代 `data-slot` 版 chart.tsx 并以合同测试锁 class；若升级 recharts 需回归图表渲染。

## Migration Plan

1. 后端先合并：新增 `edge_metrics` 表由 AutoMigrate 创建，无存量数据迁移；presence 与 admin 端点可平滑上线。
2. 前端后合：详情页切到新端点；旧 `GET /{id}/system` 端点保留但不再被详情页使用（队列/任务节不受影响）。
3. 回滚：前端回退 `getEdgeSystem` 代理即可恢复旧系统节；控制面停止写 `edge_metrics` 或直接弃表均不影响其它功能。

## Open Questions

无阻塞项。图表默认时间窗口（最近 1 小时）与保留窗口（24 小时）为可配置默认值，可在最终审视时调整。
