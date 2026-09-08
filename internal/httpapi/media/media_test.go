package media

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
)

func newTestHandler(t *testing.T, max int64) http.Handler {
	t.Helper()
	st, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatalf("localfs: %v", err)
	}
	h := &Handler{Blob: st, MaxBytes: max}
	r := chi.NewRouter()
	r.Route("/api/v1/media", h.Mount)
	return r
}

func upload(t *testing.T, h http.Handler, filename string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = fw.Write(body)
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUploadPNGAndServe(t *testing.T) {
	h := newTestHandler(t, DefaultMaxBytes)
	rec := upload(t, h, "cover.png", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(resp.Key, "previews/") {
		t.Fatalf("key prefix = %q", resp.Key)
	}
	if !strings.HasSuffix(resp.Key, ".png") {
		t.Fatalf("key ext = %q", resp.Key)
	}
	if resp.MIME != "image/png" {
		t.Fatalf("mime = %q", resp.MIME)
	}

	get := httptest.NewRequest(http.MethodGet, resp.URL, nil)
	grec := httptest.NewRecorder()
	h.ServeHTTP(grec, get)
	if grec.Code != http.StatusOK {
		t.Fatalf("get status = %d", grec.Code)
	}
	if grec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("get content-type = %q", grec.Header().Get("Content-Type"))
	}
	got, _ := io.ReadAll(grec.Body)
	if !bytes.Equal(got, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Fatalf("round-trip bytes mismatch")
	}
}

func TestUploadVideoWebM(t *testing.T) {
	h := newTestHandler(t, DefaultMaxBytes)
	rec := upload(t, h, "demo.webm", []byte("webm-bytes"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp uploadResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.MIME != "video/webm" {
		t.Fatalf("mime = %q", resp.MIME)
	}
}

func TestUploadRejectsDisallowedType(t *testing.T) {
	h := newTestHandler(t, DefaultMaxBytes)
	rec := upload(t, h, "script.txt", []byte("hello"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	h := newTestHandler(t, 16)
	// PNG magic is irrelevant; size limit trips first.
	rec := upload(t, h, "big.png", bytes.Repeat([]byte{0x61}, 64))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestServeRejectsTraversal(t *testing.T) {
	h := newTestHandler(t, DefaultMaxBytes)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/previews/..%2fapp.db", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
