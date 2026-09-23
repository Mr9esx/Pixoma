package application

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func TestWorkflowReconcilerEventSequenceContinuesAfterThousand(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:workflow_reconciler_long_sequence?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-long-workflow", "session-long-workflow", "account-a", "message-long-workflow", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	for sequence := uint64(1); sequence <= 1005; sequence++ {
		if err := repo.AppendEvent(ctx, &domain.Event{
			ID: fmt.Sprintf("workflow-legacy-%d", sequence), RunID: run.ID,
			SessionID: run.SessionID, AccountID: run.AccountID, Sequence: sequence,
			Type: "CUSTOM", Payload: json.RawMessage(`{}`), CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	reconciler := &WorkflowReconciler{Repo: repo, IDs: func() string { return "workflow-next" }, Now: func() time.Time { return now }}
	execution := &domain.WorkflowExecution{RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID}
	if err := reconciler.appendEvent(ctx, execution, EventWorkflowTaskCompleted, map[string]any{"task_id": "task-1"}); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 1005, 10)
	if err != nil || len(events) != 1 || events[0].Sequence != 1006 {
		t.Fatalf("events after 1005 = (%#v, %v)", events, err)
	}
}
