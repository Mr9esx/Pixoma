package media

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// mediaPrefix is the logical blob key prefix for admin workflow preview media,
// kept separate from the runtime inputs/jobs/outputs keyspaces.
const mediaPrefix = "previews/"

// DefaultMaxBytes caps a single preview media upload.
const DefaultMaxBytes int64 = 25 << 20 // 25 MiB

// extToMIME maps the allow-listed preview media types.
var extToMIME = map[string]string{
	"png":  "image/png",
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"webp": "image/webp",
	"gif":  "image/gif",
	"mp4":  "video/mp4",
	"webm": "video/webm",
}

// Handler serves admin media upload + authenticated preview retrieval,
// backed by blob.Store (non-public: mounted behind the admin auth gate).
type Handler struct {
	Blob     blob.Store
	MaxBytes int64
}

func (h *Handler) maxBytes() int64 {
	if h.MaxBytes <= 0 {
		return DefaultMaxBytes
	}
	return h.MaxBytes
}

// Mount registers media routes under /api/v1/media.
func (h *Handler) Mount(r chi.Router) {
	r.Post("/", h.Upload)
	r.Get("/previews/{file}", h.ServePreview)
}

type uploadResponse struct {
	Key  string `json:"key"`
	URL  string `json:"url"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}

// Upload accepts a single multipart file, validates it against the media
// allow-list and size cap, persists it to blob.Store under previews/ and
// returns an authenticated, served URL.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	max := h.maxBytes()

	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "请求必须是 multipart/form-data")
		return
	}
	part, err := reader.NextPart()
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺少上传文件")
		return
	}
	defer part.Close()

	var buf bytes.Buffer
	n, err := io.Copy(&buf, io.LimitReader(part, max+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "读取上传文件失败")
		return
	}
	if n > max {
		writeError(w, http.StatusRequestEntityTooLarge, "上传文件超过大小上限")
		return
	}

	ext := extOf(part.FileName())
	contentType := extToMIME[ext]
	if contentType == "" {
		writeError(w, http.StatusBadRequest, "不支持的文件类型，仅支持图片（png/jpeg/webp/gif）与视频（mp4/webm）")
		return
	}

	key := mediaPrefix + uuid.NewString() + "." + ext
	ref, err := h.Blob.Put(r.Context(), key, bytes.NewReader(buf.Bytes()), blob.PutOptions{MIME: contentType})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "媒体保存失败")
		return
	}

	writeJSON(w, http.StatusCreated, uploadResponse{
		Key:  ref.Key,
		URL:  "/api/v1/media/" + ref.Key,
		MIME: ref.MIME,
		Size: ref.Size,
	})
}

// ServePreview streams a preview media object from blob.Store with a correct
// Content-Type. Only previews/ keys are reachable.
func (h *Handler) ServePreview(w http.ResponseWriter, r *http.Request) {
	file := chi.URLParam(r, "file")
	if file == "" || strings.Contains(file, "/") || strings.Contains(file, "..") {
		writeError(w, http.StatusNotFound, "媒体不存在")
		return
	}
	ext := extOf(file)
	contentType := extToMIME[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	rc, err := h.Blob.Get(r.Context(), sharedkernel.BlobRef{Key: mediaPrefix + file})
	if err != nil {
		writeError(w, http.StatusNotFound, "媒体不存在")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// MIMETypeFor returns the allow-listed content type for an extension, or ""
// when the extension is not allowed. Exposed for validation/testing helpers.
func extOf(filename string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
}

func MIMETypeFor(filename string) string {
	return extToMIME[extOf(filename)]
}
