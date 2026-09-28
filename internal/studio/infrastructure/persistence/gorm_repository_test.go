package persistence_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func openRepository(t *testing.T) *persistence.GormRepository {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return persistence.NewGormRepository(gdb)
}

func TestRunLocalePersistsAcrossRepositoryRead(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("session-locale", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun("run-locale", session.ID, session.AccountID, "message-locale", now)
	if err != nil {
		t.Fatal(err)
	}
	run.Locale = "en"
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetRun(ctx, session.AccountID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Locale != "en" {
		t.Fatalf("run locale = %q", stored.Locale)
	}
}

func TestMCPConnectorUpdatePersistsDisabledState(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	connector, err := domain.NewMCPConnector("connector-1", "account-a", "知识库", "https://example.com/mcp", "encrypted", domain.ConnectorPolicyAuto, now)
	if err != nil {
		t.Fatal(err)
	}
	connector.Enabled = true
	if err := repo.CreateMCPConnector(ctx, connector); err != nil {
		t.Fatal(err)
	}
	connector.Enabled = false
	connector.UpdatedAt = now.Add(time.Second)
	if err := repo.UpdateMCPConnector(ctx, connector); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetMCPConnector(ctx, "account-a", connector.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Enabled {
		t.Fatal("连接器停用状态未保存")
	}
}

func TestSkillVersionedUpdateKeepsLatestContent(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	skill, err := domain.NewSkill("skill-1", "account-a", "sample-skill", "描述", "原内容", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}
	first, err := repo.GetSkill(ctx, "account-a", skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.GetSkill(ctx, "account-a", skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	first.Prompt = "首次修改"
	first.Version = "1.0.1"
	first.UpdatedAt = now.Add(time.Nanosecond)
	if err := repo.UpdateSkillIfUnchanged(ctx, first, now, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	second.Prompt = "过期修改"
	second.Version = "1.0.2"
	second.UpdatedAt = now.Add(2 * time.Nanosecond)
	if err := repo.UpdateSkillIfUnchanged(ctx, second, now, "1.0.0"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("过期内容保存结果 = %v", err)
	}
	if err := repo.SetSkillEnabled(ctx, "account-a", skill.ID, true); err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetSkill(ctx, "account-a", skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Prompt != "首次修改" || !current.Enabled || !current.UpdatedAt.Equal(first.UpdatedAt) {
		t.Fatalf("当前 Skill = %#v", current)
	}
}

func TestWorkflowClarificationKeepsFormAcrossRepositoryReads(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	clarification, err := domain.NewWorkflowClarification("workflow-question", "run-1", "session-1", "account-1", domain.WorkflowRequest{
		ID: "12", Name: "角色三视图", InputSchema: []byte(`{"type":"object"}`), SuggestedInputs: map[string]any{"prompt": "雨夜"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateClarification(ctx, clarification); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetClarification(ctx, "account-1", clarification.ID)
	if err != nil || stored.Workflow == nil || stored.Workflow.SuggestedInputs["prompt"] != "雨夜" {
		t.Fatalf("stored workflow request = %#v, err=%v", stored, err)
	}
	if err := stored.ResolveWorkflow("account-1", map[string]any{"prompt": "夜景"}, false, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateClarification(ctx, stored); err != nil {
		t.Fatal(err)
	}
	answered, err := repo.GetClarification(ctx, "account-1", clarification.ID)
	if err != nil || answered.Workflow == nil || answered.Workflow.SubmittedInputs["prompt"] != "夜景" {
		t.Fatalf("stored workflow answer = %#v, err=%v", answered, err)
	}
}

type legacyApprovalRow struct {
	ID         string `gorm:"primaryKey;size:64"`
	RunID      string `gorm:"size:64;not null;index;uniqueIndex:idx_studio_approvals_run_tool"`
	SessionID  string `gorm:"size:64;not null;index"`
	AccountID  string `gorm:"size:64;not null;index"`
	ToolCallID string `gorm:"size:128;not null;uniqueIndex:idx_studio_approvals_run_tool"`
	Action     string `gorm:"size:128;not null"`
	Status     string `gorm:"size:32;not null;index"`
	ResolvedBy string `gorm:"size:64"`
	CreatedAt  time.Time
	ResolvedAt time.Time
	UpdatedAt  time.Time
}

func (legacyApprovalRow) TableName() string { return "studio_approvals" }

func TestMigrateAddsApprovalDescriptionToExistingTable(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := gdb.AutoMigrate(&legacyApprovalRow{}); err != nil {
		t.Fatalf("create approval table without description: %v", err)
	}
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	if err := gdb.Create(&legacyApprovalRow{
		ID: "approval-legacy", RunID: "run-legacy", SessionID: "session-legacy", AccountID: "account-legacy",
		ToolCallID: "tool-call-legacy", Action: "workflow.execute", Status: string(domain.ApprovalPending),
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("insert legacy approval: %v", err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatalf("migrate existing approval table: %v", err)
	}
	stored, err := persistence.NewGormRepository(gdb).GetApproval(context.Background(), "account-legacy", "approval-legacy")
	if err != nil {
		t.Fatalf("GetApproval() error = %v", err)
	}
	if stored.Description != "" {
		t.Fatalf("legacy approval description = %q, want empty", stored.Description)
	}
}

func TestApprovalCheckpointSurvivesStoreRecreation(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	first := repo.Checkpoints()
	if _, exists, err := first.Get(ctx, "run-1"); err != nil || exists {
		t.Fatalf("missing checkpoint = (%v, %v)", exists, err)
	}
	if err := first.Set(ctx, "run-1", []byte("checkpoint-v1")); err != nil {
		t.Fatal(err)
	}
	second := repo.Checkpoints()
	data, exists, err := second.Get(ctx, "run-1")
	if err != nil || !exists || string(data) != "checkpoint-v1" {
		t.Fatalf("restored checkpoint = (%q, %v, %v)", data, exists, err)
	}
	if err := second.Set(ctx, "run-1", []byte("checkpoint-v2")); err != nil {
		t.Fatal(err)
	}
	data, exists, err = first.Get(ctx, "run-1")
	if err != nil || !exists || string(data) != "checkpoint-v2" {
		t.Fatalf("updated checkpoint = (%q, %v, %v)", data, exists, err)
	}
}

func TestSessionAndMessagesAreAccountScoped(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("session-1", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := session.UpdateContextSummary("历史摘要", "message-7", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateSession(ctx, session); err != nil {
		t.Fatalf("UpdateSession() with context summary error = %v", err)
	}
	if err := repo.CreateSession(ctx, session); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateSession() error = %v, want ErrAlreadyExists", err)
	}

	got, err := repo.GetSession(ctx, "account-a", session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Title != domain.DefaultSessionTitle {
		t.Fatalf("Title = %q", got.Title)
	}
	if got.ContextSummary != "历史摘要" || got.ContextSummaryThroughMessageID != "message-7" {
		t.Fatalf("context summary = %#v", got)
	}
	if _, err := repo.GetSession(ctx, "account-b", session.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account GetSession() error = %v, want ErrNotFound", err)
	}

	message := &domain.Message{
		ID: "message-1", SessionID: session.ID, AccountID: "account-a",
		Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"画一篇漫画"}]`), CreatedAt: now,
	}
	if err := repo.AppendMessage(ctx, message); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}
	messages, err := repo.ListMessages(ctx, "account-a", session.ID, 100)
	if err != nil || len(messages) != 1 || messages[0].ID != message.ID {
		t.Fatalf("ListMessages() = (%#v, %v)", messages, err)
	}
	messages, err = repo.ListMessages(ctx, "account-b", session.ID, 100)
	if err != nil || len(messages) != 0 {
		t.Fatalf("cross-account ListMessages() = (%#v, %v)", messages, err)
	}
}

func TestListLatestSessionRunsReturnsNewestOwnedRunPerSession(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	base := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	for _, value := range []struct {
		id string
	}{
		{id: "session-latest-a"},
		{id: "session-latest-b"},
		{id: "session-latest-empty"},
	} {
		session, err := domain.NewSession(value.id, "account-a", base)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
	}
	older, err := domain.NewRun("run-latest-older", "session-latest-a", "account-a", "message-a-1", base.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := older.Start(base.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := older.Succeed(base.Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	newer, err := domain.NewRun("run-latest-newer", "session-latest-a", "account-a", "message-a-2", base.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := newer.Start(base.Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	other, err := domain.NewRun("run-latest-other", "session-latest-b", "account-a", "message-b-1", base.Add(6*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Start(base.Add(7 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for _, run := range []*domain.Run{older, newer, other} {
		if err := repo.CreateRun(ctx, run); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.ListLatestSessionRuns(ctx, "account-a", []string{
		"session-latest-a", "session-latest-b", "session-latest-empty", "missing",
	})
	if err != nil {
		t.Fatalf("ListLatestSessionRuns() error = %v", err)
	}
	if got["session-latest-a"] == nil || got["session-latest-a"].ID != newer.ID {
		t.Fatalf("latest session-a run = %#v, want %s", got["session-latest-a"], newer.ID)
	}
	if got["session-latest-b"] == nil || got["session-latest-b"].ID != other.ID {
		t.Fatalf("latest session-b run = %#v, want %s", got["session-latest-b"], other.ID)
	}
	if _, ok := got["session-latest-empty"]; ok {
		t.Fatalf("empty session returned a run: %#v", got["session-latest-empty"])
	}
	if _, ok := got["missing"]; ok {
		t.Fatalf("unknown session returned a run: %#v", got["missing"])
	}
}

func TestRunSkillSnapshotRoundTrip(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-skill-snapshot", "session-skill-snapshot", "account-a", "message-skill-snapshot", now)
	if err != nil {
		t.Fatal(err)
	}
	run.SkillIDs = []string{"skill-a"}
	run.SkillSnapshot = []domain.RunSkill{
		{ID: "skill-a", Name: "分镜", Description: "编排镜头", Prompt: "输出镜头表"},
		{ID: "skill-b", Name: "文案", Description: "撰写文案", Prompt: "输出文案"},
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetRun(ctx, "account-a", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stored.SkillSnapshot, run.SkillSnapshot) {
		t.Fatalf("Skill snapshot = %#v", stored.SkillSnapshot)
	}
	if err := stored.Start(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateRun(ctx, stored); err != nil {
		t.Fatal(err)
	}
	resumed, err := repo.GetRun(ctx, "account-a", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resumed.SkillSnapshot, run.SkillSnapshot) {
		t.Fatalf("Skill snapshot after Run update = %#v", resumed.SkillSnapshot)
	}
}

func TestRunProgressIsAccountScopedAndUpsertable(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 14, 0, 0, 0, time.UTC)
	progress := &domain.RunProgress{
		RunID: "run-progress-1", SessionID: "session-progress-1", AccountID: "account-a",
		AssistantMessageID: "assistant-progress-1", AssistantText: "已经输出一半",
		ReasoningText: "先分析需求", ToolCallsJSON: json.RawMessage(`[{"id":"tool-1","name":"search","args":"{}"}]`),
		LastSequence: 7, UpdatedAt: now,
	}
	if err := repo.UpsertRunProgress(ctx, progress); err != nil {
		t.Fatalf("UpsertRunProgress() error = %v", err)
	}
	got, err := repo.GetRunProgress(ctx, "account-a", progress.RunID)
	if err != nil {
		t.Fatalf("GetRunProgress() error = %v", err)
	}
	if got == nil || got.AssistantText != progress.AssistantText || got.LastSequence != progress.LastSequence || string(got.ToolCallsJSON) != string(progress.ToolCallsJSON) {
		t.Fatalf("progress = %#v, want %#v", got, progress)
	}
	progress.AssistantText = "已经输出更多"
	progress.LastSequence = 9
	if err := repo.UpsertRunProgress(ctx, progress); err != nil {
		t.Fatalf("UpsertRunProgress() update error = %v", err)
	}
	got, err = repo.GetRunProgress(ctx, "account-a", progress.RunID)
	if err != nil || got.AssistantText != "已经输出更多" || got.LastSequence != 9 {
		t.Fatalf("updated progress = (%#v, %v)", got, err)
	}
	if _, err := repo.GetRunProgress(ctx, "account-b", progress.RunID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account GetRunProgress() error = %v, want ErrNotFound", err)
	}
	if err := repo.DeleteRunProgress(ctx, "account-a", progress.RunID); err != nil {
		t.Fatalf("DeleteRunProgress() error = %v", err)
	}
	if _, err := repo.GetRunProgress(ctx, "account-a", progress.RunID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted progress error = %v, want ErrNotFound", err)
	}
}

func TestRunEventsAreOrdered(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-1", "session-1", "account-a", "message-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	for _, event := range []*domain.Event{
		{ID: "event-2", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 2, Type: "TEXT_MESSAGE_CONTENT", Payload: json.RawMessage(`{"delta":"好"}`), CreatedAt: now.Add(2 * time.Second)},
		{ID: "event-1", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 1, Type: "RUN_STARTED", Payload: json.RawMessage(`{}`), CreatedAt: now.Add(time.Second)},
	} {
		if err := repo.AppendEvent(ctx, event); err != nil {
			t.Fatalf("AppendEvent(%s) error = %v", event.ID, err)
		}
	}
	events, err := repo.ListEventsAfter(ctx, "account-a", run.ID, 0, 100)
	if err != nil {
		t.Fatalf("ListEventsAfter() error = %v", err)
	}
	if len(events) != 2 || events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("events = %#v, want sequences [1,2]", events)
	}
	events, err = repo.ListEventsAfter(ctx, "account-a", run.ID, 1, 100)
	if err != nil || len(events) != 1 || events[0].Sequence != 2 {
		t.Fatalf("events after 1 = (%#v, %v)", events, err)
	}
}

func TestSessionTranscriptReadsEveryOwnedMessageRunAndEvent(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("session-transcript", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	message := &domain.Message{ID: "message-transcript", SessionID: session.ID, AccountID: session.AccountID, Role: domain.MessageRoleUser, ContentJSON: json.RawMessage(`[{"type":"text","text":"hello"}]`), CreatedAt: now}
	if err := repo.AppendMessage(ctx, message); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun("run-transcript", session.ID, session.AccountID, message.ID, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	for sequence := uint64(1); sequence <= 205; sequence++ {
		if err := repo.AppendEvent(ctx, &domain.Event{ID: fmt.Sprintf("event-transcript-%03d", sequence), RunID: run.ID, SessionID: session.ID, AccountID: session.AccountID, Sequence: sequence, Type: "CUSTOM", Payload: json.RawMessage(`{"sequence":1}`), CreatedAt: now.Add(time.Duration(sequence) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.AppendEvent(ctx, &domain.Event{ID: "event-model-trace", RunID: run.ID, SessionID: session.ID, AccountID: session.AccountID, Sequence: 206, Type: "MODEL_REQUEST_FINISHED", Payload: json.RawMessage(`{"response_body":{"choices":[{"message":{"content":"large raw response"}}]}}`), CreatedAt: now.Add(206 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	data, err := repo.ListSessionTranscript(ctx, session.AccountID, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Messages) != 1 || len(data.Runs) != 1 || len(data.Events) != 205 || data.Events[0].Sequence != 1 || data.Events[204].Sequence != 205 {
		t.Fatalf("transcript = messages=%d runs=%d events=%d", len(data.Messages), len(data.Runs), len(data.Events))
	}
	traceEvents, err := repo.ListRunTraceEvents(ctx, session.AccountID, run.ID)
	if err != nil || len(traceEvents) != 206 {
		t.Fatalf("full trace events = %d, %v", len(traceEvents), err)
	}
	foreign, err := repo.ListSessionTranscript(ctx, "account-b", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(foreign.Messages) != 0 || len(foreign.Runs) != 0 || len(foreign.Events) != 0 {
		t.Fatalf("cross-account transcript = %#v", foreign)
	}
}

func TestAppendEventRejectsDuplicateSequence(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	first := &domain.Event{
		ID: "event-first", RunID: "run-sequence", SessionID: "session-sequence",
		AccountID: "account-a", Sequence: 201, Type: "TOOL_CALL_START",
		Payload: json.RawMessage(`{}`), CreatedAt: now,
	}
	if err := repo.AppendEvent(ctx, first); err != nil {
		t.Fatal(err)
	}
	duplicate := *first
	duplicate.ID = "event-duplicate"
	duplicate.Type = "APPROVAL_RESOLVED"
	if err := repo.AppendEvent(ctx, &duplicate); err == nil {
		t.Fatal("duplicate sequence was silently accepted")
	}
	events, err := repo.ListEventsAfter(ctx, "account-a", first.RunID, 0, 10)
	if err != nil || len(events) != 1 || events[0].Type != first.Type {
		t.Fatalf("events after duplicate = (%#v, %v)", events, err)
	}
}

func TestAppendRunEventAllocatesSequenceBeyondTwoHundred(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-long-sequence", "session-long-sequence", "account-a", "message-long-sequence", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	for sequence := uint64(1); sequence <= 385; sequence++ {
		event := &domain.Event{
			ID: fmt.Sprintf("legacy-event-%d", sequence), RunID: run.ID,
			SessionID: run.SessionID, AccountID: run.AccountID, Sequence: sequence,
			Type: "CUSTOM", Payload: json.RawMessage(`{}`), CreatedAt: now,
		}
		if err := repo.AppendEvent(ctx, event); err != nil {
			t.Fatalf("append legacy event %d: %v", sequence, err)
		}
	}
	created, err := repo.AppendRunEvent(ctx, &domain.Event{
		ID: "next-event", RunID: run.ID, SessionID: run.SessionID,
		AccountID: run.AccountID, Type: "APPROVAL_RESOLVED",
		Payload: json.RawMessage(`{}`), CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Sequence != 386 {
		t.Fatalf("allocated sequence = %d, want 386", created.Sequence)
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 385, 10)
	if err != nil || len(events) != 1 || events[0].ID != "next-event" {
		t.Fatalf("events after 385 = (%#v, %v)", events, err)
	}
}

func TestAppendRunEventConcurrentWritersKeepUniqueSequence(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-concurrent-events", "session-concurrent-events", "account-a", "message-concurrent-events", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	const writers = 12
	var group sync.WaitGroup
	errors := make(chan error, writers)
	for index := range writers {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := repo.AppendRunEvent(ctx, &domain.Event{
				ID: fmt.Sprintf("concurrent-event-%d", index), RunID: run.ID,
				SessionID: run.SessionID, AccountID: run.AccountID,
				Type: "CUSTOM", Payload: json.RawMessage(`{}`), CreatedAt: now,
			})
			errors <- err
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 0, writers)
	if err != nil || len(events) != writers {
		t.Fatalf("events = (%d, %v), want %d", len(events), err, writers)
	}
	for index, event := range events {
		if event.Sequence != uint64(index+1) {
			t.Fatalf("event %d sequence = %d, want %d", index, event.Sequence, index+1)
		}
	}
}

func TestAssetVersionAndLibraryReferenceRoundTrip(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	asset, err := domain.NewAsset("asset-1", "session-1", "account-a", "故事大纲.md", domain.AssetDocument, domain.AssetOriginAgent, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asset.AppendVersion("version-1", "text/markdown", "studio/account-a/asset-1/v1.md", 128, now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAsset(ctx, asset); err != nil {
		t.Fatalf("CreateAsset() error = %v", err)
	}
	category, err := domain.NewLibraryCategory("category-story", "account-a", "", "故事", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateLibraryCategory(ctx, category); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveAssetToLibrary(ctx, "account-a", asset.ID, "missing-category", now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("save to missing category error = %v", err)
	}
	if err := repo.SaveAssetToLibrary(ctx, "account-a", asset.ID, "category-story", now.Add(time.Second)); err != nil {
		t.Fatalf("SaveAssetToLibrary() error = %v", err)
	}
	second, err := asset.AppendVersion("version-2", "text/markdown", "studio/account-a/asset-1/v2.md", 256, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendAssetVersion(ctx, asset.ID, "account-a", second); err != nil {
		t.Fatalf("AppendAssetVersion() error = %v", err)
	}

	got, err := repo.GetAsset(ctx, "account-a", asset.ID)
	if err != nil {
		t.Fatalf("GetAsset() error = %v", err)
	}
	if got.CurrentVersion != 2 || len(got.Versions) != 2 || got.Versions[1].BlobKey != "studio/account-a/asset-1/v2.md" {
		t.Fatalf("asset = %#v", got)
	}
	page, err := repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{CategoryID: "category-story", Limit: 100})
	if err != nil || len(page.Assets) != 1 || page.Assets[0].ID != asset.ID {
		t.Fatalf("ListLibraryAssets() = (%#v, %v)", page, err)
	}
	if page.Assets[0].CurrentVersion != 1 || len(page.Assets[0].Versions) != 1 || page.Assets[0].Versions[0].ID != "version-1" {
		t.Fatalf("library asset must retain saved v1, got %#v", page.Assets[0])
	}
	if err := repo.MoveLibraryAsset(ctx, "account-a", asset.ID, "", now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	page, err = repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{CategoryID: "category-story", Limit: 100})
	if err != nil || len(page.Assets) != 0 {
		t.Fatalf("old category items = (%#v, %v)", page, err)
	}
	page, err = repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{Limit: 100})
	if err != nil || len(page.Assets) != 1 || page.Assets[0].CurrentVersion != 1 || page.Assets[0].Versions[0].ID != "version-1" {
		t.Fatalf("moved asset changed pinned version: (%#v, %v)", page, err)
	}
	if _, err := repo.GetAsset(ctx, "account-b", asset.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account GetAsset() error = %v, want ErrNotFound", err)
	}
}

func TestListLibraryAssetsPaginatesAndSearches(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	category, err := domain.NewLibraryCategory("category-1", "account-a", "", "故事", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateLibraryCategory(ctx, category); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id, accountID, name, categoryID string
	}{
		{"asset-a", "account-a", "故事一", "category-1"},
		{"asset-b", "account-a", "故事二", "category-1"},
		{"asset-c", "account-a", "报告", ""},
		{"asset-d", "account-b", "故事三", ""},
	} {
		asset, createErr := domain.NewAsset(item.id, "session-1", item.accountID, item.name, domain.AssetDocument, domain.AssetOriginUser, now)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, createErr = asset.AppendVersion(item.id+"-version", "text/markdown", "studio/"+item.accountID+"/"+item.id, 12, now); createErr != nil {
			t.Fatal(createErr)
		}
		if createErr = repo.CreateAsset(ctx, asset); createErr != nil {
			t.Fatal(createErr)
		}
		if createErr = repo.SaveAssetToLibrary(ctx, item.accountID, item.id, item.categoryID, now); createErr != nil {
			t.Fatal(createErr)
		}
	}
	first, err := repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{Limit: 2})
	if err != nil || first.Total != 3 || len(first.Assets) != 2 || first.Assets[0].ID != "asset-c" || first.Assets[1].ID != "asset-b" {
		t.Fatalf("first page = (%#v, %v)", first, err)
	}
	second, err := repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{Limit: 2, Offset: 2})
	if err != nil || second.Total != 3 || len(second.Assets) != 1 || second.Assets[0].ID != "asset-a" {
		t.Fatalf("second page = (%#v, %v)", second, err)
	}
	searched, err := repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{Search: "故事", CategoryID: "category-1", Limit: 2})
	if err != nil || searched.Total != 2 || len(searched.Assets) != 2 || searched.Assets[0].ID != "asset-b" || searched.Assets[1].ID != "asset-a" {
		t.Fatalf("searched page = (%#v, %v)", searched, err)
	}
	literal, err := repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{Search: "%", Limit: 2})
	if err != nil || literal.Total != 0 || len(literal.Assets) != 0 {
		t.Fatalf("literal search = (%#v, %v)", literal, err)
	}
}

func TestClearSessionsKeepsLibraryAssetsAndOtherAccounts(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	for _, accountID := range []string{"account-a", "account-b"} {
		session, err := domain.NewSession("session-"+accountID, accountID, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
	}
	message := &domain.Message{ID: "message-a", SessionID: "session-account-a", AccountID: "account-a", Role: domain.MessageRoleUser, ContentJSON: []byte(`[{"type":"text","text":"秘密"}]`), CreatedAt: now}
	if err := repo.AppendMessage(ctx, message); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun("run-a", "session-account-a", "account-a", message.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := run.Succeed(now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendEvent(ctx, &domain.Event{ID: "event-a", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 1, Type: "RUN_STARTED", Payload: json.RawMessage(`{}`), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	for _, assetID := range []string{"saved-a", "private-a"} {
		asset, err := domain.NewAsset(assetID, "session-account-a", "account-a", assetID+".md", domain.AssetDocument, domain.AssetOriginAgent, now)
		if err != nil {
			t.Fatal(err)
		}
		asset.SourceRunID = run.ID
		if _, err := asset.AppendVersion(assetID+"-v1", "text/markdown", "studio/account-a/"+assetID+".md", 8, now); err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateAsset(ctx, asset); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.SaveAssetToLibrary(ctx, "account-a", "saved-a", "", now); err != nil {
		t.Fatal(err)
	}
	otherAsset, err := domain.NewAsset("asset-b", "session-account-b", "account-b", "asset-b.md", domain.AssetDocument, domain.AssetOriginUser, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherAsset.AppendVersion("asset-b-v1", "text/markdown", "studio/account-b/asset-b.md", 8, now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAsset(ctx, otherAsset); err != nil {
		t.Fatal(err)
	}
	if err := repo.ClearSessions(ctx, "account-a"); err != nil {
		t.Fatal(err)
	}
	if sessions, err := repo.ListSessions(ctx, "account-a", domain.SessionListQuery{Limit: 10}); err != nil || len(sessions) != 0 {
		t.Fatalf("cleared sessions = (%v, %v)", sessions, err)
	}
	if _, err := repo.GetSession(ctx, "account-b", "session-account-b"); err != nil {
		t.Fatalf("other account session: %v", err)
	}
	if _, err := repo.GetAsset(ctx, "account-b", otherAsset.ID); err != nil {
		t.Fatalf("other account asset: %v", err)
	}
	if _, err := repo.GetRun(ctx, "account-a", run.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cleared run: %v", err)
	}
	if messages, err := repo.ListMessages(ctx, "account-a", "session-account-a", 10); err != nil || len(messages) != 0 {
		t.Fatalf("cleared messages = (%v, %v)", messages, err)
	}
	if events, err := repo.ListEventsAfter(ctx, "account-a", run.ID, 0, 10); err != nil || len(events) != 0 {
		t.Fatalf("cleared events = (%v, %v)", events, err)
	}
	if _, err := repo.GetAsset(ctx, "account-a", "private-a"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("private asset: %v", err)
	}
	page, err := repo.ListLibraryAssets(ctx, "account-a", domain.LibraryAssetListQuery{Limit: 10})
	if err != nil || len(page.Assets) != 1 || page.Assets[0].ID != "saved-a" || page.Assets[0].SessionID != "" || page.Assets[0].SourceRunID != "" || len(page.Assets[0].Versions) != 1 {
		t.Fatalf("library assets = (%#v, %v)", page, err)
	}
}

func TestClearSessionsRemovesActiveRunAndPendingWorkflow(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	session, err := domain.NewSession("session-a", "account-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun("run-a", session.ID, "account-a", "message-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	execution, err := domain.NewWorkflowExecution("workflow-a", "account-a", session.ID, run.ID, "tool-a", "task-a", "workflow-1", "node-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateWorkflowExecution(ctx, execution); err != nil {
		t.Fatal(err)
	}
	if err := repo.ClearSessions(ctx, "account-a"); err != nil {
		t.Fatalf("ClearSessions() with active work: %v", err)
	}
	if _, err := repo.GetSession(ctx, "account-a", session.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("session after clear: %v", err)
	}
	if _, err := repo.GetRun(ctx, "account-a", run.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("run after clear: %v", err)
	}
	if _, err := repo.GetWorkflowExecutionByTask(ctx, "account-a", execution.TaskID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("workflow execution after clear: %v", err)
	}
}

func TestFlowRoundTripKeepsUserOrder(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	second, _ := domain.NewFlowNode("node-2", "session-1", "account-a", domain.FlowNodeAsset, "分镜图", 20, now)
	first, _ := domain.NewFlowNode("node-1", "session-1", "account-a", domain.FlowNodeStage, "立住角色", 10, now)
	for _, node := range []*domain.FlowNode{second, first} {
		if err := repo.SaveFlowNode(ctx, node); err != nil {
			t.Fatalf("SaveFlowNode(%s) error = %v", node.ID, err)
		}
	}
	edge, _ := domain.NewFlowEdge("edge-1", "session-1", "account-a", first.ID, second.ID, now)
	if err := repo.SaveFlowEdge(ctx, edge); err != nil {
		t.Fatalf("SaveFlowEdge() error = %v", err)
	}

	nodes, edges, err := repo.GetFlow(ctx, "account-a", "session-1")
	if err != nil {
		t.Fatalf("GetFlow() error = %v", err)
	}
	if len(nodes) != 2 || nodes[0].ID != first.ID || nodes[1].ID != second.ID {
		t.Fatalf("nodes = %#v", nodes)
	}
	if len(edges) != 1 || edges[0].SourceNodeID != first.ID || edges[0].TargetNodeID != second.ID {
		t.Fatalf("edges = %#v", edges)
	}
	nodes, edges, err = repo.GetFlow(ctx, "account-b", "session-1")
	if err != nil || len(nodes) != 0 || len(edges) != 0 {
		t.Fatalf("cross-account GetFlow() = (%#v, %#v, %v)", nodes, edges, err)
	}
}

func TestFlowNodeAssetVersionChangesRemainPinnedAfterUpsert(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	node, err := domain.NewFlowNode("asset-node", "session-1", "account-a", domain.FlowNodeAsset, "故事大纲", 10, now)
	if err != nil {
		t.Fatal(err)
	}
	node.AssetID = "story-asset"
	node.AssetVersionID = "version-1"
	node.AssetVersion = 1
	if err := repo.SaveFlowNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	node.AssetVersionID = "version-2"
	node.AssetVersion = 2
	node.UpdatedAt = now.Add(time.Minute)
	if err := repo.SaveFlowNode(ctx, node); err != nil {
		t.Fatal(err)
	}
	nodes, _, err := repo.GetFlow(ctx, "account-a", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].AssetID != "story-asset" || nodes[0].AssetVersionID != "version-2" || nodes[0].AssetVersion != 2 {
		t.Fatalf("asset version after upsert = %#v", nodes)
	}
}

func TestWorkflowExecutionIsIdempotentAndAccountScoped(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	execution := &domain.WorkflowExecution{
		ID: "workflow-execution-1", AccountID: "account-a", SessionID: "session-1", RunID: "run-1",
		ToolCallID: "tool-call-1", TaskID: "task-1", WorkflowID: "12", OperationNodeID: "operation-1",
		Status: domain.WorkflowExecutionSubmitted, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateWorkflowExecution(ctx, execution); err != nil {
		t.Fatalf("CreateWorkflowExecution() error = %v", err)
	}
	duplicate := *execution
	duplicate.ID = "workflow-execution-2"
	if err := repo.CreateWorkflowExecution(ctx, &duplicate); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("duplicate CreateWorkflowExecution() error = %v, want ErrAlreadyExists", err)
	}
	got, err := repo.GetWorkflowExecutionByTask(ctx, "account-a", "task-1")
	if err != nil || got.ID != execution.ID || got.Status != domain.WorkflowExecutionSubmitted {
		t.Fatalf("GetWorkflowExecutionByTask() = (%#v, %v)", got, err)
	}
	got, err = repo.GetWorkflowExecutionByRunTool(ctx, "account-a", "run-1", "tool-call-1")
	if err != nil || got.TaskID != "task-1" {
		t.Fatalf("GetWorkflowExecutionByRunTool() = (%#v, %v)", got, err)
	}
	if _, err := repo.GetWorkflowExecutionByTask(ctx, "account-b", "task-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account GetWorkflowExecutionByTask() error = %v, want ErrNotFound", err)
	}
}

func TestListSessionWorkflowExecutionsIncludesTerminalTasksAndIsAccountScoped(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	for _, input := range []struct {
		id, accountID, sessionID string
		createdAt                time.Time
		status                   domain.WorkflowExecutionStatus
	}{
		{"execution-pending", "account-a", "session-a", now, domain.WorkflowExecutionSubmitted},
		{"execution-failed", "account-a", "session-a", now.Add(time.Second), domain.WorkflowExecutionFailed},
		{"other-session", "account-a", "session-b", now.Add(2 * time.Second), domain.WorkflowExecutionSubmitted},
		{"other-account", "account-b", "session-a", now.Add(3 * time.Second), domain.WorkflowExecutionSubmitted},
	} {
		execution, err := domain.NewWorkflowExecution(input.id, input.accountID, input.sessionID, "run-"+input.id, "tool-"+input.id, "task-"+input.id, "12", "operation-"+input.id, input.createdAt)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateWorkflowExecution(ctx, execution); err != nil {
			t.Fatal(err)
		}
		if input.status.Terminal() {
			if err := execution.Complete(input.status, "出图失败", input.createdAt.Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			if err := repo.UpdateWorkflowExecution(ctx, execution); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, err := repo.ListSessionWorkflowExecutions(ctx, "account-a", "session-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "execution-pending" || got[1].ID != "execution-failed" || got[1].Status != domain.WorkflowExecutionFailed || got[1].ErrorMessage != "出图失败" {
		t.Fatalf("ListSessionWorkflowExecutions() = %#v", got)
	}
}
