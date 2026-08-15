package setup

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	CookieName = "pixoma_session"
	sessionTTL = 12 * time.Hour
)

type Sessions struct {
	mu   sync.Mutex
	byID map[string]session
}

type session struct {
	Username string
	Expires  time.Time
}

func NewSessions() *Sessions {
	return &Sessions{byID: map[string]session{}}
}

func (s *Sessions) Issue(username string) (plain string, err error) {
	var b [24]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[hashToken(plain)] = session{
		Username: username,
		Expires:  time.Now().Add(sessionTTL),
	}
	return plain, nil
}

func (s *Sessions) Lookup(plain string) (username string, ok bool) {
	if plain == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, exists := s.byID[hashToken(plain)]
	if !exists || time.Now().After(row.Expires) {
		if exists {
			delete(s.byID, hashToken(plain))
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
	delete(s.byID, hashToken(plain))
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

func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
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

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
