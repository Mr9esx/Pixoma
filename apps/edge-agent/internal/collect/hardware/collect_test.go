package hardware_test

import (
	"context"
	"testing"

	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/collect/hardware"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui/comfyuitest"
)

func TestCollect_MergesInspectAndComfyVRAM(t *testing.T) {
	inspect := func() (edge.HostInfo, error) {
		return edge.HostInfo{
			CPUModel: "Intel",
			CPUCores: 8,
			RAMBytes: 16 << 30,
			GPUNames: []string{"Fake GPU"},
		}, nil
	}
	comfy := &comfyuitest.Fake{}
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
	comfy := &comfyuitest.Fake{}
	h := hardware.Collect(context.Background(), inspect, comfy)
	if len(h.GPUs) == 0 || h.GPUs[0].Name == "" {
		t.Fatalf("expected comfy device, got %+v", h)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
