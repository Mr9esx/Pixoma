package bootstrap

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	consoledomain "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/domain"
	consolepersist "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/persistence"
)

// MigrateBootstrapAdmin seeds the console-users table with the bootstrap admin
// account (role Admin) if it is not already present. Idempotent and safe to run
// on any upgrade; the bootstrap credentials are preserved verbatim.
func MigrateBootstrapAdmin(ctx context.Context, gdb *gorm.DB, st *Store) error {
	if gdb == nil || st == nil {
		return nil
	}
	username, hash, mustChange := st.AdminAccount()
	if username == "" || hash == "" {
		return nil
	}
	nickname, email, avatarURL := st.AdminProfile()
	repo := consolepersist.NewConsoleUserRepository(gdb)
	if _, err := repo.GetByUsername(ctx, username); err == nil {
		return nil // already migrated
	} else if !errors.Is(err, consoledomain.ErrNotFound) {
		return err
	}
	return repo.Create(ctx, &consoledomain.ConsoleUser{
		ID:                 uuid.NewString(),
		Username:           username,
		Nickname:           nickname,
		Email:              email,
		AvatarURL:          avatarURL,
		Role:               consoledomain.RoleAdmin,
		Enabled:            true,
		MustChangePassword: mustChange,
		PasswordHash:       hash,
	})
}
