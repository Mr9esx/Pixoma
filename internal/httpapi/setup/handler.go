package setup

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
)

// Handler serves login + wizard APIs under /api/v1/setup.
type Handler struct {
	Boot     *bootstrap.Store
	Sessions *Sessions
	DataDir  string
	// OpenBusiness is used after a successful DB ping so settings can be saved.
	OpenBusiness func(driver, dsn string) (*gorm.DB, error)
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/status", h.status)
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Post("/password", h.password)
	r.Post("/database", h.database)
	r.Post("/draft", h.draft)
	r.Post("/finalize", h.finalize)
	r.Get("/settings", h.getSettings)
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
	tok, err := h.Sessions.Issue(body.Username)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	SetCookie(w, tok)
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
	if err := h.Boot.ChangePassword(user, body.OldPassword, body.NewPassword); err != nil {
		if errors.Is(err, bootstrap.ErrInvalidCredentials) {
			writeErr(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, bootstrap.ErrWeakPassword) {
			writeErr(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.Boot.SetWizardStep("database")
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
		writeErr(w, http.StatusBadRequest, "configure database first")
		return
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

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false})
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
		writeJSON(w, http.StatusOK, map[string]any{"configured": false})
		return
	}
	got.TelegramBotToken = mask(got.TelegramBotToken)
	got.BlobAccessKey = mask(got.BlobAccessKey)
	got.BlobSecretKey = mask(got.BlobSecretKey)
	writeJSON(w, http.StatusOK, map[string]any{"configured": true, "settings": got})
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
		"message":          "settings saved; restart pixoma for them to take effect",
	})
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

func mask(s string) string {
	if s == "" {
		return ""
	}
	return "********"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
