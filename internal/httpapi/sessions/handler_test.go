package sessions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openSessionsHandler(t *testing.T) (*persistence.SessionRepository, *httptest.Server) {
	t.Helper()
	dsn := "file:sessions_httpapi_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.SessionRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewSessionRepository(gdb)
	h := &sessionsapi.Handler{Repo: repo}
	r := chi.NewRouter()
	r.Route("/api/v1/sessions", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return repo, srv
}

func TestSessionsHandler_ListGetReadOnly(t *testing.T) {
	repo, srv := openSessionsHandler(t)
	ctx := context.Background()
	now := time.Now().UTC()

	s1 := domain.NewCollecting("sess-a", "tg:101", "case-a", []string{"prompt"}, now)
	s1.UserID = "user-a"
	text := "hello"
	s1.Draft["prompt"] = domain.DraftValue{Key: "prompt", Text: &text}
	if err := repo.Save(ctx, s1); err != nil {
		t.Fatal(err)
	}
	s2 := domain.NewCollecting("sess-b", "tg:202", "case-b", []string{"prompt"}, now)
	s2.UserID = "user-b"
	s2.Status = domain.StatusSubmitted
	if err := repo.Save(ctx, s2); err != nil {
		t.Fatal(err)
	}

	res, err := http.Get(srv.URL + "/api/v1/sessions?user_id=user-a")
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
