package mcp

import (
	"net/http"
	"strings"
	"sync"
)

func bindSSESessionUser(inner http.Handler) http.Handler {
	return &sseSessionGuard{inner: inner, owner: map[string]string{}}
}

type sseSessionGuard struct {
	inner http.Handler
	mu    sync.Mutex
	owner map[string]string
}

func (g *sseSessionGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, _ := IdentityFrom(r.Context())
	switch r.Method {
	case http.MethodPost:
		sid := r.URL.Query().Get("sessionid")
		g.mu.Lock()
		owner := g.owner[sid]
		g.mu.Unlock()
		if sid != "" && owner != "" && id.UserID != "" && owner != id.UserID {
			http.Error(w, GuideForbidden, http.StatusForbidden)
			return
		}
		g.inner.ServeHTTP(w, r)
	case http.MethodGet:
		if id.UserID == "" {
			g.inner.ServeHTTP(w, r)
			return
		}
		cap := &sseSessionCapture{
			ResponseWriter: w,
			onSession: func(sid string) {
				g.mu.Lock()
				g.owner[sid] = id.UserID
				g.mu.Unlock()
			},
		}
		defer func() {
			if cap.sessionID == "" {
				return
			}
			g.mu.Lock()
			delete(g.owner, cap.sessionID)
			g.mu.Unlock()
		}()
		g.inner.ServeHTTP(cap, r)
	default:
		g.inner.ServeHTTP(w, r)
	}
}

type sseSessionCapture struct {
	http.ResponseWriter
	onSession func(string)
	sessionID string
	buf       []byte
}

func (c *sseSessionCapture) Unwrap() http.ResponseWriter { return c.ResponseWriter }

func (c *sseSessionCapture) Write(p []byte) (int, error) {
	n, err := c.ResponseWriter.Write(p)
	if c.sessionID == "" {
		c.buf = append(c.buf, p...)
		if sid := sessionIDFromSSE(string(c.buf)); sid != "" {
			c.sessionID = sid
			if c.onSession != nil {
				c.onSession(sid)
			}
		}
	}
	return n, err
}

func sessionIDFromSSE(s string) string {
	const key = "sessionid="
	i := strings.Index(s, key)
	if i < 0 {
		return ""
	}
	rest := s[i+len(key):]
	cut := len(rest)
	for j, r := range rest {
		if r == '&' || r == '\n' || r == '\r' || r == ' ' {
			cut = j
			break
		}
	}
	return strings.TrimSpace(rest[:cut])
}
