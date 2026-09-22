package persistence_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func TestRunEventsAreOrderedAndIdempotent(t *testing.T) {
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
	duplicate := &domain.Event{ID: "event-2-retry", RunID: run.ID, SessionID: run.SessionID, AccountID: run.AccountID, Sequence: 2, Type: "TEXT_MESSAGE_CONTENT", Payload: json.RawMessage(`{"delta":"好"}`), CreatedAt: now.Add(3 * time.Second)}
	if err := repo.AppendEvent(ctx, duplicate); err != nil {
		t.Fatalf("idempotent AppendEvent() error = %v", err)
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
	if err := repo.SaveAssetToLibrary(ctx, "account-a", asset.ID, "folder-story", now.Add(time.Second)); err != nil {
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
	items, err := repo.ListLibraryAssets(ctx, "account-a", "folder-story", 100)
	if err != nil || len(items) != 1 || items[0].ID != asset.ID {
		t.Fatalf("ListLibraryAssets() = (%#v, %v)", items, err)
	}
	if items[0].CurrentVersion != 1 || len(items[0].Versions) != 1 || items[0].Versions[0].ID != "version-1" {
		t.Fatalf("library asset must retain saved v1, got %#v", items[0])
	}
	if _, err := repo.GetAsset(ctx, "account-b", asset.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account GetAsset() error = %v, want ErrNotFound", err)
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
