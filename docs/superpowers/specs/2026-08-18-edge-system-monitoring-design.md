---
comet_change: edge-system-monitoring
role: technical-design
canonical_spec: openspec
---

# 深度技术设计：Edge 系统监控

## 1. 背景与目标

参见 OpenSpec `docs/openspec/changes/edge-system-monitoring/`：节点详情页「系统」节此前由控制面直连 Edge 本机 ComfyUI，内网 Edge 下必现「连接被拒绝」。本次改为 Edge 在 presence 心跳中主动上报实时系统指标，控制面落库，前端用与参考 MHTML（Shadcnblocks Admin Kit dashboard-3）同源的 chart 组件与 class 展示「系统监控」图表。

已确认的边界：GPU 占用率只做 NVIDIA（`nvidia-smi`），AMD/Intel 字段留空；不引入 NVML/ROCm；不做实时推送；不改调度与健康检查；静态机器规格（右列）语义不变。

## 2. 架构与数据流

```text
┌─────────────────────────────┐      presence 心跳（5s）      ┌──────────────────────────────┐
│  Edge-Agent                 │ ─────────────────────────────▶ │  Control Plane (admin-api)   │
│  internal/metrics           │   { edge_id, comfy_running,    │  POST /agent/v1/presence     │
│  ├ CPU % (gopsutil)         │     hardware?, metrics? }      │  ├ Presence.Report           │
│  ├ Mem used/total/% (gopsutil)│                             │  └ Metrics.Append ──▶ edge_metrics 表
│  ├ Disk I/O rate (gopsutil) │                              └───────────▲──────────────────┘
│  └ GPU usage/VRAM (nvidia-smi)│                                          │ GET /api/v1/edges/{id}/metrics
│  └ Mock 合成（COMFY_MOCK=true）│                                          │ { latest, series }（window 过滤）
└─────────────────────────────┘                                          ▼
                                                          ┌──────────────────────────────┐
                                                          │  Admin Web（节点详情页）       │
                                                          │  「系统监控」chart 卡组        │
                                                          │  15s 轮询 getEdgeMetrics      │
                                                          └──────────────────────────────┘
```

采样与上报解耦在同一个 Reporter goroutine 内：presence 每 5s 一拍；距上次采样超过 `METRICS_INTERVAL`（默认 30s）才重新采集并随本次心跳携带 `metrics`，其余拍不带（服务端不落库）。

## 3. 详细设计

### 3.1 Edge 指标采集器 `apps/edge-agent/internal/metrics`

对外入口：

```go
type Sampler struct {
    Now       func() time.Time
    InspectCPU   func(ctx context.Context, interval time.Duration) ([]float64, error)
    InspectMem   func(ctx context.Context) (*mem.VirtualMemoryStat, error)
    InspectDisk  func(ctx context.Context) (map[string]disk.IOCountersStat, error)
    QueryNvidiaSMI func(ctx context.Context) ([]edge.GPUMetric, error) // nil 表示不可用
    Mock       bool // COMFY_MOCK=true 时 GPU 用合成数据
}

func (s *Sampler) Sample(ctx context.Context, since time.Time) edge.Metrics
```

- **CPU**：`cpu.PercentWithContext(ctx, interval, false)`，`interval` 为距上次采样的时长，得到两次采样间的整机占用率。首次采样无历史，取 0 并正常上报。
- **内存**：`mem.VirtualMemoryWithContext(ctx)` → `Used`、`Total`、`UsedPercent`。
- **磁盘 I/O**：`disk.IOCountersWithContext(ctx)` 汇总所有设备的 `read_bytes` / `write_bytes`，与上次采样差值除以间隔得到字节/秒速率；首次采样（`since` 为零值）时速率字段为 `nil`。
- **GPU（仅 NVIDIA）**：执行

  ```text
  nvidia-smi --query-gpu=index,name,utilization.gpu,memory.used,memory.total --format=csv,noheader,nounits
  ```

  用 `encoding/csv` 解析（`name` 可能含逗号），显存单位 MiB 换算为字节；`usage_percent`、`vram_used_bytes`、`vram_total_bytes`、`vram_usage_percent` 均由数据直接得出。命令不存在、退出非零或解析失败时整组 GPU 字段返回 `nil`（不阻断 CPU/内存/I/O）。
- **Mock**：`COMFY_MOCK=true` 时跳过 `nvidia-smi`，合成一张 `Mock GPU`（占用率 20–60% 波动、显存总量 8 GiB、已用随占用率变化），保证开发与演示环境页面有 GPU 图；CPU/内存/I/O 仍走真实采集。
- **失败语义**：任一单项失败只留空该项；`edge.Metrics` 的 GPU 与 I/O 速率用 `*float64`/切片 + `omitempty` 表达“不可用”，CPU/内存字段必有值。整体采集不返回 error（局部失败不阻塞心跳）。

领域类型放在 `internal/platform/edge/metrics.go`（agent 与控制面共用）：

```go
type GPUMetric struct {
    Name             string   `json:"name"`
    UsagePercent     *float64 `json:"usage_percent,omitempty"`
    VRAMUsedBytes    uint64   `json:"vram_used_bytes,omitempty"`
    VRAMTotalBytes   uint64   `json:"vram_total_bytes,omitempty"`
    VRAMUsagePercent *float64 `json:"vram_usage_percent,omitempty"`
}

type Metrics struct {
    CPUUsagePercent      float64     `json:"cpu_usage_percent"`
    MemUsedBytes         uint64      `json:"mem_used_bytes"`
    MemTotalBytes        uint64      `json:"mem_total_bytes"`
    MemUsagePercent      float64     `json:"mem_usage_percent"`
    GPUs                 []GPUMetric `json:"gpus,omitempty"`
    DiskReadBytesPerSec  *float64    `json:"disk_read_bytes_per_sec,omitempty"`
    DiskWriteBytesPerSec *float64    `json:"disk_write_bytes_per_sec,omitempty"`
    CollectedAt          time.Time   `json:"collected_at"`
}
```

### 3.2 presence 上报集成

`apps/edge-agent/internal/presence/reporter.go`：

- 新增字段 `MetricsInterval time.Duration`（默认 30s）与 `Sample func(ctx context.Context, since time.Time) *edge.Metrics`。
- `Reporter` 增加 `lastMetrics time.Time`；`ProbeAndReport` 中：`now := time.Now().UTC()`，若 `lastMetrics.IsZero() || now.Sub(lastMetrics) >= interval`，则采样并把快照放入上报载荷，更新 `lastMetrics`；否则载荷不带 `metrics`。
- `pull.Client.ReportPresence` 载荷新增 `metrics *edge.Metrics`（可选），JSON 序列化后经 `POST /agent/v1/presence` 发送；`RefreshHardware` 响应逻辑不变。

### 3.3 数据模型与持久化

`edge_metrics` 表（GORM `MetricsRow`，放 `internal/platform/edge/persistence/gorm_metrics.go`）：

| 列 | 类型 | 说明 |
|---|---|---|
| `id` | BIGINT 自增 | 主键 |
| `edge_id` | VARCHAR(128) | 索引 |
| `metrics_json` | TEXT | 3.1 的 `edge.Metrics` 完整快照 |
| `collected_at` | TIMESTAMP | 索引，查询窗口 |

仓库接口（`internal/platform/edge/metrics_repository.go`）：

```go
type MetricsRepository interface {
    Append(ctx context.Context, edgeID sharedkernel.EdgeID, m edge.Metrics) error
    ListSince(ctx context.Context, edgeID sharedkernel.EdgeID, since time.Time, limit int) ([]edge.Metrics, error)
}
```

- `Append`：写入新行后顺带 `DELETE FROM edge_metrics WHERE collected_at < now - retention`（幂等，无事务依赖）。
- `ListSince`：按 `collected_at ASC` 取 `limit`（默认 720）条窗口内记录；解析 `metrics_json` 失败返回错误。
- 保留窗口由 `MetricsRepository` 构造参数注入（默认 24h，控制面 `METRICS_RETENTION` 环境变量覆盖）。
- 装配：`internal/platform/appboot/boot.go` 的 AutoMigrate 模型列表追加 `&persistence.MetricsRow{}`。

### 3.4 控制面 API

**Agent presence（`internal/httpapi/agent/handler.go`）**

- 请求体新增 `Metrics *edge.Metrics \`json:"metrics,omitempty"\``；鉴权通过后，若 `h.Metrics != nil && body.Metrics != nil` 且 `body.Metrics.CollectedAt` 非零，则 `h.Metrics.Append(r.Context(), edgeID, *body.Metrics)`；写入失败返回 500。鉴权失败在任何写入之前返回 401（不落库）。

**Admin metrics（`internal/httpapi/edges/handler.go`）**

- 新增 `GET /{id}/metrics`：
  - `window` 参数：`1h`（默认）/ `6h` / `24h`；非法值返回 400。
  - Edge 不存在返回 404；存在但无数据返回 `{ "latest": null, "series": [] }`。
  - 响应：`{ "latest": edge.Metrics|null, "series": edge.Metrics[] }`（升序，最多 720 点，超出截尾保留最近点）。
- `Handler` 增加 `Metrics edge.MetricsRepository` 字段；`apps/admin-api/cmd/admin-api/main.go` 用同一个 `*gorm.DB` 构造并注入 agent 与 edges 两个 handler。

### 3.5 Admin 前端

**chart 组件**：新增 `web/admin/src/components/ui/chart.tsx`——shadcn/ui 现行 Tailwind v4 + `data-slot` 版（`ChartContainer`、`ChartTooltip`、`ChartTooltipContent`、`ChartLegend`、`ChartLegendContent`、`ChartStyle`），即参考 MHTML 渲染所依赖的同一组件；从 shadcn/ui 官方源复制（MIT），导入走项目现有 `@/lib/utils` 的 `cn`。`recharts@3.8.1` 已在依赖中。

**API 与类型**：

- `lib/api/types.ts`：`EdgeGPUMetric`、`EdgeMetrics`、`EdgeMetricsResponse { latest: EdgeMetrics|null; series: EdgeMetrics[] }`。
- `lib/api/edges.ts`：`getEdgeMetrics(id, window)` → `GET /api/v1/edges/{id}/metrics?window=...`。
- `lib/api/query-keys.ts`：`edges.metrics: (id) => ['edges', id, 'metrics']`。

**解析**：`features/edges/observation.ts` 新增 `parseMetrics(data)`，把 `latest`/`series` 规整为图表行：`{ time, cpu, memUsed, memTotal, memPct, gpus[], ioRead, ioWrite }`；可选字段缺省为 `null`，供图表判空。

**「系统监控」卡组（`observation-panel.tsx`）**：`SystemSection` 替换为图表卡组，卡片 class 逐条对照 MHTML dashboard-3：

| 卡 | 数据 | 参考卡 | 关键 class |
|---|---|---|---|
| CPU 占用率（折线图，与内存占用率并排） | series.cpu | Total Revenue（样式体系） | `bg-card flex min-w-0 flex-1 flex-col gap-4 rounded-xl border p-4 sm:gap-6 sm:p-6`；顶部大数字 `text-xl leading-tight font-semibold tracking-tight sm:text-2xl`；图高为基准 50%：`h-[100px] w-full min-w-0 sm:h-[120px] lg:h-[140px]`；`--color-cpu: var(--primary)`；Line strokeWidth 2 |
| 内存占用率（折线图，与 CPU 并排） | series.memPct + series.memUsed | Total Revenue（样式体系） | 同上；左轴 0–100%（占用率），右轴字节（内存占用，辅助序列）；`--color-memRate: var(--primary)`、`--color-memUsed: color-mix(... 75% ...)` |
| GPU 占用率（折线图，与显存占用率并排，每 GPU 一组） | series.gpus[].usage | Total Revenue（样式体系） | 同上；无 GPU 数据整组隐藏 |
| 显存占用率（折线图，与 GPU 占用率并排） | series.gpus[].vramPct + vramUsed | Total Revenue（样式体系） | 同上；左轴 0–100%，右轴字节（显存占用，辅助序列） |
| I/O（双系列面积图） | series.ioRead/ioWrite | Total Revenue 双系列 | `--color-read: var(--primary)`、`--color-write: color-mix(in oklch, var(--primary) 75%, var(--background))`；两条 Area 各自渐变；y 轴字节/秒 |

> 布局调整（2026-08-18 build 第二轮）：CPU 与内存占用率并排折线图、GPU 与显存占用率并排折线图（图高均为基准 50%），内存占用/显存占用字节作为对应占用率图的辅助序列（右轴），I/O 图高 50%；移除环形图卡。

> 数值可读性（2026-08-18 build 第三轮）：`formatBytes` 改为自适应二进制单位（B/KiB/MiB/GiB/TiB，保留 1 位小数、≥100 取整），所有字节序列（内存占用/显存占用/I/O 读/写）的 tooltip 与坐标轴 tick 均走该格式化；百分比序列 tooltip 补 `%`；图表左右 margin 收窄到 4px，百分比轴宽 36、字节轴宽 48。

> I/O 轴阶梯化（2026-08-18 build 第四轮）：`ioAxisTicks` 按显示单位（B/KiB/MiB/GiB，取 max/unit ≥ 4 的最大单位）以 1/2/5×10ⁿ 步进生成整齐 tick，Y 轴不再出现零碎小数；tooltip 仍显示精确值。

> tooltip 与内边距（2026-08-18 build 第五轮）：tooltip 每行恢复「label: 值」格式（`formatMetricValue` 处理字节/百分比）；五张图统一 `CHART_MARGIN`（top 12 / right 4 / bottom 0 / left 4）、X 轴 `height=20`、`tickMargin=4`，消除各图底部 padding 差异并给 Y 轴顶部刻度留白防裁切。

图例点抄参考：`size-2.5 rounded-full sm:size-3`（面积图）与 `size-2 rounded-full sm:size-2.5`（环形图），颜色用 `style={{ backgroundColor: 'var(--primary)' }}` 与 `color-mix` 表达式；均需暗色变体（`dark [data-chart=...]` 下的 85% 混色）。

**集成**：`detail-panel.tsx` 删除 `getEdgeSystem`/`systemQuery`，改 `getEdgeMetrics(id, '1h')` + `refetchInterval: 15000`；`ObservationPanel` 仅接收 `metricsQuery` 与 `tasksQuery`；`parseSystem` 相关代码随系统节移除（队列节未实现，保留其余逻辑）。

**i18n**：`edges.observationSystem` → `系统监控`（zh）/ `System Monitoring`（en）；`observationSystemHint` → `CPU、内存、GPU、I/O`；新增图表标签 key（CPU 占用率、已用/剩余、GPU 占用率、显存、读/写、暂无数据等）。

**空态**：`series` 为空时整组渲染空态，不渲染无数据图表。

### 3.6 配置项

| 环境变量 | 位置 | 默认 |
|---|---|---|
| `METRICS_INTERVAL` | Edge-Agent | `30s` |
| `METRICS_RETENTION` | 控制面 | `24h` |

解析失败回退默认值；`METRICS_INTERVAL < 5s` 时钳制到 5s（不与心跳节拍倒挂）。

## 4. 边界条件与错误处理

- `nvidia-smi` 不存在/非 NVIDIA/解析失败 → `GPUs` 为 nil，前端隐藏 GPU 卡。
- 首次采样：CPU 占用率 0、I/O 速率 nil、GPU 照常；第二次起 I/O 有速率。
- Edge 重启：`lastMetrics` 归零，重启后首拍即采样（I/O 因无历史仍为 nil）。
- `metrics_json` 损坏 → `ListSince` 返回错误（500），不吞数据。
- 大窗口：`limit=720` 截尾；写入侧保留窗口裁剪，避免表无限增长。
- 时钟回拨：`now.Sub(lastMetrics) < 0` 视为未到期，不采样。
- 前端无 GPU / 无历史 / 服务端 404 / 500：分别渲染隐藏、空态、错误横幅，不崩溃。

## 5. 测试策略

**Go 单测**

- `metrics`：mock 注入 CPU/内存/磁盘/nvidia-smi 输出，覆盖正常、`nvidia-smi` 缺失、首拍 I/O nil、CPU 首拍 0、`COMFY_MOCK` 合成 GPU、磁盘设备空。
- `presence.Reporter`：采样到期/未到期时载荷是否携带 `metrics`；首拍立即采样；interval 钳制。
- `pull.Client.ReportPresence`：序列化含/不含 `metrics`。
- `MetricsRepository`（gorm sqlite 内存库）：Append 落库、ListSince 排序与 limit、过期清理。
- agent presence handler：指标落库、鉴权失败不落库、坏载荷 400。
- admin metrics handler：window 解析、404、空序列、latest+series 结构。

**前端**

- `parseMetrics`：可选字段缺省、GPU 缺失、空 series。
- 合同测试：锁定「系统监控」文案与关键 chart class（`data-slot="chart"`、`aspect-video`、`h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]`、`size-2.5 rounded-full sm:size-3`、`rounded-xl border p-4 sm:p-5`、`--color-` 变量等）；断言 `detail-panel` 不再调用 `getEdgeSystem`。
- 组件测试：空态、无 GPU 降级。

**端到端**：`COMFY_MOCK=true` 起本地 Edge + 控制面，页面详情页「系统监控」出图且数据随 30s 采样推进。

## 6. 风险与缓解

| 风险 | 缓解 |
|---|---|
| 非 NVIDIA 主机 GPU 无数据 | 页面隐藏 GPU 卡；spec 明确「不可用留空」 |
| 磁盘 I/O 首拍/重启后缺速率 | 首拍 nil，后续差值计算 |
| `edge_metrics` 增长 | Append 时清理超窗记录；查询限 720 点 |
| 多实例控制面并发写 | 删除/插入幂等，无事务依赖 |
| chart 组件与 recharts 版本漂移 | 固定 shadcn data-slot 版 + 合同测试锁 class；升级 recharts 需回归 |
| nvidia-smi 解析被驱动版本差异破坏 | 只依赖稳定字段（utilization.gpu / memory.used / memory.total），解析失败整体留空 |

## 7. 迁移与回滚

- 上线：`MetricsRow` 随 AutoMigrate 创建，无存量迁移；先合并后端（presence 载荷兼容旧 Edge——无 `metrics` 字段时不落库），再合并前端切换。
- 回滚：前端回退到 `getEdgeSystem` 即可恢复旧「系统」节；控制面停止写/读 `edge_metrics` 不影响其它功能。

## 8. Implementation Divergence

- CPU 占用率采样（2026-08-18 build）：design §3.1 原写“`cpu.PercentWithContext(ctx, interval)`，interval 为距上次采样时长”。实测 gopsutil 在 `interval > 0` 时会阻塞 `Sleep(interval)`（首拍 since 为零值会传入约 64 年间隔，直接卡死 presence 上报）。实现改为 `cpu.PercentWithContext(ctx, 0, false)`（距上次调用、不阻塞），磁盘 I/O 速率仍用自维护计数器差值除以采样间隔。spec 行为不变（CPU 首拍可 0、I/O 首拍空）。回归测试：`TestSample_CPUPercentMustNotSleep`。
- Round 6 图表样式（2026-08-18 build，任务 3.20）：用户要求从 Total Revenue 卡样式改为 MHTML「Sprint health」卡样式——卡片 `p-4` + 标题 `h2.text-base.font-semibold` + 主体左图右值，右侧统计列 `grid-cols-3 … lg:grid-cols-1 lg:text-right`，三值字号统一 `text-lg leading-6 font-semibold`、说明 `text-muted-foreground text-[11px]`。图例色点+系列名常显（`inline-flex items-center gap-1.5` + `size-2 rounded-full`，单系列也显示，如 CPU 占用率、单 GPU 名称）。占用率系列（CPU/内存/GPU/显存）右侧显示「当前 / 最高 / 平均」百分比（多 GPU 时前缀 `GPU{n} ·`）；内存/显存已用字节仍作为占用率图辅助线（tooltip 走 `formatMetricValue` 字节分支，键以 `Used` 结尾同样识别为字节，右侧字节 Y 轴 `width={70}` 防标签换行）；I/O 右侧为「当前读 / 当前写 / 最高读 / 最高写 / 平均读 / 平均写」六值（`formatBytes` 人类可读单位），compact 模式两列三行 `text-xs`，右列加宽为 122px，避免卡片过高；占用率卡右列 72px（92−20）。主体左右栏 gap 统一收窄为 `gap-1.5`。GPU 不再每卡一张，而是「GPU 占用率」「显存占用率」各一张卡、多 GPU 同图多条折线（配色 `GPU_SERIES_COLORS`：primary + chart-2..5，图例显示各 GPU 名称色点）。chart 撑满卡片剩余高度（`h-full min-h-[120px] w-full min-w-0`），且 ChartContainer 必须覆盖 `aspect-auto`（否则基类 `aspect-video` 会让通栏宽图按宽度推高、I/O 卡过高），取代 Round 3 的 50% 固定图高约定；内边距与 tooltip 格式维持 Round 4/5 约定。新增纯函数 `seriesStats(values, format)`（过滤 null，返回 current/max/avg 格式化值）并单测。
- Round 7 节点信息（2026-08-19 build，任务 1.8/2.6/3.21）：agent 进程内首次心跳时间记为 `started_at`（`Reporter.startedAt` 只记一次，随每次 presence 上报），并从 Comfy `/system_stats` 的 `system.comfyui_version` 解析版本（`SystemStats.ComfyUIVersion`，Mock 为 "mock"）随 presence 上报 `comfy_version`。控制面 `edges` 表新增 `started_at`（可空时间）与 `comfy_version` 列；`Repository.UpdatePresenceInfo` 落库（nil 时间/空版本不清空旧值）；agent presence handler 落库；admin `instanceDTO` 新增 `started_at`/`comfy_version`。前端节点信息左侧栏顺序：ID → 创建时间 → 启动时间（`fieldStartedAt`，Timer 图标）→ Comfy 版本（`fieldComfyVersion`，Boxes 图标）→ 分类；启动时间仅当 `edge.enabled && presence?.edge_online` 且有值时才显示，否则 —（离线/未启用不显示旧值）。
