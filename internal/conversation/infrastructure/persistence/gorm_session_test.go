package persistence_test

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openShared(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.SessionRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestGormSession_ActiveSurviveReopenAndSubmittedKept(t *testing.T) {
	dsn := "file:sess_test?mode=memory&cache=shared"
	gdb := openShared(t, dsn)
	repo := persistence.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	s := domain.NewCollecting("s1", 100, "case-1", []string{"prompt"}, now)
	s.UserID = "user-1"
	if err := repo.Save(ctx, s); err != nil {
		t.Fatal(err)
	}

	gdb2 := openShared(t, dsn)
	repo2 := persistence.NewSessionRepository(gdb2)
	got, err := repo2.GetActiveByChat(ctx, 100)
	if err != nil || got.ID != "s1" || got.UserID != "user-1" {
		t.Fatalf("active restore: %+v %v", got, err)
	}

	got.Status = domain.StatusSubmitted
	_ = repo2.Save(ctx, got)
	if _, err := repo2.GetActiveByChat(ctx, 100); err != domain.ErrNoActiveSession {
		t.Fatalf("expected no active, got %v", err)
	}
	byID, err := repo2.GetByID(ctx, "s1")
	if err != nil || byID.Status != domain.StatusSubmitted {
		t.Fatalf("submitted row missing: %+v %v", byID, err)
	}
}
