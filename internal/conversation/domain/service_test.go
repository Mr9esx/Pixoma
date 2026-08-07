package domain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func newSvc() *domain.Service {
	repo := domain.NewMemoryRepository()
	var n int
	return domain.NewService(repo, func() sharedkernel.SessionID {
		n++
		return sharedkernel.SessionID(fmt.Sprintf("s-%d", n))
	}, func() time.Time { return time.Unix(1_700_000_000, 0).UTC() })
}

func TestStartCaseLocksAndRejectsSecond(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	s, err := svc.StartCase(ctx, 42, "c1", []string{"prompt", "seed"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != domain.StatusCollecting {
		t.Fatalf("status=%s", s.Status)
	}
	_, err = svc.StartCase(ctx, 42, "c2", []string{"a"})
	if !errors.Is(err, domain.ErrSessionLocked) {
		t.Fatalf("want locked, got %v", err)
	}
}

func TestSubmitThenSkipToConfirming(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, _ = svc.StartCase(ctx, 1, "c1", []string{"prompt", "seed"})
	prompt := "cat"
	s, err := svc.SubmitInput(ctx, 1, domain.DraftValue{Text: &prompt})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != domain.StatusCollecting || s.CurrentInputIndex != 1 {
		t.Fatalf("after prompt: %+v", s)
	}
	s, err = svc.SkipInput(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != domain.StatusConfirming {
		t.Fatalf("want confirming, got %s", s.Status)
	}
}

func TestExitUnlocks(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, _ = svc.StartCase(ctx, 7, "c1", []string{"prompt"})
	if err := svc.Exit(ctx, 7); err != nil {
		t.Fatal(err)
	}
	_, err := svc.StartCase(ctx, 7, "c2", []string{"x"})
	if err != nil {
		t.Fatalf("should unlock: %v", err)
	}
}

func TestSubmitWrongState(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, err := svc.SubmitInput(ctx, 9, domain.DraftValue{})
	if !errors.Is(err, domain.ErrNoActiveSession) {
		t.Fatalf("got %v", err)
	}
}
