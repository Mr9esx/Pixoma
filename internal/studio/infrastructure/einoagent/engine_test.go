package einoagent_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/einoagent"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

type resolver struct {
	config *domain.ResolvedModelConfig
}

type connectorResolver struct {
	connectors []studioapp.ResolvedMCPConnector
}

func (r connectorResolver) ResolveMCPConnectors(_ context.Context, _ string) ([]studioapp.ResolvedMCPConnector, error) {
	return r.connectors, nil
}

type workflowResolver struct {
	workflows []studioapp.ResolvedWorkflow
}

func (r workflowResolver) ResolveWorkflows(_ context.Context, _ string) ([]studioapp.ResolvedWorkflow, error) {
	return r.workflows, nil
}

type workflowStarter func(context.Context, studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error)

func (f workflowStarter) Start(ctx context.Context, input studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
	return f(ctx, input)
}

func (r resolver) Resolve(_ context.Context, _ string, _ string) (*domain.ResolvedModelConfig, error) {
	return r.config, nil
}

type sink struct {
	mu        sync.Mutex
	events    []string
	responses []string
	assets    []studioapp.GeneratedAsset
}

func (s *sink) Emit(_ context.Context, eventType string, _ any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, eventType)
	return nil
}

func (s *sink) AssistantMessage(_ context.Context, text string) (*domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses = append(s.responses, text)
	return &domain.Message{ID: "message_01"}, nil
}

func (s *sink) CreateAsset(_ context.Context, asset studioapp.GeneratedAsset) (*domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assets = append(s.assets, asset)
	created, err := domain.NewAsset("asset_01", "session_01", "account_01", asset.Name, asset.Kind, asset.Origin, time.Now())
	if err != nil {
		return nil, err
	}
	_, err = created.AppendVersion("version_01", asset.MIMEType, "studio/asset_01/v1", int64(len(asset.Content)), time.Now())
	return created, err
}
func (*sink) CreateFlowNode(context.Context, studioapp.FlowNodeInput) (*domain.FlowNode, error) {
	return &domain.FlowNode{ID: "node_01"}, nil
}
func (*sink) CreateFlowEdge(context.Context, string, string, string) (*domain.FlowEdge, error) {
	return nil, nil
}
func (*sink) RequestApproval(context.Context, string, string) (*domain.Approval, error) {
	return nil, nil
}

func TestEngineExecutesOneEinoAgentTurn(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/chat/completions", request.URL.Path)
		require.Equal(t, "Bearer test-key", request.Header.Get("Authorization"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, "test-model", body["model"])
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已生成创作建议。"}}],"usage":{"prompt_tokens":10,"completion_tokens":6}}`))
	}))
	defer server.Close()

	engine := &einoagent.Engine{
		Models: resolver{config: &domain.ResolvedModelConfig{
			ID: "model_01", Protocol: domain.ModelProtocolOpenAIChat, BaseURL: server.URL,
			Model: "test-model", APIKey: "test-key",
		}},
		Client: modelprovider.NewOpenAICompatibleClient(server.Client()),
	}
	output := &sink{}

	err := engine.Execute(context.Background(), studioapp.AgentRequest{
		Run:      &domain.Run{ID: "run_01", AccountID: "account_01", SessionID: "session_01", ModelConfigID: "model_01"},
		Session:  &domain.Session{ID: "session_01"},
		UserText: "给我一个漫画开头",
	}, output)

	require.NoError(t, err)
	require.Equal(t, []string{"已生成创作建议。"}, output.responses)
	require.Equal(t, []string{studioapp.EventRunStarted, studioapp.EventRunFinished}, output.events)
}

func TestEngineInvokesAllowedMCPToolAndReturnsFollowUp(t *testing.T) {
	t.Parallel()
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(_ context.Context, _ *mcp.CallToolRequest, input struct {
		Query string `json:"query" jsonschema:"Search query"`
	}) (*mcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"answer": "result for " + input.Query}, nil
	})
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, nil)
	mcpEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "Bearer connector-secret", request.Header.Get("Authorization"))
		mcpHandler.ServeHTTP(writer, request)
	}))
	defer mcpEndpoint.Close()

	var calls int
	modelEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		writer.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			require.True(t, hasToolNamed(body["tools"], "mcp_connector_01_search_reference"))
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_01","type":"function","function":{"name":"mcp_connector_01_search_reference","arguments":"{\"query\":\"rain\"}"}}]}}]}`))
			return
		}
		messages, ok := body["messages"].([]any)
		require.True(t, ok)
		require.Contains(t, messages[len(messages)-1].(map[string]any)["content"], "result for rain")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已检索到参考资料。"}}]}`))
	}))
	defer modelEndpoint.Close()

	engine := &einoagent.Engine{
		Models: resolver{config: &domain.ResolvedModelConfig{
			ID: "model_01", Protocol: domain.ModelProtocolOpenAIChat, BaseURL: modelEndpoint.URL,
			Model: "test-model", APIKey: "test-key",
		}},
		Capabilities: connectorResolver{connectors: []studioapp.ResolvedMCPConnector{{
			ID: "connector-01", Name: "Reference", URL: mcpEndpoint.URL, Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
			Tools: []domain.MCPTool{{Name: "search_reference", Description: "Search reference material", InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)}},
		}}},
		Client: modelprovider.NewOpenAICompatibleClient(modelEndpoint.Client()),
	}
	output := &sink{}

	err := engine.Execute(context.Background(), studioapp.AgentRequest{
		Run:      &domain.Run{ID: "run_01", AccountID: "account_01", SessionID: "session_01", ModelConfigID: "model_01"},
		Session:  &domain.Session{ID: "session_01", PermissionMode: domain.PermissionFullAccess},
		UserText: "检索 rain 的参考资料",
	}, output)

	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, []string{"已检索到参考资料。"}, output.responses)
	require.Equal(t, []string{studioapp.EventRunStarted, studioapp.EventToolCallStart, studioapp.EventToolCallEnd, studioapp.EventRunFinished}, output.events)
}

func TestEngineCreatesMarkdownAssetWithBuiltInTool(t *testing.T) {
	t.Parallel()
	var calls int
	modelEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		writer.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			tools, ok := body["tools"].([]any)
			require.True(t, ok)
			require.Contains(t, toolNames(tools), "create_text_asset")
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_01","type":"function","function":{"name":"create_text_asset","arguments":"{\"name\":\"story.md\",\"content\":\"# 雨夜侦探\"}"}}]}}]}`))
			return
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已创建故事大纲。"}}]}`))
	}))
	defer modelEndpoint.Close()
	engine := &einoagent.Engine{
		Models: resolver{config: &domain.ResolvedModelConfig{ID: "model_01", Protocol: domain.ModelProtocolOpenAIChat, BaseURL: modelEndpoint.URL, Model: "test-model", APIKey: "test-key"}},
		Client: modelprovider.NewOpenAICompatibleClient(modelEndpoint.Client()),
	}
	output := &sink{}
	err := engine.Execute(context.Background(), studioapp.AgentRequest{
		Run:     &domain.Run{ID: "run_01", AccountID: "account_01", SessionID: "session_01", ModelConfigID: "model_01"},
		Session: &domain.Session{ID: "session_01", PermissionMode: domain.PermissionFullAccess}, UserText: "把故事大纲写成 Markdown 文件",
	}, output)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Len(t, output.assets, 1)
	require.Equal(t, "story.md", output.assets[0].Name)
	require.Equal(t, "# 雨夜侦探", string(output.assets[0].Content))
	require.Equal(t, []string{studioapp.EventRunStarted, studioapp.EventToolCallStart, studioapp.EventToolCallEnd, studioapp.EventRunFinished}, output.events)
}

func toolNames(tools []any) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		value, _ := tool.(map[string]any)
		function, _ := value["function"].(map[string]any)
		name, _ := function["name"].(string)
		names = append(names, name)
	}
	return names
}

func hasToolNamed(raw any, expected string) bool {
	tools, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, tool := range tools {
		value, _ := tool.(map[string]any)
		if name, _ := value["name"].(string); name == expected {
			return true
		}
		function, _ := value["function"].(map[string]any)
		if name, _ := function["name"].(string); name == expected {
			return true
		}
	}
	return false
}

func TestEngineInvokesEnabledWorkflowToolAndReturnsSubmission(t *testing.T) {
	t.Parallel()
	var calls int
	modelEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		writer.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			require.True(t, hasToolNamed(body["tools"], "studio_workflow_12"))
			_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_01","type":"function","function":{"name":"studio_workflow_12","arguments":"{\"prompt\":\"rain\"}"}}]}}]}`))
			return
		}
		messages, ok := body["messages"].([]any)
		require.True(t, ok)
		require.Contains(t, messages[len(messages)-1].(map[string]any)["content"], "task-1")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已提交分镜工作流。"}}]}`))
	}))
	defer modelEndpoint.Close()

	var started studioapp.WorkflowStartInput
	engine := &einoagent.Engine{
		Models: resolver{config: &domain.ResolvedModelConfig{
			ID: "model_01", Protocol: domain.ModelProtocolOpenAIChat, BaseURL: modelEndpoint.URL,
			Model: "test-model", APIKey: "test-key",
		}},
		Workflows: workflowResolver{workflows: []studioapp.ResolvedWorkflow{{
			ID: "12", ToolName: "studio_workflow_12", Name: "分镜工作流", Description: "根据故事生成分镜",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string"}}}`),
		}}},
		WorkflowStarter: workflowStarter(func(_ context.Context, input studioapp.WorkflowStartInput) (*studioapp.WorkflowStartResult, error) {
			started = input
			return &studioapp.WorkflowStartResult{TaskID: "task-1", WorkflowID: "12"}, nil
		}),
		Client: modelprovider.NewOpenAICompatibleClient(modelEndpoint.Client()),
	}
	output := &sink{}

	err := engine.Execute(context.Background(), studioapp.AgentRequest{
		Run:      &domain.Run{ID: "run_01", AccountID: "account_01", SessionID: "session_01", ModelConfigID: "model_01"},
		Session:  &domain.Session{ID: "session_01", PermissionMode: domain.PermissionFullAccess},
		UserText: "根据故事生成分镜",
	}, output)

	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "account_01", started.AccountID)
	require.Equal(t, "run_01", started.RunID)
	require.Equal(t, map[string]any{"prompt": "rain"}, started.Inputs)
	require.Equal(t, []string{"已提交分镜工作流。"}, output.responses)
	require.Equal(t, []string{studioapp.EventRunStarted, studioapp.EventToolCallStart, studioapp.EventToolCallEnd, studioapp.EventRunFinished}, output.events)
}

func TestEngineInvokesMCPToolThroughResponsesProtocol(t *testing.T) {
	t.Parallel()
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(_ context.Context, _ *mcp.CallToolRequest, input struct {
		Query string `json:"query" jsonschema:"Search query"`
	}) (*mcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"answer": "result for " + input.Query}, nil
	})
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, nil)
	mcpEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "Bearer connector-secret", request.Header.Get("Authorization"))
		mcpHandler.ServeHTTP(writer, request)
	}))
	defer mcpEndpoint.Close()

	var calls int
	modelEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		require.Equal(t, "/responses", request.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		writer.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			require.True(t, hasToolNamed(body["tools"], "mcp_connector_01_search_reference"))
			_, _ = writer.Write([]byte(`{"output":[{"type":"reasoning","id":"rs_01","encrypted_content":"opaque-reasoning"},{"type":"function_call","call_id":"call_01","name":"mcp_connector_01_search_reference","arguments":"{\"query\":\"rain\"}"}]}`))
			return
		}
		input, ok := body["input"].([]any)
		require.True(t, ok)
		require.True(t, responsesInputIncludesToolRound(input, "rs_01", "opaque-reasoning", "call_01", "mcp_connector_01_search_reference", `{"query":"rain"}`, "result for rain"), "input = %#v", input)
		_, _ = writer.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"已检索到参考资料。"}]}]}`))
	}))
	defer modelEndpoint.Close()

	engine := &einoagent.Engine{
		Models: resolver{config: &domain.ResolvedModelConfig{
			ID: "model_01", Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: modelEndpoint.URL,
			Model: "test-model", APIKey: "test-key",
		}},
		Capabilities: connectorResolver{connectors: []studioapp.ResolvedMCPConnector{{
			ID: "connector-01", Name: "Reference", URL: mcpEndpoint.URL, Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
			Tools: []domain.MCPTool{{Name: "search_reference", Description: "Search reference material", InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)}},
		}}},
		Client: modelprovider.NewOpenAICompatibleClient(modelEndpoint.Client()),
	}
	output := &sink{}

	err := engine.Execute(context.Background(), studioapp.AgentRequest{
		Run:      &domain.Run{ID: "run_01", AccountID: "account_01", SessionID: "session_01", ModelConfigID: "model_01"},
		Session:  &domain.Session{ID: "session_01", PermissionMode: domain.PermissionFullAccess},
		UserText: "检索 rain 的参考资料",
	}, output)

	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, []string{"已检索到参考资料。"}, output.responses)
	require.Equal(t, []string{studioapp.EventRunStarted, studioapp.EventToolCallStart, studioapp.EventToolCallEnd, studioapp.EventRunFinished}, output.events)
}

func responsesInputIncludesToolRound(input []any, reasoningID, encryptedReasoning, callID, name, arguments, result string) bool {
	for index := 0; index+2 < len(input); index++ {
		reasoning, reasoningOK := input[index].(map[string]any)
		functionCall, functionCallOK := input[index+1].(map[string]any)
		functionOutput, functionOutputOK := input[index+2].(map[string]any)
		if !reasoningOK || !functionCallOK || !functionOutputOK {
			continue
		}
		if reasoning["type"] != "reasoning" || reasoning["id"] != reasoningID || reasoning["encrypted_content"] != encryptedReasoning {
			continue
		}
		if functionCall["type"] != "function_call" || functionCall["call_id"] != callID || functionCall["name"] != name || functionCall["arguments"] != arguments {
			continue
		}
		if functionOutput["type"] != "function_call_output" || functionOutput["call_id"] != callID {
			continue
		}
		output, _ := functionOutput["output"].(string)
		if strings.Contains(output, result) {
			return true
		}
	}
	return false
}
