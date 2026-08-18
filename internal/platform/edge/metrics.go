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
