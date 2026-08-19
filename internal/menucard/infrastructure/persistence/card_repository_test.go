package persistence_test

import (
	"context"
	"testing"

	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestCardRepositoryMenuAndCardsRoundTrip(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:cards_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &persistence.CardRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormCardRepository(gdb)
	ctx := context.Background()

	if err := repo.PutMenu(ctx, "ch1", mcdomain.Menu{ID: "m", Name: "主", Columns: 2}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetMenu(ctx, "ch1")
	if err != nil || got.Name != "主" {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	card := mcdomain.Card{ID: "c1", Name: "开始生成", Text: "hi", Buttons: []mcdomain.CardButton{
		{ID: "b", Label: "B", Action: mcdomain.Action{Type: "placeholder"}},
	}}
	if err := repo.CreateCard(ctx, "ch1", card); err != nil {
		t.Fatal(err)
	}
	refs, err := repo.CardReferences(ctx, "ch1", "c1")
	if err != nil || len(refs) != 0 {
		t.Fatalf("refs=%v err=%v", refs, err)
	}

	// 菜单项引用 c1 → references 非空
	menuWithRef := mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_card", CardID: "c1"}},
	}}
	if err := repo.PutMenu(ctx, "ch1", menuWithRef); err != nil {
		t.Fatal(err)
	}
	refs, err = repo.CardReferences(ctx, "ch1", "c1")
	if err != nil || len(refs) != 1 || refs[0] != "menu:mi" {
		t.Fatalf("refs=%v err=%v", refs, err)
	}
}
