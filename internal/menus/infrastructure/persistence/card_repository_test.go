package persistence_test

import (
	"context"
	"testing"

	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	mcdomain "github.com/Mr9esx/Pixoma/internal/menus/domain"
	"github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"gorm.io/gorm"
)

func TestTreeRoundTripAndStaleJSON(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:tree_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormCardRepository(gdb)
	ctx := context.Background()

	tree := mcdomain.MenuTree{
		Columns: 2,
		Items: []mcdomain.TreeButton{
			{ID: "a", Label: "图", Action: mcdomain.TreeAction{Type: "list_tasks"}},
		},
	}
	if err := repo.PutTree(ctx, "ch1", tree); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTree(ctx, "ch1")
	if err != nil || len(got.Items) != 1 || got.Items[0].ID != "a" {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	if err := gdb.Save(&persistence.MainMenuRow{
		ChannelID: "ch2",
		DocJSON:   `{"id":"m","columns":2,"items":[{"id":"x","label":"L","action":{"type":"open_card","card_id":"c1"}}]}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetTree(ctx, "ch2"); err != gorm.ErrRecordNotFound {
		t.Fatalf("stale want not found, err=%v", err)
	}
}

func TestRemoveWorkflowReferencesInTree(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:unlink_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&persistence.MainMenuRow{}, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormCardRepository(gdb)
	ctx := context.Background()

	tree := mcdomain.MenuTree{
		Columns: 2,
		Items: []mcdomain.TreeButton{
			{ID: "mi-keep", Label: "保留", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "2"}},
			{ID: "mi-drop", Label: "删除", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "1"}},
			{ID: "mi-other", Label: "无关", Action: mcdomain.TreeAction{Type: "send_text", Text: "ok"}},
			{
				ID: "mi-card", Label: "卡",
				Action: mcdomain.TreeAction{
					Type: "open_card",
					Card: &mcdomain.TreeCard{
						Text: "hi",
						Buttons: []mcdomain.TreeButton{
							{ID: "b1", Label: "B", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "1"}},
							{ID: "b2", Label: "B2", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "2"}},
						},
					},
				},
			},
		},
	}
	if err := repo.PutTree(ctx, "ch1", tree); err != nil {
		t.Fatal(err)
	}
	removed, err := repo.RemoveWorkflowReferences(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed=%+v", removed)
	}
	got, err := repo.GetTree(ctx, "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("items=%+v", got.Items)
	}
	card := got.Items[2].Action.Card
	if card == nil || len(card.Buttons) != 1 || card.Buttons[0].ID != "b2" {
		t.Fatalf("card=%+v", card)
	}
}