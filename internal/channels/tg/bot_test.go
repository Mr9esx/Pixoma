package tg

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"

	"github.com/Mr9esx/Pixoma/internal/channels/tg/tginternal"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

func TestHandleCallbackUpdateAnswersBeforeProcessing(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
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
	if got := srv.SendMessageCalls(); got != 1 {
		t.Fatalf("sendMessage calls = %d, want 1", got)
	}
	texts := srv.SendMessageTexts()
	if len(texts) < 1 || texts[0] != "未知操作" {
		t.Fatalf("sendMessage texts = %v, want first %q", texts, "未知操作")
	}
}

func TestHandleCallbackClearsMarkup(t *testing.T) {
	b, srv := tginternal.NewBot(t)
	ad := New(&BotMessenger{Bot: b})
	ad.ChannelID = "tg-default"
	if err := ad.HandleCallback(context.Background(), "tg-default:1", 42, CBMenu, ""); err != nil {
		t.Fatal(err)
	}
	if got := srv.EditMessageReplyMarkupCalls(); got != 1 {
		t.Fatalf("editMessageReplyMarkup calls = %d, want 1", got)
	}
	ids := srv.EditedMessageIDs()
	if len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("edited ids = %v, want [42]", ids)
	}
}

type identityResolverFunc func(ctx context.Context, addr sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error)

func (f identityResolverFunc) Resolve(ctx context.Context, addr sharedkernel.ChannelAddr, profile identitydomain.UpsertFrom) (string, error) {
	return f(ctx, addr, profile)
}
