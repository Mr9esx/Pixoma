package edge

import (
	"strings"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

// GPU is one graphics card reported by the host and/or Comfy.
type GPU struct {
	Name      string `json:"name"`
	VRAMBytes uint64 `json:"vram_bytes,omitempty"`
}

// Hardware is the last collected machine spec for an edge.
type Hardware struct {
	CPUModel    string    `json:"cpu_model,omitempty"`
	CPUCores    int       `json:"cpu_cores,omitempty"`
	RAMBytes    uint64    `json:"ram_bytes,omitempty"`
	GPUs        []GPU     `json:"gpus,omitempty"`
	CollectedAt time.Time `json:"collected_at"`
}

// HostInfo is CPU/RAM/GPU names from the local machine (ghw).
type HostInfo struct {
	CPUModel string
	CPUCores int
	RAMBytes uint64
	GPUNames []string
}

// HardwareEmpty reports whether spec fields are unset (CollectedAt ignored).
func HardwareEmpty(h Hardware) bool {
	return h.CPUModel == "" && h.CPUCores == 0 && h.RAMBytes == 0 && len(h.GPUs) == 0
}

// ShouldWriteHardware is true when the stored spec is empty or the operator asked to refresh.
func ShouldWriteHardware(existing Hardware, refreshRequested bool) bool {
	return refreshRequested || HardwareEmpty(existing)
}

// MergeHardware trusts ghw as the primary GPU source: one entry per host card,
// with VRAM filled from Comfy system_stats devices. Display names are extracted
// to the marketing model (e.g. "GeForce RTX 4070 SUPER"), preferring the fuller
// Comfy name ("NVIDIA GeForce RTX 4070 SUPER") when the two sources agree.
// Comfy's device list is used as a fallback only when ghw reported no cards.
//
func MergeHardware(host HostInfo, stats *comfyui.SystemStats, now time.Time) Hardware {
	h := Hardware{
		CPUModel:    host.CPUModel,
		CPUCores:    host.CPUCores,
		RAMBytes:    host.RAMBytes,
		CollectedAt: now,
	}
	devs := comfyDevices(stats)
	if len(host.GPUNames) == 0 {
		for _, dev := range devs {
			h.GPUs = append(h.GPUs, GPU{Name: extractGPUModel(dev.name), VRAMBytes: dev.vram})
		}
		return h
	}
	consumed := make([]bool, len(devs))
	for _, name := range host.GPUNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		gpu := GPU{Name: extractGPUModel(name)}
		if i, ok := matchGPUDevice(devs, consumed, name); ok {
			consumed[i] = true
			gpu.VRAMBytes = devs[i].vram
			if cleaned := extractGPUModel(devs[i].name); cleaned != "" {
				gpu.Name = cleaned
			}
		}
		h.GPUs = append(h.GPUs, gpu)
	}
	return h
}

type comfyDevice struct {
	name string
	vram uint64
}

func comfyDevices(stats *comfyui.SystemStats) []comfyDevice {
	if stats == nil || stats.Raw == nil {
		return nil
	}
	raw, ok := stats.Raw["devices"].([]any)
	if !ok {
		return nil
	}
	out := make([]comfyDevice, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		vram := asUint64(m["vram_total"])
		if name == "" && vram == 0 {
			continue
		}
		out = append(out, comfyDevice{name: name, vram: vram})
	}
	return out
}

// matchGPUDevice returns the index of the first unconsumed Comfy device that
// refers to the same physical card as the host GPU. Exact normalized names win;
// otherwise a containment match is accepted (e.g. "geforce rtx 4070 super" is
// contained in "nvidia geforce rtx 4070 super").
func matchGPUDevice(devs []comfyDevice, consumed []bool, hostName string) (int, bool) {
	key := gpuNameKey(hostName)
	fallback := -1
	for i, dev := range devs {
		if consumed[i] {
			continue
		}
		devKey := gpuNameKey(dev.name)
		if devKey == key {
			return i, true
		}
		if fallback < 0 && gpuNamesContain(key, devKey) {
			fallback = i
		}
	}
	return fallback, fallback >= 0
}

// gpuNamesContain reports whether one normalized GPU name contains the other.
func gpuNamesContain(a, b string) bool {
	if a == "" || b == "" || a == b {
		return false
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// gpuNameKey normalizes a GPU name for matching: it strips CUDA indices and
// allocator suffixes and prefers the bracketed marketing name.
func gpuNameKey(name string) string {
	s := strings.ToLower(extractGPUModel(name))
	return strings.Join(strings.Fields(s), " ")
}

// extractGPUModel returns the marketing model from common GPU name formats:
//   - "AD104 [GeForce RTX 4070 SUPER]" -> "GeForce RTX 4070 SUPER"
//   - "cuda:0 NVIDIA GeForce RTX 4070 SUPER : cudaMallocAsync" -> "NVIDIA GeForce RTX 4070 SUPER"
//   - "NVIDIA GeForce RTX 4090" -> "NVIDIA GeForce RTX 4090"
func extractGPUModel(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "cuda:"); i >= 0 {
		s = strings.TrimSpace(strings.TrimLeft(s[i+len("cuda:"):], "0123456789"))
	}
	if i := strings.LastIndex(s, " : "); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if i := strings.Index(s, "["); i >= 0 {
		if j := strings.Index(s[i:], "]"); j >= 0 {
			if inner := strings.TrimSpace(s[i+1 : i+j]); inner != "" {
				return inner
			}
		}
	}
	return s
}

func asUint64(v any) uint64 {
	switch n := v.(type) {
	case uint64:
		return n
	case uint:
		return uint64(n)
	case int:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case int64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	case float64:
		if n < 0 {
			return 0
		}
		return uint64(n)
	default:
		return 0
	}
}
