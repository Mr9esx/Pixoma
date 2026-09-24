package application_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/einoagent"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

func TestRealAgentCanAddSessionSOPStages(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	modelCalls := 0
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelCalls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if modelCalls == 1 {
			tools, _ := body["tools"].([]any)
			found := false
			for _, candidate := range tools {
				wrapper, _ := candidate.(map[string]any)
				function, _ := wrapper["function"].(map[string]any)
				if function["name"] == "edit_session_flow" {
					found = true
				}
			}
			if !found {
				t.Errorf("edit_session_flow was not sent to the model")
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call-sop","type":"function","function":{"name":"edit_session_flow","arguments":"{\"operations\":[{\"type\":\"create_stage\",\"title\":\"立住角色\"},{\"type\":\"create_stage\",\"title\":\"排好分镜\"}]}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已安排创作阶段。"}}]}`))
	}))
	defer endpoint.Close()
	ids := &idSequence{}
	models := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: ids.Next}
	model, err := models.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "测试 Agent", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: endpoint.URL, Model: "test-model", APIKey: "test-key", Enabled: true, AgentEnabled: true,
		Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	engine := &einoagent.Engine{Models: models, Client: modelprovider.NewOpenAICompatibleClient(endpoint.Client())}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: engine, IDs: ids.Next})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Queue: runner}
	turn, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a", Text: "安排漫画创作阶段", ModelConfigID: model.ID,
		PermissionMode: domain.PermissionFullAccess,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitRunStatus(t, repo, turn.Run.ID, domain.RunSucceeded)
	nodes, _, err := repo.GetFlow(context.Background(), "account-a", turn.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if modelCalls != 2 || len(nodes) != 2 || nodes[0].Title != "立住角色" || nodes[1].Title != "排好分镜" {
		t.Fatalf("modelCalls=%d nodes=%#v", modelCalls, nodes)
	}
}

func TestRealAgentFlowEditRetryDoesNotMoveNodeOrDuplicateEdge(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	modelCalls := 0
	var stageID string
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		modelCalls++
		w.Header().Set("Content-Type", "application/json")
		if modelCalls <= 2 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call-plan","type":"function","function":{"name":"edit_session_flow","arguments":"{\"operations\":[{\"type\":\"add_operation\",\"title\":\"生成分镜\",\"stage_node_id\":\"` + stageID + `\"}]}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已规划分镜步骤。"}}]}`))
	}))
	defer endpoint.Close()
	ids := &idSequence{}
	models := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: ids.Next}
	model, err := models.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "测试 Agent", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: endpoint.URL, Model: "test-model", APIKey: "test-key", Enabled: true, AgentEnabled: true,
		Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
	})
	if err != nil {
		t.Fatal(err)
	}
	engine := &einoagent.Engine{Models: models, Client: modelprovider.NewOpenAICompatibleClient(endpoint.Client())}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: engine, IDs: ids.Next})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Queue: runner}
	session, err := service.CreateSession(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	stage, err := service.CreateFlowNode(context.Background(), "account-a", session.ID, studioapp.CreateFlowNodeInput{Type: domain.FlowNodeStage, Title: "排好分镜", Position: studioapp.FlowPositionInput{X: 480, Y: 240}})
	if err != nil {
		t.Fatal(err)
	}
	stageID = stage.ID
	turn, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{AccountID: "account-a", SessionID: session.ID, Text: "规划分镜", ModelConfigID: model.ID, PermissionMode: domain.PermissionFullAccess})
	if err != nil {
		t.Fatal(err)
	}
	waitRunStatus(t, repo, turn.Run.ID, domain.RunSucceeded)
	nodes, edges, err := repo.GetFlow(context.Background(), "account-a", session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if modelCalls != 3 || len(nodes) != 2 || len(edges) != 1 {
		t.Fatalf("calls=%d nodes=%#v edges=%#v", modelCalls, nodes, edges)
	}
	if nodes[0].ID != stage.ID || nodes[0].PositionX != 480 || nodes[1].SortOrder != 10 {
		t.Fatalf("nodes moved: %#v", nodes)
	}
}

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
	wantEventTypes := []string{"RUN_STARTED", "TEXT_MESSAGE_END", "TOOL_CALL_START", "TOOL_CALL_END", "RUN_FINISHED"}
	for _, eventType := range wantEventTypes {
		if !hasEventType(events, eventType) {
			t.Errorf("events missing %q: %#v", eventType, events)
		}
	}
}

func TestAgentExecutorPersistsFinalAssistantTextWithoutDeltaEvents(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	service := &studioapp.Service{
		Repo: repo, IDs: ids.Next, Now: time.Now, Queue: &queueSpy{},
	}
	turn, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a", Text: "写一段文本",
	})
	if err != nil {
		t.Fatal(err)
	}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{
		Repo: repo, Blob: blobs, Engine: &streamingEngine{}, IDs: ids.Next,
	})
	if err := executor.Execute(context.Background(), turn.Run); err != nil {
		t.Fatal(err)
	}

	events, err := repo.ListEventsAfter(context.Background(), "account-a", turn.Run.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEventType(events, studioapp.EventTextMessageContent) {
		t.Fatalf("durable events omit text deltas: %#v", events)
	}
	for _, event := range events {
		if event.Type != studioapp.EventTextMessageEnd {
			continue
		}
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Content != "第一段第二段" {
			t.Fatalf("final text = %q, want %q", payload.Content, "第一段第二段")
		}
		return
	}
	t.Fatal("TEXT_MESSAGE_END event missing")
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
	if err := first.Run.Start(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := first.Run.Succeed(now.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateRun(context.Background(), first.Run); err != nil {
		t.Fatal(err)
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

type streamingEngine struct{}

func (*streamingEngine) Execute(ctx context.Context, _ studioapp.AgentRequest, sink studioapp.AgentSink) error {
	stream, ok := sink.(studioapp.AssistantStreamSink)
	if !ok {
		return nil
	}
	messageID, err := stream.BeginAssistantMessage(ctx)
	if err != nil {
		return err
	}
	if err := stream.AppendAssistantMessage(ctx, messageID, "第一段"); err != nil {
		return err
	}
	if err := stream.AppendAssistantMessage(ctx, messageID, "第二段"); err != nil {
		return err
	}
	_, err = stream.EndAssistantMessage(ctx, messageID, "第一段第二段")
	return err
}

func hasEventType(events []*domain.Event, want string) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}
