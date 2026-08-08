package main

import (
	"path/filepath"
	"testing"
)

func TestResolveDSN_PrefersConfigured(t *testing.T) {
	t.Setenv("DATA_DIR", "should-not-win")
	got := resolveDSN("file:shared/app.db?cache=shared")
	if got != "file:shared/app.db?cache=shared" {
		t.Fatalf("resolveDSN(configured) = %q, want configured DSN", got)
	}
}

func TestResolveDSN_FallsBackToDataDirAppDB(t *testing.T) {
	t.Setenv("DATA_DIR", "custom-data")
	got := resolveDSN("")
	want := filepath.Join("custom-data", "app.db")
	if got != want {
		t.Fatalf("resolveDSN(\"\") = %q, want %q", got, want)
	}
}

func TestResolveDSN_DefaultDataDirWhenUnset(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	got := resolveDSN("")
	want := filepath.Join("data", "app.db")
	if got != want {
		t.Fatalf("resolveDSN(\"\") with empty DATA_DIR = %q, want %q", got, want)
	}
}
