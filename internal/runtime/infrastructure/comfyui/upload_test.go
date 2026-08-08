package comfyui_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestMock_UploadImageReturnsStableFilename(t *testing.T) {
	m := &comfyui.Mock{}
	name, err := m.UploadImage(context.Background(), "user.png", "image/png", []byte("png-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "mock-upload.png" {
		t.Fatalf("got %q, want mock-upload.png", name)
	}
}

func TestHTTP_UploadImagePostsMultipartAndParsesName(t *testing.T) {
	const wantName = "uploaded-ref.png"
	var gotFilename string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/upload/image" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Errorf("Content-Type=%q err=%v", r.Header.Get("Content-Type"), err)
			http.Error(w, "bad content type", http.StatusBadRequest)
			return
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		part, err := mr.NextPart()
		if err != nil {
			t.Errorf("NextPart: %v", err)
			http.Error(w, "no part", http.StatusBadRequest)
			return
		}
		defer part.Close()
		gotFilename = part.FileName()
		gotBody, err = io.ReadAll(part)
		if err != nil {
			t.Errorf("ReadAll: %v", err)
			http.Error(w, "read", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":      wantName,
			"subfolder": "",
			"type":      "input",
		})
	}))
	t.Cleanup(srv.Close)

	h := comfyui.NewHTTP(srv.URL)
	payload := []byte("fake-png-data")
	name, err := h.UploadImage(context.Background(), "local.png", "image/png", payload)
	if err != nil {
		t.Fatal(err)
	}
	if name != wantName {
		t.Fatalf("got %q, want %q", name, wantName)
	}
	if gotFilename != "local.png" {
		t.Fatalf("multipart filename=%q want local.png", gotFilename)
	}
	if !bytes.Equal(gotBody, payload) {
		t.Fatalf("multipart body=%q want %q", gotBody, payload)
	}
}

func TestNewClient_MockUploadImage(t *testing.T) {
	c, err := comfyui.NewClient(comfyui.Options{Mock: true})
	if err != nil {
		t.Fatal(err)
	}
	name, err := c.UploadImage(context.Background(), "a.png", "image/png", []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if name != "mock-upload.png" {
		t.Fatalf("got %q, want mock-upload.png", name)
	}
}
