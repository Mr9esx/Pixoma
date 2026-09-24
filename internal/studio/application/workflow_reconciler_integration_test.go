//go:build integration

package application_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/platform/blob"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/queue"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/einoagent"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

type workflowEventChannel struct{ messages chan queue.Message }

func (p workflowEventChannel) Publish(_ context.Context, message queue.Message) error {
	p.messages <- message
	return nil
}

func TestStudioAgentWorkflowTaskProducesSessionAsset(t *testing.T) {
	ctx := context.Background()
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	require.NoError(t, err)
	tasks := runtimedomain.NewMemoryTaskRepository()
	ids := &idSequence{}
	var modelCalls atomic.Int32
	modelEndpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if modelCalls.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call-storyboard","type":"function","function":{"name":"studio_workflow_12","arguments":"{\"prompt\":\"雨夜侦探第一话\"}"}}]}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"分镜工作流已提交，完成后可在右侧查看图片。"}}]}`))
	}))
	defer modelEndpoint.Close()

	catalog := workflowCatalog{cases: []*catalogdomain.Case{{
		Document: catalogdomain.CaseDocument{
			ID: sharedkernel.CaseID(12), Name: "分镜生成", Description: "根据漫画故事生成分镜图",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{"prompt": map[string]any{"type": "string"}}, "required": []string{"prompt"}},
			Inputs:      []catalogdomain.InputField{{Key: "prompt", Type: "string", Required: true}},
		},
		Enabled: true,
	}}}
	capabilities := &studioapp.CapabilityConfigService{Repo: repo, WorkflowCatalog: catalog}
	_, err = capabilities.SetAgentWorkflowEnabled(ctx, "account-a", "12", true)
	require.NoError(t, err)
	models := &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: ids.Next}
	model, err := models.Create(ctx, studioapp.CreateModelConfigInput{
		AccountID: "account-a", Name: "测试 Agent", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: modelEndpoint.URL, Model: "test-model", APIKey: "test-key", Enabled: true, AgentEnabled: true,
		Limits:       domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
		Capabilities: domain.ModelCapabilities{Tools: true},
	})
	require.NoError(t, err)
	events := workflowEventChannel{messages: make(chan queue.Message, 1)}
	starter := &studioapp.WorkflowStarter{
		StudioRepo: repo, Catalog: catalog,
		Validator: inputValidator(func(_ catalogdomain.CaseDocument, values []catalogdomain.InputValue) error {
			if len(values) != 1 || values[0].Text == nil || *values[0].Text != "雨夜侦探第一话" {
				t.Errorf("workflow input = %#v", values)
			}
			return nil
		}),
		Tasks: tasks, Blob: blobs, Publisher: events, NewID: ids.Next,
		NewTaskID: func() sharedkernel.TaskID { return "task-storyboard" },
	}
	engine := &einoagent.Engine{
		Models: models, Workflows: capabilities, WorkflowStarter: starter,
		Client: modelprovider.NewOpenAICompatibleClient(modelEndpoint.Client()),
	}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: engine, IDs: ids.Next})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Queue: runner, Titles: staticTitleGenerator{title: "雨夜侦探分镜"}}
	turn, err := service.SendMessage(ctx, studioapp.SendMessageInput{
		AccountID: "account-a", Text: "请用分镜生成工作流把雨夜侦探第一话画成图片",
		ModelConfigID: model.ID, PermissionMode: domain.PermissionFullAccess,
	})
	require.NoError(t, err)
	waitRunStatus(t, repo, turn.Run.ID, domain.RunSucceeded)
	require.EqualValues(t, 2, modelCalls.Load())

	select {
	case event := <-events.messages:
		require.Equal(t, sharedkernel.TopicTaskCreated, event.Topic)
		var created sharedkernel.TaskCreated
		require.NoError(t, json.Unmarshal(event.Payload, &created))
		require.Equal(t, sharedkernel.TaskID("task-storyboard"), created.TaskID)
	case <-time.After(5 * time.Second):
		t.Fatal("Agent did not submit the workflow task")
	}
	assetBefore, err := repo.ListSessionAssets(ctx, "account-a", turn.Session.ID, 10)
	require.NoError(t, err)
	require.Empty(t, assetBefore, "output must not appear before task completion")

	content := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>雨夜侦探第一话</text></svg>`)
	ref, err := blobs.Put(ctx, "outputs/task-storyboard/storyboard.svg", strings.NewReader(string(content)), blob.PutOptions{MIME: "image/svg+xml"})
	require.NoError(t, err)
	task, err := tasks.Get(ctx, "task-storyboard")
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, task.MarkQueued("edge-mock", now))
	require.NoError(t, task.MarkRunning("prompt-mock", now))
	require.NoError(t, task.MarkSucceeded([]runtimedomain.OutputRef{{Key: "storyboard", Blob: ref}}, now))
	require.NoError(t, tasks.Update(ctx, task))
	reconciler := &studioapp.WorkflowReconciler{Repo: repo, Tasks: tasks, IDs: ids.Next}
	require.NoError(t, reconciler.ReconcileOnce(ctx, 10))

	assets, err := repo.ListSessionAssets(ctx, "account-a", turn.Session.ID, 10)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, domain.AssetImage, assets[0].Kind)
	require.Equal(t, domain.AssetOriginWorkflow, assets[0].Origin)
	require.Len(t, assets[0].Versions, 1)
	reader, err := blobs.Get(ctx, sharedkernel.BlobRef{Key: assets[0].Versions[0].BlobKey})
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	gotContent, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, content, gotContent)

	nodes, edges, err := repo.GetFlow(ctx, "account-a", turn.Session.ID)
	require.NoError(t, err)
	require.Len(t, nodes, 2)
	require.Len(t, edges, 1)
	require.Equal(t, domain.FlowNodeOperation, nodes[0].Type)
	require.Equal(t, domain.FlowNodeAsset, nodes[1].Type)
	require.Equal(t, assets[0].ID, nodes[1].AssetID)
	require.Equal(t, nodes[0].ID, edges[0].SourceNodeID)
	require.Equal(t, nodes[1].ID, edges[0].TargetNodeID)

	execution, err := repo.GetWorkflowExecutionByTask(ctx, "account-a", "task-storyboard")
	require.NoError(t, err)
	require.Equal(t, domain.WorkflowExecutionSucceeded, execution.Status)
}
