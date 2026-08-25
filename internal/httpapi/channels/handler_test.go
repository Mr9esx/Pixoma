package channels_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openChannelsServer(t *testing.T) (*httptest.Server, *channelapp.Service) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:ch_http_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	svc := &channelapp.Service{
		Store:         channelpersist.NewGormRepository(gdb),
		Key:           make([]byte, 32),
	}
	h := &channelsapi.Handler{Svc: svc}
	r := chi.NewRouter()
	r.Route("/api/v1/channels", func(r chi.Router) {
		h.Mount(r)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, svc
}

func post(t *testing.T, url string, body any) (*http.Response, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	res, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(res.Body).Decode(&m)
	return res, m
}

func TestChannelsHandler_CreateListDetailUpdateDelete(t *testing.T) {
	srv, _ := openChannelsServer(t)

	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	if m["token_masked"] != "1234****7890" {
		t.Fatalf("masked=%v", m["token_masked"])
	}
	if m["enabled"] != true {
		t.Fatalf("enabled=%v", m["enabled"])
	}
	id := m["id"].(string)

	// 非法平台 400
	res, m = post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "slack", "name": "x", "token": "t",
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad platform status=%d", res.StatusCode)
	}

	// 列表
	listRes, err := http.Get(srv.URL + "/api/v1/channels")
	if err != nil {
		t.Fatal(err)
	}
	var list []map[string]any
	_ = json.NewDecoder(listRes.Body).Decode(&list)
	listRes.Body.Close()
	if len(list) != 1 {
		t.Fatalf("list len=%d", len(list))
	}

	// 详情
	detailRes, err := http.Get(srv.URL + "/api/v1/channels/" + id)
	if err != nil {
		t.Fatal(err)
	}
	var detail map[string]any
	_ = json.NewDecoder(detailRes.Body).Decode(&detail)
	detailRes.Body.Close()
	if detail["name"] != "主机器人" {
		t.Fatalf("detail=%v", detail)
	}

	// 更新名称，token 留空不变
	raw, _ := json.Marshal(map[string]any{"name": "新名字", "token": ""})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/v1/channels/"+id, bytes.NewReader(raw))
	updRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var upd map[string]any
	_ = json.NewDecoder(updRes.Body).Decode(&upd)
	updRes.Body.Close()
	if upd["name"] != "新名字" || upd["token_masked"] != "1234****7890" {
		t.Fatalf("update=%v", upd)
	}

	// 启用中直接删除成功（不再要求停用）。
	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/channels/"+id, nil)
	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	delRes.Body.Close()
	if delRes.StatusCode != http.StatusOK {
		t.Fatalf("delete enabled status=%d want 200", delRes.StatusCode)
	}
}

func TestChannelsHandler_CheckReachability(t *testing.T) {
	srv, svc := openChannelsServer(t)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	id := m["id"].(string)

	var gotToken string
	svc.CheckTelegram = func(_ context.Context, token string) (channelapp.ReachabilityResult, error) {
		gotToken = token
		return channelapp.ReachabilityResult{OK: false, Kind: channelapp.ReachabilityNetwork, Message: "dial timeout"}, nil
	}

	checkRes, err := http.Post(srv.URL+"/api/v1/channels/"+id+"/check", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer checkRes.Body.Close()
	var out channelapp.ReachabilityResult
	_ = json.NewDecoder(checkRes.Body).Decode(&out)
	if checkRes.StatusCode != http.StatusOK {
		t.Fatalf("check status=%d", checkRes.StatusCode)
	}
	if gotToken != "1234567890" {
		t.Fatalf("token=%q", gotToken)
	}
	if out.OK || out.Kind != channelapp.ReachabilityNetwork {
		t.Fatalf("out=%+v", out)
	}

	notFoundRes, err := http.Post(srv.URL+"/api/v1/channels/missing/check", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	notFoundRes.Body.Close()
	if notFoundRes.StatusCode != http.StatusNotFound {
		t.Fatalf("not found status=%d", notFoundRes.StatusCode)
	}
}
