package tg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestTelegramFileHTTPClientHasTimeout(t *testing.T) {
	if telegramFileHTTPClient.Timeout != 30*time.Second {
		t.Fatalf("Timeout=%v, want 30s", telegramFileHTTPClient.Timeout)
	}
}

func TestFetchTelegramFileBytesRejectsOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		chunk := make([]byte, 64*1024)
		remaining := maxTelegramFileBytes + 1
		for remaining > 0 {
			n := len(chunk)
			if n > remaining {
				n = remaining
			}
			if _, err := w.Write(chunk[:n]); err != nil {
				return
			}
			remaining -= n
		}
	}))
	defer srv.Close()

	_, err := fetchTelegramFileBytes(context.Background(), srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("want error when body exceeds size limit")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("want size-limit error, got %v", err)
	}
}

func TestFetchTelegramFileBytesOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("png-bytes"))
	}))
	defer srv.Close()

	got, err := fetchTelegramFileBytes(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "png-bytes" {
		t.Fatalf("got %q", got)
	}
}
