package bootstrap_test

import (
	"path/filepath"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
)

func TestEnsureAgentToken_MintsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st, _, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	tok1, minted1, err := st.EnsureAgentToken()
	if err != nil {
		t.Fatal(err)
	}
	if !minted1 || tok1 == "" {
		t.Fatalf("expected mint, got minted=%v tok=%q", minted1, tok1)
	}
	ok, err := st.VerifyAgentToken(tok1)
	if err != nil || !ok {
		t.Fatalf("verify: ok=%v err=%v", ok, err)
	}

	tok2, minted2, err := st.EnsureAgentToken()
	if err != nil {
		t.Fatal(err)
	}
	if minted2 {
		t.Fatal("second EnsureAgentToken must not remint")
	}
	if tok2 != "" {
		t.Fatal("plaintext only returned on mint")
	}
	okBad, err := st.VerifyAgentToken("wrong")
	if err != nil {
		t.Fatal(err)
	}
	if okBad {
		t.Fatal("wrong token must fail")
	}
}
