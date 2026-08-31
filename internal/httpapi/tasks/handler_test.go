package tasks_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	channeldomain "github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	userdomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
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
	pending := runtimedomain.NewPending("t-pending", "sess-1", 1, "pfx", now)
	if err := tasks.Create(ctx, pending); err != nil {
		t.Fatal(err)
	}
	done := runtimedomain.NewPending("t-done", "sess-1", 1, "pfx", now)
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

	queued := runtimedomain.NewPending("t-q", "sess-1", 1, "pfx", now)
	queued.Status = sharedkernel.TaskQueued
	queued.DispatchTopic = "fast-gpu"
	if err := tasks.Create(ctx, queued); err != nil {
		t.Fatal(err)
	}
	other := runtimedomain.NewPending("t-other", "sess-1", 1, "pfx", now)
	other.Status = sharedkernel.TaskQueued
	other.DispatchTopic = "slow-gpu"
	if err := tasks.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	resQ, err := http.Get(srv.URL + "/api/v1/tasks?dispatch_topic=fast-gpu&status=queued")
	if err != nil {
		t.Fatal(err)
	}
	defer resQ.Body.Close()
	if resQ.StatusCode != http.StatusOK {
		t.Fatalf("dispatch filter status=%d", resQ.StatusCode)
	}
	var qList []map[string]any
	if err := json.NewDecoder(resQ.Body).Decode(&qList); err != nil {
		t.Fatal(err)
	}
	if len(qList) != 1 || qList[0]["id"] != "t-q" {
		t.Fatalf("dispatch filter: %+v", qList)
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

func TestTasksHandler_FilterByChannel(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:tasks_chan_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &taskpersist.TaskRow{}, &sesspersist.SessionRow{}); err != nil {
		t.Fatal(err)
	}
	tasks := taskpersist.NewTaskRepository(gdb)
	sessions := sesspersist.NewSessionRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := sessions.Save(ctx, convdomain.NewCollecting("s1", "tg:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(ctx, convdomain.NewCollecting("s2", "ig:9", 1, []string{"a"}, now)); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, runtimedomain.NewPending("t1", "s1", 1, "inputs/t1", now)); err != nil {
		t.Fatal(err)
	}
	if err := tasks.Create(ctx, runtimedomain.NewPending("t2", "s2", 1, "inputs/t2", now)); err != nil {
		t.Fatal(err)
	}

	h := &tasksapi.Handler{Tasks: tasks}
	r := chi.NewRouter()
	r.Route("/api/v1/tasks", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/tasks?channel_id=tg")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var list []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0]["id"] != "t1" {
		t.Fatalf("channel filter: %+v", list)
	}
}

func TestTasksHandler_ContextProjection(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:tasks_context_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &taskpersist.TaskRow{}, &sesspersist.SessionRow{}, &userpersist.UserRow{}, &userpersist.UserExternalIdentityRow{}, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	channelStore := channelpersist.NewGormRepository(gdb)
	if err := channelStore.Create(ctx, channeldomain.Channel{ID: "tg-default", Platform: "telegram", Name: "Default Telegram", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	userRepo := userpersist.NewUserRepository(gdb)
	user, err := userRepo.UpsertByChannelExternal(ctx, userdomain.UpsertFrom{
		ChannelID: "tg-default", ExternalUserID: "9001",
		Username: "alice", FirstName: "Alice", LastName: "A",
	})
	if err != nil {
		t.Fatal(err)
	}
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	session := convdomain.NewCollecting("s1", "tg-default:101", 1, []string{"a"}, now)
	session.UserID = user.ID
	if err := sessionRepo.Save(ctx, session); err != nil {
		t.Fatal(err)
	}
	taskRepo := taskpersist.NewTaskRepository(gdb)
	if err := taskRepo.Create(ctx, runtimedomain.NewPending("task-context", "s1", 1, "pfx", now)); err != nil {
		t.Fatal(err)
	}
	if err := taskRepo.Create(ctx, runtimedomain.NewPending("task-orphan", "missing", 2, "pfx", now)); err != nil {
		t.Fatal(err)
	}

	h := &tasksapi.Handler{
		Tasks:   taskRepo,
		Context: taskpersist.NewTaskAdminProjection(gdb),
	}
	r := chi.NewRouter()
	r.Route("/api/v1/tasks", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var list []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, row := range list {
		byID[row["id"].(string)] = row
	}
	row := byID["task-context"]
	if row == nil {
		t.Fatalf("missing context task: %+v", list)
	}
	for key, want := range map[string]any{
		"session_id":   "s1",
		"channel_id":   "tg-default",
		"channel_name": "Default Telegram",
		"user_id":      user.ID,
	} {
		if row[key] != want {
			t.Fatalf("%s = %v, want %v; task=%+v", key, row[key], want, row)
		}
	}
	userContext, ok := row["user"].(map[string]any)
	if !ok || userContext["id"] != user.ID || userContext["external_user_id"] != "9001" {
		t.Fatalf("user context: %+v", row["user"])
	}
	orphan := byID["task-orphan"]
	if orphan == nil || orphan["channel_id"] != "" || orphan["user"] != nil {
		t.Fatalf("orphan task context: %+v", orphan)
	}

	res2, err := http.Get(srv.URL + "/api/v1/tasks/task-context")
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	var detail map[string]any
	if err := json.NewDecoder(res2.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail["channel_id"] != "tg-default" || detail["user_id"] != user.ID {
		t.Fatalf("detail context: %+v", detail)
	}
}
