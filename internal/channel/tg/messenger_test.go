package tg

import (
	"testing"
)

func TestMimeFromURLExt(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"png lowercase", "https://a/x.png", "image/png"},
		{"jpg with query", "https://a/x.JPG?token=abc", "image/jpeg"},
		{"jpeg", "https://a/x.jpeG", "image/jpeg"},
		{"webp", "https://a/x.webp", "image/webp"},
		{"gif", "https://a/x.gif", "image/gif"},
		{"mp4", "https://a/x.mp4", "video/mp4"},
		{"webm", "https://a/x.webm", "video/webm"},
		{"mov", "https://a/x.mov", "video/quicktime"},
		{"no ext", "https://a/x", ""},
		{"unknown ext", "https://a/x.txt", ""},
		{"query only no ext", "https://a/x?foo=1", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mimeFromURLExt(tc.in)
			if got != tc.want {
				t.Fatalf("mimeFromURLExt(%q)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMediaKindToMIME(t *testing.T) {
	cases := map[string]string{
		"image":      "image/jpeg",
		"animation":  "image/gif",
		"video":      "video/mp4",
		"":           "",
		"unknown":    "",
		"VIDEO":      "",
	}
	for in, want := range cases {
		if got := mediaKindToMIME(in); got != want {
			t.Errorf("mediaKindToMIME(%q)=%q, want %q", in, got, want)
		}
	}
}
