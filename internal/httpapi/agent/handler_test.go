package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/agent"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type statusSpy struct {
	events []sharedkernel.TaskStatusEvent
}

func (s *statusSpy) OnStatus(_ context.Context, ev sharedkernel.TaskStatusEvent) error {
	s.events = append(s.events, ev)
	return nil
}

func mountAgent(t *testing.T, token string, tasks runtimedomain.TaskRepository, status agent.StatusApplier) *httptest.Server {
	t.Helper()
	h := &agent.Handler{
		Token:  token,
		Tasks:  tasks,
		Status: status,
		Lease:  90 * time.Second,
		Now:    func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestAgent_PresenceStoresMetrics(t *testing.T) {
	dsn := "file:agent_metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	metricsRepo := instpersist.NewMetricsRepository(gdb, 24*time.Hour)
	h := &agent.Handler{
		Token:    "secret-token",
		Presence: presence.NewStore(),
		Metrics:  metricsRepo,
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"edge_id":       "gpu-1",
		"comfy_running": true,
		"metrics": edge.Metrics{
			CPUUsagePercent: 42,
			// Use a fresh timestamp so the repository's retention cleanup
			// (24h) does not delete the row right after Append.
			CollectedAt: time.Now().UTC(),
		},
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}
	series, err := metricsRepo.ListSince(context.Background(), "gpu-1", time.Unix(0, 0), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 1 || series[0].CPUUsagePercent != 42 {
		t.Fatalf("series=%+v", series)
	}
}

func TestAgent_PresenceStoresStartedAtAndComfyVersion(t *testing.T) {
	dsn := "file:agent_presence_info_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatal(err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	now := time.Now().UTC()
	if err := repo.Upsert(context.Background(), &edge.Record{
		ID:        "gpu-1",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	h := &agent.Handler{
		Token:    "secret-token",
		Presence: presence.NewStore(),
		Edges:    repo,
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"edge_id":       "gpu-1",
		"comfy_running": true,
		"started_at":    "2026-08-18T12:00:00Z",
		"comfy_version": "v0.1.0",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}

	got, err := repo.Get(context.Background(), "gpu-1")
	if err != nil {
		t.Fatal(err)
	}
	wantStarted := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	if got.StartedAt == nil || !got.StartedAt.Equal(wantStarted) {
		t.Fatalf("started_at=%v want %v", got.StartedAt, wantStarted)
	}
	if got.ComfyVersion != "v0.1.0" {
		t.Fatalf("comfy_version=%q", got.ComfyVersion)
	}
}

func TestAgent_PresenceUnauthorizedDoesNotStoreMetrics(t *testing.T) {
	dsn := "file:agent_metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	metricsRepo := instpersist.NewMetricsRepository(gdb, 24*time.Hour)
	h := &agent.Handler{
		Token:    "secret-token",
		Presence: presence.NewStore(),
		Metrics:  metricsRepo,
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"edge_id": "gpu-1",
		"metrics": edge.Metrics{
			CPUUsagePercent: 42,
			CollectedAt:     time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC),
		},
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.StatusCode)
	}
	series, err := metricsRepo.ListSince(context.Background(), "gpu-1", time.Unix(0, 0), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 0 {
		t.Fatalf("unauthorized must not store metrics: %+v", series)
	}
}

func TestAgent_Unauthorized(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	srv := mountAgent(t, "secret-token", tasks, &statusSpy{})
	res, err := http.Get(srv.URL + "/agent/v1/jobs/claim?edge_id=gpu-1&wait=0s")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestAgent_ClaimReturnsJob(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", 1, "in", now)
	_ = task.PrepareForTopic("default", sharedkernel.BlobRef{Key: "jobs/t1/job.json"}, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	dsn := "file:agent_claim_default_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatal(err)
	}
	edgeRepo := instpersist.NewEdgeRepository(gdb)
	_ = edgeRepo.Upsert(ctx, &edge.Record{
		ID:              sharedkernel.EdgeID("gpu-1"),
		Name:            "gpu-1",
		Enabled:         true,
		SubscribeTopics: []string{"default"},
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	h := &agent.Handler{
		Token:  "secret-token",
		Tasks:  tasks,
		Edges:  edgeRepo,
		Status: &statusSpy{},
		Lease:  90 * time.Second,
		Now:    func() time.Time { return now },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?edge_id=gpu-1&wait=0s", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["task_id"] != "t1" {
		t.Fatalf("body=%v", body)
	}
	jobRef, _ := body["job_ref"].(map[string]any)
	if jobRef["key"] != "jobs/t1/job.json" {
		t.Fatalf("job_ref=%v", jobRef)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskRunning {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestAgent_ClaimEmptyWhenNone(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	srv := mountAgent(t, "secret-token", tasks, &statusSpy{})
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?edge_id=gpu-1&wait=0s", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	if res.StatusCode == http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		if len(bytes.TrimSpace(raw)) != 0 && string(raw) != "null" && string(raw) != "{}" {
			// allow empty JSON object / null for "no job"
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err == nil {
				if body["task_id"] != nil && body["task_id"] != "" {
					t.Fatalf("unexpected job %v", body)
				}
			}
		}
	}
}

func TestAgent_StatusReports(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", 1, "in", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	_ = tasks.Create(ctx, task)
	spy := &statusSpy{}
	srv := mountAgent(t, "tok", tasks, spy)

	payload, _ := json.Marshal(map[string]any{
		"edge_id": "gpu-1",
		"status":  "succeeded",
		"outputs": []map[string]any{{"key": "out.png"}},
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/jobs/t1/status", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	if len(spy.events) != 1 || spy.events[0].Status != sharedkernel.TaskSucceeded {
		t.Fatalf("events=%+v", spy.events)
	}
}

func TestAgent_StatusRejectsWrongInstance(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", 1, "in", now)
	_ = task.PrepareForClaim("gpu-2", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-2", time.Minute, now)
	_ = tasks.Create(ctx, task)
	spy := &statusSpy{}
	srv := mountAgent(t, "tok", tasks, spy)

	payload, _ := json.Marshal(map[string]any{
		"edge_id": "gpu-1",
		"status":  "succeeded",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/jobs/t1/status", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	if len(spy.events) != 0 {
		t.Fatalf("stale status must not apply: %+v", spy.events)
	}
}

func TestAgent_StatusRejectsUnclaimedTask(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t-unclaimed", "s1", 1, "in", now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	spy := &statusSpy{}
	srv := mountAgent(t, "tok", tasks, spy)

	payload, _ := json.Marshal(map[string]any{
		"edge_id": "gpu-1",
		"status":  "succeeded",
	})
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/jobs/t-unclaimed/status", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	if len(spy.events) != 0 {
		t.Fatalf("unclaimed status must not apply: %+v", spy.events)
	}
}

func TestAgent_RejectsTokenBoundToAnotherInstance(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	h := &agent.Handler{
		Verify: func(_ context.Context, edgeID sharedkernel.EdgeID, token string) bool {
			if edgeID == "gpu-1" {
				return token == "tok-a"
			}
			if edgeID == "gpu-2" {
				return token == "tok-b"
			}
			return false
		},
		Tasks: tasks,
		Lease: 90 * time.Second,
		Now:   func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?edge_id=gpu-1&wait=0s", nil)
	req.Header.Set("Authorization", "Bearer tok-b")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("foreign token status=%d", res.StatusCode)
	}

	okReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?edge_id=gpu-1&wait=0s", nil)
	okReq.Header.Set("Authorization", "Bearer tok-a")
	okRes, err := http.DefaultClient.Do(okReq)
	if err != nil {
		t.Fatal(err)
	}
	defer okRes.Body.Close()
	if okRes.StatusCode != http.StatusNoContent && okRes.StatusCode != http.StatusOK {
		t.Fatalf("matching token status=%d", okRes.StatusCode)
	}
}

func TestAgent_PresenceReportsComfy(t *testing.T) {
	store := presence.NewStore()
	now := time.Unix(1000, 0).UTC()
	store.Now = func() time.Time { return now }
	h := &agent.Handler{
		Token:    "secret-token",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: store,
		Now:      func() time.Time { return now },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"edge_id":       "gpu-1",
		"comfy_running": true,
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}
	got := store.Snapshot("gpu-1")
	if !got.EdgeOnline || !got.ComfyRunning {
		t.Fatalf("%+v", got)
	}
}

func TestAgent_PresenceUnauthorized(t *testing.T) {
	store := presence.NewStore()
	h := &agent.Handler{
		Token:    "secret-token",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: store,
		Now:      func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"edge_id":       "gpu-1",
		"comfy_running": true,
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.StatusCode)
	}
	got := store.Snapshot("gpu-1")
	if got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("unauthorized must not record presence: %+v", got)
	}
}

func TestAgent_ClaimTouchesLastSeenWithoutComfy(t *testing.T) {
	store := presence.NewStore()
	now := time.Unix(1000, 0).UTC()
	store.Now = func() time.Time { return now }
	h := &agent.Handler{
		Token:    "secret-token",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: store,
		Now:      func() time.Time { return now },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?edge_id=gpu-1&wait=0s", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}
	got := store.Snapshot("gpu-1")
	if !got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("claim must only touch: %+v", got)
	}
}

func TestAgent_PresenceWritesHardwareOnceUntilRefresh(t *testing.T) {
	dsn := "file:agent_hw_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	now := time.Now().UTC().Truncate(time.Second)
	if err := repo.Upsert(context.Background(), &edge.Record{
		ID:        "gpu-1",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	store := presence.NewStore()
	agentH := &agent.Handler{
		Token:    "secret-token",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: store,
		Edges:    repo,
		Now:      func() time.Time { return now },
	}
	edgesH := &edges.Handler{Repo: repo, EncKey: bytes.Repeat([]byte("k"), 32)}
	r := chi.NewRouter()
	r.Route("/agent/v1", agentH.Mount)
	r.Route("/api/v1/edges", edgesH.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	postPresence := func(cpu string) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{
			"edge_id":       "gpu-1",
			"comfy_running": true,
			"hardware":      map[string]any{"cpu_model": cpu},
		})
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/presence", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer secret-token")
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(res.Body)
			t.Fatalf("presence status=%d body=%s", res.StatusCode, raw)
		}
	}
	getCPU := func() string {
		t.Helper()
		res, err := http.Get(srv.URL + "/api/v1/edges/gpu-1")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("get status=%d", res.StatusCode)
		}
		var dto map[string]any
		if err := json.NewDecoder(res.Body).Decode(&dto); err != nil {
			t.Fatal(err)
		}
		hw, _ := dto["hardware"].(map[string]any)
		cpu, _ := hw["cpu_model"].(string)
		return cpu
	}

	postPresence("Intel")
	if got := getCPU(); got != "Intel" {
		t.Fatalf("first write cpu=%q", got)
	}
	postPresence("AMD")
	if got := getCPU(); got != "Intel" {
		t.Fatalf("second write must keep first cpu, got %q", got)
	}

	patchBody, _ := json.Marshal(map[string]any{"refresh_hardware": true})
	patchReq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/edges/gpu-1", bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRes, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatal(err)
	}
	defer patchRes.Body.Close()
	if patchRes.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(patchRes.Body)
		t.Fatalf("patch status=%d body=%s", patchRes.StatusCode, raw)
	}

	postPresence("AMD")
	if got := getCPU(); got != "AMD" {
		t.Fatalf("after refresh cpu=%q", got)
	}
}
