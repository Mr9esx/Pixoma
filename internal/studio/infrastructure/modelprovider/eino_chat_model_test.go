package modelprovider_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

func TestEinoChatModelImplementsEinoGeneration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已收到创作需求"}}],"usage":{"prompt_tokens":3,"completion_tokens":4}}`))
	}))
	defer server.Close()
	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{BaseURL: server.URL + "/v1/chat/completions", Model: "test", APIKey: "secret"})
	message, err := chat.Generate(context.Background(), []*schema.Message{schema.UserMessage("写一个故事")})
	if err != nil {
		t.Fatal(err)
	}
	if message.Role != schema.Assistant || message.Content != "已收到创作需求" {
		t.Fatalf("message = %#v", message)
	}
	if message.ResponseMeta == nil || message.ResponseMeta.Usage == nil || message.ResponseMeta.Usage.TotalTokens != 7 {
		t.Fatalf("usage = %#v", message.ResponseMeta)
	}
}

func TestEinoChatModelBindsNativeOpenAIToolsAndReturnsToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %#v", body["tools"])
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"create_outline","arguments":"{}"}}]}}]}`))
	}))
	defer server.Close()
	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{BaseURL: server.URL + "/v1/chat/completions", Model: "test", APIKey: "secret"})
	withTools, err := chat.WithTools([]*schema.ToolInfo{{Name: "create_outline", Desc: "Create a story outline"}})
	if err != nil {
		t.Fatal(err)
	}
	message, err := withTools.Generate(context.Background(), []*schema.Message{schema.UserMessage("创建大纲")})
	if err != nil {
		t.Fatal(err)
	}
	if len(message.ToolCalls) != 1 || message.ToolCalls[0].ID != "call-1" || message.ToolCalls[0].Function.Name != "create_outline" {
		t.Fatalf("tool calls = %#v", message.ToolCalls)
	}
}

func TestEinoAnthropicToolRoundTripKeepsSignedThinking(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if requests == 1 {
			_, _ = w.Write([]byte(`{"content":[{"type":"thinking","thinking":"查一下","signature":"sig-1"},{"type":"tool_use","id":"toolu-1","name":"list_session_assets","input":{}}]}`))
			return
		}
		messages := body["messages"].([]any)
		assistantBlocks := messages[1].(map[string]any)["content"].([]any)
		require.Equal(t, "sig-1", assistantBlocks[0].(map[string]any)["signature"])
		toolResult := messages[2].(map[string]any)["content"].([]any)
		require.Equal(t, "toolu-1", toolResult[0].(map[string]any)["tool_use_id"])
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"找到了资产"}]}`))
	}))
	defer server.Close()
	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{Protocol: domain.ModelProtocolAnthropic, BaseURL: server.URL, Model: "claude-test", APIKey: "test-secret", Thinking: domain.ThinkingConfig{Enabled: true}})
	withTools, err := chat.WithTools([]*schema.ToolInfo{{Name: "list_session_assets", Desc: "List assets"}})
	require.NoError(t, err)
	first, err := withTools.Generate(context.Background(), []*schema.Message{schema.UserMessage("查资产")})
	require.NoError(t, err)
	require.Len(t, first.ToolCalls, 1)
	second, err := withTools.Generate(context.Background(), []*schema.Message{schema.UserMessage("查资产"), first, {Role: schema.Tool, ToolCallID: "toolu-1", Content: "[]"}})
	require.NoError(t, err)
	require.Equal(t, "找到了资产", second.Content)
}

func TestEinoChatModelHonorsRuntimeToolOptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Len(t, body["tools"], 1)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已收到创作需求"}}]}`))
	}))
	defer server.Close()
	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{BaseURL: server.URL + "/v1/chat/completions", Model: "test", APIKey: "secret"})
	message, err := chat.Generate(context.Background(), []*schema.Message{schema.UserMessage("写一个故事")}, model.WithTools([]*schema.ToolInfo{{Name: "search_reference", Desc: "Search reference material"}}))
	require.NoError(t, err)
	require.Equal(t, "已收到创作需求", message.Content)
}

func TestEinoChatModelStreamsWithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Len(t, body["tools"], 1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"create_outline","arguments":"{\"genre\":\""}}]}}]}

`))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"noir\"}"}}]}}]}

`))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{BaseURL: server.URL, Model: "test", APIKey: "secret"})
	stream, err := chat.Stream(context.Background(), []*schema.Message{schema.UserMessage("创建大纲")}, model.WithTools([]*schema.ToolInfo{{Name: "create_outline", Desc: "Create a story outline"}}))
	require.NoError(t, err)
	defer stream.Close()

	var toolMessage *schema.Message
	for {
		message, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		require.NoError(t, recvErr)
		if len(message.ToolCalls) > 0 {
			toolMessage = message
		}
	}
	require.NotNil(t, toolMessage)
	require.Len(t, toolMessage.ToolCalls, 1)
	require.Equal(t, "create_outline", toolMessage.ToolCalls[0].Function.Name)
	require.Equal(t, `{"genre":"noir"}`, toolMessage.ToolCalls[0].Function.Arguments)
}
