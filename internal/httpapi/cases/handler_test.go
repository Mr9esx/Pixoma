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

func validCaseBody(id, name string) map[string]any {
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
		"menu_key": "create",
		"enabled":  true,
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
	body, _ := json.Marshal(validCaseBody("alpha-case", "Alpha Workflow"))
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
	if created["id"] != "alpha-case" || created["enabled"] != true {
		t.Fatalf("create dto=%v", created)
	}

	// POST invalid (empty name) → 400，库中无脏数据
	bad := validCaseBody("bad-case", "")
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
	if _, err := repo.Get(ctx, "bad-case"); err == nil {
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
	if len(list) != 1 || list[0]["id"] != "alpha-case" {
		t.Fatalf("list=%v", list)
	}

	// GET /{id}
	getRes, err := http.Get(srv.URL + "/api/v1/cases/alpha-case")
	if err != nil {
		t.Fatal(err)
	}
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getRes.StatusCode)
	}

	// PATCH update name
	patchBody, _ := json.Marshal(validCaseBody("alpha-case", "Alpha Updated"))
	patchReq, err := http.NewRequest(http.MethodPatch, srv.URL+"/api/v1/cases/alpha-case", bytes.NewReader(patchBody))
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
	disRes, err := http.Post(srv.URL+"/api/v1/cases/alpha-case/disable", "application/json", nil)
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
	got, err := repo.Get(ctx, sharedkernel.CaseID("alpha-case"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled {
		t.Fatal("repo still enabled after disable")
	}

	// POST /{id}/enable → enabled=true
	enRes, err := http.Post(srv.URL+"/api/v1/cases/alpha-case/enable", "application/json", nil)
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
	missing, err := http.Get(srv.URL + "/api/v1/cases/no-such")
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
	body, _ := json.Marshal(validCaseBody("dup-case", "Dup"))
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
