package app

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-telegram/bot"

	channelruntime "github.com/mr9esx/comfyui_tgbot/internal/channel/runtime"
)

func TestTgChannelFactory_CreateDoesNotCallGetMe(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	orig := newTelegramBot
	t.Cleanup(func() { newTelegramBot = orig })
	newTelegramBot = func(token string, options ...bot.Option) (*telegramBot, error) {
		opts := []bot.Option{
			bot.WithServerURL(srv.URL),
			bot.WithCheckInitTimeout(200 * time.Millisecond),
		}
		opts = append(opts, options...)
		return orig(token, opts...)
	}

	f := &tgChannelFactory{registry: &notifyRegistry{handlers: map[string]channelruntime.NotifyHandler{}}}
	_, err := f.Create(channelruntime.ChannelSnapshot{ID: "c1", Credential: "1:token"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("Create called Telegram %d times; adapter start must skip getMe", n)
	}
}
