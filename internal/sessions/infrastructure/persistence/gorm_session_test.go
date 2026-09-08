package persistence_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
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

	s := domain.NewCollecting("s1", "tg:100", 1, []string{"prompt"}, now)
	s.UserID = "user-1"
	if err := repo.Save(ctx, s); err != nil {
		t.Fatal(err)
	}

	gdb2 := openShared(t, dsn)
	repo2 := persistence.NewSessionRepository(gdb2)
	got, err := repo2.GetActiveByChat(ctx, "tg:100")
	if err != nil || got.ID != "s1" || got.UserID != "user-1" {
		t.Fatalf("active restore: %+v %v", got, err)
	}

	got.Status = domain.StatusSubmitted
	_ = repo2.Save(ctx, got)
	if _, err := repo2.GetActiveByChat(ctx, "tg:100"); err != domain.ErrNoActiveSession {
		t.Fatalf("expected no active, got %v", err)
	}
	byID, err := repo2.GetByID(ctx, "s1")
	if err != nil || byID.Status != domain.StatusSubmitted {
		t.Fatalf("submitted row missing: %+v %v", byID, err)
	}
}

func TestSessionListFilters(t *testing.T) {
	dsn := "file:sess_list_test?mode=memory&cache=shared"
	gdb := openShared(t, dsn)
	repo := persistence.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	s1 := domain.NewCollecting("sess-alice-1", "tg:100", 3, []string{"prompt"}, now)
	s1.UserID = "user-alice"
	if err := repo.Save(ctx, s1); err != nil {
		t.Fatal(err)
	}

	s2 := domain.NewCollecting("sess-bob-2", "tg:200", 4, []string{"prompt"}, now.Add(time.Second))
	s2.UserID = "user-bob"
	s2.Status = domain.StatusSubmitted
	s2.UpdatedAt = now.Add(2 * time.Second)
	if err := repo.Save(ctx, s2); err != nil {
		t.Fatal(err)
	}

	byUser, err := repo.List(ctx, domain.ListQuery{UserID: "user-alice"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byUser) != 1 || byUser[0].ID != "sess-alice-1" {
		t.Fatalf("user_id: want 1 alice, got %+v", byUser)
	}

	byStatus, err := repo.List(ctx, domain.ListQuery{Status: domain.StatusSubmitted})
	if err != nil {
		t.Fatal(err)
	}
	if len(byStatus) != 1 || byStatus[0].ID != "sess-bob-2" {
		t.Fatalf("status=submitted: want 1 bob, got %+v", byStatus)
	}

	chatID := sharedkernel.ChatID("tg:100")
	byChat, err := repo.List(ctx, domain.ListQuery{ChatID: &chatID})
	if err != nil {
		t.Fatal(err)
	}
	if len(byChat) != 1 || byChat[0].ID != "sess-alice-1" {
		t.Fatalf("chat_id: want 1 alice, got %+v", byChat)
	}

	byQCase, err := repo.List(ctx, domain.ListQuery{Q: "3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byQCase) != 1 || byQCase[0].ID != "sess-alice-1" {
		t.Fatalf("q case_id: want 1 alice, got %+v", byQCase)
	}

	byQID, err := repo.List(ctx, domain.ListQuery{Q: "sess-bob"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byQID) != 1 || byQID[0].ID != "sess-bob-2" {
		t.Fatalf("q id prefix: want 1 bob, got %+v", byQID)
	}
}

func TestSessionListActiveByCase(t *testing.T) {
	gdb := openShared(t, "file:sess_active_"+t.Name()+"?mode=memory&cache=shared")
	repo := persistence.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := repo.Save(ctx, domain.NewCollecting("s1", "tg:1", 10, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	s2 := domain.NewCollecting("s2", "tg:2", 10, []string{"a"}, now)
	s2.Status = domain.StatusSubmitted
	if err := repo.Save(ctx, s2); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, domain.NewCollecting("s3", "tg:3", 20, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListActiveByCase(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "s1" {
		t.Fatalf("ListActiveByCase(10)=%+v", got)
	}
}

func TestAutoMigrate_UpgradesLegacySessionsTable(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:sess_migrate_test?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	// 旧 schema：chat_id int 列 + 已有数据
	if err := gdb.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		chat_id INTEGER NOT NULL,
		case_id TEXT NOT NULL,
		status TEXT NOT NULL,
		current_input_index INTEGER NOT NULL DEFAULT 0,
		input_keys_json TEXT NOT NULL,
		draft_json TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := gdb.Exec(`INSERT INTO sessions (id, user_id, chat_id, case_id, status, input_keys_json, draft_json, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		"s-old-1", "u1", 100, "c1", "submitted", "[]", "{}", now, now,
	).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec(`INSERT INTO sessions (id, user_id, chat_id, case_id, status, input_keys_json, draft_json, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		"s-old-2", "u1", 100, "c1", "submitted", "[]", "{}", now, now,
	).Error; err != nil {
		t.Fatal(err)
	}

	// 新 schema AutoMigrate 必须成功（NOT NULL 新列带默认值；同 chat 多条 submitted 不冲突）
	if err := db.AutoMigrate(gdb, &persistence.SessionRow{}); err != nil {
		t.Fatalf("migrate legacy sessions: %v", err)
	}
	var count int64
	if err := gdb.Model(&persistence.SessionRow{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("rows=%d", count)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	dsn := "file:sess_notfound_test?mode=memory&cache=shared"
	gdb := openShared(t, dsn)
	repo := persistence.NewSessionRepository(gdb)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "missing-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
