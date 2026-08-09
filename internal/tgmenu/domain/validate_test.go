package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

func TestValidate_RejectsEmptyTreeDuplicateIDAndPropagatesLookupError(t *testing.T) {
	ctx := context.Background()
	exists := func(context.Context, string) (bool, error) { return true, nil }

	if err := domain.Validate(ctx, domain.MenuTree{ID: "default", BotID: "default", Items: nil}, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty items: %v", err)
	}

	allDisabled := domain.DefaultSeedTree()
	for i := range allDisabled.Items {
		allDisabled.Items[i].Enabled = false
	}
	if err := domain.Validate(ctx, allDisabled, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("no enabled: %v", err)
	}

	dupID := domain.DefaultSeedTree()
	dupID.Items[1].ID = dupID.Items[0].ID
	dupID.Items[1].Label = "other"
	if err := domain.Validate(ctx, dupID, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("dup id: %v", err)
	}

	lookupErr := errors.New("db down")
	tree := domain.DefaultSeedTree()
	tree.Items[0].Kind = domain.KindOpenCase
	tree.Items[0].CaseIDs = []string{"x"}
	tree.Items[0].Tag = ""
	failing := func(context.Context, string) (bool, error) { return false, lookupErr }
	err := domain.Validate(ctx, tree, failing)
	if err == nil || errors.Is(err, domain.ErrValidation) {
		t.Fatalf("want non-validation lookup err, got %v", err)
	}
	if !errors.Is(err, lookupErr) {
		t.Fatalf("want wrapped lookupErr, got %v", err)
	}
}

func TestValidate_RejectsDuplicateLabelAndBadReplyMedia(t *testing.T) {
	ctx := context.Background()
	exists := func(context.Context, string) (bool, error) { return true, nil }

	dup := domain.DefaultSeedTree()
	dup.Items[1].Label = dup.Items[0].Label
	if err := domain.Validate(ctx, dup, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("dup label: %v", err)
	}

	// Same label under different parents is allowed.
	okTree := domain.MenuTree{
		ID: domain.DocumentIDDefault, BotID: domain.BotIDDefault,
		Items: []domain.MenuNode{{
			ID: "parent", Label: "P", Enabled: true, Kind: domain.KindFolder,
			Children: []domain.MenuNode{
				{ID: "child", Label: "Same", Enabled: true, Kind: domain.KindFolder},
			},
		}, {
			ID: "other", Label: "Same", Enabled: true, Kind: domain.KindPlaceholder,
		}},
	}
	if err := domain.Validate(ctx, okTree, exists); err != nil {
		t.Fatalf("same label under different parents should be ok: %v", err)
	}

	emptyReply := domain.DefaultSeedTree()
	emptyReply.Items[0].Kind = domain.KindReplyMedia
	emptyReply.Items[0].Tag = ""
	emptyReply.Items[0].Reply = &domain.ReplyPayload{}
	if err := domain.Validate(ctx, emptyReply, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty reply: %v", err)
	}

	badURL := domain.DefaultSeedTree()
	badURL.Items[0].Kind = domain.KindReplyMedia
	badURL.Items[0].Tag = ""
	badURL.Items[0].Reply = &domain.ReplyPayload{Images: []string{"ftp://x/a.png"}}
	if err := domain.Validate(ctx, badURL, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad url: %v", err)
	}
}

func TestValidate_OpenCaseRequiresExistingCase(t *testing.T) {
	ctx := context.Background()
	tree := domain.DefaultSeedTree()
	tree.Items[0].Kind = domain.KindOpenCase
	tree.Items[0].CaseIDs = []string{"missing"}
	tree.Items[0].Tag = ""
	exists := func(_ context.Context, id string) (bool, error) { return id == "ok", nil }
	if err := domain.Validate(ctx, tree, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing case: %v", err)
	}
	tree.Items[0].CaseIDs = []string{"ok"}
	if err := domain.Validate(ctx, tree, exists); err != nil {
		t.Fatal(err)
	}
}
