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
