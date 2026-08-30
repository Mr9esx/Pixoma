package edges_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/edgeadmin"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestHandler_CreateListAndTasksFilter(t *testing.T) {
	dsn := "file:comfy_httpapi_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	tasks := runtimedomain.NewMemoryTaskRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	if err := tasks.Create(ctx, &runtimedomain.Task{
		ID:        "t-pending",
		SessionID: "s1",
		CaseID:    sharedkernel.CaseID(1),
		Status:    sharedkernel.TaskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, &runtimedomain.Task{
		ID:        "t-queued",
		SessionID: "s1",
		CaseID:    sharedkernel.CaseID(1),
		Status:    sharedkernel.TaskQueued,
		EdgeID:    "gpu-2",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	h := &edges.Handler{Repo: repo, Tasks: tasks, EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":      "gpu-2",
		"enabled": true,
	})
	res, err := http.Post(srv.URL+"/api/v1/edges", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", res.StatusCode)
	}

	listRes, err := http.Get(srv.URL + "/api/v1/edges")
	if err != nil {
		t.Fatal(err)
	}
	defer listRes.Body.Close()
	if listRes.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listRes.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(listRes.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range list {
		if item["id"] == "gpu-2" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("list missing gpu-2: %v", list)
	}

	tasksRes, err := http.Get(srv.URL + "/api/v1/edges/gpu-2/tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer tasksRes.Body.Close()
	if tasksRes.StatusCode != http.StatusOK {
		t.Fatalf("tasks status=%d", tasksRes.StatusCode)
	}
	var taskList []map[string]any
	if err := json.NewDecoder(tasksRes.Body).Decode(&taskList); err != nil {
		t.Fatal(err)
	}
	if len(taskList) != 1 || taskList[0]["id"] != "t-queued" {
		t.Fatalf("want only queued task, got %v", taskList)
	}

}

func TestHandler_CreateDuplicateIDReturns409(t *testing.T) {
	dsn := "file:comfy_httpapi_dup_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	h := &edges.Handler{Repo: repo, Tasks: runtimedomain.NewMemoryTaskRepository(), EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":      "gpu-dup",
		"enabled": true,
	})
	res1, err := http.Post(srv.URL+"/api/v1/edges", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res1.Body.Close()
	if res1.StatusCode != http.StatusCreated {
		t.Fatalf("first create status=%d", res1.StatusCode)
	}

	body2, _ := json.Marshal(map[string]any{
		"id":      "gpu-dup",
		"enabled": false,
	})
	res2, err := http.Post(srv.URL+"/api/v1/edges", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate create status=%d want 409", res2.StatusCode)
	}

	got, err := repo.Get(context.Background(), "gpu-dup")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled {
		t.Fatalf("existing row overwritten: %+v", got)
	}
}

func TestHandler_CreateRefreshesPoolAndDispatchTopicReceivable(t *testing.T) {
	dsn := "file:comfy_httpapi_dispatch_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	pool := edge.NewPool(repo, edge.PoolOptions{})
	bus := memory.New()
	t.Cleanup(func() { _ = bus.Close() })
	ctx := context.Background()

	var hits atomic.Int32
	h := func(_ context.Context, _ queue.Message) error {
		hits.Add(1)
		return nil
	}
	var subs queue.SubscriptionSet
	ensure := func(instances []edge.Instance) {
		for _, inst := range instances {
			topic := inst.DispatchTopic
			if topic == "" {
				topic = sharedkernel.TopicDispatch(inst.ID)
			}
			if err := subs.Ensure(ctx, bus, topic, h); err != nil {
				t.Errorf("ensure: %v", err)
			}
		}
	}
	pool.SetAfterRefresh(ensure)
	if err := pool.Refresh(ctx); err != nil {
		t.Fatal(err)
	}

	api := &edges.Handler{Repo: repo, Pool: pool, Tasks: runtimedomain.NewMemoryTaskRepository(), EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) {
		api.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":      "gpu-new",
		"enabled": true,
	})
	res, err := http.Post(srv.URL+"/api/v1/edges", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", res.StatusCode)
	}

	topic := sharedkernel.TopicDispatch("gpu-new")
	if err := bus.Publish(ctx, queue.Message{Topic: topic, Payload: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("dispatch hits=%d want 1 (topic subscribed after create/refresh)", got)
	}
}

func testEncKey() []byte {
	return bytes.Repeat([]byte("k"), 32)
}

func TestHandler_CreateMintsNameAndAgentToken(t *testing.T) {
	dsn := "file:comfy_httpapi_token_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	h := &edges.Handler{
		Repo:   repo,
		Tasks:  runtimedomain.NewMemoryTaskRepository(),
		EncKey: testEncKey(),
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"name":        "机房 A",
		"description": "夜间出图",
	})
	res, err := http.Post(srv.URL+"/api/v1/edges", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", res.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("expected generated id, got %v", created)
	}
	if created["name"] != "机房 A" || created["description"] != "夜间出图" {
		t.Fatalf("name/desc=%v", created)
	}
	tok, _ := created["agent_token"].(string)
	if tok == "" {
		t.Fatalf("expected agent_token, got %v", created)
	}

	getRes, err := http.Get(srv.URL + "/api/v1/edges/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer getRes.Body.Close()
	var got map[string]any
	if err := json.NewDecoder(getRes.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["agent_token"] != tok {
		t.Fatalf("get token=%v want %s", got["agent_token"], tok)
	}

	listRes, err := http.Get(srv.URL + "/api/v1/edges")
	if err != nil {
		t.Fatal(err)
	}
	defer listRes.Body.Close()
	var list []map[string]any
	if err := json.NewDecoder(listRes.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list=%v", list)
	}
	if _, ok := list[0]["agent_token"]; ok && list[0]["agent_token"] != "" && list[0]["agent_token"] != nil {
		t.Fatalf("list must omit agent_token: %v", list[0])
	}

	rotate, err := http.Post(srv.URL+"/api/v1/edges/"+id+"/rotate-token", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rotate.Body.Close()
	if rotate.StatusCode != http.StatusOK {
		t.Fatalf("rotate status=%d", rotate.StatusCode)
	}
	var rotated map[string]any
	if err := json.NewDecoder(rotate.Body).Decode(&rotated); err != nil {
		t.Fatal(err)
	}
	next, _ := rotated["agent_token"].(string)
	if next == "" || next == tok {
		t.Fatalf("rotate must mint a new token, got %v", rotated)
	}
}

func TestHandler_PresenceListsAllInstances(t *testing.T) {
	dsn := "file:comfy_httpapi_presence_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	tasks := runtimedomain.NewMemoryTaskRepository()
	store := presence.NewStore()
	now := time.Unix(1000, 0).UTC()
	store.Now = func() time.Time { return now }

	h := &edges.Handler{
		Repo:     repo,
		Tasks:    tasks,
		EncKey:   testEncKey(),
		Presence: store,
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":      "gpu-2",
		"enabled": true,
	})
	res, err := http.Post(srv.URL+"/api/v1/edges", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", res.StatusCode)
	}

	presRes, err := http.Get(srv.URL + "/api/v1/edges/presence")
	if err != nil {
		t.Fatal(err)
	}
	defer presRes.Body.Close()
	if presRes.StatusCode != http.StatusOK {
		t.Fatalf("unreported presence status=%d", presRes.StatusCode)
	}
	var unreported []map[string]any
	if err := json.NewDecoder(presRes.Body).Decode(&unreported); err != nil {
		t.Fatal(err)
	}
	row := presenceRow(t, unreported, "gpu-2")
	if row["edge_online"] != false || row["comfy_running"] != false {
		t.Fatalf("unreported: %v", row)
	}

	store.Report("gpu-2", true)
	presRes2, err := http.Get(srv.URL + "/api/v1/edges/presence")
	if err != nil {
		t.Fatal(err)
	}
	defer presRes2.Body.Close()
	if presRes2.StatusCode != http.StatusOK {
		t.Fatalf("reported presence status=%d", presRes2.StatusCode)
	}
	var reported []map[string]any
	if err := json.NewDecoder(presRes2.Body).Decode(&reported); err != nil {
		t.Fatal(err)
	}
	row = presenceRow(t, reported, "gpu-2")
	if row["edge_online"] != true || row["comfy_running"] != true {
		t.Fatalf("reported: %v", row)
	}

	now = now.Add(16 * time.Second)
	presRes3, err := http.Get(srv.URL + "/api/v1/edges/presence")
	if err != nil {
		t.Fatal(err)
	}
	defer presRes3.Body.Close()
	var stale []map[string]any
	if err := json.NewDecoder(presRes3.Body).Decode(&stale); err != nil {
		t.Fatal(err)
	}
	row = presenceRow(t, stale, "gpu-2")
	if row["edge_online"] != false || row["comfy_running"] != false {
		t.Fatalf("stale: %v", row)
	}
}

func presenceRow(t *testing.T, rows []map[string]any, id string) map[string]any {
	t.Helper()
	for _, row := range rows {
		if row["id"] == id {
			return row
		}
	}
	t.Fatalf("missing id %s in %v", id, rows)
	return nil
}

func TestHandler_PatchHardwareAndRefreshFlag(t *testing.T) {
	dsn := "file:comfy_httpapi_hw_" + t.Name() + "?mode=memory&cache=shared"
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
		Name:      "gpu-1",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	h := &edges.Handler{
		Repo:   repo,
		Tasks:  runtimedomain.NewMemoryTaskRepository(),
		EncKey: testEncKey(),
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"hardware": map[string]any{"cpu_model": "Ryzen", "cpu_cores": 16},
	})
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/edges/gpu-1", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch hardware status=%d", res.StatusCode)
	}
	var dto map[string]any
	if err := json.NewDecoder(res.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	hw, _ := dto["hardware"].(map[string]any)
	if hw["cpu_model"] != "Ryzen" {
		t.Fatalf("hardware=%v", hw)
	}
	if hw["collected_at"] == nil || hw["collected_at"] == "" {
		t.Fatal("hand edit must set collected_at")
	}

	flagBody, _ := json.Marshal(map[string]any{"refresh_hardware": true})
	flagReq, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/edges/gpu-1", bytes.NewReader(flagBody))
	flagReq.Header.Set("Content-Type", "application/json")
	flagRes, err := http.DefaultClient.Do(flagReq)
	if err != nil {
		t.Fatal(err)
	}
	defer flagRes.Body.Close()
	if flagRes.StatusCode != http.StatusOK {
		t.Fatalf("patch refresh status=%d", flagRes.StatusCode)
	}
	got, err := repo.Get(context.Background(), "gpu-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.HardwareRefreshRequested {
		t.Fatal("refresh flag not set")
	}
	if got.Hardware.CPUModel != "Ryzen" {
		t.Fatalf("refresh must not wipe hardware: %+v", got.Hardware)
	}
}

func TestHandler_Stats(t *testing.T) {
	dsn := "file:comfy_httpapi_stats_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	now := time.Unix(1_700_000_000, 0).UTC()
	if err := repo.Upsert(context.Background(), &edge.Record{
		ID:        "gpu-1",
		Name:      "gpu-1",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	tasks := runtimedomain.NewMemoryTaskRepository()
	add := func(id string, status sharedkernel.TaskStatus, dur time.Duration) {
		t.Helper()
		if err := tasks.Create(context.Background(), &runtimedomain.Task{
			ID:        sharedkernel.TaskID(id),
			SessionID: "s1",
			CaseID:    sharedkernel.CaseID(1),
			Status:    status,
			EdgeID:    "gpu-1",
			CreatedAt: now,
			UpdatedAt: now.Add(dur),
		}); err != nil {
			t.Fatal(err)
		}
	}
	add("t-ok-1", sharedkernel.TaskSucceeded, 100*time.Millisecond)
	add("t-ok-2", sharedkernel.TaskSucceeded, 200*time.Millisecond)
	add("t-fail", sharedkernel.TaskFailed, 300*time.Millisecond)
	add("t-run", sharedkernel.TaskRunning, time.Second)

	h := &edges.Handler{
		Repo:   repo,
		Tasks:  tasks,
		EncKey: testEncKey(),
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/edges/gpu-1/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stats status=%d", res.StatusCode)
	}
	var dto map[string]any
	if err := json.NewDecoder(res.Body).Decode(&dto); err != nil {
		t.Fatal(err)
	}
	if dto["task_count"] != float64(4) {
		t.Fatalf("task_count=%v", dto["task_count"])
	}
	if dto["runtime_ms"] != float64(600) {
		t.Fatalf("runtime_ms=%v", dto["runtime_ms"])
	}
	rate, ok := dto["success_rate"].(float64)
	if !ok {
		t.Fatalf("success_rate=%v", dto["success_rate"])
	}
	if diff := rate - 2.0/3.0; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("success_rate=%v want 2/3", rate)
	}
}

func TestHandler_MetricsEndpoint(t *testing.T) {
	dsn := "file:edges_metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}, &instpersist.MetricsRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	metricsRepo := instpersist.NewMetricsRepository(gdb, 24*time.Hour)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := repo.Upsert(ctx, &edge.Record{
		ID:        "gpu-1",
		Name:      "gpu-1",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for i, offset := range []time.Duration{-30 * time.Minute, -10 * time.Minute} {
		m := edge.Metrics{CPUUsagePercent: float64(i+1) * 10, CollectedAt: now.Add(offset)}
		if err := metricsRepo.Append(ctx, "gpu-1", m); err != nil {
			t.Fatal(err)
		}
	}

	h := &edges.Handler{Repo: repo, Metrics: metricsRepo, EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/edges/gpu-1/metrics?window=1h")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}
	var out struct {
		Latest *edge.Metrics  `json:"latest"`
		Series []edge.Metrics `json:"series"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Series) != 60 || out.Latest == nil || out.Latest.CPUUsagePercent != 20 {
		t.Fatalf("series=%+v latest=%+v", out.Series, out.Latest)
	}
	real := 0
	for _, p := range out.Series {
		if p.CPUUsagePercent > 0 {
			real++
		}
	}
	if real != 2 {
		t.Fatalf("want 2 real points among zero-filled buckets, got %d", real)
	}
	if out.Series[0].CPUUsagePercent != 0 {
		t.Fatalf("first bucket must be zero-filled, got %+v", out.Series[0])
	}

	missing, err := http.Get(srv.URL + "/api/v1/edges/nope/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status=%d", missing.StatusCode)
	}
}

func TestHandler_MetricsEmptySeries(t *testing.T) {
	dsn := "file:edges_metrics_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}, &instpersist.MetricsRow{}); err != nil {
		t.Fatal(err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	metricsRepo := instpersist.NewMetricsRepository(gdb, 24*time.Hour)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := repo.Upsert(ctx, &edge.Record{
		ID:        "gpu-2",
		Name:      "gpu-2",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	h := &edges.Handler{Repo: repo, Metrics: metricsRepo, EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/edges/gpu-2/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["latest"] != nil {
		t.Fatalf("latest must be null: %v", out["latest"])
	}
	series, ok := out["series"].([]any)
	if !ok || len(series) != 0 {
		t.Fatalf("series must be empty: %v", out["series"])
	}
}

func TestHandler_MetricsCustomRange(t *testing.T) {
	dsn := "file:edges_metrics_custom_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}, &instpersist.MetricsRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	metricsRepo := instpersist.NewMetricsRepository(gdb, 24*time.Hour)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := repo.Upsert(ctx, &edge.Record{
		ID: "gpu-custom", Name: "gpu-custom", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for i, offset := range []time.Duration{-25 * time.Minute, -5 * time.Minute} {
		m := edge.Metrics{CPUUsagePercent: float64(i+1) * 10, CollectedAt: now.Add(offset)}
		if err := metricsRepo.Append(ctx, "gpu-custom", m); err != nil {
			t.Fatal(err)
		}
	}

	h := &edges.Handler{Repo: repo, Metrics: metricsRepo, EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	from := now.Add(-30 * time.Minute).Format(time.RFC3339)
	to := now.Format(time.RFC3339)
	res, err := http.Get(srv.URL + "/api/v1/edges/gpu-custom/metrics?from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, raw)
	}
	var out struct {
		Latest *edge.Metrics  `json:"latest"`
		Series []edge.Metrics `json:"series"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Series) != 30 {
		t.Fatalf("custom 30m window must produce 30 buckets, got %d", len(out.Series))
	}
	real := 0
	for _, p := range out.Series {
		if p.CPUUsagePercent > 0 {
			real++
		}
	}
	if real != 2 || out.Latest == nil || out.Latest.CPUUsagePercent != 20 {
		t.Fatalf("real=%d latest=%+v", real, out.Latest)
	}
}

func TestHandler_MetricsCustomRangeInvalid(t *testing.T) {
	dsn := "file:edges_metrics_custom_invalid_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}, &instpersist.MetricsRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	metricsRepo := instpersist.NewMetricsRepository(gdb, 24*time.Hour)
	now := time.Now().UTC().Truncate(time.Second)
	if err := repo.Upsert(context.Background(), &edge.Record{
		ID: "gpu-bad", Name: "gpu-bad", Enabled: true, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	h := &edges.Handler{Repo: repo, Metrics: metricsRepo, EncKey: testEncKey()}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	iso := func(tm time.Time) string { return url.QueryEscape(tm.Format(time.RFC3339)) }
	cases := []string{
		"from=" + iso(now) + "&to=" + iso(now),                                // from == to
		"from=" + iso(now) + "&to=" + iso(now.Add(-time.Hour)),                // from after to
		"from=" + iso(now.Add(-time.Hour)) + "&to=" + iso(now.Add(time.Hour)), // future to
		"from=not-a-time&to=" + iso(now),                                      // bad format
	}
	for _, q := range cases {
		res, err := http.Get(srv.URL + "/api/v1/edges/gpu-bad/metrics?" + q)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("query %q status=%d want 400", q, res.StatusCode)
		}
	}
}

func TestEdge_GetIncludesStartedAtAndComfyVersion(t *testing.T) {
	dsn := "file:comfy_httpapi_presence_info_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()
	startedAt := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	if err := repo.Upsert(ctx, &edge.Record{
		ID:           "gpu-1",
		Enabled:      true,
		Capabilities: []string{"sdxl"},
		StartedAt:    &startedAt,
		ComfyVersion: "v0.1.0",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	h := &edges.Handler{
		Repo:   repo,
		Tasks:  runtimedomain.NewMemoryTaskRepository(),
		EncKey: testEncKey(),
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/edges/gpu-1")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["comfy_version"] != "v0.1.0" {
		t.Fatalf("comfy_version=%v", out["comfy_version"])
	}
	if got, ok := out["started_at"].(string); !ok || got != "2026-08-18T12:00:00Z" {
		t.Fatalf("started_at=%v", out["started_at"])
	}
}

func TestHandler_DeleteCleanup(t *testing.T) {
	h := &edges.Handler{
		DeleteWithCleanup: func(ctx context.Context, id sharedkernel.EdgeID, ack bool) (edgeadmin.DeleteSummary, error) {
			if id == "gpu-x" && !ack {
				return edgeadmin.DeleteSummary{}, edgeadmin.ErrNeedsAck
			}
			if id == "missing" {
				return edgeadmin.DeleteSummary{}, edge.ErrNotFound
			}
			return edgeadmin.DeleteSummary{FailedTasks: 1}, nil
		},
	}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	do := func(id string, body string) *http.Response {
		t.Helper()
		var rdr io.Reader
		if body != "" {
			rdr = bytes.NewBufferString(body)
		}
		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/edges/"+id, rdr)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := do("missing", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status=%d want 404", res.StatusCode)
	}
	res.Body.Close()

	res = do("gpu-x", "")
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("no-ack status=%d want 409", res.StatusCode)
	}
	res.Body.Close()

	res = do("gpu-x", `{"ack_references":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ack status=%d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if body["deleted"] != true || body["failed_tasks"] != float64(1) {
		t.Fatalf("body=%+v", body)
	}
}
