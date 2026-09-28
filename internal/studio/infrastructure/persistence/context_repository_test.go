package persistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestSessionContextProjectionAndOwnership(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("context-session", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun("context-run", session.ID, session.AccountID, "message-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	events := []struct{ kind, payload string }{
		{"CONTEXT_INJECTED", `{"source":"run_context","detail":"会话运行上下文","delta_tokens_estimated":12}`},
		{"MODEL_REQUEST_STARTED", `{"attempt_id":"context-run:model:1","turn_id":"message-1","purpose":"agent","model":"test-model","protocol":"openai_chat_compatible","context_window_tokens":1000,"at":"2026-09-28T10:00:01Z","request_body":{"messages":[{"role":"system","content":"rules"},{"role":"user","content":"hello"}],"tools":[]}}`},
		{"MODEL_FIRST_TOKEN", `{"attempt_id":"context-run:model:1","at":"2026-09-28T10:00:02Z"}`},
		{"MODEL_REQUEST_FINISHED", `{"attempt_id":"context-run:model:1","at":"2026-09-28T10:00:03Z","input_tokens":100,"output_tokens":20,"cache_read_tokens":60,"reasoning_tokens":5}`},
		{"CONTEXT_PRUNED", `{"source":"tool_results","detail":"清理工具结果","delta_tokens_estimated":-10}`},
	}
	for index, item := range events {
		if _, err := repo.AppendRunEvent(ctx, &domain.Event{
			ID: "context-event-" + string(rune('a'+index)), RunID: run.ID, SessionID: session.ID, AccountID: session.AccountID,
			Type: item.kind, Payload: json.RawMessage(item.payload), CreatedAt: now.Add(time.Duration(index) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	view, err := repo.GetContextOverview(ctx, session.AccountID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Turns != 1 || view.Steps != 1 || view.Injections != 1 || view.Prunes != 1 || view.Tokens.Input != 100 || view.Tokens.Output != 20 || view.Tokens.CacheRead != 60 || view.Tokens.Uncached != 40 {
		t.Fatalf("overview = %#v", view)
	}
	if view.Current == nil || len(view.Current.Parts) != 2 || view.Current.ProjectedTokens == nil || *view.Current.ProjectedTokens != 90 {
		t.Fatalf("current = %#v", view.Current)
	}
	request, err := repo.GetContextRequest(ctx, session.AccountID, session.ID, "context-run:model:1")
	if err != nil || request.Parts[1].Content != "hello" {
		t.Fatalf("request = %#v, %v", request, err)
	}
	if request.TurnNumber != 1 || request.StepNumber != 1 || request.Preview != "hello" {
		t.Fatalf("request position = %#v", request)
	}
	if _, err := repo.GetContextRequest(ctx, "account-b", session.ID, "context-run:model:1"); err != domain.ErrNotFound {
		t.Fatalf("cross-account request = %v", err)
	}
	if err := session.Configure("model-new", domain.PermissionAutoApprove, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	contextEvents, err := repo.ListContextEvents(ctx, session.AccountID, session.ID, "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, event := range contextEvents {
		kinds[event.Kind]++
	}
	if kinds["MODEL_SWITCHED"] != 1 || kinds["MODE_SWITCHED"] != 1 || kinds["CONTEXT_INJECTED"] != 1 || kinds["CONTEXT_PRUNED"] != 1 {
		t.Fatalf("events = %#v", kinds)
	}
}

func TestSessionContextBackfillAndCurrentTail(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("legacy-context-session", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun("legacy-context-run", session.ID, session.AccountID, "message-legacy", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	legacy := []struct{ id, kind, payload string }{
		{"legacy-start", "MODEL_REQUEST_STARTED", `{"attempt_id":"legacy-attempt","purpose":"agent","model":"legacy-model","at":"2026-09-28T11:00:00Z","request_body":{"messages":[{"role":"user","content":"old request"}]}}`},
		{"legacy-finish", "MODEL_REQUEST_FINISHED", `{"attempt_id":"legacy-attempt","at":"2026-09-28T11:00:02Z","input_tokens":20,"output_tokens":4}`},
	}
	for index, item := range legacy {
		if err := repo.AppendEvent(ctx, &domain.Event{ID: item.id, RunID: run.ID, SessionID: session.ID, AccountID: session.AccountID, Sequence: uint64(index + 1), Type: item.kind, Payload: json.RawMessage(item.payload), CreatedAt: now.Add(time.Duration(index) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.BackfillSessionContext(ctx, session.AccountID, session.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.BackfillSessionContext(ctx, session.AccountID, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AppendRunEvent(ctx, &domain.Event{ID: "legacy-assistant", RunID: run.ID, SessionID: session.ID, AccountID: session.AccountID, Type: "TEXT_MESSAGE_END", Payload: json.RawMessage(`{"content":"new assistant text"}`), CreatedAt: now.Add(3 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendMessage(ctx, &domain.Message{ID: "pending-user", SessionID: session.ID, AccountID: session.AccountID, Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"pending user message"}]`), CreatedAt: now.Add(4 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetCurrentContextRequest(ctx, session.AccountID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Parts) != 3 || current.Parts[0].Content != "old request" || current.Parts[1].Content != "new assistant text" || current.Parts[2].Content != "pending user message" || current.ProjectedTokens == nil || *current.ProjectedTokens <= 20 {
		t.Fatalf("current = %#v", current)
	}
}
