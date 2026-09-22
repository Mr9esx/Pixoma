package modelprovider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

func TestOpenAICompatibleChatUsesConfiguredBaseURLAndSecret(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-secret" {
			t.Errorf("Authorization = %q", auth)
		}
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "deepseek-v4-flash" || len(body.Messages) != 1 || body.Messages[0].Content != "你好" {
			t.Errorf("request body = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"你好，我是 Pixoma"}}],"usage":{"prompt_tokens":2,"completion_tokens":5}}`))
	}))
	defer server.Close()

	client := modelprovider.NewOpenAICompatibleClient(server.Client())
	result, err := client.Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL + "/v1", Model: "deepseek-v4-flash", APIKey: "test-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "你好"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if result.Text != "你好，我是 Pixoma" || result.InputTokens != 2 || result.OutputTokens != 5 {
		t.Fatalf("result = %#v", result)
	}
}

func TestOpenAICompatibleChatReturnsProviderErrorWithoutLeakingSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	client := modelprovider.NewOpenAICompatibleClient(server.Client())
	_, err := client.Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL, Model: "test", APIKey: "super-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Chat() error = nil")
	}
	if strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("error leaks secret: %v", err)
	}
}

func TestChatUsesOpenAIResponsesProtocolAndThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		requireNoError(t, json.NewDecoder(r.Body).Decode(&body))
		if body["model"] != "gpt-test" {
			t.Fatalf("model = %#v", body["model"])
		}
		reasoning, ok := body["reasoning"].(map[string]any)
		if !ok || reasoning["effort"] != "high" {
			t.Fatalf("reasoning = %#v", body["reasoning"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"Responses 已完成"}]}],"usage":{"input_tokens":7,"output_tokens":3}}`))
	}))
	defer server.Close()

	result, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: server.URL + "/v1", Model: "gpt-test", APIKey: "test-secret", Thinking: domain.ThinkingConfig{Enabled: true, Effort: "high"}},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "你好"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result.Text != "Responses 已完成" || result.InputTokens != 7 || result.OutputTokens != 3 {
		t.Fatalf("result = %#v", result)
	}
}

func TestChatParsesResponsesOutputAfterReasoningItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"completed",
			"output":[
				{"type":"reasoning","summary":[]},
				{"type":"message","content":[{"type":"output_text","text":"OK."}]}
			],
			"usage":{"input_tokens":11,"output_tokens":2}
		}`))
	}))
	defer server.Close()

	result, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: server.URL + "/api/v3", Model: "deepseek-v4-1-flash-260910", APIKey: "test-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "Reply with OK."}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result.Text != "OK." || result.InputTokens != 11 || result.OutputTokens != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestChatUsesResponsesFunctionToolsAndParsesFunctionCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body map[string]any
		requireNoError(t, json.NewDecoder(r.Body).Decode(&body))
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Fatalf("tools = %#v", body["tools"])
		}
		tool, ok := tools[0].(map[string]any)
		if !ok || tool["type"] != "function" || tool["name"] != "create_outline" {
			t.Fatalf("tool = %#v", tools[0])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"function_call","call_id":"call_outline_01","name":"create_outline","arguments":"{\"genre\":\"noir\"}"}]}`))
	}))
	defer server.Close()

	result, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: server.URL + "/v1", Model: "gpt-test", APIKey: "test-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "创建黑色电影大纲"}},
		Tools: []modelprovider.ToolDefinition{{
			Name: "create_outline", Description: "Create a story outline",
			Parameters: map[string]any{"type": "object", "properties": map[string]any{"genre": map[string]any{"type": "string"}}},
		}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].ID != "call_outline_01" || result.ToolCalls[0].Function.Name != "create_outline" || result.ToolCalls[0].Function.Arguments != `{"genre":"noir"}` {
		t.Fatalf("tool calls = %#v", result.ToolCalls)
	}
}

func TestChatUsesAnthropicMessagesProtocolAndThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "anthropic-secret" || r.Header.Get("Anthropic-Version") != "2023-06-01" {
			t.Fatalf("Anthropic headers are missing")
		}
		var body map[string]any
		requireNoError(t, json.NewDecoder(r.Body).Decode(&body))
		thinking, ok := body["thinking"].(map[string]any)
		if !ok || thinking["type"] != "enabled" || thinking["budget_tokens"] != float64(2048) {
			t.Fatalf("thinking = %#v", body["thinking"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"Anthropic 已完成"}],"usage":{"input_tokens":9,"output_tokens":4}}`))
	}))
	defer server.Close()

	result, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolAnthropic, BaseURL: server.URL + "/v1", Model: "claude-test", APIKey: "anthropic-secret", Thinking: domain.ThinkingConfig{Enabled: true, BudgetTokens: 2048}},
		Messages: []modelprovider.ChatMessage{{Role: "system", Content: "你是助手"}, {Role: "user", Content: "你好"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result.Text != "Anthropic 已完成" || result.InputTokens != 9 || result.OutputTokens != 4 {
		t.Fatalf("result = %#v", result)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
