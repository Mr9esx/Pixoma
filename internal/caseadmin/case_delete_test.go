package caseadmin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/caseadmin"
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type captureNotify struct {
	items []sharedkernel.UserNotify
}

func (c *captureNotify) Publish(_ context.Context, n sharedkernel.UserNotify) error {
	c.items = append(c.items, n)
	return nil
}

func newDeleteTest(t *testing.T) (*gorm.DB, *caseadmin.Service, *captureNotify, context.Context, time.Time) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:case_delete_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&casepersist.CaseRow{},
		&taskpersist.TaskRow{},
		&sesspersist.SessionRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{},
		&channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	n := &captureNotify{}
	return gdb, caseadmin.NewService(gdb, n), n, ctx, now
}

func TestDeleteCase_AckRequired(t *testing.T) {
	gdb, svc, _, ctx, now := newDeleteTest(t)
	caseRepo := casepersist.NewGormRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	sessRepo := sesspersist.NewSessionRepository(gdb)
	menuRepo := mencardpersist.NewGormCardRepository(gdb)

	if err := caseRepo.Save(ctx, &catalogdomain.Case{Document: catalogdomain.CaseDocument{ID: 10, Name: "c"}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := taskRepo.Create(ctx, runtimedomain.NewPending("t1", "s1", 10, "inputs/t1", now)); err != nil {
		t.Fatal(err)
	}
	if err := sessRepo.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 10, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	if err := menuRepo.PutMenu(ctx, "ch1", mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"10"}}},
	}}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.DeleteCase(ctx, 10, false)
	if !errors.Is(err, caseadmin.ErrNeedsAck) {
		t.Fatalf("err=%v want ErrNeedsAck", err)
	}
	if _, err := caseRepo.Get(ctx, 10); err != nil {
		t.Fatalf("case must survive rollback: %v", err)
	}
	task, err := taskRepo.Get(ctx, "t1")
	if err != nil || task.Status != sharedkernel.TaskPending {
		t.Fatalf("task must survive rollback: %+v err=%v", task, err)
	}
	sess, err := sessRepo.ListActiveByCase(ctx, 10)
	if err != nil || len(sess) != 1 {
		t.Fatalf("session must survive rollback: %v %+v", err, sess)
	}
}

func TestDeleteCase_CleanupAndDelete(t *testing.T) {
	gdb, svc, n, ctx, now := newDeleteTest(t)
	caseRepo := casepersist.NewGormRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	sessRepo := sesspersist.NewSessionRepository(gdb)
	menuRepo := mencardpersist.NewGormCardRepository(gdb)

	if err := caseRepo.Save(ctx, &catalogdomain.Case{Document: catalogdomain.CaseDocument{ID: 10, Name: "c"}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	pending := runtimedomain.NewPending("t1", "s1", 10, "inputs/t1", now)
	pending.ChatID = "tg:9"
	if err := taskRepo.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}
	running := runtimedomain.NewPending("t2", "s2", 10, "inputs/t2", now)
	if err := running.MarkQueued("local", now); err != nil {
		t.Fatal(err)
	}
	if err := running.MarkRunning("p", now); err != nil {
		t.Fatal(err)
	}
	if err := taskRepo.Create(ctx, running); err != nil {
		t.Fatal(err)
	}
	if err := sessRepo.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 10, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	if err := menuRepo.PutMenu(ctx, "ch1", mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"10"}}},
	}}); err != nil {
		t.Fatal(err)
	}

	summary, err := svc.DeleteCase(ctx, 10, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.FailedTasks != 1 || summary.TerminatedSessions != 1 || len(summary.RemovedPlacements) != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	if _, err := caseRepo.Get(ctx, 10); !errors.Is(err, catalogdomain.ErrNotFound) {
		t.Fatalf("case delete err=%v", err)
	}
	t1, err := taskRepo.Get(ctx, "t1")
	if err != nil || t1.Status != sharedkernel.TaskFailed || t1.ErrorCode != sharedkernel.TaskErrorCaseDeleted {
		t.Fatalf("pending task=%+v err=%v", t1, err)
	}
	t2, err := taskRepo.Get(ctx, "t2")
	if err != nil || t2.Status != sharedkernel.TaskRunning {
		t.Fatalf("running task must be untouched: %+v err=%v", t2, err)
	}
	sess, err := sessRepo.ListActiveByCase(ctx, 10)
	if err != nil || len(sess) != 0 {
		t.Fatalf("sessions not terminated: %+v err=%v", sess, err)
	}
	menu, err := menuRepo.GetMenu(ctx, "ch1")
	if err != nil || len(menu.Items) != 0 {
		t.Fatalf("menu items=%+v err=%v", menu.Items, err)
	}
	var kinds []string
	for _, item := range n.items {
		kinds = append(kinds, item.Kind)
	}
	if len(n.items) != 2 || !containsKind(kinds, "task_failed") || !containsKind(kinds, "session_terminated") {
		t.Fatalf("notifies=%+v", n.items)
	}
}

func containsKind(kinds []string, want string) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}
