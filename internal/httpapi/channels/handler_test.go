package channels_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func openChannelsServer(t *testing.T) (*httptest.Server, *channelapp.Service) {
	return openChannelsServerPool(t, 0)
}

func openChannelsServerPool(t *testing.T, maxOpen int) (*httptest.Server, *channelapp.Service) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:ch_http_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if maxOpen > 0 {
		sqlDB, err := gdb.DB()
		if err != nil {
			t.Fatal(err)
		}
		sqlDB.SetMaxIdleConns(maxOpen)
		sqlDB.SetMaxOpenConns(maxOpen)
	}
	if err := db.AutoMigrate(gdb, &channelpersist.ChannelRow{}); err != nil {
		t.Fatal(err)
	}
	svc := &channelapp.Service{
		Store: channelpersist.NewGormRepository(gdb),
		Key:   make([]byte, 32),
		FetchTelegram: func(context.Context, string) (json.RawMessage, error) {
			return nil, errors.New("offline")
		},
	}
	h := &channelsapi.Handler{
		Svc:      svc,
		Probe:    &channelapp.ReachabilityProbe{Svc: svc},
		ProbeCtx: context.Background(),
	}
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

func TestChannelsHandler_CheckRouteRemoved(t *testing.T) {
	srv, _ := openChannelsServer(t)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	id := m["id"].(string)
	checkRes, err := http.Post(srv.URL+"/api/v1/channels/"+id+"/check", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	checkRes.Body.Close()
	if checkRes.StatusCode != http.StatusNotFound {
		t.Fatalf("check status=%d want 404", checkRes.StatusCode)
	}
}

func TestChannelsHandler_KickProbeReturnsBeforeTelegram(t *testing.T) {
	srv, svc := openChannelsServer(t)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "开", "token": "tok",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	id := m["id"].(string)

	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	svc.CheckTelegram = func(_ context.Context, _ string) (channelapp.ReachabilityResult, error) {
		close(entered)
		<-release
		return channelapp.ReachabilityResult{OK: true, Kind: channelapp.ReachabilityOK}, nil
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	start := time.Now()
	kickRes, err := client.Post(srv.URL+"/api/v1/channels/probe", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /probe blocked on Telegram: %v", err)
	}
	defer kickRes.Body.Close()
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("POST /probe took %s, want immediate 202", elapsed)
	}
	if kickRes.StatusCode != http.StatusAccepted {
		t.Fatalf("kick status=%d want 202", kickRes.StatusCode)
	}

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("ProbeOnce did not run")
	}
	close(release)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		stored, err := svc.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if stored.LastCheckKind == string(channelapp.ReachabilityOK) && stored.LastCheckAt != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("ProbeOnce must still persist last_check")
}

func TestChannelsHandler_ListDetailExposeAdapterState(t *testing.T) {
	srv, svc := openChannelsServer(t)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	id := m["id"].(string)

	svc.AdapterStatus = func(_ context.Context, _ string) (state string, lastErr string, found bool) {
		return "error", "dial timeout", true
	}

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
	if list[0]["adapter_state"] != "error" || list[0]["adapter_error"] != "dial timeout" {
		t.Fatalf("list adapter=%v", list[0])
	}

	detailRes, err := http.Get(srv.URL + "/api/v1/channels/" + id)
	if err != nil {
		t.Fatal(err)
	}
	var detail map[string]any
	_ = json.NewDecoder(detailRes.Body).Decode(&detail)
	detailRes.Body.Close()
	if detail["adapter_state"] != "error" || detail["adapter_error"] != "dial timeout" {
		t.Fatalf("detail adapter=%v", detail)
	}
}

func TestChannelsHandler_ListGetNotBlockedDuringTelegram(t *testing.T) {
	srv, svc := openChannelsServerPool(t, 1)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "开", "token": "tok",
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	id := m["id"].(string)

	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	svc.CheckTelegram = func(_ context.Context, _ string) (channelapp.ReachabilityResult, error) {
		close(entered)
		<-release
		return channelapp.ReachabilityResult{OK: true, Kind: channelapp.ReachabilityOK}, nil
	}
	svc.AdapterStatus = func(_ context.Context, _ string) (state string, lastErr string, found bool) {
		return "running", "", true
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := svc.CheckReachability(context.Background(), id)
		errCh <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("probe did not start")
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	listRes, err := client.Get(srv.URL + "/api/v1/channels")
	if err != nil {
		t.Fatalf("GET /channels during probe: %v", err)
	}
	listRes.Body.Close()
	if listRes.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listRes.StatusCode)
	}
	getRes, err := client.Get(srv.URL + "/api/v1/channels/" + id)
	if err != nil {
		t.Fatalf("GET /channels/{id} during probe: %v", err)
	}
	getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getRes.StatusCode)
	}
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestChannelsHandler_CreateStoresExtraInfo(t *testing.T) {
	srv, _ := openChannelsServer(t)
	res, m := post(t, srv.URL+"/api/v1/channels", map[string]any{
		"platform":   "telegram",
		"name":       "Demo",
		"token":      "t",
		"extra_info": `{"username":"demo_bot"}`,
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d body=%v", res.StatusCode, m)
	}
	extra, ok := m["extra_info"].(map[string]any)
	if !ok {
		t.Fatalf("extra_info=%v", m["extra_info"])
	}
	if extra["username"] != "demo_bot" {
		t.Fatalf("username=%v", extra["username"])
	}
}
