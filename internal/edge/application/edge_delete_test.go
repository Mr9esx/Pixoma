package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	edgeapp "github.com/Mr9esx/Pixoma/internal/edge/application"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	instpersist "github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type captureNotify struct {
	items []sharedkernel.UserNotify
}

func (c *captureNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	c.items = append(c.items, n)
	return nil
}

func newEdgeTest(t *testing.T) (*gorm.DB, *edgeapp.Service, *captureNotify, context.Context, time.Time) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:edge_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&instpersist.EdgeRow{},
		&taskpersist.TaskRow{},
		&sesspersist.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	n := &captureNotify{}
	return gdb, edgeapp.NewService(gdb, n), n, ctx, now
}

func seedEdge(t *testing.T, gdb *gorm.DB, id sharedkernel.EdgeID, now time.Time) {
	t.Helper()
	repo := instpersist.NewEdgeRepository(gdb)
	if err := repo.Upsert(context.Background(), &edge.Record{
		ID:           id,
		Name:         string(id),
		Enabled:      true,
		Capabilities: []string{"comfy"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteEdge_AckRequired(t *testing.T) {
	gdb, svc, _, ctx, now := newEdgeTest(t)
	seedEdge(t, gdb, "gpu-1", now)
	tasks := taskpersist.NewTaskRepository(gdb)
	running := runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)
	if err := running.MarkQueued("gpu-1", now); err != nil {
		t.Fatal(err)
	}
	if err := running.MarkRunning("p", now); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, running); err != nil {
		t.Fatal(err)
	}

	_, err := svc.DeleteEdge(ctx, "gpu-1", false)
	if !errors.Is(err, edgeapp.ErrNeedsAck) {
		t.Fatalf("err=%v want ErrNeedsAck", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	if _, err := repo.Get(ctx, "gpu-1"); err != nil {
		t.Fatalf("edge must survive rollback: %v", err)
	}
}

func TestDeleteEdge_CleanupAndDelete(t *testing.T) {
	gdb, svc, n, ctx, now := newEdgeTest(t)
	seedEdge(t, gdb, "gpu-1", now)
	tasks := taskpersist.NewTaskRepository(gdb)
	sessions := sesspersist.NewSessionRepository(gdb)
	if err := sessions.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	running := runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)
	if err := running.MarkQueued("gpu-1", now); err != nil {
		t.Fatal(err)
	}
	if err := running.MarkRunning("p", now); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, running); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, runtimedomain.NewPending("t2", "s2", 1, "inputs/t2", now)); err != nil {
		t.Fatal(err)
	}

	summary, err := svc.DeleteEdge(ctx, "gpu-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.FailedTasks != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	if _, err := repo.Get(ctx, "gpu-1"); !errors.Is(err, edge.ErrNotFound) {
		t.Fatalf("edge delete err=%v", err)
	}
	t1, err := tasks.Get(ctx, "t1")
	if err != nil || t1.Status != sharedkernel.TaskFailed || t1.ErrorCode != sharedkernel.TaskErrorEdgeDeleted {
		t.Fatalf("running task=%+v err=%v", t1, err)
	}
	t2, err := tasks.Get(ctx, "t2")
	if err != nil || t2.Status != sharedkernel.TaskPending {
		t.Fatalf("pending task must be untouched: %+v err=%v", t2, err)
	}
	if len(n.items) != 1 || n.items[0].Kind != "task_failed" || n.items[0].ChatID != "tg:9" {
		t.Fatalf("notifies=%+v", n.items)
	}
}
