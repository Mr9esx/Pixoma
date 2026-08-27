package setup_test

import (
	"path/filepath"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
)

func TestIssueAccount_LookupAccount(t *testing.T) {
	s := setup.NewSessions("")
	tok, err := s.IssueAccount("admin", "acct-9", "admin", false)
	if err != nil {
		t.Fatal(err)
	}
	acct, ok := s.LookupAccount(tok)
	if !ok {
		t.Fatal("token should resolve")
	}
	if acct.AccountID != "acct-9" || acct.Role != "admin" || acct.Username != "admin" {
		t.Fatalf("unexpected account session: %+v", acct)
	}
}

func TestIssueAccount_RememberSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	s1 := setup.NewSessions(path)
	tok, err := s1.IssueAccount("alice", "acct-1", "viewer", true)
	if err != nil {
		t.Fatal(err)
	}
	s2 := setup.NewSessions(path)
	acct, ok := s2.LookupAccount(tok)
	if !ok {
		t.Fatal("remember-me session should survive restart")
	}
	if acct.AccountID != "acct-1" || acct.Role != "viewer" {
		t.Fatalf("account fields lost after restart: %+v", acct)
	}
}

func TestLookupAccount_UnknownToken(t *testing.T) {
	s := setup.NewSessions("")
	if _, ok := s.LookupAccount("nope"); ok {
		t.Fatal("unknown token must not resolve")
	}
}
