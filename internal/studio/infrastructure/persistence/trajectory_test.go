package persistence_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestTraceListProjectionOmitsRawProviderBodies(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	event := &domain.Event{ID: "request-1", AccountID: "account-a", SessionID: "session-a", RunID: "run-a", Sequence: 1, Type: "MODEL_REQUEST_STARTED", Payload: json.RawMessage(`{"attempt_id":"attempt-1","model":"test-model","request_body":{"messages":[{"content":"private prompt"}]}}`)}
	if err := repo.AppendEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	summary, err := repo.ListRunTraceSummaryEvents(ctx, "account-a", "run-a")
	if err != nil || len(summary) != 1 {
		t.Fatalf("summary = %#v, %v", summary, err)
	}
	if json.Valid(summary[0].Payload) == false || bytes.Contains(summary[0].Payload, []byte("private prompt")) || !bytes.Contains(summary[0].Payload, []byte("attempt-1")) {
		t.Fatalf("summary payload = %s", summary[0].Payload)
	}
	full, err := repo.ListRunTraceEvents(ctx, "account-a", "run-a")
	if err != nil || len(full) != 1 || !bytes.Contains(full[0].Payload, []byte("private prompt")) {
		t.Fatalf("full event = %#v, %v", full, err)
	}
}

func TestSessionTraceRunsPagesBeyondLegacyLimitAndScopesAccount(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("session-1", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 23; i++ {
		run, err := domain.NewRun(fmt.Sprintf("run-%02d", i), session.ID, session.AccountID, fmt.Sprintf("message-%02d", i), now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateRun(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListSessionTraceRuns(ctx, "account-a", session.ID, time.Time{}, "", 20)
	if err != nil || len(first) != 20 || first[0].ID != "run-22" {
		t.Fatalf("first page = %#v, %v", first, err)
	}
	last := first[len(first)-1]
	second, err := repo.ListSessionTraceRuns(ctx, "account-a", session.ID, last.CreatedAt, last.ID, 20)
	if err != nil || len(second) != 3 || second[0].ID != "run-02" {
		t.Fatalf("second page = %#v, %v", second, err)
	}
	if other, err := repo.ListSessionTraceRuns(ctx, "account-b", session.ID, time.Time{}, "", 20); err != nil || len(other) != 0 {
		t.Fatalf("cross-account rows = %#v, %v", other, err)
	}
	if _, err := repo.GetRun(ctx, "account-b", first[0].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account run = %v", err)
	}
}
