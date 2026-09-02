package domain_test

import (
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
)

func allowAll(_ string) bool { return true }

func validTree() domain.MenuTree {
	return domain.MenuTree{
		ID:      "ch1",
		Columns: 2,
		Items: []domain.TreeButton{
			{
				ID:    "kbd-1",
				Label: "图片",
				Action: domain.TreeAction{
					Type:       "open_workflow",
					WorkflowID: "10",
				},
			},
		},
	}
}

func TestValidateTreeAcceptsMinimalKeyboard(t *testing.T) {
	if err := domain.ValidateTree(validTree(), allowAll); err != nil {
		t.Fatalf("valid tree: %v", err)
	}
}

func TestValidateTreeRejectsEmptyRoot(t *testing.T) {
	tr := validTree()
	tr.Items = nil
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTreeRejectsSevenRootButtons(t *testing.T) {
	tr := validTree()
	tr.Items = make([]domain.TreeButton, 7)
	for i := range tr.Items {
		tr.Items[i] = domain.TreeButton{
			ID:    "kbd-" + string(rune('a'+i)),
			Label: "x",
			Action: domain.TreeAction{
				Type:       "open_workflow",
				WorkflowID: "10",
			},
		}
	}
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTreeRejectsBadColumns(t *testing.T) {
	tr := validTree()
	tr.Columns = 0
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("columns=0: %v", err)
	}
	tr.Columns = 7
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("columns=7: %v", err)
	}
}

func TestValidateTreeRejectsOpenCardWithoutCard(t *testing.T) {
	tr := validTree()
	tr.Items[0].Action = domain.TreeAction{Type: "open_card"}
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTreeRejectsCardDepthNine(t *testing.T) {
	leaf := &domain.TreeCard{Text: "n9"}
	card := leaf
	for i := 8; i >= 1; i-- {
		card = &domain.TreeCard{
			Text: "n",
			Buttons: []domain.TreeButton{{
				ID:    "d-" + string(rune('0'+i)),
				Label: "next",
				Action: domain.TreeAction{
					Type: "open_card",
					Card: card,
				},
			}},
		}
	}
	tr := validTree()
	tr.Items[0] = domain.TreeButton{
		ID:    "root-card",
		Label: "open",
		Action: domain.TreeAction{
			Type: "open_card",
			Card: card,
		},
	}
	// 9 open_card edges: root + 8 nested
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("depth 9: %v", err)
	}
}

func TestValidateTreeRejectsDuplicateIDs(t *testing.T) {
	tr := validTree()
	tr.Items = []domain.TreeButton{
		{ID: "same", Label: "a", Action: domain.TreeAction{Type: "list_tasks"}},
		{ID: "same", Label: "b", Action: domain.TreeAction{Type: "list_tasks"}},
	}
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTreeRejectsMissingWorkflow(t *testing.T) {
	tr := validTree()
	err := domain.ValidateTree(tr, func(string) bool { return false })
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateTreeRejectsBadMediaURL(t *testing.T) {
	tr := validTree()
	tr.Items[0].Action = domain.TreeAction{
		Type:  "send_media",
		Media: []domain.Media{{Kind: "image", URL: "ftp://x"}},
	}
	if err := domain.ValidateTree(tr, allowAll); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestDefaultMenuTreeHasWorkflowAndHelp(t *testing.T) {
	tr := domain.DefaultMenuTree("ch-9")
	if tr.ID != "ch-9" {
		t.Fatalf("id=%s", tr.ID)
	}
	if err := domain.ValidateTree(tr, func(id string) bool { return id == "default-image" }); err != nil {
		t.Fatalf("default invalid: %v", err)
	}
	if len(tr.Items) != 2 {
		t.Fatalf("items=%d", len(tr.Items))
	}
	if tr.Items[0].Action.Type != "open_workflow" || tr.Items[0].Action.WorkflowID != "default-image" {
		t.Fatalf("first=%+v", tr.Items[0].Action)
	}
	if tr.Items[1].Action.Type != "send_text" || tr.Items[1].Action.Text == "" {
		t.Fatalf("second=%+v", tr.Items[1].Action)
	}
}
