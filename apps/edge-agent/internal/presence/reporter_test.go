package presence_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/controlplane"
	"github.com/Mr9esx/Pixoma/apps/edge-agent/internal/presence"
	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui"
	"github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/comfyui/comfyuitest"
)

func TestReporter_FakeComfyReportsRunning(t *testing.T) {
	var running any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		running = body["comfy_running"]
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	comfy := &comfyuitest.Fake{}
	r := &presence.Reporter{
		Client: controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		Comfy:  comfy,
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if running != true {
		t.Fatalf("fake must report comfy_running=true, got %v", running)
	}
}

func TestReporter_SendsStartedAtAndComfyVersion(t *testing.T) {
	var startedAt any
	var version any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		startedAt = body["started_at"]
		version = body["comfy_version"]
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	comfy := &comfyuitest.Fake{}
	r := &presence.Reporter{
		Client: controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		Comfy:  comfy,
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	first, ok := startedAt.(string)
	if !ok || first == "" {
		t.Fatalf("started_at=%v", startedAt)
	}
	if version != "fake" {
		t.Fatalf("comfy_version=%v want fake", version)
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if second, ok := startedAt.(string); !ok || second != first {
		t.Fatalf("started_at must stay the first heartbeat: first=%q got=%v", first, startedAt)
	}
}

func TestReporter_UnreachableComfyReportsFalse(t *testing.T) {
	var running any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		running = body["comfy_running"]
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dead.Close()
	comfy := comfyui.NewHTTP(dead.URL)
	r := &presence.Reporter{
		Client: controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		Comfy:  comfy,
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if running != false {
		t.Fatalf("unreachable comfy must report comfy_running=false, got %v", running)
	}
}

func TestReporter_SendsHardwareFirstThenOmits(t *testing.T) {
	var bodies []map[string]any
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		n++
		if n == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"refresh_hardware": false})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"refresh_hardware": false})
	}))
	t.Cleanup(srv.Close)
	r := &presence.Reporter{
		Client:       controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		SendHardware: true,
		Collect: func(ctx context.Context) edge.Hardware {
			return edge.Hardware{CPUModel: "Intel"}
		},
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("reports=%d", len(bodies))
	}
	if _, ok := bodies[0]["hardware"]; !ok {
		t.Fatalf("first body missing hardware: %v", bodies[0])
	}
	if _, ok := bodies[1]["hardware"]; ok {
		t.Fatalf("second body should omit hardware: %v", bodies[1])
	}
}

func TestReporter_ResendsHardwareWhenRefreshRequested(t *testing.T) {
	var bodies []map[string]any
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		n++
		refresh := n == 1
		_ = json.NewEncoder(w).Encode(map[string]any{"refresh_hardware": refresh})
	}))
	t.Cleanup(srv.Close)
	r := &presence.Reporter{
		Client:       controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		SendHardware: true,
		Collect: func(ctx context.Context) edge.Hardware {
			return edge.Hardware{CPUModel: "Intel"}
		},
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := bodies[0]["hardware"]; !ok {
		t.Fatalf("first missing hardware: %v", bodies[0])
	}
	if _, ok := bodies[1]["hardware"]; !ok {
		t.Fatalf("second should include hardware after refresh: %v", bodies[1])
	}
}

func TestReporter_MetricsCadence(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]any{"refresh_hardware": false})
	}))
	t.Cleanup(srv.Close)
	r := &presence.Reporter{
		Client:          controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		MetricsInterval: 30 * time.Second,
		Sample: func(ctx context.Context, since time.Time) edge.Metrics {
			return edge.Metrics{CPUUsagePercent: 10, CollectedAt: time.Now().UTC()}
		},
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("reports=%d", len(bodies))
	}
	if _, ok := bodies[0]["metrics"]; !ok {
		t.Fatalf("first report must carry metrics: %v", bodies[0])
	}
	if _, ok := bodies[1]["metrics"]; ok {
		t.Fatalf("second report before interval must omit metrics: %v", bodies[1])
	}
}

func TestReporter_FirstSampleImmediate(t *testing.T) {
	// 首次上报（lastMetrics 零值）必须立即采样；且 since 传零值
	var gotSince time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	r := &presence.Reporter{
		Client:          controlplane.NewClient(srv.URL, "tok", "gpu-1"),
		MetricsInterval: time.Hour,
		Sample: func(_ context.Context, since time.Time) edge.Metrics {
			gotSince = since
			return edge.Metrics{CPUUsagePercent: 10, CollectedAt: time.Now().UTC()}
		},
	}
	if err := r.ProbeAndReport(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !gotSince.IsZero() {
		t.Fatalf("first sample must receive zero since, got %v", gotSince)
	}
}
