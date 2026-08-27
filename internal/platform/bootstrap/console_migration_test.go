package bootstrap_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/consoleuser/domain"
	consolepersist "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openConsoleGDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := "console_mig_" + strings.ReplaceAll(t.Name(), "/", "_")
	gdb, err := db.Open(db.Options{DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &consolepersist.ConsoleUserRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestMigrateBootstrapAdmin_SeedsAdmin(t *testing.T) {
	st, creds, err := bootstrap.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	gdb := openConsoleGDB(t)
	ctx := context.Background()

	if err := bootstrap.MigrateBootstrapAdmin(ctx, gdb, st); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := consolepersist.NewConsoleUserRepository(gdb)
	admin, err := repo.GetByUsername(ctx, creds.Username)
	if err != nil {
		t.Fatalf("get migrated admin: %v", err)
	}
	if admin.Role != domain.RoleAdmin || !admin.Enabled {
		t.Fatalf("admin role/enabled wrong: %+v", admin)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(creds.Password)); err != nil {
		t.Fatalf("password hash not preserved: %v", err)
	}
	if !admin.MustChangePassword {
		t.Fatalf("expected must_change_password propagated")
	}
	// Idempotent.
	if err := bootstrap.MigrateBootstrapAdmin(ctx, gdb, st); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	n, err := repo.CountAdmins(ctx)
	if err != nil || n != 1 {
		t.Fatalf("expected exactly 1 admin after idempotent migrate, got %d err=%v", n, err)
	}
}
