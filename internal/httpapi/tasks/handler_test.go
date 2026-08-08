package tasks_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestTasksHandler_ListGetCancel(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	orch := orchestrator.New(tasks, nil, nil, notify.Nop{})
	h := &tasksapi.Handler{Tasks: tasks, Cancel: orch}
	r := chi.NewRouter()
	r.Route("/api/v1/tasks", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	ctx := t.Context()
	now := time.Now().UTC()
	pending := runtimedomain.NewPending("t-pending", "sess-1", "case-1", "pfx", now)
	if err := tasks.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}
	done := runtimedomain.NewPending("t-done", "sess-1", "case-1", "pfx", now)
	done.Status = sharedkernel.TaskSucceeded
	if err := tasks.Create(ctx, done); err != nil {
		t.Fatal(err)
	}

	res, err := http.Get(srv.URL + "/api/v1/tasks?status=pending")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", res.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["id"] != "t-pending" {
		t.Fatalf("pending list: %+v", list)
	}

	cres, err := http.Post(srv.URL+"/api/v1/tasks/t-pending/cancel", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cres.Body.Close()
	if cres.StatusCode != http.StatusOK {
		t.Fatalf("cancel pending status=%d", cres.StatusCode)
	}
	var cancelled map[string]any
	if err := json.NewDecoder(cres.Body).Decode(&cancelled); err != nil {
		t.Fatal(err)
	}
	if cancelled["status"] != string(sharedkernel.TaskCancelled) {
		t.Fatalf("want cancelled, got %+v", cancelled)
	}

	cres2, err := http.Post(srv.URL+"/api/v1/tasks/t-done/cancel", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cres2.Body.Close()
	if cres2.StatusCode != http.StatusConflict {
		t.Fatalf("cancel succeeded want 409, got %d", cres2.StatusCode)
	}

	cres3, err := http.Post(srv.URL+"/api/v1/tasks/missing/cancel", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cres3.Body.Close()
	if cres3.StatusCode != http.StatusNotFound {
		t.Fatalf("cancel missing want 404, got %d", cres3.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tasks", nil)
	cres4, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer cres4.Body.Close()
	if cres4.StatusCode != http.StatusMethodNotAllowed && cres4.StatusCode != http.StatusNotFound {
		t.Fatalf("POST / create should be disallowed, got %d", cres4.StatusCode)
	}

	gres, err := http.Get(srv.URL + "/api/v1/tasks/t-pending")
	if err != nil {
		t.Fatal(err)
	}
	defer gres.Body.Close()
	if gres.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", gres.StatusCode)
	}
}
