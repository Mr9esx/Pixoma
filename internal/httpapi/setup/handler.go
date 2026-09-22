package setup

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	consoledomain "github.com/Mr9esx/Pixoma/internal/adminusers/domain"
	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/factory"
	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/response"
	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
	settingsinfra "github.com/Mr9esx/Pixoma/internal/settings/infrastructure"
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
	// ConsoleUsers authenticates login/change-password against system accounts
	// once the platform is initialized. Nil falls back to bootstrap auth.
	ConsoleUsers consoledomain.Repository
	// LoginAttempts and RegistrationAttempts are lazily initialized in-process
	// fixed-window limiters. They are safe for the single-process default.
	LoginAttempts        *AttemptLimiter
	RegistrationAttempts *AttemptLimiter
}

// MountAuth serves console self-registration under /api/v1/auth.
func (h *Handler) MountAuth(r chi.Router) {
	r.Get("/registration", h.registrationStatus)
	r.Post("/register", h.register)
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/status", h.status)
	r.Get("/me", h.me)
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Post("/password", h.password)
	r.Post("/profile", h.profile)
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

// me returns the authenticated console user's display profile so the admin UI
// can render the sidebar user card. Falls back to bootstrap admin profile when
// the platform has not been migrated to console accounts yet.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	acct, ok := h.Sessions.LookupAccount(TokenFromRequest(r))
	if !ok {
		response.Fail(w, apierr.ErrSetupSessionUnauthorized, "unauthorized")
		return
	}
	nickname := acct.Username
	role := acct.Role
	if role == "" {
		role = consoledomain.RoleAdmin
	}
	email, avatarURL := "", ""
	if h.ConsoleUsers != nil {
		if u, err := h.ConsoleUsers.GetByUsername(r.Context(), acct.Username); err == nil {
			// 平滑升级：把尚未绑定账号身份的会话（如初始化向导签发）就地补绑，
			// 避免前端仍需重新登录才能获得完整角色权限。
			if acct.AccountID == "" {
				_ = h.Sessions.BindAccount(TokenFromRequest(r), u.ID, u.Role)
			}
			if u.Nickname != "" {
				nickname = u.Nickname
			}
			if u.Role != "" {
				role = u.Role
			}
			email, avatarURL = u.Email, u.AvatarURL
		}
	} else {
		if pn, pe, pa := h.Boot.AdminProfile(); pn != "" || pa != "" {
			nickname, email, avatarURL = pn, pe, pa
		}
	}
	response.OKStatus(w, http.StatusOK, map[string]any{
		"username":   acct.Username,
		"nickname":   nickname,
		"role":       role,
		"email":      email,
		"avatar_url": avatarURL,
	})
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	user, authed := h.user(r)
	response.OKStatus(w, http.StatusOK, statusDTO{
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
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	limiter := h.loginLimiter()
	loginKey := clientKey(r, strings.TrimSpace(body.Username))
	if !limiter.Allowed(loginKey) {
		response.Fail(w, apierr.ErrSetupLoginTooManyRequests, "too many login attempts")
		return
	}
	initialized := h.Boot.Initialized()
	var tok string
	var mustChange bool
	if initialized && h.ConsoleUsers != nil {
		u, err := h.ConsoleUsers.GetByUsername(r.Context(), body.Username)
		if err != nil {
			limiter.Record(loginKey)
			response.Fail(w, apierr.ErrSetupLoginUnauthorized, "invalid credentials")
			return
		}
		if !u.Enabled {
			limiter.Record(loginKey)
			response.Fail(w, apierr.ErrSetupAccountDisabled, "account disabled")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)); err != nil {
			limiter.Record(loginKey)
			response.Fail(w, apierr.ErrSetupLoginUnauthorized, "invalid credentials")
			return
		}
		limiter.Reset(loginKey)
		mustChange = u.MustChangePassword
		if !mustChange {
			u.LastLoginAt = time.Now().UTC()
			_ = h.ConsoleUsers.Update(r.Context(), u)
		}
		tok, err = h.Sessions.IssueAccount(u.Username, u.ID, u.Role, body.Remember)
		if err != nil {
			response.FailErr(w, apierr.ErrSetupLoginFailed, err)
			return
		}
		body.Username = u.Username
	} else {
		ok, err := h.Boot.VerifyPassword(body.Username, body.Password)
		if err != nil {
			response.FailErr(w, apierr.ErrSetupLoginFailed, err)
			return
		}
		if !ok {
			limiter.Record(loginKey)
			response.Fail(w, apierr.ErrSetupLoginUnauthorized, "invalid credentials")
			return
		}
		limiter.Reset(loginKey)
		tok, err = h.Sessions.Issue(body.Username, body.Remember)
		if err != nil {
			response.FailErr(w, apierr.ErrSetupLoginFailed, err)
			return
		}
		mustChange = h.Boot.MustChangePassword()
	}
	SetCookie(w, r, tok, body.Remember)
	response.OKStatus(w, http.StatusOK, map[string]any{
		"ok":                   true,
		"token":                tok,
		"username":             body.Username,
		"must_change_password": mustChange,
		"initialized":          initialized,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.Sessions.Revoke(TokenFromRequest(r))
	ClearCookie(w, r)
	response.OKStatus(w, http.StatusOK, map[string]any{"ok": true})
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
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	initialized := h.Boot.Initialized()
	var err error
	var consolePasswordChanged bool
	if initialized && h.ConsoleUsers != nil {
		err = h.setConsolePassword(r.Context(), user, body.OldPassword, body.NewPassword)
	} else if h.Boot.MustChangePassword() {
		err = h.Boot.SetPassword(user, body.NewPassword)
		consolePasswordChanged = true
	} else {
		err = h.Boot.ChangePassword(user, body.OldPassword, body.NewPassword)
		consolePasswordChanged = true
	}
	if err != nil {
		if errors.Is(err, bootstrap.ErrInvalidCredentials) {
			response.FailErr(w, apierr.ErrSetupSessionUnauthorized, err)
			return
		}
		if errors.Is(err, bootstrap.ErrWeakPassword) {
			response.Fail(w, apierr.ErrSetupChangePasswordPasswordTooShort, "password must be at least 8 characters")
			return
		}
		if errors.Is(err, bootstrap.ErrPasswordAlreadySet) {
			response.Fail(w, apierr.ErrSetupChangePasswordPasswordAlreadySet, "password already set")
			return
		}
		response.FailErr(w, apierr.ErrSetupChangePasswordFailed, err)
		return
	}
	// 未走 console 分支（如初始化向导改密）时，bootstrap 是唯一密码源，
	// 需把新密码同步到 console_users，否则初始化完成后登录会按旧值校验而失败。
	if consolePasswordChanged {
		if err := h.syncConsolePassword(r.Context(), user, body.NewPassword); err != nil {
			slog.Error("sync console password failed", "error", err)
		}
	}
	if !initialized {
		_ = h.Boot.SetWizardStep("profile")
	}
	// 改密成功后清除落盘的初始明文密码，旧默认密码不再可恢复。
	if err := bootstrap.RemoveStoredPassword(filepath.Join(h.DataDir, "bootstrap.db")); err != nil {
		// 仅记录，不阻断响应。
		_ = err
	}
	response.OKStatus(w, http.StatusOK, map[string]any{"ok": true, "must_change_password": false})
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
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	driver := strings.ToLower(strings.TrimSpace(body.Driver))
	if driver == "" {
		driver = settingsdomain.DriverSQLite
	}
	dsn := strings.TrimSpace(body.DSN)
	if dsn == "" {
		response.Fail(w, apierr.ErrSetupTestDatabaseDSNRequired, "dsn required")
		return
	}
	if driver == settingsdomain.DriverSQLite {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil && filepath.Dir(dsn) != "." {
			response.FailErr(w, apierr.ErrSetupTestDatabaseOpenFailed, err)
			return
		}
	}
	gdb, err := h.openDB(driver, dsn)
	if err != nil {
		response.Fail(w, apierr.ErrSetupTestDatabaseUnreachable, "database unreachable: "+err.Error())
		return
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		response.FailErr(w, apierr.ErrSetupTestDatabaseOpenFailed, err)
		return
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		response.Fail(w, apierr.ErrSetupTestDatabasePingFailed, "database ping failed: "+err.Error())
		return
	}
	_ = sqlDB.Close()
	if err := h.Boot.SetAppDB(driver, dsn); err != nil {
		response.FailErr(w, apierr.ErrSetupTestDatabaseFailed, err)
		return
	}
	_ = h.Boot.SetWizardStep("placement")
	response.OKStatus(w, http.StatusOK, map[string]any{"ok": true, "driver": driver})
}

func (h *Handler) draft(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	var body settingsdomain.Settings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil {
		response.FailErr(w, apierr.ErrSetupSaveDraftFailed, err)
		return
	}
	if strings.TrimSpace(dsn) == "" {
		d := strings.ToLower(strings.TrimSpace(body.DBDriver))
		if d == "" {
			d = settingsdomain.DriverSQLite
		}
		dbDSN := strings.TrimSpace(body.DBDSN)
		if dbDSN == "" {
			response.Fail(w, apierr.ErrSetupSaveDraftDatabaseNotConfigured, "configure database first")
			return
		}
		if err := db.EnsureDatabase(d, dbDSN); err != nil {
			response.Fail(w, apierr.ErrSetupTestDatabaseUnreachable, "database unreachable: "+err.Error())
			return
		}
		if err := h.Boot.SetAppDB(d, dbDSN); err != nil {
			response.FailErr(w, apierr.ErrSetupSaveDraftFailed, err)
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
		response.FailErr(w, apierr.ErrSetupSaveDraftInvalid, err)
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		response.FailErr(w, apierr.ErrSetupSaveDraftInvalid, err)
		return
	}
	defer func() { _ = cleanup() }()
	if err := st.Save(body); err != nil {
		response.FailErr(w, apierr.ErrSetupSaveDraftInvalid, err)
		return
	}
	step := "storage"
	if body.Placement == settingsdomain.PlacementRemote {
		step = "edge"
	}
	_ = h.Boot.SetWizardStep(step)
	response.OKStatus(w, http.StatusOK, map[string]any{
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
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	driver := strings.ToLower(strings.TrimSpace(body.BlobDriver))
	placement := settingsdomain.PlacementLocal
	if driver == "s3" || driver == "tos" || driver == "sharedfs" {
		placement = settingsdomain.PlacementRemote
	}
	cfg := settingsdomain.Settings{
		Placement:      placement,
		DBDriver:       settingsdomain.DriverSQLite,
		DBDSN:          "data/app.db",
		BlobDriver:     driver,
		BlobRoot:       body.BlobRoot,
		BlobEndpoint:   body.BlobEndpoint,
		BlobRegion:     body.BlobRegion,
		BlobBucket:     body.BlobBucket,
		BlobAccessKey:  body.BlobAccessKey,
		BlobSecretKey:  body.BlobSecretKey,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
	}
	if err := cfg.Validate(); err != nil {
		response.FailErr(w, apierr.ErrSetupTestBlobConfigInvalid, err)
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
		response.Fail(w, apierr.ErrSetupBlobBucketMissing, body.BlobBucket)
		return
	}
	if errors.Is(err, blob.ErrBucketNotFound) {
		err = factory.EnsureBucket(r.Context(), opts)
	}
	if err != nil {
		response.Fail(w, apierr.ErrSetupTestBlobCheckFailed, "blob check failed: "+err.Error())
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		response.OKStatus(w, http.StatusOK, map[string]any{"configured": false, "public_url": h.PublicURL})
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		response.FailErr(w, apierr.ErrSetupLoadSettingsInvalid, err)
		return
	}
	defer func() { _ = cleanup() }()
	got, err := st.Load()
	if err != nil {
		response.OKStatus(w, http.StatusOK, map[string]any{"configured": false, "public_url": h.PublicURL})
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]any{
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
		response.Fail(w, apierr.ErrSetupSaveSettingsSetupNotFinalized, "finalize setup first")
		return
	}
	var body settingsdomain.Settings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		response.Fail(w, apierr.ErrSetupSaveDraftDatabaseNotConfigured, "configure database first")
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		response.FailErr(w, apierr.ErrSetupSaveSettingsInvalid, err)
		return
	}
	defer func() { _ = cleanup() }()
	existing, err := st.Load()
	if err != nil {
		response.Fail(w, apierr.ErrSetupSaveSettingsSettingsNotSaved, "save settings first")
		return
	}
	merged := mergePlatformSettings(existing, body)
	merged.AllowSelfRegistration = body.AllowSelfRegistration
	if strings.TrimSpace(body.DefaultUserAccess) != "" {
		merged.DefaultUserAccess = settingsdomain.NormalizeDefaultUserAccess(body.DefaultUserAccess)
	}
	merged.DBDriver = driver
	merged.DBDSN = dsn
	if err := merged.Validate(); err != nil {
		response.FailErr(w, apierr.ErrSetupSaveSettingsInvalid, err)
		return
	}
	if err := st.Save(merged); err != nil {
		response.FailErr(w, apierr.ErrSetupSaveSettingsInvalid, err)
		return
	}
	if err := h.Boot.SetRestartRequired(true); err != nil {
		response.FailErr(w, apierr.ErrSetupSaveSettingsFailed, err)
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]any{
		"ok":               true,
		"restart_required": true,
		"restarting":       h.Restart != nil,
		"message":          "reloading pixoma",
	})
	h.scheduleRestart()
}

func mergePlatformSettings(existing, in settingsdomain.Settings) settingsdomain.Settings {
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
	out.MediaMaxBytes = in.MediaMaxBytes
	if strings.TrimSpace(in.DefaultUserAccess) != "" {
		out.DefaultUserAccess = settingsdomain.NormalizeDefaultUserAccess(in.DefaultUserAccess)
	}
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
		response.Fail(w, apierr.ErrSetupFinalizeDefaultPasswordUnchanged, "change default password first")
		return
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		response.Fail(w, apierr.ErrSetupSaveDraftDatabaseNotConfigured, "configure database first")
		return
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		response.FailErr(w, apierr.ErrSetupFinalizeInvalid, err)
		return
	}
	defer func() { _ = cleanup() }()
	cfg, err := st.Load()
	if err != nil {
		response.Fail(w, apierr.ErrSetupSaveSettingsSettingsNotSaved, "save settings first")
		return
	}
	if err := cfg.Validate(); err != nil {
		response.FailErr(w, apierr.ErrSetupFinalizeInvalid, err)
		return
	}
	if err := h.Boot.MarkInitialized(); err != nil {
		response.FailErr(w, apierr.ErrSetupFinalizeFailed, err)
		return
	}
	if err := h.Boot.SetRestartRequired(true); err != nil {
		response.FailErr(w, apierr.ErrSetupFinalizeFailed, err)
		return
	}
	_ = h.Boot.SetWizardStep("done")
	response.OKStatus(w, http.StatusOK, map[string]any{
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
	acct, ok := resolveAccount(r.Context(), h.Sessions, h.ConsoleUsers, TokenFromRequest(r))
	if !ok {
		return "", false
	}
	return acct.Username, true
}

func (h *Handler) requireSession(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, ok := h.user(r)
	if !ok {
		response.Fail(w, apierr.ErrSetupSessionUnauthorized, "unauthorized")
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

func (h *Handler) settingsStore(driver, dsn string) (*settingsinfra.Store, func() error, error) {
	key, err := h.Boot.EncKey()
	if err != nil {
		return nil, nil, err
	}
	gdb, err := h.openDB(driver, dsn)
	if err != nil {
		return nil, nil, err
	}
	st, err := settingsinfra.NewStore(gdb, key)
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

// setConsolePassword changes a console account password. When the account still
// must change password (e.g. admin reset), the old password is not required.
func (h *Handler) setConsolePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return bootstrap.ErrWeakPassword
	}
	u, err := h.ConsoleUsers.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, consoledomain.ErrNotFound) {
			return bootstrap.ErrInvalidCredentials
		}
		return err
	}
	if !u.MustChangePassword {
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
			return bootstrap.ErrInvalidCredentials
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.MustChangePassword = false
	return h.ConsoleUsers.Update(ctx, u)
}

// syncConsolePassword mirrors a bootstrap-side password change onto the console
// account row (if one exists), so login — which validates against console_users
// once the platform is initialized — accepts the new secret.
func (h *Handler) syncConsolePassword(ctx context.Context, username, plain string) error {
	if h.ConsoleUsers == nil {
		return nil
	}
	u, err := h.ConsoleUsers.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, consoledomain.ErrNotFound) {
			return nil
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.MustChangePassword = false
	return h.ConsoleUsers.Update(ctx, u)
}

// register creates a console account (role Viewer) when self-registration is
// enabled, then signs in the new account.
// profile stores the first-run admin「如何称呼您」profile (nickname/email/avatar).
// It persists in the bootstrap store and, when the console account already
// exists, mirrors onto the migrated console_users row.
func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSession(w, r); !ok {
		return
	}
	var body struct {
		Nickname  string `json:"nickname"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	body.Email = strings.TrimSpace(body.Email)
	if body.Email != "" && !validEmail(body.Email) {
		response.Fail(w, apierr.ErrSetupSaveProfileInvalidEmail, "invalid email")
		return
	}
	if err := h.Boot.SetAdminProfile(body.Nickname, body.Email, body.AvatarURL); err != nil {
		response.FailErr(w, apierr.ErrSetupSaveProfileFailed, err)
		return
	}
	if h.ConsoleUsers != nil {
		username, _, _ := h.Boot.AdminAccount()
		if u, err := h.ConsoleUsers.GetByUsername(r.Context(), username); err == nil {
			u.Nickname = body.Nickname
			u.Email = body.Email
			u.AvatarURL = body.AvatarURL
			_ = h.ConsoleUsers.Update(r.Context(), u)
		}
	}
	if !h.Boot.Initialized() {
		_ = h.Boot.SetWizardStep("database")
	}
	response.OKStatus(w, http.StatusOK, map[string]any{"ok": true})
}

// registrationStatus reports whether public self-registration is currently on.
func (h *Handler) registrationStatus(w http.ResponseWriter, r *http.Request) {
	response.OKStatus(w, http.StatusOK, map[string]bool{"enabled": h.selfRegistrationEnabled()})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if !h.selfRegistrationEnabled() {
		response.Fail(w, apierr.ErrSetupRegisterRegistrationDisabled, "registration disabled")
		return
	}
	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Nickname string `json:"nickname"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Fail(w, apierr.ErrSetupLoginInvalidJSON, "invalid json")
		return
	}
	limiter := h.registrationLimiter()
	registrationKey := clientKey(r, "")
	if !limiter.Allowed(registrationKey) {
		response.Fail(w, apierr.ErrSetupRegisterTooManyRequests, "too many registrations")
		return
	}
	limiter.Record(registrationKey)
	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.TrimSpace(body.Email)
	if body.Username == "" {
		response.Fail(w, apierr.ErrSetupRegisterAccountNameRequired, "账号名不能为空")
		return
	}
	if len(body.Password) < 8 {
		response.Fail(w, apierr.ErrSetupChangePasswordPasswordTooShort, "password must be at least 8 characters")
		return
	}
	if body.Email != "" && !validEmail(body.Email) {
		response.Fail(w, apierr.ErrSetupSaveProfileInvalidEmail, "invalid email")
		return
	}
	var newUser *consoledomain.ConsoleUser
	if h.ConsoleUsers != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			response.Fail(w, apierr.ErrSetupRegisterHashPasswordFailed, "failed to hash password")
			return
		}
		newUser = &consoledomain.ConsoleUser{
			Username:     body.Username,
			Email:        body.Email,
			Nickname:     body.Nickname,
			Role:         consoledomain.RoleViewer,
			Enabled:      true,
			PasswordHash: string(hash),
		}
		if err := h.ConsoleUsers.Create(r.Context(), newUser); err != nil {
			if errors.Is(err, consoledomain.ErrDuplicate) {
				response.Fail(w, apierr.ErrSetupRegisterAlreadyTaken, "username or email already taken")
				return
			}
			response.Fail(w, apierr.ErrSetupRegisterCreateFailed, "create failed")
			return
		}
	} else {
		response.Fail(w, apierr.ErrSetupRegisterServiceUnavailable, "unavailable")
		return
	}
	tok, err := h.Sessions.IssueAccount(newUser.Username, newUser.ID, newUser.Role, false)
	if err != nil {
		response.FailErr(w, apierr.ErrSetupRegisterFailed, err)
		return
	}
	SetCookie(w, r, tok, false)
	response.OKStatus(w, http.StatusOK, map[string]any{
		"ok": true, "token": tok, "username": newUser.Username, "role": newUser.Role,
	})
}

func (h *Handler) loginLimiter() *AttemptLimiter {
	if h.LoginAttempts == nil {
		h.LoginAttempts = NewAttemptLimiter(5, time.Minute)
	}
	return h.LoginAttempts
}

func (h *Handler) registrationLimiter() *AttemptLimiter {
	if h.RegistrationAttempts == nil {
		h.RegistrationAttempts = NewAttemptLimiter(10, time.Minute)
	}
	return h.RegistrationAttempts
}

// selfRegistrationEnabled reports whether the "open registration" setting is on.
func (h *Handler) selfRegistrationEnabled() bool {
	if h == nil || h.Boot == nil || !h.Boot.Initialized() {
		return false
	}
	driver, dsn, err := h.Boot.AppDB()
	if err != nil || strings.TrimSpace(dsn) == "" {
		return false
	}
	st, cleanup, err := h.settingsStore(driver, dsn)
	if err != nil {
		return false
	}
	defer func() { _ = cleanup() }()
	cfg, err := st.Load()
	if err != nil {
		return false
	}
	return cfg.AllowSelfRegistration
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

const maskedSecret = "********"
