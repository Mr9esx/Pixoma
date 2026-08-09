package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

func TestValidate_FolderCaseMustExist(t *testing.T) {
	tree := domain.DefaultSeedTree()
	tree.Items[0].CaseIDs = []string{"missing"}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return false, nil
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidate_FolderChildMustBeFolder(t *testing.T) {
	tree := domain.MenuTree{
		ID: domain.DocumentIDDefault, BotID: domain.BotIDDefault,
		Items: []domain.MenuNode{{
			ID: "root", Label: "Root", Enabled: true, Kind: domain.KindFolder,
			Children: []domain.MenuNode{{
				ID: "bad", ParentID: "root", Label: "Bad", Enabled: true, Kind: domain.KindOpenCase, CaseIDs: []string{"c1"},
			}},
		}},
	}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return true, nil
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidate_OpenCaseNeedsExactlyOne(t *testing.T) {
	tree := domain.MenuTree{
		ID: domain.DocumentIDDefault, BotID: domain.BotIDDefault,
		Items: []domain.MenuNode{{
			ID: "x", Label: "X", Enabled: true, Kind: domain.KindOpenCase, CaseIDs: nil,
		}},
	}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return true, nil
	})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildTree_RoundTrip(t *testing.T) {
	flat := []domain.MenuItem{
		{ID: "r", Label: "R", Enabled: true, Kind: domain.KindFolder, Row: 0, Col: 0},
		{ID: "c", ParentID: "r", Label: "C", Enabled: true, Kind: domain.KindFolder, Row: 0, Col: 0},
	}
	nodes, err := domain.BuildTree(flat)
	if err != nil || len(nodes) != 1 || len(nodes[0].Children) != 1 {
		t.Fatalf("%v %+v", err, nodes)
	}
	back := domain.Flatten(nodes)
	if len(back) != 2 {
		t.Fatalf("%d", len(back))
	}
}
