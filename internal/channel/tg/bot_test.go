package tg

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"

	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestHandleCallbackUpdateAnswersBeforeProcessing(t *testing.T) {
	out := &textCaptureOutbound{}
	ad := New(out)
	ad.ChannelID = "tg-default"
	var order []string
	ad.Users = identityResolverFunc(func(_ context.Context, _ sharedkernel.ChannelAddr, _ identitydomain.UpsertFrom) (string, error) {
		order = append(order, "resolve")
		return "user-1", nil
	})
	cq := &models.CallbackQuery{
		ID:   "cb-1",
		From: models.User{ID: 100},
		Data: "unknown",
	}

	if err := handleCallbackUpdate(context.Background(), ad, cq, func(_ context.Context, callbackID string) error {
		if callbackID != "cb-1" {
			t.Fatalf("callbackID=%q", callbackID)
		}
		order = append(order, "answer")
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(order) != 2 || order[0] != "answer" || order[1] != "resolve" {
		t.Fatalf("order=%q", order)
	}
	if len(out.texts) != 1 || out.texts[0] != "未知操作" {
		t.Fatalf("texts=%q", out.texts)
	}
}

type identityResolverFunc func(ctx context.Context, addr sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error)

func (f identityResolverFunc) Resolve(ctx context.Context, addr sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error) {
	return f(ctx, addr, profile)
}
