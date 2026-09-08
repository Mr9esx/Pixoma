package application

import (
	"context"
	"log/slog"

	edgehw "github.com/Mr9esx/Pixoma/apps/edge-agent/internal/collect/hardware"
	edgemetrics "github.com/Mr9esx/Pixoma/apps/edge-agent/internal/collect/metrics"
	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/controlplane"
	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/presence"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/factory"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/actuator"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
)

func Run(ctx context.Context, cfg Config) error {
	blobStore, err := factory.New(cfg.BlobDriver, cfg.BlobLocalRoot)
	if err != nil {
		return err
	}

	comfy, err := comfyui.NewClient(comfyui.Options{BaseURL: cfg.ComfyURL})
	if err != nil {
		return err
	}

	client := controlplane.NewClient(cfg.ControlPlaneURL, cfg.AgentToken, cfg.EdgeID)
	worker := &actuator.Worker{
		EdgeID: sharedkernel.EdgeID(cfg.EdgeID),
		Comfy:  comfy,
		Blob:   blobStore,
		Status: controlplane.NewStatusPublisher(client),
	}
	loop := &controlplane.Loop{Client: client, Worker: worker, Wait: cfg.ClaimWait}
	sampler := edgemetrics.NewSampler()
	reporter := &presence.Reporter{
		Client: client,
		Comfy:  comfy,
		Collect: func(ctx context.Context) edge.Hardware {
			return edgehw.Collect(ctx, edgehw.InspectGHW, comfy)
		},
		Sample:          sampler.Sample,
		MetricsInterval: cfg.MetricsInterval,
		SendHardware:    true,
	}

	slog.Info("pixoma-edge-agent running",
		"edge_id", cfg.EdgeID,
		"control_plane", cfg.ControlPlaneURL,
		"blob_driver", cfg.BlobDriver,
	)
	go func() {
		_ = reporter.Run(ctx)
	}()
	return loop.Run(ctx)
}
