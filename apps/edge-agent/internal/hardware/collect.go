package hardware

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jaypipes/ghw"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

// Inspect reads host CPU/RAM/GPU names. Production uses InspectGHW; tests inject a stub.
type Inspect func() (edge.HostInfo, error)

// InspectGHW collects CPU, memory, and GPU product names via ghw.
func InspectGHW() (edge.HostInfo, error) {
	var info edge.HostInfo
	var errs []error

	cpu, err := ghw.CPU(ghw.WithDisableWarnings())
	if err != nil {
		errs = append(errs, err)
	} else if cpu != nil {
		info.CPUCores = int(cpu.TotalCores)
		for _, proc := range cpu.Processors {
			if proc == nil {
				continue
			}
			if m := strings.TrimSpace(proc.Model); m != "" {
				info.CPUModel = m
				break
			}
		}
	}

	mem, err := ghw.Memory(ghw.WithDisableWarnings())
	if err != nil {
		errs = append(errs, err)
	} else if mem != nil && mem.TotalPhysicalBytes > 0 {
		info.RAMBytes = uint64(mem.TotalPhysicalBytes)
	}

	gpu, err := ghw.GPU(ghw.WithDisableWarnings())
	if err != nil {
		errs = append(errs, err)
	} else if gpu != nil {
		for _, card := range gpu.GraphicsCards {
			if card == nil || card.DeviceInfo == nil || card.DeviceInfo.Product == nil {
				continue
			}
			name := strings.TrimSpace(card.DeviceInfo.Product.Name)
			if name == "" {
				continue
			}
			info.GPUNames = append(info.GPUNames, name)
		}
	}

	if info.CPUModel == "" && info.CPUCores == 0 && info.RAMBytes == 0 && len(info.GPUNames) == 0 && len(errs) > 0 {
		return info, errors.Join(errs...)
	}
	return info, nil
}

// Collect merges host inspect with Comfy system_stats. Inspect or Comfy failures leave that side empty.
func Collect(ctx context.Context, inspect Inspect, comfy comfyui.Client) edge.Hardware {
	var host edge.HostInfo
	if inspect != nil {
		if h, err := inspect(); err == nil {
			host = h
		}
	}
	var stats *comfyui.SystemStats
	if comfy != nil {
		st, err := comfy.SystemStats(ctx)
		if err == nil {
			stats = st
		}
	}
	return edge.MergeHardware(host, stats, time.Now().UTC())
}
