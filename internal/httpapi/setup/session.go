package setup

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	CookieName = "pixoma_session"

	// sessionTTL applies to ordinary (non-"remember me") sessions, kept in
	// memory only and lost on restart.
	sessionTTL = 12 * time.Hour

	// rememberTTL applies to "remember me" sessions, persisted to disk so they
	// survive a service restart (auto-login).
	rememberTTL = 30 * 24 * time.Hour

	// sessionStoreFile is the on-disk store for remember-me sessions, inside the
	// DATA_DIR passed to NewSessions. Tokens are stored hashed.
	sessionStoreFile = "sessions.json"
)

// Sessions issues and validates admin login tokens. Ordinary sessions live in
// memory only; "remember me" sessions are additionally persisted to storePath
// so they survive a service restart.
type Sessions struct {
	mu        sync.Mutex
	byID      map[string]session
	storePath string // "" = pure in-memory (no persistence)
}

type session struct {
	Username  string    `json:"username"`
	AccountID string    `json:"account_id,omitempty"`
	Role      string    `json:"role,omitempty"`
	Expires   time.Time `json:"expires"`
	Remember  bool      `json:"remember"`
}

// AccountSession is the account-scoped identity bound to a validated token.
type AccountSession struct {
	AccountID string
	Username  string
	Role      string
}

// SessionStoreFile returns the basename of the on-disk remember-me session
// store, so callers can build the full path as filepath.Join(dataDir,
// SessionStoreFile()).
func SessionStoreFile() string { return sessionStoreFile }

// NewSessions creates a session store. When storePath is non-empty it loads any
// previously persisted remember-me sessions (surviving restarts) and writes new
// remember-me sessions back to disk.
func NewSessions(storePath string) *Sessions {
	s := &Sessions{
		byID:      map[string]session{},
		storePath: storePath,
	}
	s.load()
	return s
}

func (s *Sessions) Issue(username string, remember bool) (plain string, err error) {
	var b [24]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b[:])
	ttl := sessionTTL
	if remember {
		ttl = rememberTTL
	}
	s.mu.Lock()
	s.byID[hashToken(plain)] = session{
		Username: username,
		Expires:  time.Now().Add(ttl),
		Remember: remember,
	}
	// Persist only remember-me sessions; ordinary ones stay in memory.
	persist := remember && s.storePath != ""
	s.mu.Unlock()
	if persist {
		if err := s.save(); err != nil {
			return "", err
		}
	}
	return plain, nil
}

// IssueAccount issues a session bound to a console account id and role.
func (s *Sessions) IssueAccount(username, accountID, role string, remember bool) (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(b[:])
	ttl := sessionTTL
	if remember {
		ttl = rememberTTL
	}
	s.mu.Lock()
	s.byID[hashToken(plain)] = session{
		Username:  username,
		AccountID: accountID,
		Role:      role,
		Expires:   time.Now().Add(ttl),
		Remember:  remember,
	}
	persist := remember && s.storePath != ""
	s.mu.Unlock()
	if persist {
		if err := s.save(); err != nil {
			return "", err
		}
	}
	return plain, nil
}

// LookupAccount resolves a token to its account identity. Tokens issued without
// an account id (e.g. legacy persisted sessions) return an empty AccountID/Role.
func (s *Sessions) LookupAccount(plain string) (acct AccountSession, ok bool) {
	if plain == "" {
		return AccountSession{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashToken(plain)
	row, exists := s.byID[key]
	if !exists || time.Now().After(row.Expires) {
		if exists {
			delete(s.byID, key)
			if row.Remember && s.storePath != "" {
				_ = s.save()
			}
		}
		return AccountSession{}, false
	}
	return AccountSession{
		AccountID: row.AccountID,
		Username:  row.Username,
		Role:      row.Role,
	}, true
}

func (s *Sessions) Lookup(plain string) (username string, ok bool) {
	if plain == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashToken(plain)
	row, exists := s.byID[key]
	if !exists || time.Now().After(row.Expires) {
		if exists {
			delete(s.byID, key)
			if row.Remember && s.storePath != "" {
				_ = s.save()
			}
		}
		return "", false
	}
	return row.Username, true
}

func (s *Sessions) Revoke(plain string) {
	if plain == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashToken(plain)
	row, exists := s.byID[key]
	if !exists {
		return
	}
	delete(s.byID, key)
	if row.Remember && s.storePath != "" {
		_ = s.save()
	}
}

func TokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(CookieName); err == nil {
		if v := c.Value; v != "" {
			return v
		}
	}
	const prefix = "Bearer "
	hdr := r.Header.Get("Authorization")
	if len(hdr) > len(prefix) && subtle.ConstantTimeCompare([]byte(hdr[:len(prefix)]), []byte(prefix)) == 1 {
		return hdr[len(prefix):]
	}
	return ""
}

func SetCookie(w http.ResponseWriter, token string, remember bool) {
	maxAge := int(sessionTTL.Seconds())
	if remember {
		maxAge = int(rememberTTL.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// save writes only remember-me sessions to disk (token hashes only), atomically
// and with 0600 permissions. Caller must hold s.mu.
func (s *Sessions) save() error {
	if s.storePath == "" {
		return nil
	}
	rows := map[string]session{}
	for k, v := range s.byID {
		if v.Remember && time.Now().Before(v.Expires) {
			rows[k] = v
		}
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := s.storePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.storePath)
}

// load restores persisted remember-me sessions from disk. Called once at
// construction.
func (s *Sessions) load() {
	if s.storePath == "" {
		return
	}
	b, err := os.ReadFile(s.storePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// Corrupt/unreadable store should not break startup; fall back empty.
			return
		}
		return
	}
	var rows map[string]session
	if err := json.Unmarshal(b, &rows); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range rows {
		if v.Remember && now.Before(v.Expires) {
			s.byID[k] = v
		}
	}
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
