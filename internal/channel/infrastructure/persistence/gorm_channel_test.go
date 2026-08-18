package persistence

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:ch_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	return gdb
}

func TestChannelCRUD(t *testing.T) {
	repo := NewGormRepository(openTestDB(t))
	ctx := context.Background()

	ch := domain.Channel{
		ID: "tg-default", Platform: string(domain.PlatformTelegram),
		Name: "主机器人", CredentialCiphertext: "cipher", Enabled: true,
	}
	if err := repo.Create(ctx, ch); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, "tg-default")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "主机器人" || got.CredentialCiphertext != "cipher" || !got.Enabled {
		t.Fatalf("get: %+v", got)
	}

	got.Name = "改名字"
	got.CredentialCiphertext = "cipher2"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	again, _ := repo.Get(ctx, "tg-default")
	if again.Name != "改名字" || again.CredentialCiphertext != "cipher2" {
		t.Fatalf("update: %+v", again)
	}

	list, err := repo.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}

	if err := repo.Delete(ctx, "tg-default"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, "tg-default"); err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestChannelGetNotFound(t *testing.T) {
	repo := NewGormRepository(openTestDB(t))
	if _, err := repo.Get(context.Background(), "nope"); err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
