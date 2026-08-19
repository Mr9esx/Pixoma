package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

func TestValidate_FolderCaseMustExist(t *testing.T) {
	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].Params = map[string]any{"case_ids": []any{"missing"}}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return false, nil
	}, func(context.Context, string) (bool, error) { return true, nil }, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidate_GroupChildCanBeCapabilityEntry(t *testing.T) {
	tree := domain.MenuTree{
		ChannelID: "tg-default",
		Items: []domain.MenuNode{{
			ID: "root", Label: "Root", Enabled: true,
			Children: []domain.MenuNode{{
				ID: "bad", ParentID: "root", Label: "Bad", Enabled: true, CapabilityID: "open_case", Params: map[string]any{"case_ids": []any{"c1"}},
			}},
		}},
	}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return true, nil
	}, func(context.Context, string) (bool, error) { return true, nil }, nil)
	if err != nil {
		t.Fatalf("group child capability entry should be ok, got %v", err)
	}
}

func TestValidate_OpenCaseNeedsExactlyOne(t *testing.T) {
	tree := domain.MenuTree{
		ChannelID: "tg-default",
		Items: []domain.MenuNode{{
			ID: "x", Label: "X", Enabled: true, CapabilityID: "open_case",
			Params: map[string]any{"case_ids": []any{}},
		}},
	}
	err := domain.Validate(context.Background(), tree, func(context.Context, string) (bool, error) {
		return true, nil
	}, func(context.Context, string) (bool, error) { return true, nil }, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestBuildTree_RoundTrip(t *testing.T) {
	flat := []domain.MenuItem{
		{ID: "r", Label: "R", Enabled: true, Order: 0},
		{ID: "c", ParentID: "r", Label: "C", Enabled: true, Order: 0},
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
