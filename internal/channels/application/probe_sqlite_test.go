package application

import (
	"context"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
	"github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func TestCheckReachability_SingleOpenConnDoesNotBlockList(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:ch_probe_" + t.Name() + "?mode=memory&cache=shared"})
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
	if err := db.AutoMigrate(gdb, &persistence.ChannelRow{}); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	svc := &Service{
		Store:         persistence.NewGormRepository(gdb),
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
	if _, err := svc.Get(context.Background(), "on"); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("List/Get blocked %s during CheckTelegram with MaxOpenConns(1)", elapsed)
	}
	close(release)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
