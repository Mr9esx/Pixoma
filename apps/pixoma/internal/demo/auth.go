package demo

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	"github.com/go-chi/chi/v5"
)

const (
	demoUsername = "admin"
	demoPassword = "123456"
)

type accountKey struct{}

type demoAccount struct {
	Username string
	Nickname string
	Role     string
	LiveDemo bool
}

type AuthStore struct {
	mu       sync.RWMutex
	sessions map[string]demoAccount
}

func NewAuthStore() *AuthStore {
	return &AuthStore{sessions: make(map[string]demoAccount)}
}

func (s *AuthStore) issue(username string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = demoAccount{
		Username: username,
		Nickname: "Live Demo Admin",
		Role:     "admin",
		LiveDemo: true,
	}
	return token, nil
}

func (s *AuthStore) lookup(token string) (demoAccount, bool) {
	if token == "" {
		return demoAccount{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.sessions[token]
	return account, ok
}

func (s *AuthStore) revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *AuthStore) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := setup.TokenFromRequest(r)
		account, ok := s.lookup(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}
		session := setup.AccountSession{AccountID: "demo-admin", Username: account.Username, Role: account.Role}
		next.ServeHTTP(w, r.WithContext(setup.WithAccount(r.Context(), session)))
	})
}

func accountFromContext(r *http.Request) (demoAccount, bool) {
	session, ok := setup.AccountFromContext(r.Context())
	if !ok {
		return demoAccount{}, false
	}
	account := demoAccount{Username: session.Username, Nickname: "Live Demo Admin", Role: session.Role, LiveDemo: true}
	return account, ok
}

type SetupHandler struct {
	auth *AuthStore
}

func NewSetupHandler(auth *AuthStore) *SetupHandler {
	return &SetupHandler{auth: auth}
}

func Mount(r chi.Router, auth *AuthStore) {
	handler := NewSetupHandler(auth)
	r.Use(Readonly)
	r.Get("/status", handler.status)
	r.Post("/login", handler.login)
	r.Group(func(protected chi.Router) {
		protected.Use(auth.Handler)
		protected.Get("/me", handler.me)
		protected.Post("/logout", handler.logout)
	})
	r.Post("/password", writeReadonly)
}

func (h *SetupHandler) status(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.auth.lookup(setup.TokenFromRequest(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"initialized":          true,
		"authenticated":        authenticated,
		"must_change_password": false,
		"username":             demoUsername,
		"live_demo":            true,
	})
}

func (h *SetupHandler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json", "invalid_json")
		return
	}
	if body.Username != demoUsername || body.Password != demoPassword {
		writeError(w, http.StatusUnauthorized, "invalid credentials", "invalid_credentials")
		return
	}
	token, err := h.auth.issue(body.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "internal_error")
		return
	}
	setup.SetCookie(w, r, token, body.Remember)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                   true,
		"token":                token,
		"username":             body.Username,
		"must_change_password": false,
		"initialized":          true,
		"live_demo":            true,
	})
}

func (h *SetupHandler) me(w http.ResponseWriter, r *http.Request) {
	account, ok := accountFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"username":   account.Username,
		"nickname":   account.Nickname,
		"role":       account.Role,
		"email":      "",
		"avatar_url": "",
		"live_demo":  account.LiveDemo,
	})
}

func (h *SetupHandler) logout(w http.ResponseWriter, r *http.Request) {
	h.auth.revoke(setup.TokenFromRequest(r))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
