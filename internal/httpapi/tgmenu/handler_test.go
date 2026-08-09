package tgmenu_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	tgmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tgmenu"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
	tgmenuapp "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	tgmenupersist "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/infrastructure/persistence"
)

func openTgMenuServer(t *testing.T) *httptest.Server {
	t.Helper()
	dsn := "file:tgmenu_http_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &tgmenupersist.MenuRow{}, &casepersist.CaseRow{}); err != nil {
		t.Fatal(err)
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	_ = caseRepo.Create(t.Context(), &catalogdomain.Case{
		Enabled: true,
		Document: catalogdomain.CaseDocument{
			ID:   sharedkernel.CaseID("c1"),
			Name: "C1",
			Inputs: []catalogdomain.InputField{
				{Key: "prompt", Type: "string", Required: true},
			},
			Bindings: catalogdomain.ComfyBindings{WorkflowJSON: map[string]any{"1": map[string]any{}}},
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string"},
				},
			},
		},
	})
	menuStore := tgmenupersist.NewGormRepository(gdb)
	svc := &tgmenuapp.Service{
		Store: menuStore,
		Cases: tgmenuapp.CatalogCaseChecker{Repo: caseRepo},
	}
	h := &tgmenuapi.Handler{Svc: svc}
	r := chi.NewRouter()
	r.Route("/api/v1/tg-menu", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestTgMenuHandler_GetPutValidation(t *testing.T) {
	srv := openTgMenuServer(t)

	res, err := http.Get(srv.URL + "/api/v1/tg-menu")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("get status=%d body=%s", res.StatusCode, raw)
	}
	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	items, _ := got["items"].([]any)
	if len(items) != 6 {
		t.Fatalf("seed items=%d", len(items))
	}

	bad := map[string]any{
		"items": []map[string]any{{
			"id": "btn-image", "label": "🖼 图片", "row": 0, "col": 0, "enabled": true,
			"action": "open_case", "case_id": "missing",
		}},
	}
	body, _ := json.Marshal(bad)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/tg-menu", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for missing case, got %d", res.StatusCode)
	}

	ok := map[string]any{
		"items": []map[string]any{{
			"id": "btn-image", "label": "🖼 图片", "row": 0, "col": 0, "enabled": true,
			"action": "open_case", "case_id": "c1",
		}},
	}
	body, _ = json.Marshal(ok)
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/api/v1/tg-menu", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("put ok status=%d body=%s", res.StatusCode, raw)
	}

	ftp := map[string]any{
		"items": []map[string]any{{
			"id": "btn-help", "label": "🆘 帮助", "row": 0, "col": 0, "enabled": true,
			"action": "reply_media",
			"reply":  map[string]any{"images": []string{"ftp://x/a.png"}},
		}},
	}
	body, _ = json.Marshal(ftp)
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/api/v1/tg-menu", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for ftp url, got %d", res.StatusCode)
	}
}
