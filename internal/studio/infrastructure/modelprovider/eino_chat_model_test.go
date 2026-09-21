package modelprovider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"

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
	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{BaseURL: server.URL + "/v1", Model: "test", APIKey: "secret"})
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
	chat := modelprovider.NewEinoChatModel(modelprovider.NewOpenAICompatibleClient(server.Client()), domain.ResolvedModelConfig{BaseURL: server.URL + "/v1", Model: "test", APIKey: "secret"})
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
