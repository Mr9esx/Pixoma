package pull_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/apps/edge-agent/internal/pull"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/localfs"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestClient_ClaimUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	c := pull.NewClient(srv.URL, "bad", "gpu-1")
	_, err := c.Claim(context.Background(), 0)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
}

func TestClient_ClaimEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c := pull.NewClient(srv.URL, "tok", "gpu-1")
	job, err := c.Claim(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if job != nil {
		t.Fatalf("want nil, got %+v", job)
	}
}

func TestLoop_ClaimExecuteReportStatus(t *testing.T) {
	ctx := context.Background()
	var claimed atomic.Bool
	var statuses []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/agent/v1/jobs/claim":
			if claimed.Load() {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			claimed.Store(true)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task_id": "t-edge",
				"edge_id": "gpu-1",
				"job_ref": map[string]any{"key": "jobs/t-edge/job.json"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/agent/v1/jobs/t-edge/status":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if s, ok := body["status"].(string); ok {
				statuses = append(statuses, s)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/agent/v1/jobs/t-edge/heartbeat":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	store, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	job := actuator.JobPackage{
		TaskID:   "t-edge",
		EdgeID:   "gpu-1",
		Workflow: map[string]any{"1": map[string]any{"inputs": map[string]any{}}},
	}
	raw, _ := json.Marshal(job)
	if _, err := store.Put(ctx, "jobs/t-edge/job.json", bytes.NewReader(raw), blob.PutOptions{MIME: "application/json"}); err != nil {
		t.Fatal(err)
	}

	client := pull.NewClient(srv.URL, "secret", "gpu-1")
	worker := &actuator.Worker{
		EdgeID: "gpu-1",
		Comfy:  &comfyui.Mock{},
		Blob:   store,
		Status: pull.NewStatusPublisher(client),
		Now:    func() time.Time { return time.Unix(1, 0).UTC() },
	}
	loop := &pull.Loop{
		Client: client,
		Worker: worker,
		Wait:   0,
		Once:   true,
	}
	if err := loop.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !claimed.Load() {
		t.Fatal("expected claim")
	}
	foundOK := false
	for _, s := range statuses {
		if s == string(sharedkernel.TaskSucceeded) {
			foundOK = true
		}
	}
	if !foundOK {
		t.Fatalf("want succeeded status, got %v", statuses)
	}
}

func TestClient_ReportPresence(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/agent/v1/presence" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c := pull.NewClient(srv.URL, "tok", "gpu-1")
	refresh, err := c.ReportPresence(context.Background(), true, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if refresh {
		t.Fatal("204 should mean refresh=false")
	}
	if got["edge_id"] != "gpu-1" || got["comfy_running"] != true {
		t.Fatalf("%v", got)
	}
	if _, ok := got["hardware"]; ok {
		t.Fatalf("nil hardware must omit key: %v", got)
	}
}

func TestClient_ReportPresence_SendsHardwareAndReadsRefresh(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"refresh_hardware": true})
	}))
	t.Cleanup(srv.Close)
	c := pull.NewClient(srv.URL, "tok", "gpu-1")
	hw := edge.Hardware{CPUModel: "Intel"}
	refresh, err := c.ReportPresence(context.Background(), false, &hw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !refresh {
		t.Fatal("expected refresh_hardware")
	}
	raw, ok := got["hardware"].(map[string]any)
	if !ok || raw["cpu_model"] != "Intel" {
		t.Fatalf("hardware=%v", got["hardware"])
	}
}

func TestClient_ReportPresence_SendsMetrics(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	c := pull.NewClient(srv.URL, "tok", "gpu-1")
	usage := 42.5
	m := edge.Metrics{
		CPUUsagePercent: usage,
		CollectedAt:     time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC),
	}
	if _, err := c.ReportPresence(context.Background(), true, nil, &m); err != nil {
		t.Fatal(err)
	}
	raw, ok := got["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("metrics missing: %v", got)
	}
	if raw["cpu_usage_percent"] != 42.5 {
		t.Fatalf("cpu_usage_percent=%v", raw["cpu_usage_percent"])
	}
}
