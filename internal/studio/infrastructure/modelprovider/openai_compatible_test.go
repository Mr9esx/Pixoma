package modelprovider_test

import (
	"context"
	"encoding/json"
	"io"
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
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL + "/v1/chat/completions", Model: "deepseek-v4-flash", APIKey: "test-secret"},
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

func TestOpenAICompatibleChatEmitsDurableRequestTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", "provider-request-1")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"已完成"}}],"usage":{"prompt_tokens":7,"completion_tokens":3},"api_key":"provider-secret","credentials":{"token":"another-secret"}}`))
	}))
	defer server.Close()
	traces := make([]modelprovider.TraceEvent, 0, 2)
	result, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL, Model: "trace-test", APIKey: "test-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "追踪请求"}},
		Trace: func(_ context.Context, event modelprovider.TraceEvent) error {
			traces = append(traces, event)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result.InputTokens != 7 || result.OutputTokens != 3 {
		t.Fatalf("result = %#v", result)
	}
	if len(traces) != 2 || traces[0].Phase != modelprovider.TraceRequestStarted || traces[1].Phase != modelprovider.TraceRequestFinished {
		t.Fatalf("traces = %#v", traces)
	}
	if traces[0].RequestBody == nil || strings.Contains(string(traces[0].RequestBody), "test-secret") {
		t.Fatalf("request trace = %#v", traces[0])
	}
	if traces[1].StatusCode != http.StatusOK || traces[1].ProviderRequestID != "provider-request-1" || traces[1].InputTokens != 7 || traces[1].OutputTokens != 3 || traces[1].Elapsed <= 0 || !traces[1].UsageReported || !strings.Contains(string(traces[1].ResponseBody), "已完成") {
		t.Fatalf("finish trace = %#v", traces[1])
	}
	if strings.Contains(string(traces[1].ResponseBody), "provider-secret") || strings.Contains(string(traces[1].ResponseBody), "another-secret") {
		t.Fatalf("response trace leaked a credential: %s", traces[1].ResponseBody)
	}
}

func TestOpenAICompatibleStreamEmitsFirstTokenAndFinishTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-ID", "stream-request-1")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"第一个字\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":8,\"completion_tokens\":2}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	traces := make([]modelprovider.TraceEvent, 0, 3)
	stream, err := modelprovider.NewOpenAICompatibleClient(server.Client()).StreamChat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL, Model: "trace-stream", APIKey: "test-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "流式追踪"}},
		Trace: func(_ context.Context, event modelprovider.TraceEvent) error {
			traces = append(traces, event)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(traces) != 3 || traces[0].Phase != modelprovider.TraceRequestStarted || traces[1].Phase != modelprovider.TraceFirstToken || traces[2].Phase != modelprovider.TraceRequestFinished {
		t.Fatalf("traces = %#v", traces)
	}
	if traces[1].Elapsed <= 0 || traces[2].ProviderRequestID != "stream-request-1" || traces[2].InputTokens != 8 || traces[2].OutputTokens != 2 || !traces[2].UsageReported || !strings.Contains(string(traces[2].ResponseBody), "第一个字") {
		t.Fatalf("stream trace = %#v", traces)
	}
}

func TestOpenAICompatibleTraceDoesNotInventProviderUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer server.Close()
	var traces []modelprovider.TraceEvent
	_, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL, Model: "test", APIKey: "secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "hi"}},
		Trace: func(_ context.Context, event modelprovider.TraceEvent) error {
			traces = append(traces, event)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 2 || traces[1].UsageReported {
		t.Fatalf("usage should be unknown: %#v", traces)
	}
}

func TestOpenAICompatibleFailedTraceRedactsEchoedCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key secret-value"}}`))
	}))
	defer server.Close()
	var traces []modelprovider.TraceEvent
	_, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL, Model: "test", APIKey: "secret-value"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "hi"}},
		Trace: func(_ context.Context, event modelprovider.TraceEvent) error {
			traces = append(traces, event)
			return nil
		},
	})
	if err == nil || strings.Contains(err.Error(), "secret-value") || len(traces) != 2 || traces[1].Phase != modelprovider.TraceRequestFailed || strings.Contains(traces[1].Error, "secret-value") || strings.Contains(string(traces[1].ResponseBody), "secret-value") {
		t.Fatalf("failed trace leaked credential: %#v", traces)
	}
}

func TestOpenAICompatibleChatEmitsFailedTraceForInvalidProviderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":`))
	}))
	defer server.Close()
	traces := make([]modelprovider.TraceEvent, 0, 2)
	_, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{BaseURL: server.URL, Model: "trace-invalid", APIKey: "test-secret"},
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "坏响应"}},
		Trace: func(_ context.Context, event modelprovider.TraceEvent) error {
			traces = append(traces, event)
			return nil
		},
	})
	if err == nil {
		t.Fatal("Chat() error = nil")
	}
	if len(traces) != 2 || traces[0].Phase != modelprovider.TraceRequestStarted || traces[1].Phase != modelprovider.TraceRequestFailed || !strings.Contains(traces[1].Error, "decode response") || len(traces[1].ResponseBody) == 0 {
		t.Fatalf("traces = %#v", traces)
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
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: server.URL + "/v1/responses", Model: "gpt-test", APIKey: "test-secret", Thinking: domain.ThinkingConfig{Enabled: true, Effort: "high"}},
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
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: server.URL + "/api/v3/responses", Model: "deepseek-v4-1-flash-260910", APIKey: "test-secret"},
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
		if store, ok := body["store"].(bool); !ok || store {
			t.Fatalf("store = %#v, want false", body["store"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"function_call","call_id":"call_outline_01","name":"create_outline","arguments":"{\"genre\":\"noir\"}"}]}`))
	}))
	defer server.Close()

	result, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolOpenAIResponses, BaseURL: server.URL + "/v1/responses", Model: "gpt-test", APIKey: "test-secret"},
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
		Config:   domain.ResolvedModelConfig{Protocol: domain.ModelProtocolAnthropic, BaseURL: server.URL + "/v1/messages", Model: "claude-test", APIKey: "anthropic-secret", Thinking: domain.ThinkingConfig{Enabled: true, BudgetTokens: 2048}},
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

func TestConfiguredMaxOutputTokensAreSentToEachProtocol(t *testing.T) {
	protocols := []struct {
		name     string
		protocol domain.ModelProtocol
		response string
		field    string
	}{
		{name: "openai chat", protocol: domain.ModelProtocolOpenAIChat, response: `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`, field: "max_tokens"},
		{name: "openai responses", protocol: domain.ModelProtocolOpenAIResponses, response: `{"output_text":"ok","output":[]}`, field: "max_output_tokens"},
		{name: "anthropic", protocol: domain.ModelProtocolAnthropic, response: `{"content":[{"type":"text","text":"ok"}]}`, field: "max_tokens"},
	}
	for _, tt := range protocols {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				requireNoError(t, json.NewDecoder(r.Body).Decode(&body))
				if got, ok := body[tt.field].(float64); !ok || got != 1234 {
					t.Fatalf("%s = %#v, want 1234", tt.field, body[tt.field])
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()
			_, err := modelprovider.NewOpenAICompatibleClient(server.Client()).Chat(context.Background(), modelprovider.ChatRequest{
				Config: domain.ResolvedModelConfig{
					Protocol: tt.protocol, BaseURL: server.URL, Model: "test", APIKey: "secret",
					Limits: domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1234},
				},
				Messages: []modelprovider.ChatMessage{{Role: "user", Content: "hello"}},
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
