package cases_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func openCasesHandler(t *testing.T) (*casesapi.Handler, *persistence.GormRepository, *httptest.Server) {
	t.Helper()
	dsn := "file:cases_httpapi_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.CaseRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := persistence.NewGormRepository(gdb)
	v := validation.New()
	h := &casesapi.Handler{Repo: repo, Validate: v.ValidateDocument}
	r := chi.NewRouter()
	r.Route("/api/v1/cases", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return h, repo, srv
}

func validCaseBody(id uint64, name string) map[string]any {
	return map[string]any{
		"id":   id,
		"name": name,
		"tags": []string{"text2img"},
		"inputs": []map[string]any{
			{"key": "prompt", "type": "string", "required": true},
		},
		"outputs": []map[string]any{
			{"key": "image", "type": "image"},
		},
		"bindings": map[string]any{
			"workflow": map[string]any{
				"1": map[string]any{"class_type": "CLIPTextEncode", "inputs": map[string]any{"text": "x"}},
			},
			"inputs": []map[string]any{
				{"key": "prompt", "node_id": "1", "field_path": "text"},
			},
		},
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string"},
			},
		},
		"enabled": true,
	}
}

func decodeErr(t *testing.T, res *http.Response) string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body["error"]
}

func TestCasesHandler_CRUDEnableDisable(t *testing.T) {
	_, repo, srv := openCasesHandler(t)
	ctx := context.Background()

	// POST valid → 201
	body, _ := json.Marshal(validCaseBody(1, "Alpha Workflow"))
	res, err := http.Post(srv.URL+"/api/v1/cases", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("create status=%d body=%s", res.StatusCode, raw)
	}
	var created map[string]any
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created["id"] != float64(1) || created["enabled"] != true {
		t.Fatalf("create dto=%v", created)
	}

	// POST id=0 → 201，自动分配新 id
	autoBody, _ := json.Marshal(validCaseBody(0, "Auto ID Workflow"))
	autoRes, err := http.Post(
		srv.URL+"/api/v1/cases",
		"application/json",
		bytes.NewReader(autoBody),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer autoRes.Body.Close()
	if autoRes.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(autoRes.Body)
		t.Fatalf("create with id=0 status=%d body=%s", autoRes.StatusCode, raw)
	}
	var autoCreated map[string]any
	if err := json.NewDecoder(autoRes.Body).Decode(&autoCreated); err != nil {
		t.Fatal(err)
	}
	autoID, ok := autoCreated["id"].(float64)
	if !ok || autoID <= 0 {
		t.Fatalf("want auto-assigned id > 0, got %v", autoCreated["id"])
	}

	// POST invalid (empty name) → 400，库中无脏数据
	bad := validCaseBody(999999, "")
	badBody, _ := json.Marshal(bad)
	badRes, err := http.Post(srv.URL+"/api/v1/cases", "application/json", bytes.NewReader(badBody))
	if err != nil {
		t.Fatal(err)
	}
	defer badRes.Body.Close()
	if badRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid create status=%d want 400", badRes.StatusCode)
	}
	if msg := decodeErr(t, badRes); msg == "" {
		t.Fatal("want validation error message")
	}
	if _, err := repo.Get(ctx, sharedkernel.CaseID(999999)); err == nil {
		t.Fatal("invalid case should not be persisted")
	} else if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound for dirty check, got %v", err)
	}

	// GET list ?q= → 200
	listRes, err := http.Get(srv.URL + "/api/v1/cases?q=Alpha")
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
	if len(list) != 1 || list[0]["id"] != float64(1) {
		t.Fatalf("list=%v", list)
	}

	// GET /{id}
	getRes, err := http.Get(srv.URL + "/api/v1/cases/1")
	if err != nil {
		t.Fatal(err)
	}
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getRes.StatusCode)
	}

	// PATCH update name
	patchBody, _ := json.Marshal(validCaseBody(1, "Alpha Updated"))
	patchReq, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/cases/1", bytes.NewReader(patchBody))
	if err != nil {
		t.Fatal(err)
	}
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
	var patched map[string]any
	if err := json.NewDecoder(patchRes.Body).Decode(&patched); err != nil {
		t.Fatal(err)
	}
	if patched["name"] != "Alpha Updated" {
		t.Fatalf("patched=%v", patched)
	}

	// POST /{id}/disable → enabled=false
	disRes, err := http.Post(srv.URL+"/api/v1/cases/1/disable", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer disRes.Body.Close()
	if disRes.StatusCode != http.StatusOK {
		t.Fatalf("disable status=%d", disRes.StatusCode)
	}
	var disabled map[string]any
	if err := json.NewDecoder(disRes.Body).Decode(&disabled); err != nil {
		t.Fatal(err)
	}
	if disabled["enabled"] != false {
		t.Fatalf("after disable: %v", disabled)
	}
	got, err := repo.Get(ctx, sharedkernel.CaseID(1))
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled {
		t.Fatal("repo still enabled after disable")
	}

	// POST /{id}/enable → enabled=true
	enRes, err := http.Post(srv.URL+"/api/v1/cases/1/enable", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer enRes.Body.Close()
	if enRes.StatusCode != http.StatusOK {
		t.Fatalf("enable status=%d", enRes.StatusCode)
	}
	var enabled map[string]any
	if err := json.NewDecoder(enRes.Body).Decode(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled["enabled"] != true {
		t.Fatalf("after enable: %v", enabled)
	}

	// GET missing → 404
	missing, err := http.Get(srv.URL + "/api/v1/cases/999999")
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Body.Close()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing want 404, got %d", missing.StatusCode)
	}
}

func TestCasesHandler_CreateDuplicateReturns409(t *testing.T) {
	_, _, srv := openCasesHandler(t)
	body, _ := json.Marshal(validCaseBody(2, "Dup"))
	res1, err := http.Post(srv.URL+"/api/v1/cases", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res1.Body.Close()
	if res1.StatusCode != http.StatusCreated {
		t.Fatalf("first create status=%d", res1.StatusCode)
	}

	res2, err := http.Post(srv.URL+"/api/v1/cases", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate status=%d want 409", res2.StatusCode)
	}
}

func TestCasesHandler_DeleteProtections(t *testing.T) {
	h, repo, srv := openCasesHandler(t)
	ctx := context.Background()

	menuRefs := 0
	activeSessions := 0
	activeTasks := 0
	h.CountMenuRefs = func(_ context.Context, _ string) (int, error) {
		return menuRefs, nil
	}
	h.CountActiveSessions = func(_ context.Context, _ sharedkernel.CaseID) (int, error) {
		return activeSessions, nil
	}
	h.CountActiveTasks = func(_ context.Context, _ sharedkernel.CaseID) (int, error) {
		return activeTasks, nil
	}

	doDelete := func(id string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/cases/"+id, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	// DELETE missing → 404
	res := doDelete("999999")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing delete status=%d want 404", res.StatusCode)
	}
	res.Body.Close()

	// Create enabled case.
	body, _ := json.Marshal(validCaseBody(10, "To Delete"))
	createRes, err := http.Post(srv.URL+"/api/v1/cases", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	createRes.Body.Close()
	if createRes.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d want 201", createRes.StatusCode)
	}

	// Enabled → 409 must disable first.
	res = doDelete("10")
	if res.StatusCode != http.StatusConflict || decodeErr(t, res) != "case must be disabled before deletion" {
		t.Fatalf("enabled delete status=%d want 409 disabled-first", res.StatusCode)
	}
	res.Body.Close()

	// Disable, then referenced by menu/card → 409.
	disRes, err := http.Post(srv.URL+"/api/v1/cases/10/disable", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	disRes.Body.Close()
	if disRes.StatusCode != http.StatusOK {
		t.Fatalf("disable status=%d want 200", disRes.StatusCode)
	}
	menuRefs = 1
	res = doDelete("10")
	if res.StatusCode != http.StatusConflict || decodeErr(t, res) != "case is referenced by menu or card entries" {
		t.Fatalf("menu-ref delete status=%d want 409 referenced", res.StatusCode)
	}
	res.Body.Close()

	// Active session (collecting/confirming) → 409.
	menuRefs = 0
	activeSessions = 1
	res = doDelete("10")
	if res.StatusCode != http.StatusConflict || decodeErr(t, res) != "case has active sessions" {
		t.Fatalf("active-session delete status=%d want 409 active-sessions", res.StatusCode)
	}
	res.Body.Close()

	// Active tasks → 409.
	activeSessions = 0
	menuRefs = 0
	activeTasks = 1
	res = doDelete("10")
	if res.StatusCode != http.StatusConflict || decodeErr(t, res) != "case has active tasks" {
		t.Fatalf("active-tasks delete status=%d want 409 active-tasks", res.StatusCode)
	}
	res.Body.Close()

	// No refs/tasks → 200 and row gone.
	activeSessions = 0
	activeTasks = 0
	res = doDelete("10")
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("delete status=%d want 200 body=%s", res.StatusCode, raw)
	}
	var deleted map[string]bool
	if err := json.NewDecoder(res.Body).Decode(&deleted); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if !deleted["deleted"] {
		t.Fatalf("delete dto=%v", deleted)
	}
	if _, err := repo.Get(ctx, sharedkernel.CaseID(10)); err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}
