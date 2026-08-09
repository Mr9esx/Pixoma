package persistence_test

import (
	"context"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/infrastructure/persistence"
)

func openRepo(t *testing.T) *persistence.GormRepository {
	t.Helper()
	dsn := "file:tgmenu_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.MenuRow{}); err != nil {
		t.Fatal(err)
	}
	return persistence.NewGormRepository(gdb)
}

func TestEnsureDefaultAndReplaceRoundTrip(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	doc, err := repo.EnsureDefault(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Items) != 6 {
		t.Fatalf("seed items=%d", len(doc.Items))
	}
	again, err := repo.EnsureDefault(ctx)
	if err != nil || again.Items[0].Label != doc.Items[0].Label {
		t.Fatalf("second ensure mutated: %v %+v", err, again)
	}
	doc.Items[0].Label = "🖼 图生图"
	doc.Items[0].Action = domain.ActionReplyMedia
	doc.Items[0].Tag = ""
	doc.Items[0].Reply = &domain.ReplyPayload{
		Text:   "hello",
		Images: []string{"https://example.com/a.png"},
	}
	if err := repo.Replace(ctx, doc); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, domain.DocumentIDDefault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0].Label != "🖼 图生图" || got.Items[0].Reply == nil || len(got.Items[0].Reply.Images) != 1 {
		t.Fatalf("got=%+v", got.Items[0])
	}
}
