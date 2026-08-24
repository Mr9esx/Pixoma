package persistence_test

import (
	"context"
	"testing"

	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestCardRepositoryMenuAndCardsRoundTrip(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:cards_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &persistence.CardRow{}, &channelpersist.ChannelRow{}); err != nil {
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

func TestRemoveWorkflowReferences(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:unlink_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &persistence.CardRow{}, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormCardRepository(gdb)
	ctx := context.Background()

	menu := mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi-keep", Label: "保留", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1", "2"}}},
		{ID: "mi-drop", Label: "删除", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1"}}},
		{ID: "mi-other", Label: "无关", Action: mcdomain.Action{Type: "placeholder"}},
	}}
	if err := repo.PutMenu(ctx, "ch1", menu); err != nil {
		t.Fatal(err)
	}
	card := mcdomain.Card{ID: "c1", Name: "卡", Text: "hi", Buttons: []mcdomain.CardButton{
		{ID: "b1", Label: "B", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1", "3"}}},
		{ID: "b2", Label: "B2", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"2"}}},
		{ID: "b3", Label: "B3", Action: mcdomain.Action{Type: "open_workflow", WorkflowIDs: []string{"1"}}},
	}}
	if err := repo.CreateCard(ctx, "ch1", card); err != nil {
		t.Fatal(err)
	}

	removed, err := repo.RemoveWorkflowReferences(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 4 {
		t.Fatalf("removed=%+v", removed)
	}

	gotMenu, err := repo.GetMenu(ctx, "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotMenu.Items) != 2 {
		t.Fatalf("menu items=%+v", gotMenu.Items)
	}
	if len(gotMenu.Items[0].Action.WorkflowIDs) != 1 || gotMenu.Items[0].Action.WorkflowIDs[0] != "2" {
		t.Fatalf("mi-keep workflow_ids=%v", gotMenu.Items[0].Action.WorkflowIDs)
	}
	if gotMenu.Items[1].ID != "mi-other" {
		t.Fatalf("second item=%+v", gotMenu.Items[1])
	}

	cards, err := repo.ListCards(ctx, "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || len(cards[0].Buttons) != 2 {
		t.Fatalf("cards=%+v", cards)
	}
	if len(cards[0].Buttons[0].Action.WorkflowIDs) != 1 || cards[0].Buttons[0].Action.WorkflowIDs[0] != "3" {
		t.Fatalf("b1 workflow_ids=%v", cards[0].Buttons[0].Action.WorkflowIDs)
	}
	if cards[0].Buttons[1].ID != "b2" {
		t.Fatalf("second button=%+v", cards[0].Buttons[1])
	}
}
