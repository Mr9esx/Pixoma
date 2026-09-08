package sessions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	sessionsapi "github.com/Mr9esx/Pixoma/internal/httpapi/sessions"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
	identitypersist "github.com/Mr9esx/Pixoma/internal/users/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func openSessionsHandler(t *testing.T) (*persistence.SessionRepository, *identitydomain.User, *httptest.Server) {
	t.Helper()
	dsn := "file:sessions_httpapi_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.SessionRow{}, &channelpersist.ChannelRow{}, &identitypersist.UserRow{}, &identitypersist.UserExternalIdentityRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	channelStore := channelpersist.NewGormRepository(gdb)
	for _, channel := range []channeldomain.Channel{
		{ID: "tg", Platform: "telegram", Name: "Telegram Bot", Enabled: true},
		{ID: "ig", Platform: "instagram", Name: "Instagram Bot", Enabled: true},
	} {
		if err := channelStore.Create(context.Background(), channel); err != nil {
			t.Fatal(err)
		}
	}
	repo := persistence.NewSessionRepository(gdb)
	now := time.Now().UTC()
	userRepo := identitypersist.NewUserRepository(gdb)
	user, err := userRepo.UpsertByChannelExternal(context.Background(), identitydomain.UpsertFrom{
		ChannelID: "tg", ExternalUserID: "9001", Username: "alice_session",
		FirstName: "Alice", LastName: "S", LastSeenAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	h := &sessionsapi.Handler{Repo: repo, Channels: channelStore, Context: persistence.NewSessionAdminProjection(gdb)}
	r := chi.NewRouter()
	r.Route("/api/v1/sessions", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return repo, user, srv
}

func TestSessionsHandler_ListGetReadOnly(t *testing.T) {
	repo, user, srv := openSessionsHandler(t)
	ctx := context.Background()
	now := time.Now().UTC()

	s1 := domain.NewCollecting("sess-a", "tg:101", 1, []string{"prompt"}, now)
	s1.UserID = user.ID
	text := "hello"
	s1.Draft["prompt"] = domain.DraftValue{Key: "prompt", Text: &text}
	if err := repo.Save(ctx, s1); err != nil {
		t.Fatal(err)
	}
	s2 := domain.NewCollecting("sess-b", "tg:202", 2, []string{"prompt"}, now)
	s2.UserID = "user-b"
	s2.Status = domain.StatusSubmitted
	if err := repo.Save(ctx, s2); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, domain.NewCollecting("sess-c", "ig:99", 3, []string{"prompt"}, now)); err != nil {
		t.Fatal(err)
	}

	res, err := http.Get(srv.URL + "/api/v1/sessions?user_id=" + user.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", res.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["id"] != "sess-a" {
		t.Fatalf("filter user_id: %+v", list)
	}

	resCase, err := http.Get(srv.URL + "/api/v1/sessions?case_id=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resCase.Body.Close()
	if resCase.StatusCode != http.StatusOK {
		t.Fatalf("list case_id status=%d", resCase.StatusCode)
	}
	var byCase []map[string]any
	if err := json.NewDecoder(resCase.Body).Decode(&byCase); err != nil {
		t.Fatal(err)
	}
	if len(byCase) != 1 || byCase[0]["id"] != "sess-a" {
		t.Fatalf("filter case_id: %+v", byCase)
	}

	resChan, err := http.Get(srv.URL + "/api/v1/sessions?channel_id=tg")
	if err != nil {
		t.Fatal(err)
	}
	defer resChan.Body.Close()
	if resChan.StatusCode != http.StatusOK {
		t.Fatalf("list channel_id status=%d", resChan.StatusCode)
	}
	var byChannel []map[string]any
	if err := json.NewDecoder(resChan.Body).Decode(&byChannel); err != nil {
		t.Fatal(err)
	}
	if len(byChannel) != 2 {
		t.Fatalf("filter channel_id: %+v", byChannel)
	}
	resChan2, err := http.Get(srv.URL + "/api/v1/sessions?channel_id=ig")
	if err != nil {
		t.Fatal(err)
	}
	defer resChan2.Body.Close()
	var byIg []map[string]any
	if err := json.NewDecoder(resChan2.Body).Decode(&byIg); err != nil {
		t.Fatal(err)
	}
	if len(byIg) != 1 || byIg[0]["id"] != "sess-c" {
		t.Fatalf("filter channel_id=ig: %+v", byIg)
	}

	res2, err := http.Get(srv.URL + "/api/v1/sessions/sess-a")
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", res2.StatusCode)
	}
	var detail map[string]any
	if err := json.NewDecoder(res2.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	draft, ok := detail["draft"].(map[string]any)
	if !ok || draft["prompt"] == nil {
		t.Fatalf("expected draft.prompt, got %+v", detail["draft"])
	}
	if detail["channel_id"] != "tg" || detail["channel_name"] != "Telegram Bot" {
		t.Fatalf("session context: %+v", detail)
	}
	if detail["user_id"] != user.ID {
		t.Fatalf("session user id: %+v", detail)
	}
	sessionUser, ok := detail["user"].(map[string]any)
	if !ok ||
		sessionUser["id"] != user.ID ||
		sessionUser["channel_id"] != "tg" ||
		sessionUser["external_user_id"] != "9001" ||
		sessionUser["username"] != "alice_session" {
		t.Fatalf("session related user: %+v", detail["user"])
	}

	res3, err := http.Get(srv.URL + "/api/v1/sessions/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusNotFound {
		t.Fatalf("missing get status=%d", res3.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/sessions/sess-a", nil)
	res4, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res4.Body.Close()
	if res4.StatusCode != http.StatusMethodNotAllowed && res4.StatusCode != http.StatusNotFound {
		t.Fatalf("PATCH should be disallowed, got %d", res4.StatusCode)
	}
}
