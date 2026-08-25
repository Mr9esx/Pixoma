package adminhost_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	channelsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/channels"
	menucardsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/menucards"
	statsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/stats"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

type statsFakeRepo struct{}

func (statsFakeRepo) AddTerminal(context.Context, taskstats.AddTerminalInput) error { return nil }
func (statsFakeRepo) ListDaily(context.Context, string, string) ([]taskstats.DailyRow, error) {
	return nil, nil
}
func (statsFakeRepo) ListErrors(context.Context, string, string, int) ([]taskstats.ErrorRow, error) {
	return nil, nil
}
func (statsFakeRepo) ListEdges(context.Context, string, string) ([]taskstats.EdgeRow, error) {
	return nil, nil
}
func (statsFakeRepo) ListCases(context.Context, string, string, int) ([]taskstats.CaseRow, error) {
	return nil, nil
}
func (statsFakeRepo) Prune(context.Context, string) error { return nil }

func TestStatsRouteMounted(t *testing.T) {
	h := adminhost.NewHandler(adminhost.Options{
		Stats: &statsapi.Handler{Repo: statsFakeRepo{}, Loc: time.FixedZone("CST", 8*3600)},
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/v1/stats/tasks/daily?from=2026-08-01&to=2026-08-02")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status=%d", resp.StatusCode)
	}
	var body struct {
		Days []any `json:"days"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Days) != 2 {
		t.Fatalf("days=%d, want 2 (zero-filled)", len(body.Days))
	}
}

func TestChannelDetailRouteNotShadowedByMenuMount(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:adminhost_ch_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		gdb,
		&channelpersist.ChannelRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{},
	); err != nil {
		t.Fatal(err)
	}

	chSvc := &channelapp.Service{
		Store:         channelpersist.NewGormRepository(gdb),
		Key:           make([]byte, 32),
	}
	chAPI := &channelsapi.Handler{Svc: chSvc}
	menuCardsAPI := menucardsapi.NewHandler(mencardpersist.NewGormCardRepository(gdb))

	h := adminhost.NewHandler(adminhost.Options{
		Channels:  chAPI,
		MenuCards: menuCardsAPI,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	// create a channel
	raw, _ := json.Marshal(map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	})
	createResp, err := http.Post(srv.URL+"/api/v1/channels", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}
	var created struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	createResp.Body.Close()

	// channel detail must resolve (not shadowed by the /{id} menu subroute)
	getResp, err := http.Get(srv.URL + "/api/v1/channels/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(getResp.Body)
		t.Fatalf("detail status=%d body=%s", getResp.StatusCode, body.String())
	}
	var got struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Fatalf("detail=%+v", got)
	}

	// menu subroute still works
	menuResp, err := http.Get(srv.URL + "/api/v1/channels/" + created.ID + "/menu")
	if err != nil {
		t.Fatal(err)
	}
	defer menuResp.Body.Close()
	if menuResp.StatusCode != http.StatusOK {
		t.Fatalf("menu status=%d", menuResp.StatusCode)
	}
}
