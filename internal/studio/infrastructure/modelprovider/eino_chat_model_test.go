package modelprovider_test

import (
	"context"
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
