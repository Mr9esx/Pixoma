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
	dsn := "file:runtime_task_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &convpersist.SessionRow{}, &persistence.TaskRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func seedSession(t *testing.T, gdb *gorm.DB, id string, chatID sharedkernel.ChatID) {
	t.Helper()
	now := time.Now().UTC()
	addr, err := sharedkernel.ParseChatID(string(chatID))
	if err != nil {
		t.Fatalf("chat id: %v", err)
	}
	row := convpersist.SessionRow{
		ID:             id,
		UserID:         "user-1",
		ChannelID:      addr.ChannelID,
		ChatExternalID: addr.ExternalChatID,
		CaseID:         1,
		Status:         "submitted",
		InputKeysJSON:  "[]",
		DraftJSON:      "{}",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := gdb.Create(&row).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func TestGormTask_SessionIDAndListByInstance(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s1", sharedkernel.ChatID("tg:9"))

	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	pending := domain.NewPending("t-pending", "s1", sharedkernel.CaseID(1), "inputs/t-pending", now)
	if err := tasks.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}

	queued := domain.NewPending("t-q", "s1", sharedkernel.CaseID(1), "inputs/t-q", now)
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
	seedSession(t, gdb, "s-claim", sharedkernel.ChatID("tg:7"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := tasks.Create(ctx, domain.NewPending("t-claim", "s-claim", sharedkernel.CaseID(1), "inputs/t-claim", now)); err != nil {
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
	if got.Status != sharedkernel.TaskQueued || got.EdgeID != "gpu-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestGormTask_ListByChatJoinsSession(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-chat", sharedkernel.ChatID("tg:42"))
	seedSession(t, gdb, "s-other", sharedkernel.ChatID("tg:99"))

	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	a := domain.NewPending("t-a", "s-chat", sharedkernel.CaseID(1), "inputs/t-a", now)
	b := domain.NewPending("t-b", "s-other", sharedkernel.CaseID(1), "inputs/t-b", now)
	_ = tasks.Create(ctx, a)
	_ = tasks.Create(ctx, b)

	list, err := tasks.ListByChat(ctx, sharedkernel.ChatID("tg:42"), 10)
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

func TestTaskAdminListFilters(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:runtime_task_admin_list?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &convpersist.SessionRow{}, &persistence.TaskRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	seedSession(t, gdb, "s-admin-42", sharedkernel.ChatID("tg:42"))
	seedSession(t, gdb, "s-admin-99", sharedkernel.ChatID("tg:99"))

	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	pendingA := domain.NewPending("task-admin-aaa", "s-admin-42", sharedkernel.CaseID(1), "inputs/a", now)
	pendingB := domain.NewPending("task-admin-bbb", "s-admin-99", sharedkernel.CaseID(2), "inputs/b", now.Add(time.Second))
	queued := domain.NewPending("task-other-ccc", "s-admin-42", sharedkernel.CaseID(1), "inputs/c", now.Add(2*time.Second))
	if err := queued.MarkQueued("gpu-1", now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	for _, tsk := range []*domain.Task{pendingA, pendingB, queued} {
		if err := tasks.Create(ctx, tsk); err != nil {
			t.Fatal(err)
		}
	}

	bySession, err := tasks.List(ctx, domain.AdminListQuery{SessionID: "s-admin-42"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bySession) != 2 {
		t.Fatalf("SessionID filter got %d want 2: %+v", len(bySession), idsOf(bySession))
	}

	byChat, err := tasks.List(ctx, domain.AdminListQuery{ChatID: "tg:42"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byChat) != 2 {
		t.Fatalf("ChatID JOIN filter got %d want 2: %+v", len(byChat), idsOf(byChat))
	}
	for _, tsk := range byChat {
		if tsk.SessionID != "s-admin-42" {
			t.Fatalf("ChatID filter leaked session %q", tsk.SessionID)
		}
	}

	byStatusCase, err := tasks.List(ctx, domain.AdminListQuery{
		Status: sharedkernel.TaskPending,
		CaseID: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byStatusCase) != 1 || byStatusCase[0].ID != "task-admin-aaa" {
		t.Fatalf("Status+CaseID got %+v", idsOf(byStatusCase))
	}

	byQ, err := tasks.List(ctx, domain.AdminListQuery{Q: "task-admin-"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byQ) != 2 {
		t.Fatalf("Q prefix got %d want 2: %+v", len(byQ), idsOf(byQ))
	}
}

func idsOf(list []*domain.Task) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		out = append(out, string(t.ID))
	}
	return out
}

func TestGormTask_JobRefAndLeaseRoundTrip(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-lease", sharedkernel.ChatID("tg:3"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(200, 0).UTC()

	task := domain.NewPending("t-lease", "s-lease", sharedkernel.CaseID(1), "inputs/t-lease", now)
	ref := sharedkernel.BlobRef{Key: "jobs/t-lease/job.json", MIME: "application/json"}
	if err := task.PrepareForClaim("gpu-1", ref, now); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	got, err := tasks.Get(ctx, "t-lease")
	if err != nil {
		t.Fatal(err)
	}
	if got.JobRef.Key != ref.Key || got.JobRef.MIME != ref.MIME {
		t.Fatalf("job_ref=%+v", got.JobRef)
	}
	if got.Status != sharedkernel.TaskQueued || got.EdgeID != "gpu-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestGormTask_PrepareForClaimCAS(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-prep", sharedkernel.ChatID("tg:4"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(200, 0).UTC()
	if err := tasks.Create(ctx, domain.NewPending("t-prep", "s-prep", sharedkernel.CaseID(1), "inputs/t-prep", now)); err != nil {
		t.Fatal(err)
	}
	ref := sharedkernel.BlobRef{Key: "jobs/t-prep/job.json"}
	ok, err := tasks.PrepareForClaim(ctx, "t-prep", "fast-gpu", ref, now)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	ok2, err := tasks.PrepareForClaim(ctx, "t-prep", "default", sharedkernel.BlobRef{Key: "other"}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		t.Fatal("second prepare must fail")
	}
	got, _ := tasks.Get(ctx, "t-prep")
	if got.DispatchTopic != "fast-gpu" || got.EdgeID != "" || got.JobRef.Key != ref.Key {
		t.Fatalf("got %+v", got)
	}
}

func TestGormTask_ClaimNextWithLeaseAndExpire(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-claim2", sharedkernel.ChatID("tg:5"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(300, 0).UTC()
	ref := sharedkernel.BlobRef{Key: "jobs/t-c2/job.json"}
	task := domain.NewPending("t-c2", "s-claim2", sharedkernel.CaseID(1), "inputs/t-c2", now)
	_ = task.PrepareForTopic("default", ref, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	claimed, err := tasks.ClaimNextWithLease(ctx, "gpu-1", []string{"default"}, 90*time.Second, now)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil || claimed.ID != "t-c2" || claimed.Status != sharedkernel.TaskRunning {
		t.Fatalf("claimed=%+v", claimed)
	}
	if claimed.LeaseUntil.Sub(now) != 90*time.Second {
		t.Fatalf("lease=%v", claimed.LeaseUntil)
	}

	second, err := tasks.ClaimNextWithLease(ctx, "gpu-1", []string{"default"}, time.Minute, now)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		t.Fatalf("expected nil, got %+v", second)
	}

	later := now.Add(2 * time.Minute)
	n, err := tasks.RequeueExpiredLeases(ctx, later)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("requeued=%d", n)
	}
	again, err := tasks.ClaimNextWithLease(ctx, "gpu-1", []string{"default"}, time.Minute, later)
	if err != nil || again == nil || again.ID != "t-c2" {
		t.Fatalf("reclaim=%+v err=%v", again, err)
	}
}

func TestGormTask_EmptyDispatchTopicDoesNotMatchDefault(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(800, 0).UTC()

	seed := func(t *testing.T) domain.TaskRepository {
		t.Helper()
		gdb := openTestDB(t)
		seedSession(t, gdb, "s-empty", sharedkernel.ChatID("tg:13"))
		tasks := persistence.NewTaskRepository(gdb)
		empty := domain.NewPending("empty-1", "s-empty", sharedkernel.CaseID(1), "inputs/empty-1", now)
		_ = empty.PrepareForTopic("default", sharedkernel.BlobRef{Key: "j/empty-1"}, now)
		empty.DispatchTopic = ""
		empty.CreatedAt = now
		if err := tasks.Create(ctx, empty); err != nil {
			t.Fatal(err)
		}
		def := domain.NewPending("default-1", "s-empty", sharedkernel.CaseID(1), "inputs/default-1", now)
		_ = def.PrepareForTopic("default", sharedkernel.BlobRef{Key: "j/default-1"}, now)
		def.CreatedAt = now.Add(time.Second)
		if err := tasks.Create(ctx, def); err != nil {
			t.Fatal(err)
		}
		return tasks
	}

	t.Run("claim default skips empty topic", func(t *testing.T) {
		tasks := seed(t)
		claimed, err := tasks.ClaimNextWithLease(ctx, "gpu-1", []string{"default"}, time.Minute, now)
		if err != nil {
			t.Fatal(err)
		}
		if claimed == nil || claimed.ID != "default-1" {
			t.Fatalf("claimed=%+v, want default-1 (empty topic must stay unclaimed)", claimed)
		}
		leftover, err := tasks.Get(ctx, "empty-1")
		if err != nil {
			t.Fatal(err)
		}
		if leftover.Status != sharedkernel.TaskQueued || leftover.DispatchTopic != "" {
			t.Fatalf("empty row status=%s topic=%q, want queued with empty topic", leftover.Status, leftover.DispatchTopic)
		}
	})

	t.Run("list default excludes empty topic", func(t *testing.T) {
		tasks := seed(t)
		list, err := tasks.ListByTopic(ctx, "default", domain.ListByTopicQuery{})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].ID != "default-1" {
			t.Fatalf("list = %v, want only default-1", idsOf(list))
		}
	})
}

func TestGormTask_HeartbeatLease(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-hb", sharedkernel.ChatID("tg:6"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(400, 0).UTC()
	task := domain.NewPending("t-hb", "s-hb", sharedkernel.CaseID(1), "inputs/t-hb", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	at := now.Add(30 * time.Second)
	ok, err := tasks.HeartbeatLease(ctx, "t-hb", "gpu-1", 90*time.Second, at)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	got, _ := tasks.Get(ctx, "t-hb")
	if got.LeaseUntil.Sub(at) != 90*time.Second {
		t.Fatalf("lease=%v", got.LeaseUntil)
	}
	okBad, err := tasks.HeartbeatLease(ctx, "t-hb", "gpu-2", time.Minute, at)
	if err != nil {
		t.Fatal(err)
	}
	if okBad {
		t.Fatal("wrong instance must not heartbeat")
	}
}

func TestGormTask_NullTimestampsAfterPrepareForClaim(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-null", sharedkernel.ChatID("tg:11"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(600, 0).UTC()
	if err := tasks.Create(ctx, domain.NewPending("t-null", "s-null", sharedkernel.CaseID(1), "inputs/t-null", now)); err != nil {
		t.Fatal(err)
	}
	ref := sharedkernel.BlobRef{Key: "jobs/t-null/job.json"}
	ok, err := tasks.PrepareForClaim(ctx, "t-null", "default", ref, now)
	if err != nil || !ok {
		t.Fatalf("prepare ok=%v err=%v", ok, err)
	}
	var raw struct {
		LeaseUntil *time.Time
		RequeueAt  *time.Time
	}
	if err := gdb.Model(&persistence.TaskRow{}).Where("id = ?", "t-null").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw.LeaseUntil != nil || raw.RequeueAt != nil {
		t.Fatalf("want NULL lease/requeue after prepare, got %v / %v", raw.LeaseUntil, raw.RequeueAt)
	}
}

func TestGormTask_RequeueClearsLeaseToNULL(t *testing.T) {
	gdb := openTestDB(t)
	seedSession(t, gdb, "s-req", sharedkernel.ChatID("tg:12"))
	tasks := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(700, 0).UTC()
	ref := sharedkernel.BlobRef{Key: "jobs/t-req/job.json"}
	task := domain.NewPending("t-req", "s-req", sharedkernel.CaseID(1), "inputs/t-req", now)
	_ = task.PrepareForClaim("gpu-1", ref, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	later := now.Add(2 * time.Minute)
	n, err := tasks.RequeueExpiredLeases(ctx, later)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("requeued=%d want 1", n)
	}
	var raw struct {
		Status     string
		LeaseUntil *time.Time
	}
	if err := gdb.Model(&persistence.TaskRow{}).Where("id = ?", "t-req").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw.Status != string(sharedkernel.TaskQueued) || raw.LeaseUntil != nil {
		t.Fatalf("want queued with NULL lease, got %+v", raw)
	}
}
