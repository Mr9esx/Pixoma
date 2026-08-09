package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

func TestDefaultSeed_MatchesLegacyLayout(t *testing.T) {
	doc := domain.DefaultSeed()
	if doc.ID != domain.DocumentIDDefault {
		t.Fatalf("id=%q", doc.ID)
	}
	if len(doc.Items) != 6 {
		t.Fatalf("want 6 items, got %d", len(doc.Items))
	}
	byID := map[string]domain.MenuItem{}
	for _, it := range doc.Items {
		byID[it.ID] = it
	}
	img := byID["btn-image"]
	if img.Label != "🖼 图片" || img.Action != domain.ActionListCasesByTag || img.Tag != "image" {
		t.Fatalf("btn-image: %+v", img)
	}
	for _, id := range []string{"btn-video", "btn-recharge", "btn-checkin", "btn-profile", "btn-help"} {
		if byID[id].Action != domain.ActionPlaceholder {
			t.Fatalf("%s action=%s", id, byID[id].Action)
		}
	}
}

func TestValidate_RejectsDuplicateLabelAndBadReplyMedia(t *testing.T) {
	ctx := context.Background()
	exists := func(context.Context, string) (bool, error) { return true, nil }

	dup := domain.DefaultSeed()
	dup.Items[1].Label = dup.Items[0].Label
	if err := domain.Validate(ctx, dup, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("dup label: %v", err)
	}

	emptyReply := domain.DefaultSeed()
	emptyReply.Items[0].Action = domain.ActionReplyMedia
	emptyReply.Items[0].Tag = ""
	emptyReply.Items[0].Reply = &domain.ReplyPayload{}
	if err := domain.Validate(ctx, emptyReply, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty reply: %v", err)
	}

	badURL := domain.DefaultSeed()
	badURL.Items[0].Action = domain.ActionReplyMedia
	badURL.Items[0].Tag = ""
	badURL.Items[0].Reply = &domain.ReplyPayload{Images: []string{"ftp://x/a.png"}}
	if err := domain.Validate(ctx, badURL, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad url: %v", err)
	}
}

func TestValidate_OpenCaseRequiresExistingCase(t *testing.T) {
	ctx := context.Background()
	doc := domain.DefaultSeed()
	doc.Items[0].Action = domain.ActionOpenCase
	doc.Items[0].CaseID = "missing"
	doc.Items[0].Tag = ""
	exists := func(_ context.Context, id string) (bool, error) { return id == "ok", nil }
	if err := domain.Validate(ctx, doc, exists); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing case: %v", err)
	}
	doc.Items[0].CaseID = "ok"
	if err := domain.Validate(ctx, doc, exists); err != nil {
		t.Fatal(err)
	}
}
