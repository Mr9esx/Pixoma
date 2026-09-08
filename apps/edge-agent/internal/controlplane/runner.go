package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/actuator"
)

// Loop repeatedly claims jobs and runs the worker.

type Loop struct {
	Client *Client
	Worker *actuator.Worker
	Wait   time.Duration
	Once   bool // for tests: stop after one claim attempt (empty or executed)
}

const DefaultClaimWait = 5 * time.Second

func (l *Loop) Run(ctx context.Context) error {
	if l.Client == nil || l.Worker == nil {
		return fmt.Errorf("pull: loop not configured")
	}
	wait := l.Wait
	if wait <= 0 && !l.Once {
		wait = DefaultClaimWait
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		job, err := l.Client.Claim(ctx, wait)
		if err != nil {
			if l.Once {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
				continue
			}
		}
		if job == nil {
			if l.Once {
				return nil
			}
			continue
		}
		cmd := sharedkernel.DispatchCommand{
			TaskID: job.TaskID,
			EdgeID: job.EdgeID,
			JobRef: job.JobRef,
		}
		if cmd.EdgeID == "" {
			cmd.EdgeID = sharedkernel.EdgeID(l.Client.EdgeID)
		}
		hbCtx, cancel := context.WithCancel(ctx)
		go l.heartbeat(hbCtx, job.TaskID)
		err = l.Worker.HandleDispatch(ctx, cmd)
		cancel()
		if err != nil && l.Once {
			return err
		}
		if l.Once {
			return nil
		}
	}
}

func (l *Loop) heartbeat(ctx context.Context, taskID sharedkernel.TaskID) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = l.Client.Heartbeat(ctx, taskID)
		}
	}
}
