---
change: edge-system-monitoring
design-doc: docs/superpowers/specs/2026-08-18-edge-system-monitoring-design.md
base-ref: e3e0870cd9741dd665a3ce5dbe24bf1b59ea290a
---

# Edge 系统监控 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让节点详情页「系统」节改为「系统监控」，由 Edge 在 presence 心跳中上报 CPU/内存/GPU/I/O 实时指标，控制面落库，前端用与 Shadcnblocks Admin Kit 同款 chart 组件展示图表。

**Architecture:** Edge-Agent 新增 `internal/metrics` 采集器（gopsutil + nvidia-smi），presence 心跳按 `METRICS_INTERVAL`（默认 30s）采样并随 `metrics` 字段上报；控制面写入 `edge_metrics` 表（整快照 JSON 一列），`Append` 时清理超 `METRICS_RETENTION`（默认 24h）旧行；Admin 新增 `GET /api/v1/edges/{id}/metrics`，前端新增 shadcn `chart.tsx` 并以参考 MHTML 卡片 class 渲染「系统监控」图表卡组。

**Tech Stack:** Go 1.24 + gorm（sqlite/mysql/postgres）、`github.com/shirou/gopsutil/v4`、`nvidia-smi` CLI、chi、TanStack Query/Router、recharts 3.8、shadcn/ui chart（Tailwind v4 data-slot 版）、Vitest 合同测试。

## Global Constraints

- 界面文案：节点详情页「系统」→「系统监控」；i18n zh=系统监控 / en=System Monitoring；hint=「CPU、内存、GPU、I/O」
- GPU 占用率**只做 NVIDIA**（`nvidia-smi`）；AMD/Intel 字段留空；不引入 NVML/ROCm
- `COMFY_MOCK=true` 时合成假 GPU（Mock GPU），CPU/内存/I/O 仍真实采集
- presence 心跳仍 5s 一拍；`METRICS_INTERVAL` 默认 30s、小于 5s 时钳制到 5s；`METRICS_RETENTION` 默认 24h，解析失败回退默认
- `edge_metrics` 表由 AutoMigrate 创建；`metrics_json` 存整快照（`edge.Metrics`），不拆列
- Admin `GET /api/v1/edges/{id}/metrics`：`window=1h`（默认）/`6h`/`24h`，非法值 400；未知 Edge 404；无数据 `{"latest": null, "series": []}`；最多 720 点
- 图表 class 逐条对照 MHTML dashboard-3（`data-slot="chart"`、`aspect-video`、`h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]`、图例点 `size-2.5 rounded-full sm:size-3`、`var(--primary)`/`color-mix` 配色），合同测试锁定
- 旧 `GET /{id}/system` 端点保留但前端不再使用；详情页系统节轮询 `getEdgeMetrics` + `refetchInterval: 15000`
- 未获用户要求不得 `git commit` 之外的动作；每个任务提交一次（若用户允许提交）

## File Structure

| 文件 | 职责 |
|---|---|
| `internal/platform/edge/metrics.go` | `Metrics` / `GPUMetric` 领域类型（新增） |
| `apps/edge-agent/internal/metrics/collect.go` | 采集器 `Sampler`：CPU/内存/磁盘 I/O/nvidia-smi/mock（新增） |
| `apps/edge-agent/internal/metrics/collect_test.go` | 采集器单测（新增） |
| `apps/edge-agent/internal/presence/reporter.go` | 采样节拍 + 载荷携带 `metrics`（修改） |
| `apps/edge-agent/internal/presence/reporter_test.go` | 采样到期/未到期测试（修改） |
| `apps/edge-agent/internal/pull/client.go` | `ReportPresence` 载荷加 `metrics`（修改） |
| `apps/edge-agent/internal/pull/client_test.go` | 载荷测试（修改） |
| `apps/edge-agent/cmd/edge-agent/main.go` | 装配 Sampler + `METRICS_INTERVAL`（修改） |
| `internal/platform/edge/metrics_repository.go` | `MetricsRepository` 接口（新增） |
| `internal/platform/edge/persistence/gorm_metrics.go` | `MetricsRow` + GORM 实现（新增） |
| `internal/platform/edge/persistence/gorm_metrics_test.go` | 持久化单测（新增） |
| `internal/httpapi/agent/handler.go` | presence 接收 `metrics` 落库（修改） |
| `internal/httpapi/agent/handler_test.go` | 落库/鉴权测试（修改） |
| `internal/httpapi/edges/handler.go` | `GET /{id}/metrics` 端点（修改） |
| `internal/httpapi/edges/handler_test.go` | metrics 端点测试（修改） |
| `apps/admin-api/cmd/admin-api/main.go` | 装配 MetricsRepository、`METRICS_RETENTION`（修改） |
| `internal/platform/appboot/boot.go` | AutoMigrate 追加 `MetricsRow`（修改） |
| `web/admin/src/components/ui/chart.tsx` | shadcn chart 组件（新增，从官方 registry 复制） |
| `web/admin/src/lib/api/types.ts` | `EdgeMetrics` 等类型（修改） |
| `web/admin/src/lib/api/edges.ts` | `getEdgeMetrics`（修改） |
| `web/admin/src/lib/api/query-keys.ts` | `edges.metrics`（修改） |
| `web/admin/src/features/edges/observation.ts` | `parseMetrics`（修改） |
| `web/admin/src/features/edges/observation.test.ts` | parse 测试（修改） |
| `web/admin/src/features/edges/observation-panel.tsx` | 「系统监控」图表卡组（修改） |
| `web/admin/src/features/edges/detail-panel.tsx` | 切到 `getEdgeMetrics`（修改） |
| `web/admin/src/lib/i18n/locales/zh.json` / `en.json` | 文案（修改） |
| `web/admin/src/features/edges/edges.contract.test.ts` | 锁定 chart class 与文案（修改） |
| `docs/architecture/data-model.md`、`runtime.md` | `edge_metrics` 与上报路径（修改） |
| `README.md` | 环境变量（修改） |

---

## 1. Edge 指标领域类型

### Task 1: `edge.Metrics` 领域类型

**Files:**
- Create: `internal/platform/edge/metrics.go`

**Interfaces:**
- Consumes: 无（纯类型定义）
- Produces: `edge.Metrics`、`edge.GPUMetric`、`edge.MetricsEmpty(m edge.Metrics) bool`（后续采集器、持久化、HTTP 载荷共用）

- [x] **Step 1: 创建类型文件**

```go
package edge

import "time"

// GPUMetric is one GPU's live utilization snapshot.
type GPUMetric struct {
	Name             string   `json:"name"`
	UsagePercent     *float64 `json:"usage_percent,omitempty"`
	VRAMUsedBytes    uint64   `json:"vram_used_bytes,omitempty"`
	VRAMTotalBytes   uint64   `json:"vram_total_bytes,omitempty"`
	VRAMUsagePercent *float64 `json:"vram_usage_percent,omitempty"`
}

// Metrics is one live system snapshot reported by an Edge heartbeat.
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

// MetricsEmpty reports whether a snapshot has no collected timestamp.
func MetricsEmpty(m Metrics) bool {
	return m.CollectedAt.IsZero()
}
```

- [x] **Step 2: 编译验证**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./internal/platform/edge/`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add internal/platform/edge/metrics.go
git commit -m "feat(edge): add live Metrics domain types"
```

## 2. Edge 指标采集

### Task 2: 采集器 Sampler（CPU/内存/磁盘 I/O/nvidia-smi/mock）

**Files:**
- Create: `apps/edge-agent/internal/metrics/collect.go`
- Test: `apps/edge-agent/internal/metrics/collect_test.go`

**Interfaces:**
- Consumes: `edge.Metrics` / `edge.GPUMetric`（Task 1）
- Produces: `metrics.NewSampler(mock bool) *Sampler`、`(*Sampler).Sample(ctx context.Context, since time.Time) edge.Metrics`（`since` 为零值表示首拍）；Sampler 字段全部可注入以便测试：

```go
type Sampler struct {
	Now           func() time.Time
	CPUPercent    func(ctx context.Context, interval time.Duration) (float64, error)
	VirtualMemory func(ctx context.Context) (*mem.VirtualMemoryStat, error)
	DiskIO        func(ctx context.Context) (map[string]disk.IOCountersStat, error)
	NvidiaSMI     func(ctx context.Context) ([]edge.GPUMetric, error)
	Mock          bool
	lastDisk      map[string]disk.IOCountersStat
}
```

- [x] **Step 1: 先添加依赖**

Run: `cd /Users/mr9esx/Documents/Pixoma && go get github.com/shirou/gopsutil/v4@v4.26.7`
Expected: PASS（go.mod 增加 gopsutil v4）

- [x] **Step 2: 写失败测试**（`collect_test.go`，注入 fake 采集函数，覆盖：正常全量、`NvidiaSMI` 失败、首拍 I/O nil、`Mock` 合成 GPU、CPU 首拍 0）

```go
package metrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/mr9esx/comfyui_tgbot/apps/edge-agent/internal/metrics"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
)

func floatPtr(v float64) *float64 { return &v }

func TestSample_FullSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	prev := now.Add(-30 * time.Second)
	s := metrics.NewSampler(false)
	s.Now = func() time.Time { return now }
	s.CPUPercent = func(_ context.Context, _ time.Duration) (float64, error) { return 42.5, nil }
	s.VirtualMemory = func(_ context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Used: 8 << 30, Total: 16 << 30, UsedPercent: 50}, nil
	}
	s.DiskIO = func(_ context.Context) (map[string]disk.IOCountersStat, error) {
		return map[string]disk.IOCountersStat{
			"sda": {ReadBytes: 10 << 20, WriteBytes: 4 << 20},
		}, nil
	}
	s.NvidiaSMI = func(_ context.Context) ([]edge.GPUMetric, error) {
		return []edge.GPUMetric{{Name: "RTX 4090", UsagePercent: floatPtr(60), VRAMUsedBytes: 12 << 30, VRAMTotalBytes: 24 << 30, VRAMUsagePercent: floatPtr(50)}}, nil
	}

	// 首拍（since 零值）：I/O 应为 nil
	first := s.Sample(context.Background(), time.Time{})
	if first.CPUUsagePercent != 42.5 || first.MemUsedBytes != 8<<30 || first.DiskReadBytesPerSec != nil {
		t.Fatalf("first sample: %+v", first)
	}
	if len(first.GPUs) != 1 || first.GPUs[0].Name != "RTX 4090" {
		t.Fatalf("gpus: %+v", first.GPUs)
	}

	// 第二拍：I/O 速率 = 差值 / 30s = 10MiB/30 = 349525.33...
	second := s.Sample(context.Background(), prev)
	if second.DiskReadBytesPerSec == nil || *second.DiskReadBytesPerSec < 349525 || *second.DiskReadBytesPerSec > 349526 {
		t.Fatalf("second read rate: %+v", second.DiskReadBytesPerSec)
	}
}

func TestSample_NvidiaUnavailableKeepsOtherFields(t *testing.T) {
	s := metrics.NewSampler(false)
	s.Now = func() time.Time { return time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC) }
	s.CPUPercent = func(_ context.Context, _ time.Duration) (float64, error) { return 10, nil }
	s.VirtualMemory = func(_ context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Used: 4 << 30, Total: 8 << 30, UsedPercent: 50}, nil
	}
	s.DiskIO = func(_ context.Context) (map[string]disk.IOCountersStat, error) {
		return map[string]disk.IOCountersStat{}, nil
	}
	s.NvidiaSMI = func(_ context.Context) ([]edge.GPUMetric, error) { return nil, errors.New("no nvidia-smi") }

	m := s.Sample(context.Background(), time.Time{})
	if m.CPUUsagePercent != 10 || len(m.GPUs) != 0 {
		t.Fatalf("partial sample: %+v", m)
	}
}

func TestSample_MockGPU(t *testing.T) {
	s := metrics.NewSampler(true)
	s.Now = func() time.Time { return time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC) }
	s.CPUPercent = func(_ context.Context, _ time.Duration) (float64, error) { return 30, nil }
	s.VirtualMemory = func(_ context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Used: 4 << 30, Total: 8 << 30, UsedPercent: 50}, nil
	}
	s.DiskIO = func(_ context.Context) (map[string]disk.IOCountersStat, error) {
		return map[string]disk.IOCountersStat{}, nil
	}
	m := s.Sample(context.Background(), time.Time{})
	if len(m.GPUs) != 1 || m.GPUs[0].Name != "Mock GPU" || m.GPUs[0].VRAMTotalBytes != 8<<30 {
		t.Fatalf("mock gpu: %+v", m.GPUs)
	}
}
```

- [x] **Step 3: 运行测试确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./apps/edge-agent/internal/metrics/`
Expected: FAIL（`undefined: metrics.NewSampler`）

- [x] **Step 4: 实现采集器**

```go
package metrics

import (
	"context"
	"encoding/csv"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
)

// Sampler collects one live system snapshot. Every field is injectable for tests.
type Sampler struct {
	Now           func() time.Time
	CPUPercent    func(ctx context.Context, interval time.Duration) (float64, error)
	VirtualMemory func(ctx context.Context) (*mem.VirtualMemoryStat, error)
	DiskIO        func(ctx context.Context) (map[string]disk.IOCountersStat, error)
	NvidiaSMI     func(ctx context.Context) ([]edge.GPUMetric, error)
	Mock          bool

	mu       sync.Mutex
	lastDisk map[string]disk.IOCountersStat
}

// NewSampler wires real gopsutil and nvidia-smi collectors.
func NewSampler(mock bool) *Sampler {
	s := &Sampler{Mock: mock, lastDisk: map[string]disk.IOCountersStat{}}
	s.Now = time.Now
	s.CPUPercent = func(ctx context.Context, interval time.Duration) (float64, error) {
		pcts, err := cpu.PercentWithContext(ctx, interval, false)
		if err != nil || len(pcts) == 0 {
			return 0, err
		}
		return pcts[0], nil
	}
	s.VirtualMemory = func(ctx context.Context) (*mem.VirtualMemoryStat, error) {
		return mem.VirtualMemoryWithContext(ctx)
	}
	s.DiskIO = func(ctx context.Context) (map[string]disk.IOCountersStat, error) {
		return disk.IOCountersWithContext(ctx)
	}
	s.NvidiaSMI = nvidiaSMI
	return s
}

func (s *Sampler) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

// Sample returns one snapshot. since is the previous sample time; zero means first sample.
// First sample has nil disk rates; CPU percent uses the elapsed interval.
func (s *Sampler) Sample(ctx context.Context, since time.Time) edge.Metrics {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	interval := now.Sub(since)
	if interval <= 0 {
		interval = time.Second
	}
	m := edge.Metrics{CollectedAt: now}
	if s == nil {
		return m
	}

	if s.CPUPercent != nil {
		if pct, err := s.CPUPercent(ctx, interval); err == nil {
			m.CPUUsagePercent = pct
		}
	}
	if s.VirtualMemory != nil {
		if vm, err := s.VirtualMemory(ctx); err == nil {
			m.MemUsedBytes = vm.Used
			m.MemTotalBytes = vm.Total
			m.MemUsagePercent = vm.UsedPercent
		}
	}
	if !since.IsZero() && s.DiskIO != nil {
		if counters, err := s.DiskIO(ctx); err == nil {
			var read, write uint64
			for name, cur := range counters {
				if prev, ok := s.lastDisk[name]; ok {
					if cur.ReadBytes >= prev.ReadBytes {
						read += cur.ReadBytes - prev.ReadBytes
					}
					if cur.WriteBytes >= prev.WriteBytes {
						write += cur.WriteBytes - prev.WriteBytes
					}
				}
			}
			secs := interval.Seconds()
			if secs > 0 {
				r := float64(read) / secs
				w := float64(write) / secs
				m.DiskReadBytesPerSec = &r
				m.DiskWriteBytesPerSec = &w
			}
			s.lastDisk = counters
		}
	}
	if s.Mock {
		m.GPUs = mockGPUs(now)
	} else if s.NvidiaSMI != nil {
		if gpus, err := s.NvidiaSMI(ctx); err == nil {
			m.GPUs = gpus
		}
	}
	return m
}

func nvidiaSMI(ctx context.Context) ([]edge.GPUMetric, error) {
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=index,name,utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, err
	}
	rows, err := csv.NewReader(strings.NewReader(string(out))).ReadAll()
	if err != nil {
		return nil, err
	}
	var gpus []edge.GPUMetric
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		usage, errUsage := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		usedMiB, errUsed := strconv.ParseUint(strings.TrimSpace(row[3]), 10, 64)
		totalMiB, errTotal := strconv.ParseUint(strings.TrimSpace(row[4]), 10, 64)
		if errUsage != nil || errUsed != nil || errTotal != nil || totalMiB == 0 {
			continue
		}
		usedBytes := usedMiB << 20
		totalBytes := totalMiB << 20
		vramPct := float64(usedMiB) / float64(totalMiB) * 100
		gpus = append(gpus, edge.GPUMetric{
			Name:             strings.TrimSpace(row[1]),
			UsagePercent:     &usage,
			VRAMUsedBytes:    usedBytes,
			VRAMTotalBytes:   totalBytes,
			VRAMUsagePercent: &vramPct,
		})
	}
	return gpus, nil
}

func mockGPUs(now time.Time) []edge.GPUMetric {
	phase := float64(now.Unix()%60) / 60
	usage := 20 + 40*phase
	total := uint64(8 << 30)
	used := uint64(float64(total) * (0.2 + 0.6*phase))
	vramPct := float64(used) / float64(total) * 100
	return []edge.GPUMetric{{
		Name:             "Mock GPU",
		UsagePercent:     &usage,
		VRAMUsedBytes:    used,
		VRAMTotalBytes:   total,
		VRAMUsagePercent: &vramPct,
	}}
}
```

- [x] **Step 5: 运行测试确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./apps/edge-agent/internal/metrics/`
Expected: PASS

- [x] **Step 6: 提交**

```bash
git add go.mod go.sum apps/edge-agent/internal/metrics/
git commit -m "feat(edge-agent): collect live CPU/mem/disk/GPU metrics"
```

### Task 3: presence 心跳按节拍采样并上报 metrics

**Files:**
- Modify: `apps/edge-agent/internal/presence/reporter.go`
- Modify: `apps/edge-agent/internal/presence/reporter_test.go`
- Modify: `apps/edge-agent/internal/pull/client.go`
- Modify: `apps/edge-agent/internal/pull/client_test.go`
- Modify: `apps/edge-agent/cmd/edge-agent/main.go`

**Interfaces:**
- Consumes: `metrics.NewSampler` / `(*Sampler).Sample`（Task 2）
- Produces: `presence.Reporter` 新增字段 `Sample func(ctx context.Context, since time.Time) *edge.Metrics` 与 `MetricsInterval time.Duration`；`pull.Client.ReportPresence(ctx, running bool, hw *edge.Hardware, m *edge.Metrics) (bool, error)`

- [x] **Step 1: 先写失败测试**（reporter：首拍携带 metrics、未到期不携带、到期携带）

```go
// reporter_test.go 追加
func TestProbeAndReport_MetricsCadence(t *testing.T) {
	now := time.Now().UTC()
	client := &pull.Client{BaseURL: "http://cp", Token: "tok", EdgeID: "e1", HTTP: &http.Client{}}
	reported := 0
	client.HTTP = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		reported++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})
	collect := func(_ context.Context, since time.Time) *edge.Metrics {
		return &edge.Metrics{CPUUsagePercent: 10, CollectedAt: time.Now().UTC()}
	}
	r := &presence.Reporter{
		Client:          client,
		Sample:          collect,
		MetricsInterval: 30 * time.Second,
	}
	// 首拍：立即采样并携带
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	// 未到期：不携带（载荷无 metrics 键）
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
}
```

（测试辅助 `roundTripFunc` 与载荷断言按 `pull/client_test.go` 现有风格实现；重点是断言首拍请求体含 `"metrics"` 键、未到期请求体不含。）

- [x] **Step 2: 运行测试确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./apps/edge-agent/internal/presence/`
Expected: FAIL

- [x] **Step 3: 修改 `reporter.go`**

```go
const defaultMetricsInterval = 30 * time.Second
const minMetricsInterval = 5 * time.Second

type Reporter struct {
	Client          *pull.Client
	Comfy           comfyui.Client
	Collect         func(ctx context.Context) edge.Hardware
	Sample          func(ctx context.Context, since time.Time) *edge.Metrics
	Every           time.Duration
	MetricsInterval time.Duration
	SendHardware    bool
	refreshNext     bool
	lastMetrics     time.Time
}

func (r *Reporter) metricsInterval() time.Duration {
	if r != nil && r.MetricsInterval >= minMetricsInterval {
		return r.MetricsInterval
	}
	return defaultMetricsInterval
}

func (r *Reporter) ProbeAndReport(ctx context.Context) error {
	// ...现有 running/hardware 逻辑不变...
	now := time.Now().UTC()
	var m *edge.Metrics
	if r.Sample != nil && (r.lastMetrics.IsZero() || now.Sub(r.lastMetrics) >= r.metricsInterval()) {
		collected := r.Sample(ctx, r.lastMetrics)
		r.lastMetrics = now
		m = &collected
	}
	refresh, err := r.Client.ReportPresence(ctx, running, hw, m)
	// ...rest 不变...
}
```

- [x] **Step 4: 修改 `pull/client.go` 的 `ReportPresence`**

```go
func (c *Client) ReportPresence(ctx context.Context, comfyRunning bool, hw *edge.Hardware, m *edge.Metrics) (bool, error) {
	payload := map[string]any{
		"edge_id":       c.EdgeID,
		"comfy_running": comfyRunning,
	}
	if hw != nil {
		payload["hardware"] = hw
	}
	if m != nil {
		payload["metrics"] = m
	}
	// ...其余不变...
}
```

- [x] **Step 5: 修改 `main.go` 装配**

```go
sampler := edgehw.NewSampler(comfyMock)
reporter := &presence.Reporter{
	Client:          client,
	Comfy:           comfy,
	Collect: func(ctx context.Context) edge.Hardware {
		return edgehw.Collect(ctx, edgehw.InspectGHW, comfy)
	},
	Sample:          sampler.Sample,
	MetricsInterval: envDuration("METRICS_INTERVAL", 30*time.Second),
	SendHardware:    true,
}
```

（`envDuration` 已存在于 main.go；`METRICS_INTERVAL < 5s` 由 `metricsInterval()` 钳制。）

- [x] **Step 6: 更新 pull/client_test.go 的 `ReportPresence` 调用点并补载荷断言，运行全部相关测试**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./apps/edge-agent/...`
Expected: PASS

- [x] **Step 7: 提交**

```bash
git add apps/edge-agent/
git commit -m "feat(edge-agent): report live metrics in presence heartbeat"
```

## 3. 控制面持久化与 API

### Task 4: `edge_metrics` 表与 MetricsRepository

**Files:**
- Create: `internal/platform/edge/metrics_repository.go`
- Create: `internal/platform/edge/persistence/gorm_metrics.go`
- Test: `internal/platform/edge/persistence/gorm_metrics_test.go`
- Modify: `internal/platform/appboot/boot.go`

**Interfaces:**
- Consumes: `edge.Metrics`（Task 1）
- Produces:

```go
type MetricsRepository interface {
	Append(ctx context.Context, edgeID sharedkernel.EdgeID, m edge.Metrics) error
	ListSince(ctx context.Context, edgeID sharedkernel.EdgeID, since time.Time, limit int) ([]edge.Metrics, error)
}
```

- [x] **Step 1: 写失败测试**（`gorm_metrics_test.go`：Append 后 ListSince 升序、limit 截尾、过期清理）

```go
package persistence_test

func TestMetricsRepository_AppendAndList(t *testing.T) {
	dsn := "file:metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewMetricsRepository(gdb, 24*time.Hour)
	ctx := context.Background()
	base := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		m := edge.Metrics{CPUUsagePercent: float64(i), CollectedAt: base.Add(time.Duration(i) * time.Minute)}
		if err := repo.Append(ctx, "e1", m); err != nil {
			t.Fatal(err)
		}
	}
	series, err := repo.ListSince(ctx, "e1", base.Add(-time.Hour), 720)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 3 || series[0].CPUUsagePercent != 0 || series[2].CPUUsagePercent != 2 {
		t.Fatalf("series: %+v", series)
	}
	limited, err := repo.ListSince(ctx, "e1", base.Add(-time.Hour), 2)
	if err != nil || len(limited) != 2 {
		t.Fatalf("limit: %v %v", limited, err)
	}
}

func TestMetricsRepository_PrunesOldRows(t *testing.T) {
	// retention=1h；写入 now-2h 与 now-30m 两条，Append 后再查应只剩新的一条
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/platform/edge/persistence/`
Expected: FAIL（undefined MetricsRow / NewMetricsRepository）

- [x] **Step 3: 实现接口与 GORM 实现**

```go
// internal/platform/edge/metrics_repository.go
package edge

import (
	"context"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type MetricsRepository interface {
	Append(ctx context.Context, edgeID sharedkernel.EdgeID, m Metrics) error
	ListSince(ctx context.Context, edgeID sharedkernel.EdgeID, since time.Time, limit int) ([]Metrics, error)
}
```

```go
// internal/platform/edge/persistence/gorm_metrics.go
package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type MetricsRow struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	EdgeID      string    `gorm:"size:128;index"`
	MetricsJSON string    `gorm:"type:text"`
	CollectedAt time.Time `gorm:"index"`
}

func (MetricsRow) TableName() string { return "edge_metrics" }

type MetricsRepository struct {
	db        *gorm.DB
	retention time.Duration
}

func NewMetricsRepository(db *gorm.DB, retention time.Duration) *MetricsRepository {
	return &MetricsRepository{db: db, retention: retention}
}

func (r *MetricsRepository) Append(ctx context.Context, edgeID sharedkernel.EdgeID, m edge.Metrics) error {
	if edgeID == "" || edge.MetricsEmpty(m) {
		return fmt.Errorf("edge: invalid metrics append")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("edge: metrics_json: %w", err)
	}
	row := MetricsRow{EdgeID: string(edgeID), MetricsJSON: string(raw), CollectedAt: m.CollectedAt}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-r.retention)
	return r.db.WithContext(ctx).Where("collected_at < ?", cutoff).Delete(&MetricsRow{}).Error
}

func (r *MetricsRepository) ListSince(ctx context.Context, edgeID sharedkernel.EdgeID, since time.Time, limit int) ([]edge.Metrics, error) {
	if limit <= 0 {
		limit = 720
	}
	var rows []MetricsRow
	if err := r.db.WithContext(ctx).
		Where("edge_id = ? AND collected_at >= ?", string(edgeID), since).
		Order("collected_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]edge.Metrics, 0, len(rows))
	for _, row := range rows {
		var m edge.Metrics
		if err := json.Unmarshal([]byte(row.MetricsJSON), &m); err != nil {
			return nil, fmt.Errorf("edge: metrics_json: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}
```

- [x] **Step 4: appboot AutoMigrate 追加 `MetricsRow`**

```go
// internal/platform/appboot/boot.go，MigrateEdges 分支
models = append(models, &instpersist.EdgeRow{}, &instpersist.MetricsRow{})
```

- [x] **Step 5: 运行测试确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/platform/edge/... ./internal/platform/appboot/`
Expected: PASS

- [x] **Step 6: 提交**

```bash
git add internal/platform/edge/metrics_repository.go internal/platform/edge/persistence/gorm_metrics.go internal/platform/appboot/boot.go
git commit -m "feat(edge): persist live metrics in edge_metrics table"
```

### Task 5: agent presence 落库 + admin metrics 端点

**Files:**
- Modify: `internal/httpapi/agent/handler.go`
- Modify: `internal/httpapi/agent/handler_test.go`
- Modify: `internal/httpapi/edges/handler.go`
- Modify: `internal/httpapi/edges/handler_test.go`

**Interfaces:**
- Consumes: `edge.MetricsRepository`（Task 4）
- Produces: `agent.Handler.Metrics edge.MetricsRepository`；`edges.Handler.Metrics edge.MetricsRepository`；`GET /api/v1/edges/{id}/metrics`

- [x] **Step 1: 先写失败测试**

`agent/handler_test.go` 追加：presence 携带 `metrics` → 落库可查；无 token → 不落库。

`edges/handler_test.go` 追加：

```go
func TestHandler_MetricsEndpoint(t *testing.T) {
	// 建 gorm 内存库 + EdgeRow + MetricsRow，注入 MetricsRepository
	// 先 Append 两条指标，再 GET /api/v1/edges/{id}/metrics?window=1h
	// 断言 200、series 升序 2 条、latest 为最后一条
	// 再 GET 不存在的 id → 404
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/httpapi/agent/ ./internal/httpapi/edges/`
Expected: FAIL

- [x] **Step 3: agent presence 接收并落库**

```go
// Handler 增加字段
Metrics edge.MetricsRepository

// presence handler body 增加字段
Metrics *edge.Metrics `json:"metrics"`

// 鉴权通过、Presence.Report 之后、hardware 逻辑之前/之后均可：
if h.Metrics != nil && body.Metrics != nil && !body.Metrics.CollectedAt.IsZero() {
	if err := h.Metrics.Append(r.Context(), edgeID, *body.Metrics); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
}
```

- [x] **Step 4: admin metrics 端点**

```go
// Handler 增加字段
Metrics edge.MetricsRepository

// Mount 追加
r.Get("/{id}/metrics", h.metrics)

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.EdgeID(chi.URLParam(r, "id"))
	if h.Metrics == nil {
		writeErr(w, http.StatusInternalServerError, "metrics not configured")
		return
	}
	if _, err := h.Repo.Get(r.Context(), id); errors.Is(err, edge.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "instance not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	window, err := parseMetricsWindow(r.URL.Query().Get("window"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	since := time.Now().UTC().Add(-window)
	series, err := h.Metrics.ListSince(r.Context(), id, since, 720)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var latest *edge.Metrics
	if len(series) > 0 {
		cp := series[len(series)-1]
		latest = &cp
	}
	writeJSON(w, http.StatusOK, map[string]any{"latest": latest, "series": series})
}

func parseMetricsWindow(raw string) (time.Duration, error) {
	switch strings.TrimSpace(raw) {
	case "", "1h":
		return time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "24h":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid window %q", raw)
	}
}
```

- [x] **Step 5: 运行测试确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma && go test ./internal/httpapi/...`
Expected: PASS

- [x] **Step 6: 提交**

```bash
git add internal/httpapi/agent/ internal/httpapi/edges/
git commit -m "feat(api): persist agent metrics and serve admin metrics endpoint"
```

### Task 6: admin-api 装配

**Files:**
- Modify: `apps/admin-api/cmd/admin-api/main.go`

**Interfaces:**
- Consumes: `persistence.NewMetricsRepository`、`db.AutoMigrate`（Task 4/5）

- [x] **Step 1: 装配 MetricsRepository**

```go
retention := 24 * time.Hour
if v := strings.TrimSpace(os.Getenv("METRICS_RETENTION")); v != "" {
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		retention = d
	}
}
metricsRepo := instpersist.NewMetricsRepository(gdb, retention)
// 注入 agent.Handler.Metrics 与 edges.Handler.Metrics
```

（`gdb` 即现有 AutoMigrate 后的 `*gorm.DB`；确保 `MetricsRow` 已 AutoMigrate——Task 4 已在 appboot 追加。）

- [x] **Step 2: 编译与启动冒烟**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./...`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add apps/admin-api/cmd/admin-api/main.go
git commit -m "feat(admin-api): wire metrics repository and retention config"
```

## 4. Admin 前端：chart 组件与数据层

### Task 7: 新增 shadcn chart 组件

**Files:**
- Create: `web/admin/src/components/ui/chart.tsx`

**Interfaces:**
- Consumes: `cn` from `@/lib/utils`
- Produces: `ChartContainer`、`ChartTooltip`、`ChartTooltipContent`、`ChartLegend`、`ChartLegendContent`、`ChartStyle`

- [x] **Step 1: 从官方 registry 获取组件**

```bash
cd /Users/mr9esx/Documents/Pixoma/web/admin
curl -s https://ui.shadcn.com/r/styles/new-york-v4/chart.json -o /tmp/chart.json
node -e 'const j=require("/tmp/chart.json"); process.stdout.write(j.files[0].content)' > src/components/ui/chart.tsx
```

Expected: `src/components/ui/chart.tsx` 非空，内容为 MIT 协议的 shadcn chart 组件（`"use client"`、`data-slot="chart"`、`ChartContainer` 等导出）。

- [x] **Step 2: 编译验证**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit`
Expected: PASS（如报 `TooltipValueType` 类型缺失，确认 recharts 版本为 ^3.8.1）

- [x] **Step 3: 提交**

```bash
git add web/admin/src/components/ui/chart.tsx
git commit -m "feat(admin): add shadcn chart component"
```

### Task 8: API 类型、客户端与 query key

**Files:**
- Modify: `web/admin/src/lib/api/types.ts`
- Modify: `web/admin/src/lib/api/edges.ts`
- Modify: `web/admin/src/lib/api/query-keys.ts`

**Interfaces:**
- Produces: `EdgeGPUMetric`、`EdgeMetrics`、`EdgeMetricsResponse`；`getEdgeMetrics(id: string, window?: '1h'|'6h'|'24h'): Promise<EdgeMetricsResponse>`；`queryKeys.edges.metrics(id)`

- [x] **Step 1: types.ts 追加**

```ts
export type EdgeGPUMetric = {
  name: string
  usage_percent?: number | null
  vram_used_bytes?: number
  vram_total_bytes?: number
  vram_usage_percent?: number | null
}

export type EdgeMetrics = {
  cpu_usage_percent: number
  mem_used_bytes: number
  mem_total_bytes: number
  mem_usage_percent: number
  gpus?: EdgeGPUMetric[]
  disk_read_bytes_per_sec?: number | null
  disk_write_bytes_per_sec?: number | null
  collected_at: string
}

export type EdgeMetricsResponse = {
  latest: EdgeMetrics | null
  series: EdgeMetrics[]
}
```

- [x] **Step 2: edges.ts 追加**

```ts
export function getEdgeMetrics(id: string, window: '1h' | '6h' | '24h' = '1h') {
  return apiFetch<EdgeMetricsResponse>(
    `/api/v1/edges/${encodeURIComponent(id)}/metrics?window=${window}`,
  )
}
```

- [x] **Step 3: query-keys.ts 追加**

```ts
metrics: (id: string) => ['edges', id, 'metrics'] as const,
```

- [x] **Step 4: 类型检查**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit`
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add web/admin/src/lib/api/
git commit -m "feat(admin): add edge metrics api types and client"
```

### Task 9: parseMetrics 解析层

**Files:**
- Modify: `web/admin/src/features/edges/observation.ts`
- Modify: `web/admin/src/features/edges/observation.test.ts`

**Interfaces:**
- Produces: `MetricsPoint`、`parseMetrics(data: unknown): { latest: MetricsPoint | null; series: MetricsPoint[] }`

- [x] **Step 1: 先写失败测试**

```ts
// observation.test.ts 追加
describe('parseMetrics', () => {
  it('maps optional gpu and io fields', () => {
    const out = parseMetrics({
      latest: {
        cpu_usage_percent: 30,
        mem_used_bytes: 8 * 1024 ** 3,
        mem_total_bytes: 16 * 1024 ** 3,
        mem_usage_percent: 50,
        gpus: [{ name: 'RTX 4090', usage_percent: 60, vram_used_bytes: 12 * 1024 ** 3, vram_total_bytes: 24 * 1024 ** 3 }],
        disk_read_bytes_per_sec: 1024,
        collected_at: '2026-08-18T12:00:00Z',
      },
      series: [],
    })
    expect(out.latest?.cpu).toBe(30)
    expect(out.latest?.gpus[0]?.name).toBe('RTX 4090')
    expect(out.latest?.ioRead).toBe(1024)
  })

  it('defaults missing gpu and io to null', () => {
    const out = parseMetrics({ latest: { cpu_usage_percent: 10, mem_used_bytes: 1, mem_total_bytes: 2, mem_usage_percent: 50, collected_at: '2026-08-18T12:00:00Z' }, series: [] })
    expect(out.latest?.gpus).toEqual([])
    expect(out.latest?.ioRead).toBeNull()
  })
})
```

- [x] **Step 2: 运行测试确认失败**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx vitest run src/features/edges/observation.test.ts`
Expected: FAIL（undefined parseMetrics）

- [x] **Step 3: 实现 parseMetrics**

```ts
export type MetricsPoint = {
  time: number
  cpu: number | null
  memUsed: number | null
  memTotal: number | null
  memPct: number | null
  gpus: EdgeGPUMetric[]
  ioRead: number | null
  ioWrite: number | null
}

function asMetrics(v: unknown): EdgeMetrics | null {
  if (!isRecord(v)) return null
  return v as unknown as EdgeMetrics
}

function toPoint(m: EdgeMetrics): MetricsPoint {
  return {
    time: Date.parse(m.collected_at),
    cpu: typeof m.cpu_usage_percent === 'number' ? m.cpu_usage_percent : null,
    memUsed: typeof m.mem_used_bytes === 'number' ? m.mem_used_bytes : null,
    memTotal: typeof m.mem_total_bytes === 'number' ? m.mem_total_bytes : null,
    memPct: typeof m.mem_usage_percent === 'number' ? m.mem_usage_percent : null,
    gpus: Array.isArray(m.gpus) ? m.gpus.filter((g) => isRecord(g)) as EdgeGPUMetric[] : [],
    ioRead: typeof m.disk_read_bytes_per_sec === 'number' ? m.disk_read_bytes_per_sec : null,
    ioWrite: typeof m.disk_write_bytes_per_sec === 'number' ? m.disk_write_bytes_per_sec : null,
  }
}

export function parseMetrics(data: unknown): { latest: MetricsPoint | null; series: MetricsPoint[] } {
  if (!isRecord(data)) return { latest: null, series: [] }
  const latest = asMetrics(data.latest)
  const series = Array.isArray(data.series) ? data.series.flatMap((s) => { const m = asMetrics(s); return m ? [toPoint(m)] : [] }) : []
  return { latest: latest ? toPoint(latest) : null, series }
}
```

- [x] **Step 4: 运行测试确认通过**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx vitest run src/features/edges/observation.test.ts`
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add web/admin/src/features/edges/observation.ts web/admin/src/features/edges/observation.test.ts
git commit -m "feat(admin): parse edge metrics into chart points"
```

## 5. Admin 前端：「系统监控」图表卡组

### Task 10: CPU 面积图卡

**Files:**
- Modify: `web/admin/src/features/edges/observation-panel.tsx`

**Interfaces:**
- Consumes: `parseMetrics` / `MetricsPoint`（Task 9）、`ChartContainer` 等（Task 7）

- [x] **Step 1: 新增 CPU 面积图卡组件**

```tsx
function CpuCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const latest = series.at(-1)
  const data = series.map((p) => ({ time: p.time, cpu: p.cpu }))
  return (
    <div className='bg-card flex min-w-0 flex-1 flex-col gap-4 rounded-xl border p-4 sm:gap-6 sm:p-6'>
      <div className='flex flex-wrap items-center gap-2 sm:gap-4'>
        <div className='flex flex-1 flex-col gap-1'>
          <p className='text-xl leading-tight font-semibold tracking-tight sm:text-2xl'>
            {latest?.cpu != null ? `${latest.cpu.toFixed(1)}%` : '—'}
          </p>
          <p className='text-muted-foreground text-xs'>{t('edges.monitorCpu')}</p>
        </div>
        <div className='hidden items-center gap-3 sm:flex sm:gap-5'>
          <div className='flex items-center gap-1.5'>
            <div className='size-2.5 rounded-full sm:size-3' style={{ backgroundColor: 'var(--primary)' }} />
            <span className='text-muted-foreground text-[10px] sm:text-xs'>{t('edges.monitorCpu')}</span>
          </div>
        </div>
      </div>
      <div className='h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]'>
        <ChartContainer config={{ cpu: { label: t('edges.monitorCpu'), color: 'var(--primary)' } }} className='h-full w-full'>
          <AreaChart data={data} margin={{ left: 12, right: 12 }}>
            <defs>
              <linearGradient id='cpuGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop offset='0%' stopColor='var(--color-cpu)' stopOpacity={0.3} />
                <stop offset='100%' stopColor='var(--color-cpu)' stopOpacity={0.05} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis dataKey='time' tickFormatter={(v: number) => new Date(v).toLocaleTimeString()} tickLine={false} axisLine={false} />
            <YAxis domain={[0, 100]} tickFormatter={(v: number) => `${v}%`} tickLine={false} axisLine={false} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area dataKey='cpu' type='natural' fill='url(#cpuGradient)' stroke='var(--color-cpu)' strokeWidth={2} />
          </AreaChart>
        </ChartContainer>
      </div>
    </div>
  )
}
```

（`ChartContainer` 的 `config` 键 `cpu` 会生成 `--color-cpu` CSS 变量；`ChartStyle` 由 ChartContainer 内部注入。若 recharts v3 需要 `accessibilityLayer` 属性，按 shadcn 官方示例补充。）

- [x] **Step 2: 类型检查**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/edges/observation-panel.tsx
git commit -m "feat(admin): render CPU usage area chart card"
```

### Task 11: 内存环形图卡

**Files:**
- Modify: `web/admin/src/features/edges/observation-panel.tsx`

- [x] **Step 1: 新增内存环形图卡**（抄「Sales by Category」卡结构）

```tsx
function MemCard({ point }: { point: MetricsPoint | null }) {
  const { t } = useTranslation()
  const pct = point?.memPct ?? 0
  const used = point?.memUsed ?? 0
  const total = point?.memTotal ?? 0
  const free = Math.max(total - used, 0)
  const data = [
    { name: t('edges.monitorMemUsed'), value: used, fill: 'var(--color-used)' },
    { name: t('edges.monitorMemFree'), value: free, fill: 'var(--color-free)' },
  ]
  return (
    <div className='bg-card flex flex-1 flex-col gap-4 rounded-xl border p-4 sm:p-5'>
      <div className='flex items-center justify-between'>
        <div>
          <span className='text-sm font-medium sm:text-base'>{t('edges.monitorMem')}</span>
          <p className='text-muted-foreground text-[10px] sm:text-xs'>{formatBytes(total)}</p>
        </div>
      </div>
      <div className='flex flex-1 items-center gap-4 sm:gap-6'>
        <div className='relative size-[100px] shrink-0 sm:size-[120px]'>
          <ChartContainer config={{ used: { label: t('edges.monitorMemUsed'), color: 'var(--primary)' }, free: { label: t('edges.monitorMemFree'), color: 'color-mix(in oklch, var(--primary) 75%, var(--background))' } }} className='h-full w-full'>
            <PieChart>
              <ChartTooltip content={<ChartTooltipContent hideLabel />} />
              <Pie data={data} dataKey='value' nameKey='name' innerRadius={30} outerRadius={49.5} strokeWidth={0}>
                {data.map((entry) => <Cell key={entry.name} fill={entry.fill} />)}
              </Pie>
            </PieChart>
          </ChartContainer>
          <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center'>
            <span className='text-sm font-semibold sm:text-base'>{pct.toFixed(1)}%</span>
            <span className='text-muted-foreground text-[8px] sm:text-[10px]'>{t('edges.monitorMemUsed')}</span>
          </div>
        </div>
        <div className='flex flex-1 flex-col gap-2 sm:gap-3'>
          {data.map((entry) => (
            <div key={entry.name} className='flex items-center justify-between gap-2'>
              <div className='flex items-center gap-2'>
                <div className='size-2 rounded-full sm:size-2.5' style={{ backgroundColor: entry.fill }} />
                <span className='text-muted-foreground text-[10px] sm:text-xs'>{entry.name}</span>
              </div>
              <div className='flex items-center gap-2 text-[10px] sm:text-xs'>
                <span className='font-medium tabular-nums'>{formatBytes(entry.value)}</span>
                <span className='text-muted-foreground tabular-nums'>
                  {total > 0 ? `${((entry.value / total) * 100).toFixed(1)}%` : '—'}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
```

- [x] **Step 2: 类型检查与测试**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit && npx vitest run src/features/edges/`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/edges/observation-panel.tsx
git commit -m "feat(admin): render memory donut card"
```

### Task 12: GPU 占用率与显存环形图卡

**Files:**
- Modify: `web/admin/src/features/edges/observation-panel.tsx`

- [x] **Step 1: 新增 GPU 卡组**（每 GPU 一张环形图；无 GPU 数据返回 null）

```tsx
function GpuCards({ point }: { point: MetricsPoint | null }) {
  const { t } = useTranslation()
  const gpus = point?.gpus ?? []
  if (gpus.length === 0) return null
  return (
    <>
      {gpus.map((gpu, i) => {
        const usage = gpu.usage_percent ?? 0
        const used = gpu.vram_used_bytes ?? 0
        const total = gpu.vram_total_bytes ?? 0
        return (
          <div key={`${gpu.name}-${i}`} className='bg-card flex flex-1 flex-col gap-4 rounded-xl border p-4 sm:p-5'>
            <span className='text-sm font-medium sm:text-base'>{gpu.name}</span>
            <div className='flex flex-1 items-center gap-4 sm:gap-6'>
              <div className='relative size-[100px] shrink-0 sm:size-[120px]'>
                {/* 占用率环形图：value=usage, free=100-usage，中心 usage% */}
                <ChartContainer config={{ usage: { label: t('edges.monitorGpuUsage'), color: 'var(--primary)' } }} className='h-full w-full'>
                  <PieChart>
                    <Pie data={[{ name: 'usage', value: usage, fill: 'var(--color-usage)' }, { name: 'idle', value: Math.max(100 - usage, 0), fill: 'color-mix(in oklch, var(--primary) 75%, var(--background))' }]} dataKey='value' innerRadius={30} outerRadius={49.5} strokeWidth={0}>
                      <Cell fill='var(--color-usage)' />
                      <Cell fill='color-mix(in oklch, var(--primary) 75%, var(--background))' />
                    </Pie>
                  </PieChart>
                </ChartContainer>
                <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center'>
                  <span className='text-sm font-semibold sm:text-base'>{usage.toFixed(1)}%</span>
                  <span className='text-muted-foreground text-[8px] sm:text-[10px]'>{t('edges.monitorGpuUsage')}</span>
                </div>
              </div>
              {/* 显存环形图：value=used/free，中心 formatBytes(used) */}
            </div>
          </div>
        )
      })}
    </>
  )
}
```

（显存环形图与占用率环形图结构相同，数据用 `vram_used_bytes` / `vram_total_bytes`，中心显示 `formatBytes(used)`。）

- [x] **Step 2: 类型检查与测试**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit && npx vitest run src/features/edges/`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/edges/observation-panel.tsx
git commit -m "feat(admin): render per-GPU usage and VRAM donuts"
```

### Task 13: I/O 双系列面积图卡

**Files:**
- Modify: `web/admin/src/features/edges/observation-panel.tsx`

- [x] **Step 1: 新增 I/O 面积图卡**（双系列 read/write，对照 This Year/Prev Year）

```tsx
function IOCard({ series }: { series: MetricsPoint[] }) {
  const { t } = useTranslation()
  const data = series.map((p) => ({ time: p.time, read: p.ioRead, write: p.ioWrite }))
  return (
    <div className='bg-card flex min-w-0 flex-1 flex-col gap-4 rounded-xl border p-4 sm:gap-6 sm:p-6'>
      <div className='flex flex-wrap items-center gap-2 sm:gap-4'>
        <div className='flex flex-1 flex-col gap-1'>
          <p className='text-muted-foreground text-xs'>{t('edges.monitorIo')}</p>
        </div>
        <div className='hidden items-center gap-3 sm:flex sm:gap-5'>
          <div className='flex items-center gap-1.5'>
            <div className='size-2.5 rounded-full sm:size-3' style={{ backgroundColor: 'var(--primary)' }} />
            <span className='text-muted-foreground text-[10px] sm:text-xs'>{t('edges.monitorIoRead')}</span>
          </div>
          <div className='flex items-center gap-1.5'>
            <div className='size-2.5 rounded-full sm:size-3' style={{ backgroundColor: 'color-mix(in oklch, var(--primary) 75%, var(--background))' }} />
            <span className='text-muted-foreground text-[10px] sm:text-xs'>{t('edges.monitorIoWrite')}</span>
          </div>
        </div>
      </div>
      <div className='h-[200px] w-full min-w-0 sm:h-[240px] lg:h-[280px]'>
        <ChartContainer config={{
          read: { label: t('edges.monitorIoRead'), color: 'var(--primary)' },
          write: { label: t('edges.monitorIoWrite'), color: 'color-mix(in oklch, var(--primary) 75%, var(--background))' },
        }} className='h-full w-full'>
          <AreaChart data={data} margin={{ left: 12, right: 12 }}>
            <defs>
              <linearGradient id='readGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop offset='0%' stopColor='var(--color-read)' stopOpacity={0.3} />
                <stop offset='100%' stopColor='var(--color-read)' stopOpacity={0.05} />
              </linearGradient>
              <linearGradient id='writeGradient' x1='0' y1='0' x2='0' y2='1'>
                <stop offset='0%' stopColor='var(--color-write)' stopOpacity={0.2} />
                <stop offset='100%' stopColor='var(--color-write)' stopOpacity={0.02} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis dataKey='time' tickFormatter={(v: number) => new Date(v).toLocaleTimeString()} tickLine={false} axisLine={false} />
            <YAxis tickFormatter={(v: number) => formatBytes(v)} tickLine={false} axisLine={false} width={60} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area dataKey='read' type='natural' fill='url(#readGradient)' stroke='var(--color-read)' strokeWidth={2} />
            <Area dataKey='write' type='natural' fill='url(#writeGradient)' stroke='var(--color-write)' strokeWidth={2} />
          </AreaChart>
        </ChartContainer>
      </div>
    </div>
  )
}
```

- [x] **Step 2: 类型检查与测试**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit && npx vitest run src/features/edges/`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/edges/observation-panel.tsx
git commit -m "feat(admin): render disk I/O read/write area chart"
```

### Task 14: 卡组组装、空态与详情页接入

**Files:**
- Modify: `web/admin/src/features/edges/observation-panel.tsx`
- Modify: `web/admin/src/features/edges/detail-panel.tsx`

- [x] **Step 1: `observation-panel.tsx` 组装 MonitoringSection**

```tsx
function MonitoringSection({ data }: { data: ReturnType<typeof parseMetrics> }) {
  const { t } = useTranslation()
  return (
    <section className='flex flex-col gap-4'>
      <SectionHead title={t('edges.observationSystem')} hint={t('edges.observationSystemHint')} />
      {data.series.length === 0 ? (
        <EmptyState className='py-6' message={t('edges.monitorEmpty')} />
      ) : (
        <div className='flex flex-col gap-4 xl:flex-row'>
          <CpuCard series={data.series} />
          <div className='flex w-full flex-col gap-4 xl:w-[410px]'>
            <MemCard point={data.latest} />
          </div>
        </div>
      )}
      {data.series.length > 0 ? <GpuCards point={data.latest} /> : null}
      {data.series.length > 0 ? <IOCard series={data.series} /> : null}
    </section>
  )
}
```

`ObservationPanel` 的 props 从 `systemQuery` 改为 `metricsQuery: QueryView<EdgeMetricsResponse>`，`BlockBody` 内渲染 `MonitoringSection data={parseMetrics(metricsQuery.data)}`；删除 `SystemSection`、`parseSystem` 相关字段与「连接被拒绝」错误分支。

- [x] **Step 2: `detail-panel.tsx` 切换数据源**

```tsx
// 删除 import getEdgeSystem；新增 getEdgeMetrics
const metricsQuery = useQuery({
  queryKey: queryKeys.edges.metrics(id),
  queryFn: () => getEdgeMetrics(id, '1h'),
  refetchInterval: 15000,
})
// <ObservationPanel metricsQuery={metricsQuery} tasksQuery={tasksQuery} />
```

- [x] **Step 3: 类型检查与相关测试**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b --noEmit && npx vitest run src/features/edges/`
Expected: PASS（若既有合同测试断言 `getEdgeSystem` 存在，先改断言——见 Task 16）

- [x] **Step 4: 提交**

```bash
git add web/admin/src/features/edges/observation-panel.tsx web/admin/src/features/edges/detail-panel.tsx
git commit -m "feat(admin): wire system monitoring cards into edge detail"
```

### Task 15: i18n 文案

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/zh.json`
- Modify: `web/admin/src/lib/i18n/locales/en.json`

- [x] **Step 1: 更新文案**

```jsonc
// zh.json
"observationSystem": "系统监控",
"observationSystemHint": "CPU、内存、GPU、I/O",
"monitorCpu": "CPU 占用率",
"monitorMem": "内存",
"monitorMemUsed": "已用",
"monitorMemFree": "剩余",
"monitorGpuUsage": "GPU 占用率",
"monitorVram": "显存",
"monitorIo": "I/O",
"monitorIoRead": "读",
"monitorIoWrite": "写",
"monitorEmpty": "暂无监控数据"

// en.json 对应英文
```

- [x] **Step 2: 提交**

```bash
git add web/admin/src/lib/i18n/locales/
git commit -m "feat(admin): system monitoring copy (zh/en)"
```

### Task 16: 合同测试锁定 class 与文案

**Files:**
- Modify: `web/admin/src/features/edges/edges.contract.test.ts`

- [x] **Step 1: 追加合同断言**

```ts
it('locks system monitoring chart classes and copy', () => {
  const observe = read('observation-panel.tsx')
  const detail = read('detail-panel.tsx')
  const zh = read('../../lib/i18n/locales/zh.json')
  const en = read('../../lib/i18n/locales/en.json')
  expect(observe).toMatch(/data-slot="chart"/)
  expect(observe).toMatch(/aspect-video/)
  expect(observe).toMatch(/h-\[200px\] w-full min-w-0 sm:h-\[240px\] lg:h-\[280px\]/)
  expect(observe).toMatch(/size-2\.5 rounded-full sm:size-3/)
  expect(observe).toMatch(/size-\[100px\] shrink-0 sm:size-\[120px\]/)
  expect(observe).toMatch(/rounded-xl border p-4 sm:p-5/)
  expect(observe).toMatch(/color-mix\(in oklch, var\(--primary\) 75%, var\(--background\)\)/)
  expect(observe).toMatch(/var\(--primary\)/)
  expect(observe).not.toMatch(/getEdgeSystem/)
  expect(detail).not.toMatch(/getEdgeSystem/)
  expect(zh).toMatch(/"observationSystem": "系统监控"/)
  expect(en).toMatch(/"observationSystem": "System Monitoring"/)
})
```

同时把既有契约测试里对「系统节」的旧断言（如 `parseSystem`、`errorRefused` 相关）更新为 metrics 语义。

- [x] **Step 2: 运行全部前端测试**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx vitest run`
Expected: PASS

- [x] **Step 3: 提交**

```bash
git add web/admin/src/features/edges/edges.contract.test.ts
git commit -m "test(admin): lock system monitoring chart contract"
```

## 6. 文档与全量验证

### Task 17: 架构文档同步

**Files:**
- Modify: `docs/architecture/data-model.md`
- Modify: `docs/architecture/runtime.md`

- [x] **Step 1: 补充 `edge_metrics` 表与上报路径**

在 `data-model.md` 的 edges 章节后追加 `edge_metrics`（列、索引、保留窗口默认 24h）；在 `runtime.md` 的 Edge/控制面数据流中说明 presence 携带 `metrics`、`GET /api/v1/edges/{id}/metrics`。

- [x] **Step 2: 提交**

```bash
git add docs/architecture/
git commit -m "docs(architecture): edge_metrics table and metrics reporting path"
```

### Task 18: README 环境变量

**Files:**
- Modify: `README.md`

- [x] **Step 1: 追加环境变量说明**

`METRICS_INTERVAL`（Edge，默认 30s）与 `METRICS_RETENTION`（控制面，默认 24h）。

- [x] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: METRICS_INTERVAL and METRICS_RETENTION env vars"
```

### Task 19: 全量验证与收尾

**Files:**
- 无源码改动（验证 + 勾选 tasks.md）

- [x] **Step 1: Go 全量构建与测试**

Run: `cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./...`
Expected: 全部 PASS

- [x] **Step 2: 前端构建与测试**

Run: `cd /Users/mr9esx/Documents/Pixoma/web/admin && npx tsc -b && npx vitest run`
Expected: 全部 PASS

- [x] **Step 3: Mock 模式端到端冒烟**

```bash
cd /Users/mr9esx/Documents/Pixoma && COMFY_MOCK=true METRICS_INTERVAL=5s make dev
```

打开节点详情页确认「系统监控」节出现 CPU/内存/GPU/I/O 图表且 15s 轮询推进；确认页面不再出现「连接被拒绝」。

- [x] **Step 4: 勾选 tasks.md 全部任务并提交**

```bash
git add docs/openspec/changes/edge-system-monitoring/tasks.md
git commit -m "chore(edge-system-monitoring): mark all build tasks complete"
```

### Task 20: 图表改为 Sprint health 样式（Round 6）

**Files:**
- `web/admin/src/features/edges/observation-panel.tsx`
- `web/admin/src/features/edges/observation.ts`
- `web/admin/src/features/edges/observation.test.ts`
- `web/admin/src/features/edges/edges.contract.test.ts`
- `web/admin/src/lib/i18n/locales/zh.json` / `en.json`

- [x] **Step 1: RED — 新增 `seriesStats` 单测与合同测试 Sprint health 断言，运行确认失败**
- [x] **Step 2: GREEN — 实现 `seriesStats`、i18n（当前/最高/平均）、Sprint health 卡片壳（标题 + 左图右值 92px 统计列）**
- [x] **Step 3: 全量验证**

```bash
cd /Users/mr9esx/Documents/Pixoma/web/admin && npx vitest run && npx tsc -b && npm run build
cd /Users/mr9esx/Documents/Pixoma && go build ./... && go test ./...
```

- [x] **Step 4: 代码审查（standard）→ 勾选 tasks.md 3.20 并提交**
