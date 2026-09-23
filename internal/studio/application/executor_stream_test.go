package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func TestExecutionWriterAppendsImmutableTextVersion(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:executor_update_text_asset?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := gdb.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("update-run", "update-session", "account-a", "message-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	asset, err := domain.NewAsset("story-asset", run.SessionID, run.AccountID, "story.md", domain.AssetDocument, domain.AssetOriginUser, now)
	if err != nil {
		t.Fatal(err)
	}
	firstRef, err := blobs.Put(ctx, "studio/account-a/update-session/story-asset/v1.md", strings.NewReader("# Original"), blob.PutOptions{MIME: "text/markdown"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := asset.AppendVersion("story-v1", "text/markdown", firstRef.Key, firstRef.Size, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAsset(ctx, asset); err != nil {
		t.Fatal(err)
	}
	i := 0
	writer := &executionWriter{executor: &AgentExecutor{repo: repo, blob: blobs, ids: func() string { i++; return fmt.Sprintf("update-event-%d", i) }, now: func() time.Time { return now }}, run: run}
	updated, next, err := writer.AppendTextAssetVersion(ctx, asset.ID, first.ID, "update-story", []byte("# Revised"))
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentVersion != 2 || next.Version != 2 || next.ID == first.ID || next.BlobKey == first.BlobKey {
		t.Fatalf("updated=%#v next=%#v", updated, next)
	}
	repeated, same, err := writer.AppendTextAssetVersion(ctx, asset.ID, first.ID, "update-story", []byte("# Revised"))
	if err != nil || repeated.CurrentVersion != 2 || same.ID != next.ID {
		t.Fatalf("repeat=(%#v, %#v, %v)", repeated, same, err)
	}
	stored, err := repo.GetAsset(ctx, run.AccountID, asset.ID)
	if err != nil || len(stored.Versions) != 2 {
		t.Fatalf("stored=(%#v, %v)", stored, err)
	}
	for _, check := range []struct{ key, want string }{{first.BlobKey, "# Original"}, {next.BlobKey, "# Revised"}} {
		reader, getErr := blobs.Get(ctx, sharedkernel.BlobRef{Key: check.key})
		if getErr != nil {
			t.Fatal(getErr)
		}
		content, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil || string(content) != check.want {
			t.Fatalf("content=(%q, %v), want %q", content, readErr, check.want)
		}
	}
	if _, _, err := writer.AppendTextAssetVersion(ctx, asset.ID, first.ID, "different-action", []byte("# Other")); err == nil {
		t.Fatal("stale version update succeeded")
	}
}

func TestExecutionWriterReusesAssetForStableAction(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:executor_asset_action?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	run, err := domain.NewRun("asset-action-run", "asset-action-session", "account-a", "message-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	writer := &executionWriter{executor: &AgentExecutor{repo: repo, blob: blobs, ids: func() string { return fmt.Sprintf("random-%d", time.Now().UnixNano()) }, now: time.Now}, run: run}
	input := GeneratedAsset{ActionID: "create-story", Name: "story.md", Kind: domain.AssetDocument, Origin: domain.AssetOriginAgent, MIMEType: "text/markdown", Content: []byte("# Story")}
	first, err := writer.CreateAsset(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := writer.CreateAsset(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("asset ids = %s, %s", first.ID, second.ID)
	}
	assets, err := repo.ListSessionAssets(ctx, run.AccountID, run.SessionID, 10)
	if err != nil || len(assets) != 1 {
		t.Fatalf("assets = (%d, %v), want one", len(assets), err)
	}
}

func TestExecutionWriterSequenceContinuesAfterTwoHundredEvents(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:executor_long_sequence?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-executor-long", "session-executor-long", "account-a", "trigger-message", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	for sequence := uint64(1); sequence <= 205; sequence++ {
		if err := repo.AppendEvent(ctx, &domain.Event{
			ID: fmt.Sprintf("legacy-executor-event-%d", sequence), RunID: run.ID,
			SessionID: run.SessionID, AccountID: run.AccountID, Sequence: sequence,
			Type: "CUSTOM", Payload: json.RawMessage(`{}`), CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	executor := &AgentExecutor{repo: repo, ids: func() string { return "new-executor-event" }, now: func() time.Time { return now }}
	writer := &executionWriter{executor: executor, run: run}
	if err := writer.Emit(ctx, EventToolCallStart, map[string]any{"tool_call_id": "new-tool", "tool_name": "search"}); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 205, 10)
	if err != nil || len(events) != 1 || events[0].Sequence != 206 || events[0].Type != EventToolCallStart {
		t.Fatalf("events after 205 = (%#v, %v)", events, err)
	}
}

func TestExecutionWriterPersistsTextBatchesBeforeMessageEnd(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:executor_durable_text?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-durable-text", "session-durable-text", "account-a", "trigger-message", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	id := 0
	executor := &AgentExecutor{repo: repo, ids: func() string {
		id++
		return fmt.Sprintf("durable-id-%d", id)
	}, now: func() time.Time { return now }}
	writer := &executionWriter{executor: executor, run: run}
	messageID, err := writer.BeginAssistantMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.AppendAssistantMessage(ctx, messageID, "先"); err != nil {
		t.Fatal(err)
	}
	if err := writer.AppendAssistantMessage(ctx, messageID, "后"); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 0, 10)
	if err != nil || len(events) != 2 || events[1].Type != EventTextMessageContent {
		t.Fatalf("events before end = (%#v, %v)", events, err)
	}
	if _, err := writer.EndAssistantMessage(ctx, messageID, "先后"); err != nil {
		t.Fatal(err)
	}
	events, err = repo.ListEventsAfter(ctx, run.AccountID, run.ID, 0, 10)
	if err != nil || len(events) != 4 || events[2].Type != EventTextMessageContent || events[3].Type != EventTextMessageEnd {
		t.Fatalf("events after end = (%#v, %v)", events, err)
	}
	var first, second map[string]string
	if err := json.Unmarshal(events[1].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(events[2].Payload, &second); err != nil {
		t.Fatal(err)
	}
	if first["delta"] != "先" || second["delta"] != "后" {
		t.Fatalf("durable text = %q + %q", first["delta"], second["delta"])
	}
}

func TestExecutionWriterBatchesReasoningAndFlushesBeforeEnd(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:executor_durable_reasoning?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-durable-reasoning", "session-durable-reasoning", "account-a", "trigger-message", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	id := 0
	writer := &executionWriter{executor: &AgentExecutor{repo: repo, ids: func() string {
		id++
		return fmt.Sprintf("reasoning-event-%d", id)
	}, now: func() time.Time { return now }}, run: run}
	for _, step := range []struct {
		typeName string
		payload  map[string]any
	}{
		{EventReasoningStart, map[string]any{}},
		{EventReasoningMessageStart, map[string]any{"message_id": "reasoning-1"}},
		{EventReasoningMessageContent, map[string]any{"message_id": "reasoning-1", "delta": "先"}},
		{EventReasoningMessageContent, map[string]any{"message_id": "reasoning-1", "delta": "想"}},
	} {
		if err := writer.Emit(ctx, step.typeName, step.payload); err != nil {
			t.Fatal(err)
		}
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 0, 10)
	if err != nil || len(events) != 3 {
		t.Fatalf("events before reasoning end = (%#v, %v)", events, err)
	}
	if err := writer.Emit(ctx, EventReasoningMessageEnd, map[string]any{"message_id": "reasoning-1"}); err != nil {
		t.Fatal(err)
	}
	events, err = repo.ListEventsAfter(ctx, run.AccountID, run.ID, 0, 10)
	if err != nil || len(events) != 5 || events[3].Type != EventReasoningMessageContent || events[4].Type != EventReasoningMessageEnd {
		t.Fatalf("events after reasoning end = (%#v, %v)", events, err)
	}
}

func TestExecutionWriterPersistsInFlightProgress(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:executor_stream_progress?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	base := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-progress-writer", "session-progress-writer", "account-a", "trigger-message", base)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	now := base
	id := 0
	executor := &AgentExecutor{
		repo: repo,
		ids: func() string {
			id++
			if id == 1 {
				return "assistant-progress-message"
			}
			return "event-progress-" + strconv.Itoa(id)
		},
		now: func() time.Time { return now },
	}
	writer := &executionWriter{executor: executor, run: run}
	messageID, err := writer.BeginAssistantMessage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.AppendAssistantMessage(context.Background(), messageID, "已经输出一半"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Emit(context.Background(), EventReasoningMessageStart, map[string]any{"message_id": "reasoning-1"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Emit(context.Background(), EventReasoningMessageContent, map[string]any{"message_id": "reasoning-1", "delta": "先分析需求"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Emit(context.Background(), EventToolCallStart, map[string]any{"tool_call_id": "tool-1", "tool_name": "search"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Emit(context.Background(), EventToolCallArgs, map[string]any{"tool_call_id": "tool-1", "delta": `{"q":"雨夜"}`}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Emit(context.Background(), EventToolCallResult, map[string]any{"tool_call_id": "tool-1", "content": "找到资料", "is_error": false}); err != nil {
		t.Fatal(err)
	}
	now = base.Add(time.Second)
	writer.maybePersistProgress(context.Background(), true)

	progress, err := repo.GetRunProgress(context.Background(), run.AccountID, run.ID)
	if err != nil {
		t.Fatalf("GetRunProgress() error = %v", err)
	}
	if progress.AssistantText != "已经输出一半" || progress.ReasoningText != "先分析需求" {
		t.Fatalf("progress text = %#v", progress)
	}
	if !strings.Contains(string(progress.ToolCallsJSON), "找到资料") || progress.LastSequence == 0 {
		t.Fatalf("progress tool/sequence = %s/%d", progress.ToolCallsJSON, progress.LastSequence)
	}
	var calls map[string]runProgressToolCall
	if err := json.Unmarshal(progress.ToolCallsJSON, &calls); err != nil || calls["tool-1"].Args == "" {
		t.Fatalf("progress tool calls = %#v, err=%v", calls, err)
	}

	if _, err := writer.EndAssistantMessage(context.Background(), messageID, "已经输出一半"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetRunProgress(context.Background(), run.AccountID, run.ID); err == nil {
		t.Fatal("run progress remains after assistant message completed")
	}
}
