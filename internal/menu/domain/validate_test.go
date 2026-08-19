package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

func exists(context.Context, string) (bool, error) { return true, nil }

func TestValidate_RejectsEmptyTreeDuplicateID(t *testing.T) {
	ctx := context.Background()

	if err := domain.Validate(ctx, domain.MenuTree{ChannelID: "tg-default", Items: nil}, exists, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty items: %v", err)
	}

	allDisabled := domain.DefaultSeedTree("tg-default")
	for i := range allDisabled.Items {
		allDisabled.Items[i].Enabled = false
	}
	if err := domain.Validate(ctx, allDisabled, exists, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("no enabled: %v", err)
	}

	dupID := domain.DefaultSeedTree("tg-default")
	dupID.Items[1].ID = dupID.Items[0].ID
	dupID.Items[1].Label = "other"
	if err := domain.Validate(ctx, dupID, exists, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("dup id: %v", err)
	}
}

func TestValidate_RejectsUnknownCapability(t *testing.T) {
	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].CapabilityID = "nope"
	tree.Items[0].Params = map[string]any{"case_ids": []any{"x"}}
	err := domain.Validate(context.Background(), tree, exists, func(context.Context, string) (bool, error) { return false, nil }, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown capability must be rejected, got %v", err)
	}
}

func TestValidate_RejectsRootOverLimit(t *testing.T) {
	tree := domain.MenuTree{ChannelID: "tg-default", Items: []domain.MenuNode{}}
	for i := 0; i < 7; i++ {
		tree.Items = append(tree.Items, domain.MenuNode{
			ID: string(rune('a' + i)), Label: string(rune('a' + i)), Order: i, Enabled: true,
		})
	}
	err := domain.Validate(context.Background(), tree, exists, exists, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("7 roots must be rejected, got %v", err)
	}
}

func TestValidate_RejectsNestedGroup(t *testing.T) {
	tree := domain.MenuTree{
		ChannelID: "tg-default",
		Items: []domain.MenuNode{{
			ID: "g", Label: "组", Order: 0, Enabled: true,
			Children: []domain.MenuNode{{
				ID: "g2", Label: "子组", Order: 0, Enabled: true,
				Children: []domain.MenuNode{{ID: "e", Label: "入口", Order: 0, Enabled: true}},
			}},
		}},
	}
	err := domain.Validate(context.Background(), tree, exists, exists, nil)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("nested group must be rejected, got %v", err)
	}
}

func TestValidate_RejectsDuplicateOrderWithinParent(t *testing.T) {
	ctx := context.Background()

	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[1].Order = tree.Items[0].Order
	if err := domain.Validate(ctx, tree, exists, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("duplicate order must be rejected, got %v", err)
	}

	ok := domain.MenuTree{
		ChannelID: "tg-default",
		Items: []domain.MenuNode{
			{ID: "a", Label: "A", Order: 0, Enabled: true,
				Children: []domain.MenuNode{{ID: "a1", Label: "A1", Order: 0, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "x"}}}},
			{ID: "b", Label: "B", Order: 1, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "x"}},
		},
	}
	if err := domain.Validate(ctx, ok, exists, exists, nil); err != nil {
		t.Fatalf("same order under different parents should be ok: %v", err)
	}
}

func TestValidate_OpenCaseRequiresExistingCase(t *testing.T) {
	ctx := context.Background()
	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].Params = map[string]any{"case_ids": []any{"missing"}}
	existsID := func(_ context.Context, id string) (bool, error) { return id == "ok", nil }
	if err := domain.Validate(ctx, tree, existsID, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing case: %v", err)
	}
	tree.Items[0].Params = map[string]any{"case_ids": []any{"ok"}}
	if err := domain.Validate(ctx, tree, existsID, exists, nil); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_LeafWithoutCapabilityRejected(t *testing.T) {
	tree := domain.MenuTree{Items: []domain.MenuNode{
		{ID: "a", Label: "A", Order: 0, Enabled: true},
	}}
	err := domain.Validate(context.Background(), tree, nil, nil, nil)
	if err == nil {
		t.Fatal("expected leaf-without-capability error")
	}
}

func TestValidate_LeafWithCapabilityAccepted(t *testing.T) {
	tree := domain.MenuTree{Items: []domain.MenuNode{
		{ID: "a", Label: "A", Order: 0, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "x"}},
	}}
	err := domain.Validate(context.Background(), tree, nil, func(_ context.Context, id string) (bool, error) {
		return id == "reply_text", nil
	}, func(_ context.Context, _ string, _ map[string]any) error { return nil })
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidate_ReplyMediaValidation(t *testing.T) {
	tree := domain.MenuTree{
		ChannelID: "tg-default",
		Items: []domain.MenuNode{{
			ID: "r", Label: "R", Order: 0, Enabled: true, Reply: &domain.ReplyPayload{},
		}},
	}
	if err := domain.Validate(context.Background(), tree, exists, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty reply: %v", err)
	}
	tree.Items[0].Reply = &domain.ReplyPayload{Images: []string{"ftp://x/a.png"}}
	if err := domain.Validate(context.Background(), tree, exists, exists, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad url: %v", err)
	}
}

func TestValidate_ParamsValidatorCalled(t *testing.T) {
	tree := domain.DefaultSeedTree("tg-default")
	called := false
	validateParams := func(_ context.Context, capabilityID string, _ map[string]any) error {
		called = true
		if capabilityID != "open_case" {
			t.Fatalf("capability=%q", capabilityID)
		}
		return errors.New("boom")
	}
	if err := domain.Validate(context.Background(), tree, exists, exists, validateParams); err == nil {
		t.Fatal("params validator error must propagate")
	}
	if !called {
		t.Fatal("params validator not called")
	}
}
