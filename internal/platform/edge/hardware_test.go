package edge

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestShouldWriteHardware(t *testing.T) {
	if !ShouldWriteHardware(Hardware{}, false) {
		t.Fatal("empty should write")
	}
	existing := Hardware{CPUModel: "X"}
	if ShouldWriteHardware(existing, false) {
		t.Fatal("filled without refresh must not write")
	}
	if !ShouldWriteHardware(existing, true) {
		t.Fatal("refresh should write")
	}
}

func TestMergeHardware_FillsVRAMByName(t *testing.T) {
	host := HostInfo{CPUModel: "Intel", CPUCores: 8, RAMBytes: 16 << 30, GPUNames: []string{"NVIDIA GeForce RTX 4090"}}
	st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
		"devices": []any{map[string]any{"name": "NVIDIA GeForce RTX 4090", "vram_total": float64(24 << 30)}},
	}}
	h := MergeHardware(host, st, time.Unix(0, 0).UTC())
	if h.CPUModel != "Intel" || h.CPUCores != 8 || h.RAMBytes != 16<<30 {
		t.Fatalf("cpu/ram: %+v", h)
	}
	if len(h.GPUs) != 1 || h.GPUs[0].VRAMBytes != 24<<30 {
		t.Fatalf("gpus: %+v", h.GPUs)
	}
}

func TestMergeHardware_IgnoresComfyDevicesWhenGHWHasCards(t *testing.T) {
	host := HostInfo{GPUNames: []string{"Card-A"}}
	st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
		"devices": []any{map[string]any{"name": "Card-B", "vram_total": float64(1 << 30)}},
	}}
	h := MergeHardware(host, st, time.Time{})
	if len(h.GPUs) != 1 {
		t.Fatalf("len=%d", len(h.GPUs))
	}
	if h.GPUs[0].Name != "Card-A" || h.GPUs[0].VRAMBytes != 0 {
		t.Fatalf("gpus: %+v", h.GPUs)
	}
}

func TestMergeHardware_AcceptsIntVRAM(t *testing.T) {
	host := HostInfo{GPUNames: []string{"Card"}}
	st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
		"devices": []any{map[string]any{"name": "Card", "vram_total": int(2 << 30)}},
	}}
	h := MergeHardware(host, st, time.Time{})
	if len(h.GPUs) != 1 || h.GPUs[0].VRAMBytes != 2<<30 {
		t.Fatalf("gpus: %+v", h.GPUs)
	}
}

func TestMergeHardware_MergesGHWAndComfyNames(t *testing.T) {
	host := HostInfo{GPUNames: []string{"AD104 [GeForce RTX 4070 SUPER]"}}
	st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
		"devices": []any{map[string]any{
			"name":       "cuda:0 NVIDIA GeForce RTX 4070 SUPER : cudaMallocAsync",
			"vram_total": float64(12 << 30),
		}},
	}}
	h := MergeHardware(host, st, time.Unix(0, 0).UTC())
	if len(h.GPUs) != 1 {
		t.Fatalf("len=%d", len(h.GPUs))
	}
	if h.GPUs[0].Name != "NVIDIA GeForce RTX 4070 SUPER" {
		t.Fatalf("name=%q", h.GPUs[0].Name)
	}
	if h.GPUs[0].VRAMBytes != 12<<30 {
		t.Fatalf("vram=%d", h.GPUs[0].VRAMBytes)
	}
}

func TestMergeHardware_KeepsIdenticalCards(t *testing.T) {
	host := HostInfo{GPUNames: []string{"GeForce RTX 4090", "GeForce RTX 4090"}}
	st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
		"devices": []any{
			map[string]any{"name": "cuda:0 NVIDIA GeForce RTX 4090 : cudaMallocAsync", "vram_total": float64(24 << 30)},
			map[string]any{"name": "cuda:1 NVIDIA GeForce RTX 4090 : cudaMallocAsync", "vram_total": float64(24 << 30)},
		},
	}}
	h := MergeHardware(host, st, time.Time{})
	if len(h.GPUs) != 2 {
		t.Fatalf("len=%d", len(h.GPUs))
	}
	for i, gpu := range h.GPUs {
		if gpu.Name != "NVIDIA GeForce RTX 4090" || gpu.VRAMBytes != 24<<30 {
			t.Fatalf("gpus[%d]=%+v", i, gpu)
		}
	}
}

func TestMergeHardware_ExtractsGHWBracketName(t *testing.T) {
	host := HostInfo{GPUNames: []string{"AD104 [GeForce RTX 4070 SUPER]"}}
	h := MergeHardware(host, nil, time.Time{})
	if len(h.GPUs) != 1 || h.GPUs[0].Name != "GeForce RTX 4070 SUPER" {
		t.Fatalf("gpus: %+v", h.GPUs)
	}
}

func TestMergeHardware_FallsBackToComfyDevices(t *testing.T) {
	host := HostInfo{}
	st := &comfyui.SystemStats{Reachable: true, Raw: map[string]any{
		"devices": []any{map[string]any{
			"name":       "cuda:0 NVIDIA GeForce RTX 4060 : cudaMallocAsync",
			"vram_total": float64(8 << 30),
		}},
	}}
	h := MergeHardware(host, st, time.Time{})
	if len(h.GPUs) != 1 || h.GPUs[0].Name != "NVIDIA GeForce RTX 4060" || h.GPUs[0].VRAMBytes != 8<<30 {
		t.Fatalf("gpus: %+v", h.GPUs)
	}
}

func TestExtractGPUModel(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"AD104 [GeForce RTX 4070 SUPER]", "GeForce RTX 4070 SUPER"},
		{"cuda:0 NVIDIA GeForce RTX 4070 SUPER : cudaMallocAsync", "NVIDIA GeForce RTX 4070 SUPER"},
		{"cuda:1 NVIDIA GeForce RTX 4090 : cudaMalloc", "NVIDIA GeForce RTX 4090"},
		{"NVIDIA GeForce RTX 4090", "NVIDIA GeForce RTX 4090"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := extractGPUModel(tc.in); got != tc.want {
			t.Fatalf("extractGPUModel(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}
