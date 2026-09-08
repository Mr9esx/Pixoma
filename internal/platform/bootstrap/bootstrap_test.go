package bootstrap_test

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
)

func TestOpen_CreatesUninitializedStoreWithDefaultAdmin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bootstrap.db")

	st, creds, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if st.Initialized() {
		t.Fatal("expected uninitialized")
	}
	if creds.Username == "" || creds.Password == "" {
		t.Fatalf("expected default credentials, got %+v", creds)
	}
	if !st.MustChangePassword() {
		t.Fatal("expected must change password")
	}
	ok, err := st.VerifyPassword(creds.Username, creds.Password)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("default password should verify")
	}
}

func TestOpen_IdempotentSamePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st1, c1, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st1.Close()

	st2, c2, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()

	if c2.Password != "" {
		t.Fatal("re-open must not mint a new plaintext password")
	}
	if c2.Username != c1.Username {
		t.Fatalf("username changed: %q vs %q", c1.Username, c2.Username)
	}
	ok, err := st2.VerifyPassword(c1.Username, c1.Password)
	if err != nil || !ok {
		t.Fatalf("original password should still work: ok=%v err=%v", ok, err)
	}
}

func TestMarkInitialized_AndGate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st, _, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.EnsureReadyForBusiness(); err == nil {
		t.Fatal("expected error when uninitialized")
	}
	if err := st.MarkInitialized(); err != nil {
		t.Fatal(err)
	}
	if !st.Initialized() {
		t.Fatal("expected initialized")
	}
	if err := st.EnsureReadyForBusiness(); err != nil {
		t.Fatal(err)
	}
}

func TestRestartRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st, _, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if st.RestartRequired() {
		t.Fatal("fresh store must not require restart")
	}
	if err := st.SetRestartRequired(true); err != nil {
		t.Fatal(err)
	}
	if !st.RestartRequired() {
		t.Fatal("expected restart required")
	}
	if err := st.SetRestartRequired(false); err != nil {
		t.Fatal(err)
	}
	if st.RestartRequired() {
		t.Fatal("cleared flag should be false")
	}
}

func TestChangePassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st, creds, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.ChangePassword(creds.Username, creds.Password, "new-secret-pass"); err != nil {
		t.Fatal(err)
	}
	if st.MustChangePassword() {
		t.Fatal("must-change should clear after change")
	}
	ok, err := st.VerifyPassword(creds.Username, "new-secret-pass")
	if err != nil || !ok {
		t.Fatalf("new password: ok=%v err=%v", ok, err)
	}
	ok, err = st.VerifyPassword(creds.Username, creds.Password)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("old password must fail")
	}
}

func TestSetPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st, creds, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.SetPassword(creds.Username, "new-secret-pass"); err != nil {
		t.Fatal(err)
	}
	if st.MustChangePassword() {
		t.Fatal("must-change should clear after set")
	}
	ok, err := st.VerifyPassword(creds.Username, "new-secret-pass")
	if err != nil || !ok {
		t.Fatalf("new password: ok=%v err=%v", ok, err)
	}
	if err := st.SetPassword(creds.Username, "another-secret"); !errors.Is(err, bootstrap.ErrPasswordAlreadySet) {
		t.Fatalf("second set: %v", err)
	}
}

func TestSetAdminProfile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bootstrap.db")
	st, _, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	if err := st.SetAdminProfile(" 小P ", " a@b.com ", " http://x/a.png "); err != nil {
		t.Fatalf("set profile: %v", err)
	}
	nick, email, avatar := st.AdminProfile()
	if nick != "小P" || email != "a@b.com" || avatar != "http://x/a.png" {
		t.Fatalf("profile trimmed round-trip wrong: %q %q %q", nick, email, avatar)
	}
}

func TestOpenUsesPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data", "bootstrap.db")
	st, _, err := bootstrap.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("data dir mode = %o, want 0700", info.Mode().Perm())
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("bootstrap mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestEncKeyUsesEnvironmentOverride(t *testing.T) {
	dir := t.TempDir()
	st, _, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	want := make([]byte, 32)
	for i := range want {
		want[i] = byte(i + 1)
	}
	t.Setenv("PIXOMA_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString(want))
	got, err := st.EncKey()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("key mismatch: got %v", got)
	}
}
