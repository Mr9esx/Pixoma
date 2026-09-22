package application_test

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestMockAgentCompletesConversationWorkflowAndAssets(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	now := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{
		Repo:   repo,
		Blob:   blobs,
		Engine: studioapp.NewMockEngine(),
		IDs:    ids.Next,
		Now:    func() time.Time { return now },
	})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	service := &studioapp.Service{
		Repo: repo, IDs: ids.Next, Now: func() time.Time { return now },
		Titles: staticTitleGenerator{title: "雨夜侦探漫画分镜"}, Queue: runner,
	}

	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID:      "account-a",
		Text:           "先写故事大纲，再调用分镜工作流生成一张分镜预览图",
		PermissionMode: domain.PermissionFullAccess,
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	waitRunStatus(t, repo, result.Run.ID, domain.RunSucceeded)

	messages, err := repo.ListMessages(context.Background(), "account-a", result.Session.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) < 2 || messages[0].Role != domain.MessageRoleUser || messages[1].Role != domain.MessageRoleAssistant {
		t.Fatalf("messages = %#v", messages)
	}
	assets, err := repo.ListSessionAssets(context.Background(), "account-a", result.Session.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 || assets[0].Kind != domain.AssetDocument || assets[1].Kind != domain.AssetImage {
		t.Fatalf("assets = %#v", assets)
	}
	imageAsset := assets[1]
	reader, err := blobs.Get(context.Background(), sharedkernel.BlobRef{Key: imageAsset.Versions[0].BlobKey})
	if err != nil {
		t.Fatalf("read generated image: %v", err)
	}
	defer reader.Close()
	raw, _ := io.ReadAll(reader)
	if !strings.Contains(string(raw), "<svg") {
		t.Fatalf("generated image is not SVG: %q", raw)
	}

	nodes, edges, err := repo.GetFlow(context.Background(), "account-a", result.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 4 || len(edges) != 3 {
		t.Fatalf("flow nodes=%d edges=%d", len(nodes), len(edges))
	}
	if nodes[0].Type != domain.FlowNodeStage || nodes[1].Type != domain.FlowNodeAsset || nodes[2].Type != domain.FlowNodeOperation || nodes[3].Type != domain.FlowNodeAsset {
		t.Fatalf("flow node types = [%s %s %s %s]", nodes[0].Type, nodes[1].Type, nodes[2].Type, nodes[3].Type)
	}
	events, err := repo.ListEventsAfter(context.Background(), "account-a", result.Run.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	wantEventTypes := []string{"RUN_STARTED", "TEXT_MESSAGE_CONTENT", "TOOL_CALL_START", "TOOL_CALL_END", "RUN_FINISHED"}
	for _, eventType := range wantEventTypes {
		if !hasEventType(events, eventType) {
			t.Errorf("events missing %q: %#v", eventType, events)
		}
	}
}

func TestAgentExecutorLoadsTheSkillSelectedForRun(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	skill, err := domain.NewSkill(ids.Next(), "account-a", "漫画分镜", "拆分镜头", "先输出镜头表", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	skill.Enabled = true
	if err := repo.CreateSkill(context.Background(), skill); err != nil {
		t.Fatal(err)
	}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: time.Now, Queue: &queueSpy{}}
	run, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{AccountID: "account-a", Text: "写分镜", SkillIDs: []string{skill.ID}})
	if err != nil {
		t.Fatal(err)
	}
	capture := &captureEngine{}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: capture, IDs: ids.Next})
	if err := executor.Execute(context.Background(), run.Run); err != nil {
		t.Fatal(err)
	}
	if len(capture.request.Skills) != 1 || capture.request.Skills[0].Prompt != skill.Prompt {
		t.Fatalf("request skills = %#v", capture.request.Skills)
	}
}

func TestAgentExecutorUsesTheAssetVersionSnapshottedWhenTheRunWasCreated(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	asset, err := domain.NewAsset("asset-1", "session-1", "account-a", "故事大纲.md", domain.AssetDocument, domain.AssetOriginUser, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	first, err := asset.AppendVersion("version-1", "text/markdown", "studio/account-a/asset-1/v1.md", 128, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAsset(context.Background(), asset); err != nil {
		t.Fatal(err)
	}
	session, err := domain.NewSession("session-1", "account-a", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: time.Now, Queue: &queueSpy{}}
	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a", SessionID: "session-1", Text: "基于故事大纲生成分镜", SelectedAssetIDs: []string{asset.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := asset.AppendVersion("version-2", "text/markdown", "studio/account-a/asset-1/v2.md", 256, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AppendAssetVersion(context.Background(), asset.ID, "account-a", second); err != nil {
		t.Fatal(err)
	}
	capture := &captureEngine{}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: capture, IDs: ids.Next})
	if err := executor.Execute(context.Background(), result.Run); err != nil {
		t.Fatal(err)
	}
	if len(capture.request.Assets) != 1 || capture.request.Assets[0].CurrentVersion != first.Version || len(capture.request.Assets[0].Versions) != 1 || capture.request.Assets[0].Versions[0].ID != first.ID {
		t.Fatalf("request asset must retain v1, got %#v", capture.request.Assets)
	}
}

func TestAgentExecutorReplaysCompletedHistoricalToolCalls(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	now := time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)
	clock := now
	service := &studioapp.Service{
		Repo: repo, IDs: ids.Next, Now: func() time.Time { return clock },
		Titles: staticTitleGenerator{title: "工具历史"}, Queue: &queueSpy{},
	}
	first, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{AccountID: "account-a", Text: "查雨夜资料"})
	if err != nil {
		t.Fatal(err)
	}
	assistant := &domain.Message{
		ID: "assistant-1", SessionID: first.Session.ID, AccountID: first.Session.AccountID, RunID: first.Run.ID,
		Role: domain.MessageRoleAssistant, ContentJSON: json.RawMessage(`[{"type":"text","text":"已找到资料"}]`), CreatedAt: now.Add(time.Second),
	}
	if err := repo.AppendMessage(context.Background(), assistant); err != nil {
		t.Fatal(err)
	}
	for _, event := range []*domain.Event{
		{ID: "event-1", RunID: first.Run.ID, SessionID: first.Session.ID, AccountID: first.Session.AccountID, Sequence: 1, Type: studioapp.EventToolCallStart, Payload: json.RawMessage(`{"tool_call_id":"call-1","tool_name":"search"}`), CreatedAt: now.Add(time.Millisecond)},
		{ID: "event-2", RunID: first.Run.ID, SessionID: first.Session.ID, AccountID: first.Session.AccountID, Sequence: 2, Type: studioapp.EventToolCallArgs, Payload: json.RawMessage(`{"tool_call_id":"call-1","delta":"{\"query\":\"雨夜\"}"}`), CreatedAt: now.Add(2 * time.Millisecond)},
		{ID: "event-3", RunID: first.Run.ID, SessionID: first.Session.ID, AccountID: first.Session.AccountID, Sequence: 3, Type: studioapp.EventToolCallResult, Payload: json.RawMessage(`{"tool_call_id":"call-1","content":"三条资料"}`), CreatedAt: now.Add(3 * time.Millisecond)},
		{ID: "event-4", RunID: first.Run.ID, SessionID: first.Session.ID, AccountID: first.Session.AccountID, Sequence: 4, Type: studioapp.EventTextMessageEnd, Payload: json.RawMessage(`{"message_id":"assistant-1"}`), CreatedAt: now.Add(time.Second)},
	} {
		if err := repo.AppendEvent(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	clock = now.Add(2 * time.Second)
	second, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{AccountID: "account-a", SessionID: first.Session.ID, Text: "基于资料继续"})
	if err != nil {
		t.Fatal(err)
	}
	capture := &captureEngine{}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: capture, IDs: ids.Next, Now: func() time.Time { return clock }})
	if err := executor.Execute(context.Background(), second.Run); err != nil {
		t.Fatal(err)
	}
	history := capture.request.History
	if len(history) != 4 {
		t.Fatalf("history length = %d, want 4: %#v", len(history), history)
	}
	if history[0].Role != "user" || history[1].Role != "assistant" || len(history[1].ToolCalls) != 1 || history[2].Role != "tool" || history[3].Content != "已找到资料" {
		t.Fatalf("replayed history = %#v", history)
	}
	if len(capture.request.HistoryMessageIDs) != len(history) {
		t.Fatalf("history boundaries = %#v, history = %#v", capture.request.HistoryMessageIDs, history)
	}
}

type captureEngine struct{ request studioapp.AgentRequest }

func (e *captureEngine) Execute(_ context.Context, request studioapp.AgentRequest, _ studioapp.AgentSink) error {
	e.request = request
	return nil
}

func hasEventType(events []*domain.Event, want string) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}
