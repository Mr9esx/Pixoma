package domain_test

import (
	"testing"

	domain "github.com/Mr9esx/Pixoma/internal/menus/domain"
)

func TestCompileFoldsRootByColumns(t *testing.T) {
	tr := domain.MenuTree{
		ID: "c", Columns: 2,
		Items: []domain.TreeButton{
			{ID: "a", Label: "A", Action: domain.TreeAction{Type: "list_tasks"}},
			{ID: "b", Label: "B", Action: domain.TreeAction{Type: "list_tasks"}},
			{ID: "c1", Label: "C", Action: domain.TreeAction{Type: "list_tasks"}},
		},
	}
	got := domain.Compile(tr)
	if len(got.Rows) != 2 || len(got.Rows[0]) != 2 || len(got.Rows[1]) != 1 {
		t.Fatalf("rows=%v", got.Rows)
	}
	if got.Rows[1][0].ID != "c1" {
		t.Fatalf("last=%+v", got.Rows[1][0])
	}
}

func TestCompileIndexesOpenCardByButtonID(t *testing.T) {
	inner := domain.TreeCard{Text: "inner"}
	tr := domain.MenuTree{
		ID: "c", Columns: 1,
		Items: []domain.TreeButton{{
			ID:    "open",
			Label: "卡",
			Action: domain.TreeAction{
				Type: "open_card",
				Card: &domain.TreeCard{
					Text: "outer",
					Buttons: []domain.TreeButton{{
						ID:     "deep",
						Label:  "再开",
						Action: domain.TreeAction{Type: "open_card", Card: &inner},
					}},
				},
			},
		}},
	}
	got := domain.Compile(tr)
	if got.CardByOpenerID["open"].Text != "outer" {
		t.Fatalf("open=%+v", got.CardByOpenerID["open"])
	}
	if got.CardByOpenerID["deep"].Text != "inner" {
		t.Fatalf("deep=%+v", got.CardByOpenerID["deep"])
	}
}

func TestWalkWorkflowPlacementsCollectsPaths(t *testing.T) {
	tr := domain.MenuTree{
		ID: "c", Columns: 1,
		Items: []domain.TreeButton{
			{ID: "w1", Label: "根流", Action: domain.TreeAction{Type: "open_workflow", WorkflowID: "10"}},
			{
				ID:    "card",
				Label: "卡片",
				Action: domain.TreeAction{
					Type: "open_card",
					Card: &domain.TreeCard{
						Text: "t",
						Buttons: []domain.TreeButton{{
							ID:     "w2",
							Label:  "卡流",
							Action: domain.TreeAction{Type: "open_workflow", WorkflowID: "20"},
						}},
					},
				},
			},
		},
	}
	ps := domain.WalkWorkflowPlacements(tr)
	if len(ps) != 2 {
		t.Fatalf("n=%d %+v", len(ps), ps)
	}
	if ps[0].WorkflowID != "10" || ps[0].Kind != "keyboard" || len(ps[0].Labels) != 1 {
		t.Fatalf("p0=%+v", ps[0])
	}
	if ps[1].WorkflowID != "20" || ps[1].Kind != "card_button" {
		t.Fatalf("p1=%+v", ps[1])
	}
	if len(ps[1].Labels) != 2 || ps[1].Labels[0] != "卡片" || ps[1].Labels[1] != "卡流" {
		t.Fatalf("path=%v", ps[1].Labels)
	}
}
