package setup_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
)

func TestWizard_GateAndSQLiteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	boot, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()

	sess := setup.NewSessions("")
	restarted := make(chan struct{}, 1)
	h := &setup.Handler{
		Boot:         boot,
		Sessions:     sess,
		DataDir:      dir,
		PublicURL:    "http://192.168.31.162:8082",
		RestartAfter: time.Millisecond,
		Restart:      func() { restarted <- struct{}{} },
	}
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
		Placement:      settings.PlacementLocal,
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       filepath.Join(dir, "blob"),
		DBDriver:       "sqlite",
		DBDSN:          dsn,
		ComfyMock:      true,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
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
	if !strings.Contains(rec.Body.String(), `"restarting":true`) {
		t.Fatalf("want auto-reload, got %s", rec.Body.String())
	}
	select {
	case <-restarted:
	case <-time.After(time.Second):
		t.Fatal("expected Restart after finalize")
	}
	if !boot.Initialized() {
		t.Fatal("expected initialized")
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil))
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "restart_required") {
		t.Fatalf("after finalize without restart: %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "restart_required") {
		t.Fatalf("session still blocked until restart: %d %s", rec.Code, rec.Body.String())
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
	if !strings.Contains(rec.Body.String(), `"public_url":"http://192.168.31.162:8082"`) {
		t.Fatalf("settings missing public_url: %s", rec.Body.String())
	}
}

func TestPutSettings_RequiresInitialized(t *testing.T) {
	dir := t.TempDir()
	boot, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()

	sess := setup.NewSessions("")
	h := &setup.Handler{Boot: boot, Sessions: sess, DataDir: dir}
	r := chi.NewRouter()
	r.Use((&setup.Gate{Boot: boot, Sessions: sess}).Middleware)
	r.Route("/api/v1/setup", h.Mount)

	loginBody, _ := json.Marshal(map[string]string{
		"username": creds.Username,
		"password": creds.Password,
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/login", bytes.NewReader(loginBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(settings.Settings{
		Placement:  settings.PlacementLocal,
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   filepath.Join(dir, "blob"),
		DBDriver:   "sqlite",
		DBDSN:      filepath.Join(dir, "app.db"),
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/setup/settings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 before init, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestSettings_AfterInit_PasswordAndUpdate(t *testing.T) {
	env := completeWizard(t)
	defer env.boot.Close()

	if err := env.boot.SetRestartRequired(false); err != nil {
		t.Fatal(err)
	}
	step := env.boot.WizardStep()

	pwOnlyNew, _ := json.Marshal(map[string]string{"new_password": "later-secret-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/password", bytes.NewReader(pwOnlyNew))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("password without old after init: %d %s", rec.Code, rec.Body.String())
	}

	pwOK, _ := json.Marshal(map[string]string{
		"old_password": "new-secret-9",
		"new_password": "later-secret-1",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/password", bytes.NewReader(pwOK))
	env.auth(req)
	rec = httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}
	if env.boot.WizardStep() != step {
		t.Fatalf("password must not move wizard step after init, got %q want %q", env.boot.WizardStep(), step)
	}

	ok, err := env.boot.VerifyPassword(env.username, "later-secret-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected new password to work")
	}

	newRoot := filepath.Join(env.dir, "blob-2")
	putBody, _ := json.Marshal(settings.Settings{
		Placement:      settings.PlacementLocal,
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       newRoot,
		DBDriver:       "mysql",
		DBDSN:          "user:pass@tcp(127.0.0.1:3306)/pixoma",
		ComfyMock:      false,
		ComfyUIBaseURL: "http://should-not-stick:9",
		BlobAccessKey:  "********",
		BlobSecretKey:  "********",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/setup/settings", bytes.NewReader(putBody))
	env.auth(req)
	rec = httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put settings: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"restarting":true`) {
		t.Fatalf("want auto-reload after settings save, got %s", rec.Body.String())
	}
	if !env.boot.RestartRequired() {
		t.Fatal("expected restart_required after settings save")
	}
	select {
	case <-env.restarted:
	case <-time.After(time.Second):
		t.Fatal("expected Restart after put settings")
	}

	got := loadSettings(t, env)
	if got.BlobRoot != newRoot {
		t.Fatalf("blob root: got %q want %q", got.BlobRoot, newRoot)
	}
	if got.DBDriver != "sqlite" {
		t.Fatalf("db driver must stay sqlite, got %q", got.DBDriver)
	}
	if got.DBDSN != env.dsn {
		t.Fatalf("db dsn must stay, got %q", got.DBDSN)
	}
	if !got.ComfyMock {
		t.Fatal("comfy_mock must stay as stored, not follow the settings form")
	}
	if got.ComfyUIBaseURL != "http://127.0.0.1:8188" {
		t.Fatalf("comfy url is not a settings field, got %q", got.ComfyUIBaseURL)
	}
}

type wizardEnv struct {
	t         *testing.T
	dir       string
	boot      *bootstrap.Store
	router    http.Handler
	token     string
	username  string
	dsn       string
	restarted chan struct{}
}

func (e *wizardEnv) auth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
}

func completeWizard(t *testing.T) *wizardEnv {
	t.Helper()
	dir := t.TempDir()
	boot, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	sess := setup.NewSessions("")
	restarted := make(chan struct{}, 2)
	h := &setup.Handler{
		Boot:         boot,
		Sessions:     sess,
		DataDir:      dir,
		RestartAfter: time.Millisecond,
		Restart:      func() { restarted <- struct{}{} },
	}
	r := chi.NewRouter()
	r.Use((&setup.Gate{Boot: boot, Sessions: sess}).Middleware)
	r.Route("/api/v1/setup", h.Mount)

	loginBody, _ := json.Marshal(map[string]string{
		"username": creds.Username,
		"password": creds.Password,
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/login", bytes.NewReader(loginBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}
	auth := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		req.Header.Set("Content-Type", "application/json")
	}

	pw, _ := json.Marshal(map[string]string{"new_password": "new-secret-9"})
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

	okDraft, _ := json.Marshal(settings.Settings{
		Placement:      settings.PlacementLocal,
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       filepath.Join(dir, "blob"),
		DBDriver:       "sqlite",
		DBDSN:          dsn,
		ComfyMock:      true,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
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
	select {
	case <-restarted:
	case <-time.After(time.Second):
		t.Fatal("expected Restart after finalize")
	}

	return &wizardEnv{
		t:         t,
		dir:       dir,
		boot:      boot,
		router:    r,
		token:     loginResp.Token,
		username:  creds.Username,
		dsn:       dsn,
		restarted: restarted,
	}
}

func envWithoutDB(t *testing.T) *wizardEnv {
	t.Helper()
	dir := t.TempDir()
	boot, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	sess := setup.NewSessions("")
	h := &setup.Handler{Boot: boot, Sessions: sess, DataDir: dir}
	r := chi.NewRouter()
	r.Use((&setup.Gate{Boot: boot, Sessions: sess}).Middleware)
	r.Route("/api/v1/setup", h.Mount)

	loginBody, _ := json.Marshal(map[string]string{
		"username": creds.Username,
		"password": creds.Password,
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/login", bytes.NewReader(loginBody)))
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}
	auth := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		req.Header.Set("Content-Type", "application/json")
	}
	pw, _ := json.Marshal(map[string]string{"new_password": "new-secret-9"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/password", bytes.NewReader(pw))
	auth(req)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("password: %d %s", rec.Code, rec.Body.String())
	}
	return &wizardEnv{
		t:        t,
		dir:      dir,
		boot:     boot,
		router:   r,
		token:    loginResp.Token,
		username: creds.Username,
	}
}

func loadSettings(t *testing.T, env *wizardEnv) settings.Settings {
	t.Helper()
	key, err := env.boot.EncKey()
	if err != nil {
		t.Fatal(err)
	}
	driver, dsn, err := env.boot.AppDB()
	if err != nil {
		t.Fatal(err)
	}
	gdb, err := db.Open(db.Options{Driver: driver, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	st, err := settings.NewStore(gdb, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestDatabaseUnreachableDoesNotChangeAppDB(t *testing.T) {
	env := completeWizard(t)
	body, _ := json.Marshal(map[string]string{
		"driver": "mysql",
		"dsn":    "user:pass@tcp(127.0.0.1:1)/pixoma",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/database", bytes.NewReader(body))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d %s", rec.Code, rec.Body.String())
	}
	driver, dsn, err := env.boot.AppDB()
	if err != nil {
		t.Fatal(err)
	}
	if driver != "sqlite" || dsn != env.dsn {
		t.Fatalf("app db changed: driver=%q dsn=%q", driver, dsn)
	}
}

func TestBlobTest_LocalFSSuccess(t *testing.T) {
	env := completeWizard(t)
	body, _ := json.Marshal(map[string]any{
		"blob_driver": "localfs",
		"blob_root":   t.TempDir(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(body))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("local fs: %d %s", rec.Code, rec.Body.String())
	}
}

func TestBlobTest_BucketNotFoundAndCreate(t *testing.T) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	srv := httptest.NewServer(faker.Server())
	t.Cleanup(srv.Close)

	env := completeWizard(t)
	cfg := map[string]any{
		"blob_driver":     "s3",
		"blob_endpoint":   srv.URL,
		"blob_region":     "us-east-1",
		"blob_bucket":     "pixoma-new",
		"blob_access_key": "AKIA_TEST",
		"blob_secret_key": "testsecret",
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(body))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"bucket_not_found"`) {
		t.Fatalf("want bucket_not_found, got %d %s", rec.Code, rec.Body.String())
	}

	cfg["auto_create_bucket"] = true
	body, _ = json.Marshal(cfg)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(body))
	env.auth(req)
	rec = httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
}

func TestBlobTest_SharedFSSuccess(t *testing.T) {
	env := completeWizard(t)
	body, _ := json.Marshal(map[string]any{
		"blob_driver": "sharedfs",
		"blob_root":   t.TempDir(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/blob-test", bytes.NewReader(body))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("sharedfs: %d %s", rec.Code, rec.Body.String())
	}
}

func TestDraft_LazyConfiguresDatabase(t *testing.T) {
	env := envWithoutDB(t)
	dsn := filepath.Join(env.dir, "app.db")
	body, _ := json.Marshal(settings.Settings{
		Placement:      settings.PlacementLocal,
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       filepath.Join(env.dir, "blob"),
		DBDriver:       "sqlite",
		DBDSN:          dsn,
		ComfyMock:      true,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/draft", bytes.NewReader(body))
	env.auth(req)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("draft: %d %s", rec.Code, rec.Body.String())
	}
	driver, gotDSN, err := env.boot.AppDB()
	if err != nil || driver != "sqlite" || gotDSN != dsn {
		t.Fatalf("app db not configured: driver=%q dsn=%q err=%v", driver, gotDSN, err)
	}
}
