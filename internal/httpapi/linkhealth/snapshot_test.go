package linkhealth_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	catalogpersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channelapp "github.com/mr9esx/comfyui_tgbot/internal/channel/application"
	channeldomain "github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	linkhealthapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/linkhealth"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
)

func TestCollect_NotBlockedDuringTelegramProbe(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:lh_collect_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(gdb,
		&channelpersist.ChannelRow{},
		&catalogpersist.CaseRow{},
		&topicpersist.TopicRow{},
		&instpersist.EdgeRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{},
	); err != nil {
		t.Fatal(err)
	}

	channelStore := channelpersist.NewGormRepository(gdb)
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	svc := &channelapp.Service{
		Store: channelStore,
		Key:   make([]byte, 32),
		FetchTelegram: func(context.Context, string) (json.RawMessage, error) {
			return nil, errors.New("offline")
		},
		CheckTelegram: func(_ context.Context, _ string) (channelapp.ReachabilityResult, error) {
			close(entered)
			<-release
			return channelapp.ReachabilityResult{OK: true, Kind: channelapp.ReachabilityOK}, nil
		},
	}
	ch, err := svc.Create(context.Background(), "on", channeldomain.PlatformTelegram, "开", "tok", "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	ch.LastCheckKind = string(channelapp.ReachabilityOK)
	ch.LastCheckAt = &now
	if err := channelStore.Update(context.Background(), ch); err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := svc.CheckReachability(context.Background(), "on")
		errCh <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("probe did not start")
	}

	start := time.Now()
	snap, err := linkhealthapi.Collect(context.Background(), linkhealthapi.Source{
		Channels: channelStore,
		Adapter: func(_ context.Context, _ string) (state string, lastErr string, found bool) {
			return "running", "", true
		},
		Menus:  mencardpersist.NewGormCardRepository(gdb),
		Cases:  catalogpersist.NewGormRepository(gdb),
		Topics: topicpersist.NewTopicRepository(gdb),
		Edges:  instpersist.NewEdgeRepository(gdb),
	})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("Collect blocked %s during CheckTelegram with MaxOpenConns(1)", elapsed)
	}
	if len(snap.Channels) != 1 || snap.Channels[0].LastCheckKind != string(channelapp.ReachabilityOK) {
		t.Fatalf("Collect must read stored last_check: %+v", snap.Channels)
	}
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
