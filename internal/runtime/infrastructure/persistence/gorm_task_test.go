package persistence_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	convpersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:runtime_task_test?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &convpersist.SessionRow{}, &persistence.TaskRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func seedSession(t *testing.T, gdb *gorm.DB, id string, chatID int64) {
	t.Helper()
	now := time.Now().UTC()
	row := convpersist.SessionRow{
		ID:            id,
		UserID:        "user-1",
		ChatID:        chatID,
		CaseID:        "c1",
		Status:        "submitted",
		InputKeysJSON: "[]",
		DraftJSON:     "{}",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := gdb.Create(&row).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func TestGormTask_SessionIDAndListByInstance(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s1", 9)

	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	pending := domain.NewPending("t-pending", "s1", "c1", "inputs/t-pending", now)
	if err := tasks.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}

	queued := domain.NewPending("t-q", "s1", "c1", "inputs/t-q", now)
	if err := queued.MarkQueued("gpu-1", now); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, queued); err != nil {
		t.Fatal(err)
	}

	list, err := tasks.ListByInstance(ctx, "gpu-1", domain.ListByInstanceQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "t-q" {
		t.Fatalf("got %+v", list)
	}
	got, err := tasks.Get(ctx, "t-pending")
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionID != "s1" {
		t.Fatalf("session_id=%q", got.SessionID)
	}
}

func TestGormTask_ClaimQueuedCAS(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-claim", 7)
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := tasks.Create(ctx, domain.NewPending("t-claim", "s-claim", "c1", "inputs/t-claim", now)); err != nil {
		t.Fatal(err)
	}

	ok, err := tasks.ClaimQueued(ctx, "t-claim", "gpu-1", now)
	if err != nil || !ok {
		t.Fatalf("first claim ok=%v err=%v", ok, err)
	}
	ok2, err := tasks.ClaimQueued(ctx, "t-claim", "gpu-2", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		t.Fatal("second claim must fail")
	}
	got, err := tasks.Get(ctx, "t-claim")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != sharedkernel.TaskQueued || got.InstanceID != "gpu-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestGormTask_ListByChatJoinsSession(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-chat", 42)
	seedSession(t, gdb, "s-other", 99)

	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	a := domain.NewPending("t-a", "s-chat", "c1", "inputs/t-a", now)
	b := domain.NewPending("t-b", "s-other", "c1", "inputs/t-b", now)
	_ = tasks.Create(ctx, a)
	_ = tasks.Create(ctx, b)

	list, err := tasks.ListByChat(ctx, sharedkernel.ChatID(42), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "t-a" {
		t.Fatalf("ListByChat got %+v", list)
	}
	if list[0].SessionID != "s-chat" {
		t.Fatalf("session_id=%q", list[0].SessionID)
	}
}
