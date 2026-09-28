package studio_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/httpapi/apitest"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func TestStudioProjectRoutesKeepSessionsWhenDeleted(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	createdSession, err := domain.NewSession("session-1", "account-a", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(t.Context(), createdSession); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	(&studioapi.Handler{Repo: repo}).Mount(router)

	created := request(t, router, http.MethodPost, "/projects", map[string]string{"name": "产品图"}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	if !strings.Contains(created.Body.String(), `"name":"产品图"`) {
		t.Fatalf("project JSON = %s", created.Body.String())
	}
	var project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(apitest.DataBytes(created), &project); err != nil {
		t.Fatal(err)
	}
	if project.ID == "" || project.Name != "产品图" {
		t.Fatalf("project = %#v", project)
	}
	listed := request(t, router, http.MethodGet, "/projects", nil, "account-a")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"name":"产品图"`) {
		t.Fatalf("projects = %d %s", listed.Code, listed.Body.String())
	}
	renamed := request(t, router, http.MethodPatch, "/projects/"+project.ID, map[string]string{"name": "分镜"}, "account-a")
	if renamed.Code != http.StatusOK || !strings.Contains(renamed.Body.String(), `"name":"分镜"`) {
		t.Fatalf("rename = %d %s", renamed.Code, renamed.Body.String())
	}
	foreignList := request(t, router, http.MethodGet, "/projects", nil, "account-b")
	if string(apitest.DataBytes(foreignList)) != "[]" {
		t.Fatalf("foreign projects = %s", foreignList.Body.String())
	}
	duplicate := request(t, router, http.MethodPost, "/projects", map[string]string{"name": "分镜"}, "account-a")
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate project = %d %s", duplicate.Code, duplicate.Body.String())
	}
	foreignMove := request(t, router, http.MethodPatch, "/sessions/session-1/project", map[string]string{"project_id": project.ID}, "account-b")
	if foreignMove.Code != http.StatusNotFound {
		t.Fatalf("foreign move = %d", foreignMove.Code)
	}
	move := request(t, router, http.MethodPatch, "/sessions/session-1/project", map[string]string{"project_id": project.ID}, "account-a")
	if move.Code != http.StatusOK {
		t.Fatalf("move = %d %s", move.Code, move.Body.String())
	}
	projectSessions := request(t, router, http.MethodGet, "/sessions?project_id="+project.ID+"&limit=5", nil, "account-a")
	var sessions []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(apitest.DataBytes(projectSessions), &sessions); err != nil || len(sessions) != 1 || sessions[0].ID != "session-1" {
		t.Fatalf("project sessions = %v, %v", sessions, err)
	}
	deleted := request(t, router, http.MethodDelete, "/projects/"+project.ID, nil, "account-a")
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete = %d", deleted.Code)
	}
	recent := request(t, router, http.MethodGet, "/sessions?project_id=&limit=30", nil, "account-a")
	if err := json.Unmarshal(apitest.DataBytes(recent), &sessions); err != nil || len(sessions) != 1 || sessions[0].ID != "session-1" {
		t.Fatalf("recent sessions = %v, %v", sessions, err)
	}
}
