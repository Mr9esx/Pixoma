package hardware_test

import (
	"context"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/edge-agent/internal/hardware"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestCollect_MergesInspectAndMockVRAM(t *testing.T) {
	inspect := func() (edge.HostInfo, error) {
		return edge.HostInfo{
			CPUModel: "Intel",
			CPUCores: 8,
			RAMBytes: 16 << 30,
			GPUNames: []string{"Mock GPU"},
		}, nil
	}
	comfy, err := comfyui.NewClient(comfyui.Options{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	h := hardware.Collect(context.Background(), inspect, comfy)
	if h.CPUModel != "Intel" || h.CPUCores != 8 || h.RAMBytes != 16<<30 {
		t.Fatalf("host: %+v", h)
	}
	if len(h.GPUs) == 0 || h.GPUs[0].VRAMBytes == 0 {
		t.Fatalf("gpus: %+v", h.GPUs)
	}
}

func TestCollect_InspectErrorStillUsesComfy(t *testing.T) {
	inspect := func() (edge.HostInfo, error) {
		return edge.HostInfo{}, errString("no ghw")
	}
	comfy, err := comfyui.NewClient(comfyui.Options{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	h := hardware.Collect(context.Background(), inspect, comfy)
	if len(h.GPUs) == 0 || h.GPUs[0].Name == "" {
		t.Fatalf("expected comfy device, got %+v", h)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
