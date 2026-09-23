package einoagent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type reasoningStreamSink struct {
	events []string
	text   string
	begin  int
}

func (s *reasoningStreamSink) Emit(_ context.Context, eventType string, _ any) error {
	s.events = append(s.events, eventType)
	return nil
}

func (*reasoningStreamSink) AssistantMessage(context.Context, string) (*domain.Message, error) {
	return nil, nil
}

func (*reasoningStreamSink) CreateAsset(context.Context, studioapp.GeneratedAsset) (*domain.Asset, error) {
	return nil, nil
}

func (*reasoningStreamSink) CreateFlowNode(context.Context, studioapp.FlowNodeInput) (*domain.FlowNode, error) {
	return nil, nil
}

func (*reasoningStreamSink) CreateFlowEdge(context.Context, string, string, string) (*domain.FlowEdge, error) {
	return nil, nil
}

func (*reasoningStreamSink) RequestApproval(context.Context, string, string) (*domain.Approval, error) {
	return nil, nil
}

func (s *reasoningStreamSink) BeginAssistantMessage(context.Context) (string, error) {
	s.begin++
	return "assistant-1", nil
}

func (s *reasoningStreamSink) AppendAssistantMessage(_ context.Context, _ string, delta string) error {
	s.text += delta
	return nil
}

func (s *reasoningStreamSink) EndAssistantMessage(_ context.Context, _ string, text string) (*domain.Message, error) {
	s.text = text
	return &domain.Message{ID: "assistant-1"}, nil
}

func TestConsumeAssistantStreamWaitsForTextAfterReasoning(t *testing.T) {
	reader, writer := schema.Pipe[*schema.Message](4)
	go func() {
		writer.Send(&schema.Message{Role: schema.Assistant, Extra: map[string]any{"pixoma.reasoning": "先分析"}}, nil)
		writer.Send(&schema.Message{Role: schema.Assistant, Extra: map[string]any{"pixoma.reasoning": "需求。"}}, nil)
		writer.Send(&schema.Message{Role: schema.Assistant, Content: "答案正文。"}, nil)
		writer.Close()
	}()

	sink := &reasoningStreamSink{}
	responded, err := consumeAssistantStream(context.Background(), sink, sink, reader)

	require.NoError(t, err)
	require.True(t, responded)
	require.Equal(t, 1, sink.begin)
	require.Equal(t, "答案正文。", sink.text)
	require.Equal(t, 1, countEvent(sink.events, studioapp.EventReasoningStart))
	require.Equal(t, 1, countEvent(sink.events, studioapp.EventReasoningMessageStart))
	require.Equal(t, 2, countEvent(sink.events, studioapp.EventReasoningMessageContent))
	require.Equal(t, 1, countEvent(sink.events, studioapp.EventReasoningMessageEnd))
	require.Equal(t, 1, countEvent(sink.events, studioapp.EventReasoningEnd))
}

func countEvent(events []string, target string) int {
	count := 0
	for _, event := range events {
		if event == target {
			count++
		}
	}
	return count
}
