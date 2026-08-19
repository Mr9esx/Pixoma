package channelmenu_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/capability"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	channelmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channelmenu"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	menuapp "github.com/mr9esx/comfyui_tgbot/internal/menu/application"
	menupersist "github.com/mr9esx/comfyui_tgbot/internal/menu/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func openServer(t *testing.T) *httptest.Server {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:cm_http_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&channelpersist.ChannelRow{},
		&menupersist.ChannelMenuRow{},
		&menupersist.ChannelMenuItemRow{},
		&menupersist.ChannelMenuItemCaseRow{},
		&menupersist.ChannelMenuExtraRow{},
		&casepersist.CaseRow{},
	); err != nil {
		t.Fatal(err)
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	_ = caseRepo.Create(t.Context(), &catalogdomain.Case{
		Enabled: true,
		Document: catalogdomain.CaseDocument{
			ID:       sharedkernel.CaseID("c1"),
			Name:     "C1",
			Inputs:   []catalogdomain.InputField{{Key: "prompt", Type: "string", Required: true}},
			Bindings: catalogdomain.ComfyBindings{WorkflowJSON: map[string]any{"1": map[string]any{}}},
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{
				"prompt": map[string]any{"type": "string"},
			}},
		},
	})
	chSvc := &channelapp.Service{
		Store:         channelpersist.NewGormRepository(gdb),
		Key:           make([]byte, 32),
		HasActiveRefs: func(context.Context, string) (bool, error) { return false, nil },
	}
	menuSvc := &menuapp.Service{
		Store: menupersist.NewGormRepository(gdb),
		Cases: menuapp.CatalogCaseChecker{Repo: caseRepo},
	}
	h := &channelmenuapi.Handler{Channels: chSvc, Svc: menuSvc}
	chH := &channelsapi.Handler{Svc: chSvc}
	r := chi.NewRouter()
	r.Route("/api/v1/channels", func(r chi.Router) {
		chH.Mount(r)
		r.Route("/{id}/menu", func(r chi.Router) {
			h.MountMenu(r)
		})
	})
	r.Route("/api/v1/cases", func(r chi.Router) {
		r.Get("/{id}/menu-placements", h.ListPlacements)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestChannelMenuHandler_ScopedTreeAndExtras(t *testing.T) {
	srv := openServer(t)

	// 未知渠道 404
	res, _ := http.Get(srv.URL + "/api/v1/channels/nope/menu")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown channel status=%d", res.StatusCode)
	}
	res.Body.Close()

	// 创建渠道
	raw, _ := json.Marshal(map[string]any{"platform": "telegram", "name": "主", "token": "1234567890"})
	createRes, err := http.Post(srv.URL+"/api/v1/channels", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	_ = json.NewDecoder(createRes.Body).Decode(&created)
	createRes.Body.Close()
	id := created["id"].(string)

	// 空菜单 → 种子
	menuRes, _ := http.Get(srv.URL + "/api/v1/channels/" + id + "/menu")
	var tree map[string]any
	_ = json.NewDecoder(menuRes.Body).Decode(&tree)
	menuRes.Body.Close()
	if tree["channel_id"] != id {
		t.Fatalf("channel_id=%v", tree["channel_id"])
	}
	items := tree["items"].([]any)
	if len(items) != 6 {
		t.Fatalf("seed items=%d", len(items))
	}

	// 非法树 400
	bad := map[string]any{"items": []any{}}
	rawBad, _ := json.Marshal(bad)
	putRes, err := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/channels/"+id+"/menu", bytes.NewReader(rawBad))
	if err != nil {
		t.Fatal(err)
	}
	badResp, err := http.DefaultClient.Do(putRes)
	if err != nil {
		t.Fatal(err)
	}
	badResp.Body.Close()
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty tree status=%d", badResp.StatusCode)
	}

	// extras 写入读取
	extras := map[string]any{"btn-image": []any{
		map[string]any{"channel_id": id, "menu_item_id": "btn-image", "extra_type": "tg_root_layout", "extra_json": `{"columns":2}`},
	}}
	rawExtras, _ := json.Marshal(extras)
	extrasReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/channels/"+id+"/menu/extras", bytes.NewReader(rawExtras))
	extrasResp, err := http.DefaultClient.Do(extrasReq)
	if err != nil {
		t.Fatal(err)
	}
	extrasResp.Body.Close()
	if extrasResp.StatusCode != http.StatusOK {
		t.Fatalf("put extras status=%d", extrasResp.StatusCode)
	}
	gotExtrasRes, _ := http.Get(srv.URL + "/api/v1/channels/" + id + "/menu/extras")
	var gotExtras map[string][]any
	_ = json.NewDecoder(gotExtrasRes.Body).Decode(&gotExtras)
	gotExtrasRes.Body.Close()
	if len(gotExtras["btn-image"]) != 1 {
		t.Fatalf("extras=%v", gotExtras)
	}

	// 预览 DTO
	prevRes, _ := http.Get(srv.URL + "/api/v1/channels/" + id + "/menu/preview")
	if prevRes.StatusCode != http.StatusOK {
		t.Fatalf("preview status=%d", prevRes.StatusCode)
	}
	var prev map[string]any
	_ = json.NewDecoder(prevRes.Body).Decode(&prev)
	prevRes.Body.Close()
	main := prev["main_keyboard"].([]any)
	if len(main) == 0 {
		t.Fatalf("preview main=%v", prev)
	}

	// 载荷缺 menu_item_id 时以 map key 为准
	extrasNoID := map[string]any{"btn-image": []any{
		map[string]any{"extra_type": "tg_root_layout", "extra_json": `{"columns":3}`},
	}}
	rawExtrasNoID, _ := json.Marshal(extrasNoID)
	noIDReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/channels/"+id+"/menu/extras", bytes.NewReader(rawExtrasNoID))
	noIDResp, err := http.DefaultClient.Do(noIDReq)
	if err != nil {
		t.Fatal(err)
	}
	noIDResp.Body.Close()
	if noIDResp.StatusCode != http.StatusOK {
		t.Fatalf("put extras without menu_item_id status=%d", noIDResp.StatusCode)
	}
	gotExtrasRes2, _ := http.Get(srv.URL + "/api/v1/channels/" + id + "/menu/extras")
	var gotExtras2 map[string][]any
	_ = json.NewDecoder(gotExtrasRes2.Body).Decode(&gotExtras2)
	gotExtrasRes2.Body.Close()
	if len(gotExtras2["btn-image"]) != 1 {
		t.Fatalf("extras after no-id put=%v", gotExtras2)
	}

	// 非法 extras 400
	badExtras := map[string]any{"btn-image": []any{
		map[string]any{"extra_type": "tg_root_layout", "extra_json": "not-json"},
	}}
	rawBadExtras, _ := json.Marshal(badExtras)
	badExtrasReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/channels/"+id+"/menu/extras", bytes.NewReader(rawBadExtras))
	badExtrasResp, err := http.DefaultClient.Do(badExtrasReq)
	if err != nil {
		t.Fatal(err)
	}
	badExtrasResp.Body.Close()
	if badExtrasResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad extras status=%d", badExtrasResp.StatusCode)
	}

	// placements 含渠道
	plRes, _ := http.Get(srv.URL + "/api/v1/cases/c1/menu-placements")
	var placements []map[string]any
	_ = json.NewDecoder(plRes.Body).Decode(&placements)
	plRes.Body.Close()
	if len(placements) != 0 {
		t.Fatalf("placements expected empty, got %v", placements)
	}
}

func TestListCapabilitiesIncludesParamsSchema(t *testing.T) {
	reg := capability.NewRegistry()
	if err := reg.Register(capability.ReplyText{}); err != nil {
		t.Fatal(err)
	}
	h := &channelmenuapi.Handler{Capabilities: reg}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/capabilities", nil)
	w := httptest.NewRecorder()
	h.ListCapabilities(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("caps=%+v", out)
	}
	schema, ok := out[0]["params_schema"].(map[string]any)
	if !ok {
		t.Fatalf("params_schema missing: %+v", out[0])
	}
	if out[0]["display_name"] != "提示文字" {
		t.Fatalf("display_name=%v", out[0]["display_name"])
	}
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["text"]; !ok {
		t.Fatalf("schema=%v", schema)
	}
}
