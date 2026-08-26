package setup

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
)

// Handler serves login + wizard APIs under /api/v1/setup.
type Handler struct {
	Boot     *bootstrap.Store
	Sessions *Sessions
	DataDir  string
	// PublicURL is the externally reachable control-plane address that edge
	// agents should connect to (PUBLIC_URL / HTTP_ADDR). It is exposed to the
	// admin UI so generated deploy commands never point at a frontend origin
	// that cannot serve the agent API (e.g. Vite dev on localhost:5173).
	PublicURL string
	// OpenBusiness is used after a successful DB ping so settings can be saved.
	OpenBusiness func(driver, dsn string) (*gorm.DB, error)
	// Restart reloads pixoma after finalize. Nil skips auto-reload (tests).
	Restart func()
	// RestartAfter delays Restart so the HTTP response can flush. Zero means 400ms.
	RestartAfter time.Duration
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/status", h.status)
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Post("/password", h.password)
	r.Post("/database", h.database)
	r.Post("/draft", h.draft)
	r.Post("/blob-test", h.blobTest)
	r.Post("/finalize", h.finalize)
	r.Get("/settings", h.getSettings)
	r.Put("/settings", h.putSettings)
}

type statusDTO struct {
	Initialized        bool   `json:"initialized"`
	Authenticated      bool   `json:"authenticated"`
	MustChangePassword bool   `json:"must_change_password"`
	Username           string `json:"username,omitempty"`
	WizardStep         string `json:"wizard_step,omitempty"`
	RestartRequired    bool   `json:"restart_required,omitempty"`
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	user, authed := h.user(r)
	writeJSON(w, http.StatusOK, statusDTO{
		Initialized:        h.Boot.Initialized(),
		Authenticated:      authed,
		MustChangePassword: h.Boot.MustChangePassword(),
		Username:           user,
		WizardStep:         h.Boot.WizardStep(),
		RestartRequired:    h.Boot.RestartRequired(),
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	ok, err := h.Boot.VerifyPassword(body.Username, body.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, err := h.Sessions.Issue(body.Username, body.Remember)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	SetCookie(w, tok, body.Remember)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                   true,
		"token":                tok,
		"username":             body.Username,
		"must_change_password": h.Boot.MustChangePassword(),
		"initialized":          h.Boot.Initialized(),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.Sessions.Revoke(TokenFromRequest(r))
	ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) password(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireSession(w, r)
	if !ok {
		return
	}
	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	var err error
	if h.Boot.MustChangePassword() {
		err = h.Boot.SetPassword(user, body.NewPassword)
	} else {
		err = h.Boot.ChangePassword(user, body.OldPassword, body.NewPassword)
	}
	if err != nil {
		if errors.Is(err, bootstrap.ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, bootstrap.ErrWeakPassword) {
			writeErr(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		if errors.Is(err, bootstrap.ErrPasswordAlreadySet) {
			writeErr(w, http.StatusBadRequest, "password already set")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !h.Boot.Initialized() {
		_ = h.Boot.SetWizardStep("database")
	}
	// 改密成功后清除落盘的初始明文密码，旧默认密码不再可恢复。
	if err := bootstrap.RemoveStoredPassword(filepath.Join(h.DataDir, "bootstrap.db")); err != nil {
		// 仅记录，不阻断响应。
		_ = err
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "must_change_password": false})
}

func (h *Handler) database(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	var body struct {
		Driver string `json:"driver"`
		DSN    string `json:"dsn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	driver := strings.ToLower(strings.TrimSpace(body.Driver))
	if driver == "" {
		driver = settings.DriverSQLite
	}
	dsn := strings.TrimSpace(body.DSN)
	if dsn == "" {
		writeErr(w, http.StatusBadRequest, "dsn required")
		return
	}
	if driver == settings.DriverSQLite {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil && filepath.Dir(dsn) != "." {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	gdb, err := h.openDB(driver, dsn)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "database unreachable: "+err.Error())
		return
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		writeErr(w, http.StatusBadRequest, "database ping failed: "+err.Error())
		return
	}
	_ = sqlDB.Close()
	if err := h.Boot.SetAppDB(driver, dsn); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.Boot.SetWizardStep("placement")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "driver": driver})
}

func (h *Handler) draft(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	var body settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.TrimSpace(dsn) == "" {
		d := strings.ToLower(strings.TrimSpace(body.DBDriver))
		if d == "" {
			d = settings.DriverSQLite
		}
		dbDSN := strings.TrimSpace(body.DBDSN)
		if dbDSN == "" {
			writeErr(w, http.StatusBadRequest, "configure database first")
			return
		}
		if err := db.EnsureDatabase(d, dbDSN); err != nil {
			writeErr(w, http.StatusBadRequest, "database unreachable: "+err.Error())
			return
		}
		if err := h.Boot.SetAppDB(d, dbDSN); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		driver, dsn = d, dbDSN
	}
	if strings.TrimSpace(body.DBDriver) == "" {
		body.DBDriver = driver
	}
	if strings.TrimSpace(body.DBDSN) == "" {
		body.DBDSN = dsn
	}
	if err := body.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = cleanup() }()
	if err := st.Save(body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	step := "storage"
	if body.Placement == settings.PlacementRemote {
		step = "edge"
	}
	_ = h.Boot.SetWizardStep(step)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"placement": body.Placement,
	})
}

func (h *Handler) blobTest(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	var body struct {
		BlobDriver       string `json:"blob_driver"`
		BlobRoot         string `json:"blob_root"`
		BlobEndpoint     string `json:"blob_endpoint"`
		BlobRegion       string `json:"blob_region"`
		BlobBucket       string `json:"blob_bucket"`
		BlobAccessKey    string `json:"blob_access_key"`
		BlobSecretKey    string `json:"blob_secret_key"`
		AutoCreateBucket bool   `json:"auto_create_bucket"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	driver := strings.ToLower(strings.TrimSpace(body.BlobDriver))
	placement := settings.PlacementLocal
	if driver == "s3" || driver == "tos" || driver == "sharedfs" {
		placement = settings.PlacementRemote
	}
	cfg := settings.Settings{
		Placement:      placement,
		DBDriver:       settings.DriverSQLite,
		DBDSN:          "data/app.db",
		BlobDriver:     driver,
		BlobRoot:       body.BlobRoot,
		BlobEndpoint:   body.BlobEndpoint,
		BlobRegion:     body.BlobRegion,
		BlobBucket:     body.BlobBucket,
		BlobAccessKey:  body.BlobAccessKey,
		BlobSecretKey:  body.BlobSecretKey,
		ComfyMock:      true,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
	}
	if err := cfg.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	opts := factory.CheckOptions{
		Driver:    driver,
		LocalRoot: body.BlobRoot,
		Endpoint:  body.BlobEndpoint,
		Region:    body.BlobRegion,
		Bucket:    body.BlobBucket,
		AccessKey: body.BlobAccessKey,
		SecretKey: body.BlobSecretKey,
	}
	err := factory.Check(r.Context(), opts)
	if errors.Is(err, blob.ErrBucketNotFound) && !body.AutoCreateBucket {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     false,
			"code":   "bucket_not_found",
			"bucket": body.BlobBucket,
		})
		return
	}
	if errors.Is(err, blob.ErrBucketNotFound) {
		err = factory.EnsureBucket(r.Context(), opts)
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, "blob check failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "public_url": h.PublicURL})
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = cleanup() }()
	got, err := st.Load()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "public_url": h.PublicURL})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"settings":   got,
		"public_url": h.PublicURL,
	})
}

func (h *Handler) putSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	if !h.Boot.Initialized() {
		writeErr(w, http.StatusBadRequest, "finalize setup first")
		return
	}
	var body settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		writeErr(w, http.StatusBadRequest, "configure database first")
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = cleanup() }()
	existing, err := st.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "save settings first")
		return
	}
	merged := mergePlatformSettings(existing, body)
	merged.DBDriver = driver
	merged.DBDSN = dsn
	if err := merged.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := st.Save(merged); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Boot.SetRestartRequired(true); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"restart_required": true,
		"restarting":       h.Restart != nil,
		"message":          "reloading pixoma",
	})
	h.scheduleRestart()
}

func mergePlatformSettings(existing, in settings.Settings) settings.Settings {
	out := existing
	if p := strings.TrimSpace(in.Placement); p != "" {
		out.Placement = p
	}
	if b := strings.TrimSpace(in.BlobDriver); b != "" {
		out.BlobDriver = b
	}
	out.BlobRoot = in.BlobRoot
	out.BlobEndpoint = in.BlobEndpoint
	out.BlobRegion = in.BlobRegion
	out.BlobBucket = in.BlobBucket
	out.BlobAccessKey = unmaskSecret(in.BlobAccessKey)
	out.BlobSecretKey = unmaskSecret(in.BlobSecretKey)
	out.ProxyKind = in.ProxyKind
	out.ProxyHost = in.ProxyHost
	out.ProxyPort = in.ProxyPort
	return out
}

func unmaskSecret(s string) string {
	if s == maskedSecret {
		return ""
	}
	return s
}

func (h *Handler) finalize(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	if h.Boot.MustChangePassword() {
		writeErr(w, http.StatusBadRequest, "change default password first")
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		writeErr(w, http.StatusBadRequest, "configure database first")
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = cleanup() }()
	cfg, err := st.Load()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "save settings first")
		return
	}
	if err := cfg.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Boot.MarkInitialized(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.Boot.SetRestartRequired(true); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.Boot.SetWizardStep("done")
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"initialized":      true,
		"restart_required": true,
		"restarting":       h.Restart != nil,
		"message":          "reloading pixoma",
	})
	h.scheduleRestart()
}

func (h *Handler) scheduleRestart() {
	if h == nil || h.Restart == nil {
		return
	}
	delay := h.RestartAfter
	if delay <= 0 {
		delay = 400 * time.Millisecond
	}
	time.AfterFunc(delay, h.Restart)
}

func (h *Handler) user(r *http.Request) (string, bool) {
	return h.Sessions.Lookup(TokenFromRequest(r))
}

func (h *Handler) requireSession(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, ok := h.user(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return user, true
}

func (h *Handler) openDB(driver, dsn string) (*gorm.DB, error) {
	if err := db.EnsureDatabase(driver, dsn); err != nil {
		return nil, err
	}
	if h.OpenBusiness != nil {
		return h.OpenBusiness(driver, dsn)
	}
	return db.Open(db.Options{Driver: driver, DSN: dsn})
}

func (h *Handler) settingsStore(driver, dsn string) (*settings.Store, func() error, error) {
	key, err := h.Boot.EncKey()
	if err != nil {
		return nil, nil, err
	}
	gdb, err := h.openDB(driver, dsn)
	if err != nil {
		return nil, nil, err
	}
	st, err := settings.NewStore(gdb, key)
	if err != nil {
		if sqlDB, e := gdb.DB(); e == nil {
			_ = sqlDB.Close()
		}
		return nil, nil, err
	}
	cleanup := func() error {
		sqlDB, e := gdb.DB()
		if e != nil {
			return e
		}
		return sqlDB.Close()
	}
	return st, cleanup, nil
}

const maskedSecret = "********"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
