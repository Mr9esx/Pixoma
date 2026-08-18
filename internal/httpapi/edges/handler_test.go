package comfyinstances_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/comfyinstances"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
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
	if err := db.AutoMigrate(gdb, &instpersist.InstanceRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewInstanceRepository(gdb)
	pool := instance.NewPool(repo, instance.PoolOptions{Mock: true})
	tasks := runtimedomain.NewMemoryTaskRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	if err := tasks.Create(ctx, &runtimedomain.Task{
		ID:        "t-pending",
		SessionID: "s1",
		CaseID:    "c1",
		Status:    sharedkernel.TaskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, &runtimedomain.Task{
		ID:         "t-queued",
		SessionID:  "s1",
		CaseID:     "c1",
		Status:     sharedkernel.TaskQueued,
		InstanceID: "gpu-2",
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatal(err)
	}

	h := &comfyinstances.Handler{Repo: repo, Pool: pool, Tasks: tasks, Mock: true}
	r := chi.NewRouter()
	r.Route("/api/v1/comfy-instances", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":       "gpu-2",
		"base_url": "http://127.0.0.1:8189",
		"enabled":  true,
	})
	res, err := http.Post(srv.URL+"/api/v1/comfy-instances", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", res.StatusCode)
	}

	listRes, err := http.Get(srv.URL + "/api/v1/comfy-instances")
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

	tasksRes, err := http.Get(srv.URL + "/api/v1/comfy-instances/gpu-2/tasks")
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

	sysRes, err := http.Get(srv.URL + "/api/v1/comfy-instances/gpu-2/system")
	if err != nil {
		t.Fatal(err)
	}
	defer sysRes.Body.Close()
	if sysRes.StatusCode != http.StatusOK {
		t.Fatalf("system status=%d", sysRes.StatusCode)
	}
	var sys map[string]any
	if err := json.NewDecoder(sysRes.Body).Decode(&sys); err != nil {
		t.Fatal(err)
	}
	if sys["mock"] != true || sys["reachable"] != true {
		t.Fatalf("system=%v", sys)
	}

	missing, err := http.Get(srv.URL + "/api/v1/comfy-instances/no-such/system")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing instance want 404, got %d", missing.StatusCode)
	}
}

func TestHandler_CreateDuplicateIDReturns409(t *testing.T) {
	dsn := "file:comfy_httpapi_dup_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.InstanceRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewInstanceRepository(gdb)
	h := &comfyinstances.Handler{Repo: repo, Pool: instance.NewPool(repo, instance.PoolOptions{Mock: true}), Tasks: runtimedomain.NewMemoryTaskRepository(), Mock: true}
	r := chi.NewRouter()
	r.Route("/api/v1/comfy-instances", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":       "gpu-dup",
		"base_url": "http://127.0.0.1:8188",
		"enabled":  true,
	})
	res1, err := http.Post(srv.URL+"/api/v1/comfy-instances", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res1.Body.Close()
	if res1.StatusCode != http.StatusCreated {
		t.Fatalf("first create status=%d", res1.StatusCode)
	}

	body2, _ := json.Marshal(map[string]any{
		"id":       "gpu-dup",
		"base_url": "http://127.0.0.1:9999",
		"enabled":  false,
	})
	res2, err := http.Post(srv.URL+"/api/v1/comfy-instances", "application/json", bytes.NewReader(body2))
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
	if got.BaseURL != "http://127.0.0.1:8188" || !got.Enabled {
		t.Fatalf("existing row overwritten: %+v", got)
	}
}

func TestHandler_CreateRefreshesPoolAndDispatchTopicReceivable(t *testing.T) {
	dsn := "file:comfy_httpapi_dispatch_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.InstanceRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := instpersist.NewInstanceRepository(gdb)
	pool := instance.NewPool(repo, instance.PoolOptions{Mock: true})
	bus := memory.New()
	t.Cleanup(func() { _ = bus.Close() })
	ctx := context.Background()

	var hits atomic.Int32
	h := func(_ context.Context, _ queue.Message) error {
		hits.Add(1)
		return nil
	}
	var subs queue.SubscriptionSet
	ensure := func(instances []instance.Instance) {
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

	api := &comfyinstances.Handler{Repo: repo, Pool: pool, Tasks: runtimedomain.NewMemoryTaskRepository(), Mock: true}
	r := chi.NewRouter()
	r.Route("/api/v1/comfy-instances", func(r chi.Router) {
		api.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	body, _ := json.Marshal(map[string]any{
		"id":       "gpu-new",
		"base_url": "http://127.0.0.1:8199",
		"enabled":  true,
	})
	res, err := http.Post(srv.URL+"/api/v1/comfy-instances", "application/json", bytes.NewReader(body))
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
