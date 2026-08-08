package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/apps/admin-api/internal/server"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/comfyinstances"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
)

func TestNewHandler_Healthz(t *testing.T) {
	h := server.NewHandler(server.Options{
		CORSOrigins: []string{"http://localhost:5173"},
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "ok" {
		t.Fatalf("body=%q, want ok", body)
	}
}

func TestNewHandler_CORSPreflight(t *testing.T) {
	origin := "http://localhost:5173"
	h := server.NewHandler(server.Options{
		CORSOrigins: []string{origin},
	})
	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("Access-Control-Allow-Origin=%q, want %q", got, origin)
	}
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d", rec.Code)
	}
}

func TestNewHandler_WithoutInstances_ListNotFound(t *testing.T) {
	h := server.NewHandler(server.Options{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/comfy-instances", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404 when Instances unset", rec.Code)
	}
}

func TestNewHandler_MountsResourceRoutes(t *testing.T) {
	dsn := "file:admin_api_resources_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb,
		&persistence.CaseRow{},
		&userpersist.UserRow{},
		&sesspersist.SessionRow{},
		&taskpersist.TaskRow{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	casesH := &casesapi.Handler{
		Repo:     persistence.NewGormRepository(gdb),
		Validate: validation.New().ValidateDocument,
	}
	usersH := &usersapi.Handler{Repo: userpersist.NewUserRepository(gdb)}
	sessionsH := &sessionsapi.Handler{Repo: sesspersist.NewSessionRepository(gdb)}
	taskRepo := taskpersist.NewTaskRepository(gdb)
	orch := orchestrator.New(taskRepo, nil, nil, notify.Nop{})
	tasksH := &tasksapi.Handler{Tasks: taskRepo, Cancel: orch}

	h := server.NewHandler(server.Options{
		Cases:    casesH,
		Users:    usersH,
		Sessions: sessionsH,
		Tasks:    tasksH,
	})

	for _, path := range []string{
		"/healthz",
		"/api/v1/cases",
		"/api/v1/users",
		"/api/v1/sessions",
		"/api/v1/tasks",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d, want 200", path, rec.Code)
		}
	}
}

func TestNewHandler_MountsComfyInstances(t *testing.T) {
	dsn := "file:admin_api_mount_" + t.Name() + "?mode=memory&cache=shared"
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
	instAPI := &comfyinstances.Handler{
		Repo:  repo,
		Pool:  pool,
		Tasks: tasks,
		Mock:  true,
	}

	h := server.NewHandler(server.Options{Instances: instAPI})

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/comfy-instances", nil)
	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d, want 200", listRec.Code)
	}

	body, _ := json.Marshal(map[string]any{
		"id":       "gpu-admin",
		"base_url": "http://127.0.0.1:8188",
		"enabled":  true,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/comfy-instances", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated && createRec.Code != http.StatusOK {
		t.Fatalf("create status=%d", createRec.Code)
	}

	sysReq := httptest.NewRequest(http.MethodGet, "/api/v1/comfy-instances/gpu-admin/system", nil)
	sysRec := httptest.NewRecorder()
	h.ServeHTTP(sysRec, sysReq)
	if sysRec.Code != http.StatusOK {
		t.Fatalf("system status=%d, want 200", sysRec.Code)
	}
}
