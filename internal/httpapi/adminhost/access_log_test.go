package adminhost_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Mr9esx/Pixoma/internal/httpapi/adminhost"
)

func TestAccessLogRecordsSafeRouteAndRequestID(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	router := chi.NewRouter()
	router.Use(middleware.RequestID, adminhost.AccessLog)
	inner := chi.NewRouter()
	inner.Use(middleware.RequestID)
	var innerID string
	inner.Route("/api/v1/tasks", func(r chi.Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			innerID = middleware.GetReqID(r.Context())
			w.WriteHeader(http.StatusInternalServerError)
		})
	})
	router.Mount("/", inner)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/private-task?token=secret", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Header().Get("X-Request-Id") == "" || innerID != recorder.Header().Get("X-Request-Id") {
		t.Fatal("请求编号未在路由之间传递")
	}
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["level"] != "ERROR" || entry["route"] != "/api/v1/tasks/{id}" || entry["status"] != float64(http.StatusInternalServerError) || entry["request_id"] != recorder.Header().Get("X-Request-Id") {
		t.Fatalf("请求日志字段不正确：%s", output.String())
	}
	if bytes.Contains(output.Bytes(), []byte("private-task")) || bytes.Contains(output.Bytes(), []byte("secret")) {
		t.Fatalf("请求日志包含请求内容：%s", output.String())
	}
}
