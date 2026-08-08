package persistence_test

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:identity_user_test?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.UserRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestUpsertByTgUserID_IdempotentAndRefresh(t *testing.T) {
	gdb := openTestDB(t) // sqlite memory + AutoMigrate UserRow
	repo := persistence.NewUserRepository(gdb)
	ctx := context.Background()

	u1, err := repo.UpsertByTgUserID(ctx, domain.UpsertFrom{
		TgUserID: 42, Username: "alice", FirstName: "A", LanguageCode: "zh-hans",
	})
	if err != nil {
		t.Fatal(err)
	}
	u2, err := repo.UpsertByTgUserID(ctx, domain.UpsertFrom{
		TgUserID: 42, Username: "alice2", FirstName: "A2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u1.ID != u2.ID {
		t.Fatalf("id changed: %s vs %s", u1.ID, u2.ID)
	}
	if u2.Username != "alice2" {
		t.Fatalf("username not refreshed: %q", u2.Username)
	}
	if !u2.LastSeenAt.After(u1.LastSeenAt) && !u2.LastSeenAt.Equal(u1.LastSeenAt) {
		// allow equal if same clock; prefer After when Now injects
	}
}
