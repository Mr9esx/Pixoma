package scheduling

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/tasks/application/orchestrator"
)

const schedulerStaleAfter = 2 * time.Minute

// SubscribeTaskCreated wires in-process TaskCreated messages to claimable dispatch.
func SubscribeTaskCreated(ctx context.Context, bus queue.Bus, orch *orchestrator.Service) error {
	if bus == nil || orch == nil {
		return nil
	}
	return bus.Subscribe(ctx, sharedkernel.TopicTaskCreated, func(ctx context.Context, msg queue.Message) error {
		var ev sharedkernel.TaskCreated
		if err := json.Unmarshal(msg.Payload, &ev); err != nil {
			return err
		}
		return orch.OnTaskCreated(ctx, ev)
	})
}

// RunScheduler periodically prepares pending tasks and reclaims stale ones.
func RunScheduler(ctx context.Context, orch *orchestrator.Service) {
	if orch == nil {
		return
	}
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				orch.Storm.ResetTick()
				if err := orch.SchedulePending(ctx, 32); err != nil {
					slog.Warn("schedule pending", "err", err)
				}
				if err := orch.ReconcileStale(ctx, schedulerStaleAfter, 16); err != nil {
					slog.Warn("reconcile stale", "err", err)
				}
			}
		}
	}()
}
