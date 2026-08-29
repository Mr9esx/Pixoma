package channeltext_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/channeltext"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func newTestHandler(t *testing.T) (http.Handler, *text.Store) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:ctest?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dbc, _ := gdb.DB(); _ = dbc.Close() })
	st, err := text.NewStore(gdb)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	h := &channeltext.Handler{Store: st}
	r.Route("/api/v1/text-templates", h.Mount)
	return r, st
}

type dto struct {
	Key     string   `json:"key"`
	Default string   `json:"default"`
	Value   string   `json:"value"`
	Vars    []string `json:"variables"`
}

func TestList_ReturnsFullCatalog(t *testing.T) {
	ctx := context.Background()
	h, st := newTestHandler(t)
	if err := st.Save(ctx, text.GlobalDefaultID, map[string]string{text.KeyWorkflowDone: "完成 {{ task_id }}"}); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/text-templates/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	var items []dto
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	var found *dto
	for i := range items {
		if items[i].Key == text.KeyWorkflowDone {
			found = &items[i]
			break
		}
	}
	if found == nil {
		t.Fatal("workflow_done missing from catalog")
	}
	if found.Value != "完成 {{ task_id }}" {
		t.Fatalf("stored value: %q", found.Value)
	}
	if len(items) < 8 {
		t.Fatalf("expected a full catalog, got %d items", len(items))
	}
}

func TestSave_UpdatesAndResets(t *testing.T) {
	h, _ := newTestHandler(t)

	body, _ := json.Marshal(map[string]any{"templates": map[string]string{text.KeyWelcome: "Hi {{ n }}"}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/text-templates/", bytes.NewReader(body))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("save status: %d body=%s", rr.Code, rr.Body.String())
	}
	var items []dto
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	byKey := map[string]dto{}
	for _, it := range items {
		byKey[it.Key] = it
	}
	if byKey["welcome"].Value != "Hi {{ n }}" {
		t.Fatalf("welcome after save: %q", byKey["welcome"].Value)
	}
}

func TestChannelScopedRoutes_ScopesToChannelID(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:ctest_ch?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dbc, _ := gdb.DB(); _ = dbc.Close() })
	st, err := text.NewStore(gdb)
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	h := &channeltext.Handler{Store: st}
	r.Route("/api/v1/channels", func(r chi.Router) {
		r.Route("/{id}", func(r chi.Router) { h.MountChannel(r) })
	})

	body, _ := json.Marshal(map[string]any{"templates": map[string]string{text.KeyWelcome: "消息平台欢迎"}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/channels/ch-1/text-templates", bytes.NewReader(body))
	h2 := http.Handler(r)
	h2.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("channel save status: %d body=%s", rr.Code, rr.Body.String())
	}
	// platform default must remain untouched
	if got := st.Render(context.Background(), "ch-1", text.KeyWelcome, nil); got != "消息平台欢迎" {
		t.Fatalf("channel render: got %q", got)
	}
	if got := st.Render(context.Background(), "other-ch", text.KeyWelcome, nil); got != "欢迎使用 ComfyUI Bot\n请选择功能：" {
		t.Fatalf("other channel should keep default: got %q", got)
	}
}
