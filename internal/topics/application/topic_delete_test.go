package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	casepersist "github.com/Mr9esx/Pixoma/internal/cases/infrastructure/persistence"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	instpersist "github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	topicdomain "github.com/Mr9esx/Pixoma/internal/topics/domain"
	topicpersist "github.com/Mr9esx/Pixoma/internal/topics/infrastructure/persistence"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	topicapp "github.com/Mr9esx/Pixoma/internal/topics/application"
)

type captureNotify struct {
	items []sharedkernel.UserNotify
}

func (c *captureNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	c.items = append(c.items, n)
	return nil
}

func newTopicTest(t *testing.T) (*gorm.DB, *topicapp.Service, *captureNotify, context.Context, time.Time) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:topic_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&topicpersist.TopicRow{},
		&casepersist.CaseRow{},
		&instpersist.EdgeRow{},
		&taskpersist.TaskRow{},
		&sesspersist.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	n := &captureNotify{}
	return gdb, topicapp.NewService(gdb, n), n, ctx, now
}

func seedTopicData(t *testing.T, gdb *gorm.DB, now time.Time) {
	t.Helper()
	ctx := context.Background()
	topicRepo := topicpersist.NewTopicRepository(gdb)
	if err := topicRepo.Create(ctx, topicdomain.Topic{Key: "default", Name: "默认", Enabled: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := topicRepo.Create(ctx, topicdomain.Topic{Key: "fast-gpu", Name: "F", Enabled: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	caseRepo := casepersist.NewGormRepository(gdb)
	if err := caseRepo.Save(ctx, &catalogdomain.Case{Document: catalogdomain.CaseDocument{
		ID:   1,
		Name: "c1",
		Routing: &catalogdomain.RoutingConfig{Rules: []catalogdomain.RoutingRule{
			{Topic: "fast-gpu"},
			{Topic: "default"},
		}},
	}, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	edgeRepo := instpersist.NewEdgeRepository(gdb)
	if err := edgeRepo.Upsert(ctx, &edge.Record{
		ID:              "gpu-1",
		Name:            "gpu-1",
		Enabled:         true,
		Capabilities:    []string{"comfy"},
		SubscribeTopics: []string{"fast-gpu", "default"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatal(err)
	}

	taskRepo := taskpersist.NewTaskRepository(gdb)
	sessions := sesspersist.NewSessionRepository(gdb)
	if err := sessions.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	queued := runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)
	queued.Status = sharedkernel.TaskQueued
	queued.DispatchTopic = "fast-gpu"
	queued.JobRef = sharedkernel.BlobRef{Key: "jobs/t1/job.json"}
	if err := taskRepo.Create(ctx, queued); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteTopic_AckRequired(t *testing.T) {
	gdb, svc, _, ctx, now := newTopicTest(t)
	seedTopicData(t, gdb, now)

	_, err := svc.DeleteTopic(ctx, "fast-gpu", false)
	if !errors.Is(err, topicapp.ErrNeedsAck) {
		t.Fatalf("err=%v want ErrNeedsAck", err)
	}
	topicRepo := topicpersist.NewTopicRepository(gdb)
	if _, err := topicRepo.Get(ctx, "fast-gpu"); err != nil {
		t.Fatalf("topic must survive rollback: %v", err)
	}
}

func TestDeleteTopic_CleanupAndDelete(t *testing.T) {
	gdb, svc, n, ctx, now := newTopicTest(t)
	seedTopicData(t, gdb, now)

	summary, err := svc.DeleteTopic(ctx, "fast-gpu", true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.RemovedCaseRules != 1 || summary.RemovedEdgeSubs != 1 || summary.FailedTasks != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	topicRepo := topicpersist.NewTopicRepository(gdb)
	if _, err := topicRepo.Get(ctx, "fast-gpu"); !errors.Is(err, topicdomain.ErrTopicNotFound) {
		t.Fatalf("topic delete err=%v", err)
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	c, err := caseRepo.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Document.Routing.Rules) != 1 || c.Document.Routing.Rules[0].Topic != "default" {
		t.Fatalf("case rules=%+v", c.Document.Routing.Rules)
	}
	edgeRepo := instpersist.NewEdgeRepository(gdb)
	rec, err := edgeRepo.Get(ctx, "gpu-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.SubscribeTopics) != 1 || rec.SubscribeTopics[0] != "default" {
		t.Fatalf("edge topics=%+v", rec.SubscribeTopics)
	}
	taskRepo := taskpersist.NewTaskRepository(gdb)
	t1, err := taskRepo.Get(ctx, "t1")
	if err != nil || t1.Status != sharedkernel.TaskFailed || t1.ErrorCode != sharedkernel.TaskErrorTopicDeleted {
		t.Fatalf("queued task=%+v err=%v", t1, err)
	}
	if len(n.items) != 1 || n.items[0].Kind != "task_failed" || n.items[0].ChatID != "tg:9" {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func TestDeleteTopic_DefaultProtected(t *testing.T) {
	gdb, svc, _, ctx, now := newTopicTest(t)
	seedTopicData(t, gdb, now)
	_, err := svc.DeleteTopic(ctx, "default", true)
	if !errors.Is(err, topicapp.ErrDefaultProtected) {
		t.Fatalf("err=%v want ErrDefaultProtected", err)
	}
}
