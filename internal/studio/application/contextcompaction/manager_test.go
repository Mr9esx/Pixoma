package contextcompaction

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestBudgetReservesOutputSystemToolsAndSafetyMargin(t *testing.T) {
	budget := Budget{
		ContextWindowTokens: 1000,
		MaxInputTokens:      900,
		MaxOutputTokens:     200,
		ReservedTokens:      100,
	}
	if got := budget.ConversationTokens(); got != 630 {
		t.Fatalf("ConversationTokens() = %d, want 630", got)
	}
}

func TestManagePreemptivelySummarizesAtConfiguredRatioAndKeepsCurrentMessage(t *testing.T) {
	messages := []*schema.Message{
		schema.UserMessage(strings.Repeat("早期需求 ", 40)),
		schema.AssistantMessage(strings.Repeat("早期结果 ", 40), nil),
		schema.UserMessage("当前用户消息必须保留"),
	}
	called := false
	result, err := Manage(context.Background(), messages, Options{
		Budget:       Budget{ContextWindowTokens: 220, MaxInputTokens: 220, MaxOutputTokens: 40},
		RecentRounds: 1,
		TriggerRatio: 0.8,
		Summarize: func(_ context.Context, early []*schema.Message) (string, error) {
			called = true
			if len(early) == 0 {
				t.Fatal("summary received no early messages")
			}
			return "保留的历史摘要", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called || !result.AutoCompactApplied || result.Summary != "保留的历史摘要" {
		t.Fatalf("summary result = %#v, called=%v", result, called)
	}
	if result.RetainedFrom != 2 || len(result.Messages) != 1 || result.Messages[0].Content != "当前用户消息必须保留" {
		t.Fatalf("retained messages = %#v, retainedFrom=%d", result.Messages, result.RetainedFrom)
	}
}

func TestManageClearsOldReadToolResultsBeforeDroppingConversation(t *testing.T) {
	messages := []*schema.Message{
		schema.UserMessage("读取文件"),
		{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "read_asset", Arguments: `{"id":"a"}`}}}},
		{Role: schema.Tool, Name: "read_asset", ToolCallID: "call-1", Content: strings.Repeat("asset ", 200)},
		schema.UserMessage("继续"),
	}
	result, err := Manage(context.Background(), messages, Options{
		Budget:       Budget{ContextWindowTokens: 180, MaxInputTokens: 180, MaxOutputTokens: 40},
		RecentRounds: 2,
		TriggerRatio: 0.8,
		ClearToolResult: func(message *schema.Message) bool {
			return message.Name == "read_asset"
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.MicroCompactApplied {
		t.Fatalf("expected micro compaction, got %#v", result)
	}
	if !strings.Contains(result.Messages[2].Content, "工具结果已清除") {
		t.Fatalf("old tool result was not cleared: %#v", result.Messages[2])
	}
}

func TestManageNeverLeavesOrphanToolMessageAfterHardTruncation(t *testing.T) {
	messages := []*schema.Message{
		schema.UserMessage("旧消息"),
		schema.AssistantMessage(strings.Repeat("旧结果 ", 100), nil),
		schema.UserMessage("当前消息"),
		{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "read_asset"}}}},
		{Role: schema.Tool, Name: "read_asset", ToolCallID: "call-1", Content: strings.Repeat("结果 ", 100)},
	}
	result, err := Manage(context.Background(), messages, Options{
		Budget:       Budget{ContextWindowTokens: 80, MaxInputTokens: 80, MaxOutputTokens: 20},
		RecentRounds: 1,
		TriggerRatio: 0.8,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range result.Messages {
		if message.Role == schema.Tool {
			t.Fatalf("orphan tool message survived: %#v", result.Messages)
		}
	}
	if result.Messages[len(result.Messages)-1].Content != "结果" && result.Messages[len(result.Messages)-1].Content != "当前消息" {
		t.Fatalf("latest message was lost: %#v", result.Messages)
	}
}
