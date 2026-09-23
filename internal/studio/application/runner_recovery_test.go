package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type recoveryExecutor struct{}

func (recoveryExecutor) Execute(context.Context, *domain.Run) error { return nil }

func TestRecoverRunsPagesQueuedAndInterruptsUnsafeRunning(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 205; i++ {
		run, err := domain.NewRun(fmt.Sprintf("queued-%03d", i), "session-recovery", "account-a", "message-recovery", now)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateRun(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	unsafe, err := domain.NewRun("running-unsafe", "session-recovery", "account-a", "message-recovery", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := unsafe.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, unsafe); err != nil {
		t.Fatal(err)
	}
	runner := studioapp.NewBackgroundRunner(repo, recoveryExecutor{}, studioapp.RunnerOptions{Workers: 4, QueueSize: 256})
	t.Cleanup(runner.Close)
	count, err := runner.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 205 {
		t.Fatalf("requeued %d runs, want 205", count)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		last, err := repo.GetRun(ctx, "account-a", "queued-204")
		if err != nil {
			t.Fatal(err)
		}
		if last.Status == domain.RunSucceeded {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("last queued run did not recover: %s", last.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	stale, err := repo.GetRun(ctx, "account-a", unsafe.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stale.Status != domain.RunFailed || stale.ErrorCode != "interrupted_on_restart" {
		t.Fatalf("unsafe running run = %#v", stale)
	}
}
