package application_test

import (
	"context"
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

func hasEventType(events []*domain.Event, want string) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}
