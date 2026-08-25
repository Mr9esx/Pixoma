package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestMigrateLegacyTasks(t *testing.T) {
	dsn := "file:migrate_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.TaskRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewTaskRepository(gdb)
	ctx := context.Background()
	now := time.Unix(500, 0).UTC()

	// queued with stale lease → migrates to default, holder cleared
	stale := domain.NewPending("t-stale", "s1", sharedkernel.CaseID(1), "in", now)
	_ = stale.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j1"}, now.Add(-time.Minute))
	stale.LeaseUntil = now.Add(-time.Second)
	_ = repo.Create(ctx, stale)

	// queued with valid lease → untouched
	fresh := domain.NewPending("t-fresh", "s2", sharedkernel.CaseID(1), "in", now)
	_ = fresh.PrepareForClaim("gpu-2", sharedkernel.BlobRef{Key: "j2"}, now.Add(-time.Minute))
	fresh.LeaseUntil = now.Add(time.Hour)
	_ = repo.Create(ctx, fresh)

	// running with expired lease → untouched by legacy migration
	running := domain.NewPending("t-run", "s3", sharedkernel.CaseID(1), "in", now)
	_ = running.PrepareForClaim("gpu-3", sharedkernel.BlobRef{Key: "j3"}, now.Add(-time.Minute))
	_ = running.ClaimWithLease("gpu-3", time.Minute, now.Add(-2*time.Minute))
	_ = repo.Create(ctx, running)

	n, err := persistence.MigrateLegacyTasks(ctx, gdb, now)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 1 {
		t.Fatalf("migrated %d rows, want 1", n)
	}

	got, _ := repo.Get(ctx, "t-stale")
	if got.DispatchTopic != "default" || got.EdgeID != "" || got.Status != sharedkernel.TaskQueued {
		t.Fatalf("stale not migrated: %+v", got)
	}
	fresh2, _ := repo.Get(ctx, "t-fresh")
	if fresh2.EdgeID != "gpu-2" || fresh2.DispatchTopic != "" {
		t.Fatalf("fresh task touched: %+v", fresh2)
	}
	run2, _ := repo.Get(ctx, "t-run")
	if run2.Status != sharedkernel.TaskRunning {
		t.Fatalf("running task touched: %+v", run2)
	}

	var raw struct {
		LeaseUntil *time.Time
		RequeueAt  *time.Time
	}
	if err := gdb.Model(&persistence.TaskRow{}).Where("id = ?", "t-stale").Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw.LeaseUntil != nil || raw.RequeueAt != nil {
		t.Fatalf("want NULL lease/requeue after migrate, got %v / %v", raw.LeaseUntil, raw.RequeueAt)
	}
}
