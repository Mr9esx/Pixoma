package application

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/domain"
)

func TestReachabilityProbe_ProbeOncePersistsEnabledOnly(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	var probed atomic.Int32
	svc := &Service{
		Store:         store,
		Key:           make([]byte, 32),
		FetchTelegram: offlineFetch,
		CheckTelegram: func(_ context.Context, token string) (ReachabilityResult, error) {
			probed.Add(1)
			return ReachabilityResult{OK: true, Kind: ReachabilityOK, Message: token}, nil
		},
	}
	if _, err := svc.Create(context.Background(), "on", domain.PlatformTelegram, "开", "tok-on", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "off", domain.PlatformTelegram, "关", "tok-off", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Disable(context.Background(), "off"); err != nil {
		t.Fatal(err)
	}

	probe := &ReachabilityProbe{Svc: svc}
	if err := probe.ProbeOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if probed.Load() != 1 {
		t.Fatalf("probed=%d want 1", probed.Load())
	}
	on, err := svc.Get(context.Background(), "on")
	if err != nil {
		t.Fatal(err)
	}
	if on.LastCheckKind != string(ReachabilityOK) || on.LastCheckAt == nil {
		t.Fatalf("enabled last check: %+v", on)
	}
	off, err := svc.Get(context.Background(), "off")
	if err != nil {
		t.Fatal(err)
	}
	if off.LastCheckKind != "" || off.LastCheckAt != nil {
		t.Fatalf("disabled must not be probed: %+v", off)
	}
}

func TestReachabilityProbe_ProbeOnceContinuesAfterOneFailure(t *testing.T) {
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{
		Store:         store,
		Key:           make([]byte, 32),
		FetchTelegram: offlineFetch,
		CheckTelegram: func(_ context.Context, token string) (ReachabilityResult, error) {
			if token == "bad" {
				return ReachabilityResult{}, errors.New("decrypt boom")
			}
			return ReachabilityResult{OK: false, Kind: ReachabilityNetwork, Message: "i/o timeout"}, nil
		},
	}
	if _, err := svc.Create(context.Background(), "bad", domain.PlatformTelegram, "坏", "bad", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "ok", domain.PlatformTelegram, "好", "good", ""); err != nil {
		t.Fatal(err)
	}

	probe := &ReachabilityProbe{Svc: svc}
	if err := probe.ProbeOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), "ok")
	if err != nil {
		t.Fatal(err)
	}
	if got.LastCheckKind != string(ReachabilityNetwork) || got.LastCheckAt == nil {
		t.Fatalf("second channel must still persist: %+v", got)
	}
}

func TestReachabilityProbe_OverlappingProbeOnceDoesNotStampede(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	var probed atomic.Int32
	var enterOnce atomic.Bool
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{
		Store:         store,
		Key:           make([]byte, 32),
		FetchTelegram: offlineFetch,
		CheckTelegram: func(_ context.Context, _ string) (ReachabilityResult, error) {
			probed.Add(1)
			if enterOnce.CompareAndSwap(false, true) {
				close(entered)
			}
			<-release
			return ReachabilityResult{OK: true, Kind: ReachabilityOK}, nil
		},
	}
	if _, err := svc.Create(context.Background(), "on", domain.PlatformTelegram, "开", "tok", ""); err != nil {
		t.Fatal(err)
	}

	probe := &ReachabilityProbe{Svc: svc}
	done := make(chan error, 1)
	go func() { done <- probe.ProbeOnce(context.Background()) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first ProbeOnce did not start")
	}
	secondDone := make(chan error, 1)
	go func() { secondDone <- probe.ProbeOnce(context.Background()) }()
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("overlapping ProbeOnce blocked instead of returning")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if probed.Load() != 1 {
		t.Fatalf("overlapping ProbeOnce must not stampede Telegram, probed=%d", probed.Load())
	}
}

func TestCheckReachability_ListNotBlockedDuringTelegram(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{
		Store:         store,
		Key:           make([]byte, 32),
		FetchTelegram: offlineFetch,
		CheckTelegram: func(_ context.Context, _ string) (ReachabilityResult, error) {
			close(entered)
			<-release
			return ReachabilityResult{OK: true, Kind: ReachabilityOK}, nil
		},
	}
	if _, err := svc.Create(context.Background(), "on", domain.PlatformTelegram, "开", "tok", ""); err != nil {
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
	if _, err := svc.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("List blocked %s during CheckTelegram", elapsed)
	}
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestReachabilityProbe_DefaultIntervalIs30s(t *testing.T) {
	if got := (&ReachabilityProbe{}).interval(); got != 30*time.Second {
		t.Fatalf("interval=%s want 30s", got)
	}
}

func TestReachabilityProbe_ProbeOnceLogsSuccess(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	store := &memStore{rows: map[string]domain.Channel{}}
	svc := &Service{
		Store:         store,
		Key:           make([]byte, 32),
		FetchTelegram: offlineFetch,
		CheckTelegram: func(_ context.Context, token string) (ReachabilityResult, error) {
			return ReachabilityResult{OK: true, Kind: ReachabilityOK, Message: token}, nil
		},
	}
	if _, err := svc.Create(context.Background(), "on", domain.PlatformTelegram, "开", "tok-on", ""); err != nil {
		t.Fatal(err)
	}
	if err := (&ReachabilityProbe{Svc: svc}).ProbeOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "channel reachability probe") || !strings.Contains(got, "channel=on") {
		t.Fatalf("success log missing: %s", got)
	}
}
