package studio_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func TestSessionContextRoutesScopeRequestsToOwner(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:context_routes?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB, err := gdb.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}()
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	session, err := domain.NewSession("context-route-session", "account-a", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	(&studioapi.Handler{Repo: repo, Context: repo}).Mount(router)

	request := func(account, path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(setupapi.WithAccount(req.Context(), setupapi.AccountSession{AccountID: account}))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	base := "/sessions/context-route-session/context"
	for _, path := range []string{base, base + "/requests", base + "/events"} {
		owner := request("account-a", path)
		if owner.Code != http.StatusOK {
			t.Fatalf("owner %s = %d: %s", path, owner.Code, owner.Body.String())
		}
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(owner.Body.Bytes(), &envelope); err != nil || len(envelope.Data) == 0 {
			t.Fatalf("response %s = %s, %v", path, owner.Body.String(), err)
		}
		other := request("account-b", path)
		if other.Code != http.StatusNotFound {
			t.Fatalf("other %s = %d", path, other.Code)
		}
	}
}
