package persistence_test

import (
	"context"
	"strings"
	"testing"

	"gorm.io/gorm"

	consoledomain "github.com/Mr9esx/Pixoma/internal/adminusers/domain"
	"github.com/Mr9esx/Pixoma/internal/adminusers/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func openConsoleDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := "console_user_" + strings.ReplaceAll(t.Name(), "/", "_")
	gdb, err := db.Open(db.Options{DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.ConsoleUserRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestConsoleUser_CreateGetAndList(t *testing.T) {
	gdb := openConsoleDB(t)
	repo := persistence.NewConsoleUserRepository(gdb)
	ctx := context.Background()

	u := &consoledomain.ConsoleUser{
		ID: "acct-1", Username: "alice", Email: "a@example.com",
		Nickname: "爱丽丝", Role: consoledomain.RoleAdmin,
		Enabled: true, PasswordHash: "hash",
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if got.Nickname != "爱丽丝" || got.Role != consoledomain.RoleAdmin || !got.Enabled {
		t.Fatalf("unexpected user: %+v", got)
	}

	_ = &consoledomain.ConsoleUser{
		ID: "acct-2", Username: "bob", Email: "b@example.com",
		Nickname: "Bob", Role: consoledomain.RoleViewer, Enabled: true, PasswordHash: "h",
	}
	if err := repo.Create(ctx, &consoledomain.ConsoleUser{
		ID: "acct-2", Username: "bob", Email: "b@example.com", Nickname: "Bob",
		Role: consoledomain.RoleViewer, Enabled: true, PasswordHash: "h",
	}); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	list, err := repo.List(ctx, consoledomain.ListQuery{Q: "ali"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != "acct-1" {
		t.Fatalf("expected 1 (alice) match, got %d: %+v", len(list), list)
	}
}

func TestConsoleUser_DuplicateUsernameRejected(t *testing.T) {
	gdb := openConsoleDB(t)
	repo := persistence.NewConsoleUserRepository(gdb)
	ctx := context.Background()
	if err := repo.Create(ctx, &consoledomain.ConsoleUser{
		ID: "a", Username: "dup", Email: "dup@example.com", Role: consoledomain.RoleViewer, Enabled: true, PasswordHash: "h",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	err := repo.Create(ctx, &consoledomain.ConsoleUser{
		ID: "b", Username: "dup", Email: "other@example.com", Role: consoledomain.RoleViewer, Enabled: true, PasswordHash: "h",
	})
	if err != consoledomain.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestConsoleUser_UpdateAndCountAdminsAndDelete(t *testing.T) {
	gdb := openConsoleDB(t)
	repo := persistence.NewConsoleUserRepository(gdb)
	ctx := context.Background()
	if err := repo.Create(ctx, &consoledomain.ConsoleUser{
		ID: "a", Username: "boss", Role: consoledomain.RoleAdmin, Enabled: true, PasswordHash: "h",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Create(ctx, &consoledomain.ConsoleUser{
		ID: "b", Username: "op", Role: consoledomain.RoleOperator, Enabled: true, PasswordHash: "h",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	n, err := repo.CountAdmins(ctx)
	if err != nil || n != 1 {
		t.Fatalf("count admins: n=%d err=%v", n, err)
	}
	if err := repo.Update(ctx, &consoledomain.ConsoleUser{
		ID: "b", Username: "op", Role: consoledomain.RoleOperator, Enabled: false, PasswordHash: "h2",
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	u, _ := repo.GetByID(ctx, "b")
	if u.Enabled || u.PasswordHash != "h2" {
		t.Fatalf("update not reflected: %+v", u)
	}
	if err := repo.Delete(ctx, "b"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = repo.GetByID(ctx, "b")
	if err != consoledomain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
