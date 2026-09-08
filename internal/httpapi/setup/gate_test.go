package setup_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/adminusers/domain"
	"github.com/Mr9esx/Pixoma/internal/adminusers/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func TestGate_MediaReachableWithSetupSessionBeforeInit(t *testing.T) {
	boot, creds, err := bootstrap.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()

	sess := setup.NewSessions("")
	token, err := sess.Issue(creds.Username, false)
	if err != nil {
		t.Fatal(err)
	}

	gate := &setup.Gate{Boot: boot, Sessions: sess}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	for _, tc := range []struct {
		name  string
		path  string
		token string
		want  int
	}{{
		name:  "media upload allowed with setup session",
		path:  "/api/v1/media/",
		token: token,
		want:  http.StatusOK,
	}, {
		name: "media refused without a session",
		path: "/api/v1/media/",
		want: http.StatusUnauthorized,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(""))
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			gate.Middleware(next).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("code = %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
			if tc.want == http.StatusOK && !called {
				t.Fatal("expected next handler to be called")
			}
		})
	}
}

func TestGate_InitializedRejectsViewerWritesAndSetupAdministration(t *testing.T) {
	boot, creds, err := bootstrap.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()
	if err := boot.SetPassword(creds.Username, "secret123"); err != nil {
		t.Fatal(err)
	}
	if err := boot.MarkInitialized(); err != nil {
		t.Fatal(err)
	}

	gdb, err := db.Open(db.Options{DSN: "file:gate_rbac?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.ConsoleUserRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewConsoleUserRepository(gdb)
	if err := repo.Create(context.Background(), &domain.ConsoleUser{
		ID: "acct-viewer", Username: "viewer", Role: domain.RoleViewer, Enabled: true, PasswordHash: "unused",
	}); err != nil {
		t.Fatal(err)
	}

	sess := setup.NewSessions("")
	token, err := sess.IssueAccount("viewer", "acct-viewer", domain.RoleViewer, false)
	if err != nil {
		t.Fatal(err)
	}
	gate := &setup.Gate{Boot: boot, Sessions: sess, ConsoleUsers: repo}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	for _, tc := range []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "viewer can read", method: http.MethodGet, path: "/api/v1/tasks", want: http.StatusOK},
		{name: "viewer cannot write", method: http.MethodPost, path: "/api/v1/tasks", want: http.StatusForbidden},
		{name: "viewer cannot administer setup", method: http.MethodPut, path: "/api/v1/setup/settings", want: http.StatusForbidden},
		{name: "viewer can change own password", method: http.MethodPost, path: "/api/v1/setup/password", want: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			gate.Middleware(next).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("code = %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestGate_DisabledAccountSessionIsRejected(t *testing.T) {
	boot, creds, err := bootstrap.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()
	if err := boot.SetPassword(creds.Username, "secret123"); err != nil {
		t.Fatal(err)
	}
	if err := boot.MarkInitialized(); err != nil {
		t.Fatal(err)
	}

	gdb, err := db.Open(db.Options{DSN: "file:gate_disabled?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.ConsoleUserRow{}); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewConsoleUserRepository(gdb)
	account := &domain.ConsoleUser{
		ID: "acct-op", Username: "op", Role: domain.RoleOperator, Enabled: true, PasswordHash: "unused",
	}
	if err := repo.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}

	sess := setup.NewSessions("")
	token, err := sess.IssueAccount("op", "acct-op", domain.RoleOperator, false)
	if err != nil {
		t.Fatal(err)
	}
	gate := &setup.Gate{Boot: boot, Sessions: sess, ConsoleUsers: repo}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	gate.Middleware(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("enabled account code = %d, want 200", rec.Code)
	}

	account.Enabled = false
	if err := repo.Update(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	gate.Middleware(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled account code = %d, want 401", rec.Code)
	}
}

func TestGate_UpgradesBootstrapSessionAfterRestart(t *testing.T) {
	boot, creds, err := bootstrap.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()
	if err := boot.MarkInitialized(); err != nil {
		t.Fatal(err)
	}

	sess := setup.NewSessions("")
	token, err := sess.Issue(creds.Username, true)
	if err != nil {
		t.Fatal(err)
	}
	repo := &upgradingAccountRepository{user: &domain.ConsoleUser{
		ID:       "admin-1",
		Username: creds.Username,
		Role:     domain.RoleAdmin,
		Enabled:  true,
	}}
	gate := &setup.Gate{Boot: boot, Sessions: sess, ConsoleUsers: repo}

	var account setup.AccountSession
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		account, ok = setup.AccountFromContext(r.Context())
		if !ok {
			http.Error(w, "missing account", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	gate.Middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want %d (body %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if account.AccountID != repo.user.ID {
		t.Fatalf("account ID = %q, want %q", account.AccountID, repo.user.ID)
	}
	bound, ok := sess.LookupAccount(token)
	if !ok {
		t.Fatal("session was not retained")
	}
	if bound.AccountID != repo.user.ID {
		t.Fatalf("persisted session account ID = %q, want %q", bound.AccountID, repo.user.ID)
	}
}

type upgradingAccountRepository struct {
	user *domain.ConsoleUser
}

func (r *upgradingAccountRepository) Create(context.Context, *domain.ConsoleUser) error {
	return nil
}

func (r *upgradingAccountRepository) GetByUsername(_ context.Context, username string) (*domain.ConsoleUser, error) {
	if r.user.Username == username {
		return r.user, nil
	}
	return nil, errors.New("not found")
}

func (r *upgradingAccountRepository) GetByID(_ context.Context, id string) (*domain.ConsoleUser, error) {
	if r.user.ID == id {
		return r.user, nil
	}
	return nil, errors.New("not found")
}

func (r *upgradingAccountRepository) List(context.Context, domain.ListQuery) ([]*domain.ConsoleUser, error) {
	return []*domain.ConsoleUser{r.user}, nil
}

func (r *upgradingAccountRepository) CountAdmins(context.Context) (int64, error) {
	return 1, nil
}

func (r *upgradingAccountRepository) Update(context.Context, *domain.ConsoleUser) error {
	return nil
}

func (r *upgradingAccountRepository) Delete(context.Context, string) error {
	return nil
}
