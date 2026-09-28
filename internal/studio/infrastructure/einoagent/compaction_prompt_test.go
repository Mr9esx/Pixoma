package einoagent

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/studio/application/contextcompaction"
)

func TestContextCheckpointIsUserMessageAndKeepsBoundary(t *testing.T) {
	checkpoint := contextCheckpointMessage("已确认采用英文", "zh")
	if checkpoint.Role != schema.User || !isContextCheckpoint(checkpoint) || !strings.Contains(checkpoint.Content, "<session_context_summary>") {
		t.Fatalf("checkpoint = %#v", checkpoint)
	}
	historical := withBoundary(schema.UserMessage("继续"), "message-1")
	if messageBoundary(historical, "fallback") != "message-1" {
		t.Fatalf("boundary = %#v", historical.Extra)
	}
}

func TestCompactForRetryPreservesContextCheckpoint(t *testing.T) {
	checkpoint := contextCheckpointMessage("用户目标", "zh")
	input := []*schema.Message{
		schema.SystemMessage("固定提示词"),
		checkpoint,
		schema.UserMessage("旧问题"),
		schema.AssistantMessage(strings.Repeat("旧回答", 200), nil),
		schema.UserMessage("新问题"),
	}
	output, ok := compactForRetry(context.Background(), input, contextcompaction.Budget{ContextWindowTokens: 1500, MaxInputTokens: 1400, MaxOutputTokens: 100})
	if !ok || len(output) < 3 || !isContextCheckpoint(output[1]) || output[len(output)-1].Content != "新问题" {
		t.Fatalf("retry context = %#v, compacted = %v", output, ok)
	}
}
