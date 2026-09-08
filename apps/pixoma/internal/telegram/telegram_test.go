package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-telegram/bot"

	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
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

	f := &tgChannelFactory{registry: &notifyRegistry{handlers: map[string]channelapp.NotifyHandler{}}}
	_, err := f.Create(channelapp.ChannelSnapshot{ID: "c1", Credential: "1:token"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("Create called Telegram %d times; adapter start must skip getMe", n)
	}
}

func TestTelegramHTTPClient_HasNoRequestTimeout(t *testing.T) {
	c := telegramHTTPClient()
	if c.Timeout != 0 {
		t.Fatalf("Timeout=%s; getUpdates long-polls for ~pollTimeout, Client.Timeout equal to that floods awaiting headers", c.Timeout)
	}
}

func TestTgBotWrapper_StopWaitsForGetUpdatesToExit(t *testing.T) {
	hit := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.Contains(r.URL.Path, "getUpdates") {
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
			return
		}
		select {
		case <-hit:
		default:
			close(hit)
		}
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
	}))
	t.Cleanup(srv.Close)

	botInst, err := bot.New("1:token", bot.WithServerURL(srv.URL), bot.WithSkipGetMe())
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	w := &tgBotWrapper{
		registry:  &notifyRegistry{handlers: map[string]channelapp.NotifyHandler{}},
		channelID: "c1",
		bot:       botInst,
	}
	if err := w.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-hit:
	case <-time.After(2 * time.Second):
		t.Fatal("getUpdates never started")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := w.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("Stop returned while bot.Start was still running")
	}
}

func TestBotRuntime_WaitBlocksUntilProbeFinishes(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	key := make([]byte, 32)
	ct, err := channeldomain.EncryptCredential(key, channeldomain.Credential{BotToken: "1:token"})
	if err != nil {
		t.Fatal(err)
	}
	store := &runtimeMemStore{rows: map[string]channeldomain.Channel{
		"c1": {
			ID: "c1", Platform: string(channeldomain.PlatformTelegram),
			Name: "tg", CredentialCiphertext: ct, Enabled: true,
		},
	}}
	svc := &channelapp.Service{
		Store: store,
		Key:   key,
		CheckTelegram: func(ctx context.Context, token string) (channelapp.ReachabilityResult, error) {
			close(entered)
			<-release
			return channelapp.ReachabilityResult{Kind: channelapp.ReachabilityOK}, nil
		},
	}

	orig := newTelegramBot
	t.Cleanup(func() { newTelegramBot = orig })
	newTelegramBot = func(token string, options ...bot.Option) (*telegramBot, error) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "getUpdates") {
				time.Sleep(20 * time.Millisecond)
				_, _ = w.Write([]byte(`{"ok":true,"result":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
		}))
		t.Cleanup(srv.Close)
		opts := []bot.Option{bot.WithServerURL(srv.URL), bot.WithSkipGetMe()}
		opts = append(opts, options...)
		return orig(token, opts...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt, err := StartBotRuntime(ctx, BotDeps{Channels: svc})
	if err != nil {
		t.Fatalf("StartBotRuntime: %v", err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("probe never started")
	}
	cancel()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer waitCancel()
	if err := rt.Wait(waitCtx); err == nil {
		t.Fatal("Wait returned while probe still in Telegram")
	}

	close(release)
	waitCtx, waitCancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	if err := rt.Wait(waitCtx); err != nil {
		t.Fatalf("Wait after probe finished: %v", err)
	}
}

type runtimeMemStore struct {
	mu   sync.Mutex
	rows map[string]channeldomain.Channel
}

func (s *runtimeMemStore) Create(_ context.Context, ch channeldomain.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[ch.ID] = ch
	return nil
}
func (s *runtimeMemStore) Get(_ context.Context, id string) (channeldomain.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch, ok := s.rows[id]
	if !ok {
		return channeldomain.Channel{}, channeldomain.ErrNotFound
	}
	return ch, nil
}
func (s *runtimeMemStore) List(context.Context) ([]channeldomain.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]channeldomain.Channel, 0, len(s.rows))
	for _, ch := range s.rows {
		out = append(out, ch)
	}
	return out, nil
}
func (s *runtimeMemStore) Update(_ context.Context, ch channeldomain.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[ch.ID] = ch
	return nil
}
func (s *runtimeMemStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, id)
	return nil
}
