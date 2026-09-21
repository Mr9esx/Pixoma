package einoagent_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/einoagent"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

type resolver struct {
	config *domain.ResolvedModelConfig
}

func (r resolver) Resolve(_ context.Context, _ string, _ string) (*domain.ResolvedModelConfig, error) {
	return r.config, nil
}

type sink struct {
	mu        sync.Mutex
	events    []string
	responses []string
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

func (*sink) CreateAsset(context.Context, studioapp.GeneratedAsset) (*domain.Asset, error) {
	return nil, nil
}
func (*sink) CreateFlowNode(context.Context, studioapp.FlowNodeInput) (*domain.FlowNode, error) {
	return nil, nil
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
