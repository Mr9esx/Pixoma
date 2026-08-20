package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestStormGuard_ScheduleAndReconcilePoolsIndependent(t *testing.T) {
	guard := orchestrator.NewStormGuard(orchestrator.StormConfig{
		SchedulePerTick:  1,
		ReconcilePerTick: 1,
	})
	if !guard.AllowSchedule(1) {
		t.Fatal("schedule bucket should allow first")
	}
	if guard.AllowSchedule(1) {
		t.Fatal("schedule bucket should be exhausted")
	}
	if !guard.AllowReconcile(1) {
		t.Fatal("reconcile bucket must be independent of schedule")
	}
}

func TestReconcileStale_RespectsStormLimit(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(500, 0).UTC()

	// Three stale queued tasks without job_ref (accepted phase) → re-pended by reconcile.
	for i := 0; i < 3; i++ {
		task := runtimedomain.NewPending(sharedkernel.TaskID(string(rune('a'+i))), "s", sharedkernel.CaseID(1), "in", now.Add(-time.Hour))
		task.Status = sharedkernel.TaskQueued
		task.UpdatedAt = now.Add(-time.Hour)
		if err := tasks.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}

	svc := orchestrator.New(tasks, nil, &captureBus{}, &memNotify{})
	svc.Now = func() time.Time { return now }
	svc.Storm = orchestrator.NewStormGuard(orchestrator.StormConfig{ReconcilePerTick: 3})

	if err := svc.ReconcileStale(ctx, time.Minute, 10); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	pending := 0
	for _, id := range []sharedkernel.TaskID{"a", "b", "c"} {
		got, _ := tasks.Get(ctx, id)
		if got.Status == sharedkernel.TaskPending {
			pending++
		}
	}
	if pending != 2 {
		t.Fatalf("reconciled %d tasks in one tick, want 2 (storm-limited)", pending)
	}
}
