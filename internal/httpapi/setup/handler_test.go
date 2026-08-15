package setup_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
)

func TestWizard_GateAndSQLiteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	boot, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()

	sess := setup.NewSessions()
	h := &setup.Handler{Boot: boot, Sessions: sess, DataDir: dir}
	r := chi.NewRouter()
	r.Use((&setup.Gate{Boot: boot, Sessions: sess}).Middleware)
	r.Route("/api/v1/setup", h.Mount)
	r.Mount("/", adminhost.NewHandler(adminhost.Options{}))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("uninitialized business API: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not_initialized") {
		t.Fatalf("want not_initialized, got %s", rec.Body.String())
	}

	loginBody, _ := json.Marshal(map[string]string{
		"username": creds.Username,
		"password": creds.Password,
	})
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/login", bytes.NewReader(loginBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token              string `json:"token"`
		MustChangePassword bool   `json:"must_change_password"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}
	if loginResp.Token == "" || !loginResp.MustChangePassword {
		t.Fatalf("login resp %+v", loginResp)
	}
	auth := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		req.Header.Set("Content-Type", "application/json")
	}

	pw, _ := json.Marshal(map[string]string{
		"old_password": creds.Password,
		"new_password": "new-secret-9",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/password", bytes.NewReader(pw))
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("password: %d %s", rec.Code, rec.Body.String())
	}

	dsn := filepath.Join(dir, "app.db")
	dbBody, _ := json.Marshal(map[string]string{"driver": "sqlite", "dsn": dsn})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/database", bytes.NewReader(dbBody))
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("database: %d %s", rec.Code, rec.Body.String())
	}

	bad, _ := json.Marshal(settings.Settings{
		Placement:  settings.PlacementRemote,
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   filepath.Join(dir, "blob"),
		DBDriver:   "sqlite",
		DBDSN:      dsn,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/draft", bytes.NewReader(bad))
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("remote localfs should fail: %d %s", rec.Code, rec.Body.String())
	}

	okDraft, _ := json.Marshal(settings.Settings{
		Placement:         settings.PlacementLocal,
		BlobDriver:        botconfig.BlobDriverLocalFS,
		BlobRoot:          filepath.Join(dir, "blob"),
		DBDriver:          "sqlite",
		DBDSN:             dsn,
		ComfyMock:         true,
		ComfyUIBaseURL:    "http://127.0.0.1:8188",
		DefaultInstanceID: "local",
		AutoSpawnEdge:     true,
		TelegramBotToken:  "tg-test-token",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/draft", bytes.NewReader(okDraft))
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("draft: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/finalize", nil)
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("finalize: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "restart_required") {
		t.Fatalf("want restart hint, got %s", rec.Body.String())
	}
	if !boot.Initialized() {
		t.Fatal("expected initialized")
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("initialized without session: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/settings", nil)
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"configured":true`) {
		t.Fatalf("settings not readable: %s", rec.Body.String())
	}
}
