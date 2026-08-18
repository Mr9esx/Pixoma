package comfyui_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
)

func TestMock_SystemStatsAndQueue(t *testing.T) {
	m := &comfyui.Mock{}
	st, err := m.SystemStats(context.Background())
	if err != nil || st == nil || !st.Mock || !st.Reachable {
		t.Fatalf("%+v %v", st, err)
	}
	devices, _ := st.Raw["devices"].([]any)
	if len(devices) == 0 {
		t.Fatal("mock SystemStats devices empty")
	}
	dev, _ := devices[0].(map[string]any)
	if _, ok := dev["vram_total"]; !ok {
		t.Fatalf("mock device missing vram_total: %+v", dev)
	}
	if st.ComfyUIVersion != "mock" {
		t.Fatalf("mock comfyui_version=%q want mock", st.ComfyUIVersion)
	}
	q, err := m.Queue(context.Background())
	if err != nil || q == nil || !q.Mock || !q.Reachable {
		t.Fatalf("%+v %v", q, err)
	}
	if q.Running == nil || q.Pending == nil {
		t.Fatalf("queue slices must be non-nil: running=%v pending=%v", q.Running, q.Pending)
	}
}

func TestHTTP_SystemStatsAndQueue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/system_stats", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"system": map[string]any{"comfyui_version": "0.1"},
		})
	})
	mux.HandleFunc("/queue", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"queue_running": []any{[]any{1, "prompt-a"}},
			"queue_pending": []any{[]any{2, "prompt-b"}},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := comfyui.NewHTTP(srv.URL)
	st, err := c.SystemStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.Reachable || st.Mock || st.Raw == nil {
		t.Fatalf("stats=%+v", st)
	}
	if st.ComfyUIVersion != "0.1" {
		t.Fatalf("comfyui_version=%q want 0.1", st.ComfyUIVersion)
	}
	q, err := c.Queue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !q.Reachable || len(q.Running) != 1 || len(q.Pending) != 1 {
		t.Fatalf("queue=%+v", q)
	}
}

func TestHTTP_SystemStatsUnreachable(t *testing.T) {
	c := comfyui.NewHTTP("http://127.0.0.1:1")
	c.Client = &http.Client{Timeout: 50 * time.Millisecond}
	st, err := c.SystemStats(context.Background())
	if err != nil {
		t.Fatalf("expected reachable=false without error, got err=%v", err)
	}
	if st == nil || st.Reachable || st.Error == "" {
		t.Fatalf("want unreachable with error, got %+v", st)
	}
}
