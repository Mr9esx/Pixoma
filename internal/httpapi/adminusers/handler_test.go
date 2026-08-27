package adminusers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	consoledomain "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/domain"
	consolepersist "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/persistence"
	adminusersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminusers"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func newRouter(t *testing.T) (*chi.Mux, *consolepersist.ConsoleUserRepository) {
	t.Helper()
	name := "adminusers_" + strings.ReplaceAll(t.Name(), "/", "_")
	gdb, err := db.Open(db.Options{DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &consolepersist.ConsoleUserRow{}); err != nil {
		t.Fatal(err)
	}
	repo := consolepersist.NewConsoleUserRepository(gdb)
	h := &adminusersapi.Handler{Repo: repo}
	r := chi.NewRouter()
	h.Mount(r)
	return r, repo
}

func req(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	return rec
}

func seed(repo *consolepersist.ConsoleUserRepository, id, name, role string) error {
	return repo.Create(context.Background(), &consoledomain.ConsoleUser{
		ID: id, Username: name, Role: role, Enabled: true, PasswordHash: "h",
	})
}

func TestCreate_ListAndNoPasswordLeak(t *testing.T) {
	r, _ := newRouter(t)
	rec := req(t, r, http.MethodPost, "/", map[string]any{
		"username": "newbie", "email": "n@example.com", "nickname": "新用户", "password": "secret123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	rec = req(t, r, http.MethodGet, "/?q=newbie", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d", rec.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["username"] != "newbie" {
		t.Fatalf("list result: %v", got)
	}
	if _, has := got[0]["password_hash"]; has {
		t.Fatal("password_hash leaked")
	}
}

func TestCreate_WeakPasswordAndDuplicate(t *testing.T) {
	r, repo := newRouter(t)
	if rec := req(t, r, http.MethodPost, "/", map[string]any{"username": "u", "password": "short"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("weak password: %d %s", rec.Code, rec.Body.String())
	}
	if err := seed(repo, "a", "dup", consoledomain.RoleViewer); err != nil {
		t.Fatal(err)
	}
	if rec := req(t, r, http.MethodPost, "/", map[string]any{"username": "dup", "password": "secret123"}); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPatch_ResetPasswordAndRole(t *testing.T) {
	r, repo := newRouter(t)
	if err := seed(repo, "acct-1", "viewer1", consoledomain.RoleViewer); err != nil {
		t.Fatal(err)
	}
	rec := req(t, r, http.MethodPatch, "/acct-1", map[string]any{"role": "operator", "password": "newsecret9"})
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	u, _ := repo.GetByID(context.Background(), "acct-1")
	if u.Role != "operator" || !u.MustChangePassword {
		t.Fatalf("patch not applied: %+v", u)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("newsecret9")); err != nil {
		t.Fatalf("password not reset: %v", err)
	}
}

func TestDelete_RefuseSoleAdmin(t *testing.T) {
	r, repo := newRouter(t)
	if err := seed(repo, "boss", "boss", consoledomain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if rec := req(t, r, http.MethodDelete, "/boss", nil); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 deleting sole admin, got %d %s", rec.Code, rec.Body.String())
	}
	if err := seed(repo, "op", "op", consoledomain.RoleOperator); err != nil {
		t.Fatal(err)
	}
	// Still the only admin: must remain protected even with other users.
	if rec := req(t, r, http.MethodDelete, "/boss", nil); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 still for sole admin, got %d", rec.Code)
	}
	// Non-admin deletion proceeds.
	if rec := req(t, r, http.MethodDelete, "/op", nil); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 deleting non-admin, got %d %s", rec.Code, rec.Body.String())
	}
}
