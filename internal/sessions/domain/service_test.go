package domain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/sessions/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func newSvc() *domain.Service {
	repo := domain.NewMemoryRepository()
	var n int
	return domain.NewService(repo, func() sharedkernel.SessionID {
		n++
		return sharedkernel.SessionID(fmt.Sprintf("s-%d", n))
	}, func() time.Time { return time.Unix(1_700_000_000, 0).UTC() })
}

func TestStartCaseReplacesActiveSession(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	s, err := svc.StartCase(ctx, "tg:42", "user-1", 1, []string{"prompt", "seed"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != domain.StatusCollecting {
		t.Fatalf("status=%s", s.Status)
	}
	// 重新开启新会话：旧会话自动退出，新会话正常开始。
	next, err := svc.StartCase(ctx, "tg:42", "user-1", 2, []string{"a"})
	if err != nil {
		t.Fatalf("second start must auto-exit old session: %v", err)
	}
	if next.CaseID != 2 || next.Status != domain.StatusCollecting {
		t.Fatalf("new session=%+v", next)
	}
	cur, err := svc.Get(ctx, "tg:42")
	if err != nil {
		t.Fatal(err)
	}
	if cur.ID != next.ID || cur.CaseID != 2 {
		t.Fatalf("active session must be the new one, got %+v", cur)
	}
}

func TestStartCaseWritesUserID(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	s, err := svc.StartCase(ctx, "tg:42", "user-abc", 1, []string{"prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if s.UserID != "user-abc" {
		t.Fatalf("UserID=%q", s.UserID)
	}
	got, err := svc.Get(ctx, "tg:42")
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != "user-abc" {
		t.Fatalf("persisted UserID=%q", got.UserID)
	}
}

func TestStartCaseRejectsEmptyUserID(t *testing.T) {
	svc := newSvc()
	_, err := svc.StartCase(context.Background(), "tg:1", "", 1, []string{"a"})
	if !errors.Is(err, domain.ErrEmptyUserID) {
		t.Fatalf("want ErrEmptyUserID, got %v", err)
	}
}

func TestSubmitThenSkipToConfirming(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, _ = svc.StartCase(ctx, "tg:1", "u1", 1, []string{"prompt", "seed"})
	prompt := "cat"
	s, err := svc.SubmitInput(ctx, "tg:1", domain.DraftValue{Text: &prompt})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != domain.StatusCollecting || s.CurrentInputIndex != 1 {
		t.Fatalf("after prompt: %+v", s)
	}
	s, err = svc.SkipInput(ctx, "tg:1")
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
	_, _ = svc.StartCase(ctx, "tg:7", "u1", 1, []string{"prompt"})
	if err := svc.Exit(ctx, "tg:7"); err != nil {
		t.Fatal(err)
	}
	_, err := svc.StartCase(ctx, "tg:7", "u1", 2, []string{"x"})
	if err != nil {
		t.Fatalf("should unlock: %v", err)
	}
}

func TestSubmitWrongState(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	_, err := svc.SubmitInput(ctx, "tg:9", domain.DraftValue{})
	if !errors.Is(err, domain.ErrNoActiveSession) {
		t.Fatalf("got %v", err)
	}
}

func TestMemoryKeepsSubmittedByID(t *testing.T) {
	repo := domain.NewMemoryRepository()
	svc := domain.NewService(repo, func() sharedkernel.SessionID { return "s-keep" }, func() time.Time {
		return time.Unix(1, 0).UTC()
	})
	ctx := context.Background()
	s, err := svc.StartCase(ctx, "tg:3", "u1", 1, []string{"prompt"})
	if err != nil {
		t.Fatal(err)
	}
	prompt := "x"
	s, err = svc.SubmitInput(ctx, "tg:3", domain.DraftValue{Text: &prompt})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSubmitted(time.Unix(2, 0).UTC()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, s); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetActiveByChat(ctx, "tg:3"); !errors.Is(err, domain.ErrNoActiveSession) {
		t.Fatalf("want no active, got %v", err)
	}
	byID, err := repo.GetByID(ctx, "s-keep")
	if err != nil || byID.Status != domain.StatusSubmitted || byID.UserID != "u1" {
		t.Fatalf("submitted missing: %+v %v", byID, err)
	}
}
