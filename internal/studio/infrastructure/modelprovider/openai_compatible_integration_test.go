//go:build integration

package modelprovider_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
)

// Run with PIXOMA_STUDIO_MODEL_URL, PIXOMA_STUDIO_MODEL_ID and
// PIXOMA_STUDIO_MODEL_API_KEY set in the environment. This is intentionally
// excluded from ordinary test runs because it makes billable network calls.
func TestOpenAICompatibleLiveAgentModel(t *testing.T) {
	endpoint := os.Getenv("PIXOMA_STUDIO_MODEL_URL")
	modelID := os.Getenv("PIXOMA_STUDIO_MODEL_ID")
	apiKey := os.Getenv("PIXOMA_STUDIO_MODEL_API_KEY")
	if endpoint == "" || modelID == "" || apiKey == "" {
		t.Skip("set PIXOMA_STUDIO_MODEL_URL, PIXOMA_STUDIO_MODEL_ID and PIXOMA_STUDIO_MODEL_API_KEY to run the live provider test")
	}
	config := domain.ResolvedModelConfig{
		Protocol:     domain.ModelProtocolOpenAIChat,
		BaseURL:      endpoint,
		Model:        modelID,
		APIKey:       apiKey,
		Capabilities: domain.ModelCapabilities{Tools: true, Streaming: true},
	}
	client := modelprovider.NewOpenAICompatibleClient(&http.Client{Timeout: 45 * time.Second})

	t.Run("native tool calling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		err := (modelprovider.ConnectionTester{Client: client}).Test(ctx, config)
		require.NoError(t, err)
	})

	t.Run("streaming text through Eino adapter", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		chat := modelprovider.NewEinoChatModel(client, config)
		stream, err := chat.Stream(ctx, []*schema.Message{schema.UserMessage("请简短回复：Pixoma 已连接")})
		require.NoError(t, err)
		defer stream.Close()
		var answer strings.Builder
		for {
			message, receiveErr := stream.Recv()
			if receiveErr == io.EOF {
				break
			}
			require.NoError(t, receiveErr)
			if message != nil {
				answer.WriteString(message.Content)
			}
		}
		require.NotEmpty(t, strings.TrimSpace(answer.String()))
	})
}
