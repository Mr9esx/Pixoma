package tg

import (
	"strings"
	"testing"
)

func TestPhotoUploadNameUsesBasename(t *testing.T) {
	got := photoUploadName("outputs/t1/0_out.png")
	if got != "0_out.png" {
		t.Fatalf("got %q, want 0_out.png", got)
	}
	if strings.Contains(got, "/") {
		t.Fatalf("upload name must not contain /: %q", got)
	}
	if photoUploadName("") != "result.png" {
		t.Fatalf("empty key fallback: %q", photoUploadName(""))
	}
}
