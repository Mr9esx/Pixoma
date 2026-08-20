package setup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
)

// TestRememberMe_SurvivesRestart proves the core "remember me" promise: a
// remember-me session issued before a "restart" (new Sessions instance backed by
// the same store path) is still valid afterwards, while an ordinary session is
// not.
func TestRememberMe_SurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, setup.SessionStoreFile())

	s1 := setup.NewSessions(storePath)

	rememberTok, err := s1.Issue("admin", true)
	if err != nil {
		t.Fatalf("issue remember token: %v", err)
	}
	plainTok, err := s1.Issue("admin", false)
	if err != nil {
		t.Fatalf("issue plain token: %v", err)
	}

	// Both are valid in the original instance.
	if u, ok := s1.Lookup(rememberTok); !ok || u != "admin" {
		t.Fatalf("remember token invalid in s1: ok=%v u=%q", ok, u)
	}
	if _, ok := s1.Lookup(plainTok); !ok {
		t.Fatalf("plain token invalid in s1")
	}

	// Simulate restart: new instance, same store.
	s2 := setup.NewSessions(storePath)

	if u, ok := s2.Lookup(rememberTok); !ok || u != "admin" {
		t.Fatalf("remember token lost after restart: ok=%v u=%q", ok, u)
	}
	if _, ok := s2.Lookup(plainTok); ok {
		t.Fatalf("plain (non-remember) token survived restart; should be gone")
	}
}

// TestRememberMe_RevokeRemovesFromStore verifies that revoking a remember-me
// session also removes it from the on-disk store, so a later instance cannot
// resurrect it.
func TestRememberMe_RevokeRemovesFromStore(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, setup.SessionStoreFile())

	s1 := setup.NewSessions(storePath)
	tok, err := s1.Issue("admin", true)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	s1.Revoke(tok)

	s2 := setup.NewSessions(storePath)
	if _, ok := s2.Lookup(tok); ok {
		t.Fatalf("revoked remember token still valid after restart")
	}
}

// TestRememberMe_StoreFilePermission ensures the store file is written
// owner-only.
func TestRememberMe_StoreFilePermission(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, setup.SessionStoreFile())

	s := setup.NewSessions(storePath)
	if _, err := s.Issue("admin", true); err != nil {
		t.Fatalf("issue token: %v", err)
	}

	fi, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("stat store: %v", err)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		t.Fatalf("store permissions too open: %v", perm)
	}
}
