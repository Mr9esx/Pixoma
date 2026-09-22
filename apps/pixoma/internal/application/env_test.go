package application

import "testing"

func TestOptionsFromEnvUsesPortWhenHTTPAddrIsUnset(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("PORT", "30808")
	t.Setenv("PUBLIC_URL", "")

	opts := OptionsFromEnv()
	if opts.HTTPAddr != "127.0.0.1:30808" {
		t.Fatalf("HTTPAddr = %q, want %q", opts.HTTPAddr, "127.0.0.1:30808")
	}
	if opts.PublicURL != "http://127.0.0.1:30808" {
		t.Fatalf("PublicURL = %q, want %q", opts.PublicURL, "http://127.0.0.1:30808")
	}
}

func TestOptionsFromEnvPrefersHTTPAddrOverPort(t *testing.T) {
	t.Setenv("HTTP_ADDR", "0.0.0.0:30809")
	t.Setenv("PORT", "30808")

	opts := OptionsFromEnv()
	if opts.HTTPAddr != "0.0.0.0:30809" {
		t.Fatalf("HTTPAddr = %q, want explicit HTTP_ADDR", opts.HTTPAddr)
	}
}
