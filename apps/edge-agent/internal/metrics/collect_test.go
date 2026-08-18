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
	readBytes := uint64(10 << 20)
	s.DiskIO = func(_ context.Context) (map[string]disk.IOCountersStat, error) {
		readBytes += 10 << 20
		return map[string]disk.IOCountersStat{
			"sda": {ReadBytes: readBytes, WriteBytes: 4 << 20},
		}, nil
	}
	s.NvidiaSMI = func(_ context.Context) ([]edge.GPUMetric, error) {
		return []edge.GPUMetric{{
			Name:             "RTX 4090",
			UsagePercent:     floatPtr(60),
			VRAMUsedBytes:    12 << 30,
			VRAMTotalBytes:   24 << 30,
			VRAMUsagePercent: floatPtr(50),
		}}, nil
	}

	// 首拍（since 零值）：I/O 应为 nil
	first := s.Sample(context.Background(), time.Time{})
	if first.CPUUsagePercent != 42.5 || first.MemUsedBytes != 8<<30 || first.DiskReadBytesPerSec != nil {
		t.Fatalf("first sample: %+v", first)
	}
	if len(first.GPUs) != 1 || first.GPUs[0].Name != "RTX 4090" {
		t.Fatalf("gpus: %+v", first.GPUs)
	}

	// 第二拍：I/O 速率 = 差值 / 30s = 10MiB/30 ≈ 349525.33
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

func TestSample_CPUPercentMustNotSleep(t *testing.T) {
	// gopsutil cpu.PercentWithContext(interval>0) blocks for the interval;
	// the sampler must always ask for a non-blocking since-last-call sample.
	s := metrics.NewSampler(false)
	s.Now = func() time.Time { return time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC) }
	var got time.Duration
	s.CPUPercent = func(_ context.Context, interval time.Duration) (float64, error) {
		got = interval
		return 0, nil
	}
	s.VirtualMemory = func(_ context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Used: 1, Total: 2, UsedPercent: 50}, nil
	}
	s.DiskIO = func(_ context.Context) (map[string]disk.IOCountersStat, error) {
		return map[string]disk.IOCountersStat{}, nil
	}
	_ = s.Sample(context.Background(), time.Time{})
	if got != 0 {
		t.Fatalf("cpu percent interval must be 0 (non-blocking), got %v", got)
	}
}
