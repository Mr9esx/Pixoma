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

	mu       sync.Mutex
	lastDisk map[string]disk.IOCountersStat
}

// NewSampler wires real gopsutil and nvidia-smi collectors.
func NewSampler() *Sampler {
	s := &Sampler{lastDisk: map[string]disk.IOCountersStat{}}
	s.Now = time.Now
	s.CPUPercent = func(ctx context.Context, interval time.Duration) (float64, error) {
		// interval>0 makes gopsutil sleep for the whole interval, which would
		// stall the presence reporter; use the since-last-call variant instead.
		pcts, err := cpu.PercentWithContext(ctx, 0, false)
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
		// Pass 0 so the collector uses the since-last-call variant; a positive
		// interval makes gopsutil sleep that long and stalls the heartbeat.
		if pct, err := s.CPUPercent(ctx, 0); err == nil {
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
	if s.DiskIO != nil {
		if counters, err := s.DiskIO(ctx); err == nil {
			if !since.IsZero() {
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
			}
			s.lastDisk = counters
		}
	}
	if s.NvidiaSMI != nil {
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
	return parseNvidiaSMIOutput(out)
}

func parseNvidiaSMIOutput(out []byte) ([]edge.GPUMetric, error) {
	reader := csv.NewReader(strings.NewReader(string(out)))
	// nvidia-smi pads fields with a space before quoted names; without this,
	// a GPU name containing a comma would make the whole parse fail.
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
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
