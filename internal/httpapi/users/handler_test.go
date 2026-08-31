package users_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	"github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openUsersHandler(t *testing.T) (*persistence.UserRepository, *httptest.Server) {
	t.Helper()
	dsn := "file:users_httpapi_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.UserRow{}, &persistence.UserExternalIdentityRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewUserRepository(gdb)
	h := &usersapi.Handler{Repo: repo}
	r := chi.NewRouter()
	r.Route("/api/v1/users", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return repo, srv
}

func TestUsersHandler_ListGetReadOnly(t *testing.T) {
	repo, srv := openUsersHandler(t)
	ctx := context.Background()

	u, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "9001",
		Username: "alice_admin", FirstName: "Alice", LastName: "A", LanguageCode: "zh",
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.Get(srv.URL + "/api/v1/users")
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
	found := false
	for _, row := range list {
		if row["id"] == u.ID {
			found = true
			if row["username"] != "alice_admin" {
				t.Fatalf("username=%v", row["username"])
			}
		}
	}
	if !found {
		t.Fatalf("user %s not in list: %+v", u.ID, list)
	}

	res2, err := http.Get(srv.URL + "/api/v1/users/" + u.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", res2.StatusCode)
	}

	res3, err := http.Get(srv.URL + "/api/v1/users/missing-id")
	if err != nil {
		t.Fatal(err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusNotFound {
		t.Fatalf("missing get status=%d", res3.StatusCode)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	res4, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res4.Body.Close()
	if res4.StatusCode != http.StatusMethodNotAllowed && res4.StatusCode != http.StatusNotFound {
		t.Fatalf("POST should be disallowed, got %d", res4.StatusCode)
	}
}

func TestUsersHandler_UpdateAccess(t *testing.T) {
	repo, srv := openUsersHandler(t)
	ctx := context.Background()

	u, err := repo.UpsertByChannelExternal(ctx, domain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "9002", Username: "bob_access",
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/users/"+u.ID+"/access", strings.NewReader(`{"access":"always_allowed"}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(res)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("update status=%d", response.StatusCode)
	}
	var updated map[string]any
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated["access"] != string(domain.UserAccessAlwaysAllowed) {
		t.Fatalf("access=%v", updated["access"])
	}

	stored, err := repo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Access != domain.UserAccessAlwaysAllowed {
		t.Fatalf("stored access=%q", stored.Access)
	}

	res2, err := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/users/"+u.ID+"/access", strings.NewReader(`{"access":"invalid"}`))
	if err != nil {
		t.Fatal(err)
	}
	response2, err := http.DefaultClient.Do(res2)
	if err != nil {
		t.Fatal(err)
	}
	defer response2.Body.Close()
	if response2.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid access status=%d", response2.StatusCode)
	}
}
