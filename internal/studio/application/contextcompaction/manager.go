// Package contextcompaction manages the messages sent to a Studio model.
// It intentionally has no database or provider dependencies so its loss
// behavior can be tested independently from the Agent runtime.
package contextcompaction

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/schema"
)

const (
	defaultTriggerRatio = 0.8
	defaultRecentRounds = 3
	clearedToolPrefix   = "[工具结果已清除以释放上下文空间]"
)

type Budget struct {
	ContextWindowTokens int
	MaxInputTokens      int
	MaxOutputTokens     int
	ReservedTokens      int
}

// ConversationTokens leaves output capacity, fixed prompt/tool capacity and a
// 10% estimation safety margin outside the conversation budget.
func (b Budget) ConversationTokens() int {
	inputLimit := b.ContextWindowTokens - b.MaxOutputTokens
	if b.MaxInputTokens > 0 && (inputLimit <= 0 || b.MaxInputTokens < inputLimit) {
		inputLimit = b.MaxInputTokens
	}
	if inputLimit <= 0 {
		return 0
	}
	available := inputLimit - maxInt(b.ReservedTokens, 0)
	if available <= 0 {
		return 0
	}
	return int(math.Floor(float64(available) * 0.9))
}

type Summarizer func(context.Context, []*schema.Message) (string, error)

type Options struct {
	Budget          Budget
	RecentRounds    int
	TriggerRatio    float64
	Force           bool
	Summarize       Summarizer
	ClearToolResult func(*schema.Message) bool
}

type Result struct {
	Messages            []*schema.Message
	Summary             string
	Compressed          bool
	HardTruncated       bool
	MicroCompactApplied bool
	AutoCompactApplied  bool
	ActiveLayer         string
	// RetainedFrom is the index in the input messages from which retained
	// messages originate. It is used by the caller to persist a summary
	// boundary without putting database identifiers into schema.Message.
	RetainedFrom int
}

func Manage(ctx context.Context, input []*schema.Message, options Options) (Result, error) {
	messages := cloneMessages(input)
	if len(messages) == 0 {
		return Result{Messages: messages}, nil
	}
	budget := options.Budget.ConversationTokens()
	if budget <= 0 {
		return Result{Messages: keepLatestUser(messages), Compressed: true, HardTruncated: true, ActiveLayer: "hard_truncation", RetainedFrom: latestUserIndex(messages)}, nil
	}
	ratio := options.TriggerRatio
	if ratio <= 0 || ratio > 1 {
		ratio = defaultTriggerRatio
	}
	messageTokens := EstimateTokens(messages)
	threshold := int(math.Floor(float64(budget) * ratio))
	shouldManage := options.Force || messageTokens > budget || (options.Summarize != nil && messageTokens >= threshold)
	if !shouldManage {
		return Result{Messages: messages}, nil
	}

	result := Result{Messages: messages}
	if options.ClearToolResult != nil {
		compacted, applied := microCompact(messages, options.ClearToolResult)
		if applied {
			result.Messages = compacted
			result.Compressed = true
			result.MicroCompactApplied = true
			result.ActiveLayer = "micro_compact"
			messageTokens = EstimateTokens(compacted)
		}
	}

	recentRounds := options.RecentRounds
	if recentRounds <= 0 {
		recentRounds = defaultRecentRounds
	}
	if options.Summarize != nil {
		if messageTokens <= budget && messageTokens < threshold {
			return result, nil
		}
		for _, rounds := range windowCandidates(recentRounds) {
			early, recent, split := splitByRounds(result.Messages, rounds)
			if len(early) == 0 {
				continue
			}
			summary, err := options.Summarize(ctx, early)
			if err != nil {
				return Result{}, fmt.Errorf("studio: summarize model context: %w", err)
			}
			if strings.TrimSpace(summary) == "" {
				return Result{}, fmt.Errorf("studio: model context summary is empty")
			}
			retained := sanitizeToolMessages(recent)
			if EstimateTokens(retained)+estimateStringTokens(summary)+8 > budget {
				continue
			}
			result.Summary = strings.TrimSpace(summary)
			result.Messages = retained
			result.Compressed = true
			result.AutoCompactApplied = true
			result.ActiveLayer = "auto_compact"
			result.RetainedFrom = split
			return result, nil
		}
		if messageTokens <= budget {
			return result, nil
		}
		return Result{}, fmt.Errorf("studio: conversation exceeds the model context after summarization")
	}

	for _, rounds := range windowCandidates(recentRounds) {
		_, recent, split := splitByRounds(result.Messages, rounds)
		if len(recent) == 0 {
			recent = result.Messages
			split = 0
		}
		if EstimateTokens(recent) <= budget {
			result.Messages = sanitizeToolMessages(recent)
			result.Compressed = true
			result.ActiveLayer = "sliding_window"
			result.RetainedFrom = split
			return result, nil
		}
	}

	truncated, split := hardTruncate(result.Messages, budget)
	result.Messages = sanitizeToolMessages(truncated)
	result.Compressed = true
	result.HardTruncated = true
	result.ActiveLayer = "hard_truncation"
	result.RetainedFrom = split
	return result, nil
}

func microCompact(messages []*schema.Message, clear func(*schema.Message) bool) ([]*schema.Message, bool) {
	latestUser := latestUserIndex(messages)
	seen := make(map[string]bool)
	clearAt := make(map[int]struct{})
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Role != schema.Tool || !clear(message) {
			continue
		}
		// Tool results before the current user turn are historical data. Keep
		// results produced after that turn because they may still be needed by
		// the model to finish the current tool loop. If there is no user turn,
		// keep the newest result for each tool as a conservative fallback.
		if latestUser >= 0 {
			if i < latestUser && strings.TrimSpace(message.Content) != "" {
				clearAt[i] = struct{}{}
			}
			continue
		}
		if seen[message.Name] {
			if strings.TrimSpace(message.Content) != "" {
				clearAt[i] = struct{}{}
			}
			continue
		}
		seen[message.Name] = true
	}
	if len(clearAt) == 0 {
		return messages, false
	}
	out := make([]*schema.Message, len(messages))
	for i, message := range messages {
		if _, ok := clearAt[i]; !ok {
			out[i] = message
			continue
		}
		copyMessage := *message
		copyMessage.Content = fmt.Sprintf("%s 工具: %s", clearedToolPrefix, message.Name)
		out[i] = &copyMessage
	}
	return out, true
}

func splitByRounds(messages []*schema.Message, recentRounds int) (early, recent []*schema.Message, split int) {
	if recentRounds <= 0 {
		recentRounds = 1
	}
	count := 0
	split = len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.User {
			count++
			if count >= recentRounds {
				split = i
				break
			}
		}
	}
	if count < recentRounds {
		return nil, messages, 0
	}
	return messages[:split], messages[split:], split
}

func windowCandidates(recentRounds int) []int {
	seen := map[int]struct{}{}
	result := make([]int, 0, 3)
	for _, candidate := range []int{recentRounds, recentRounds / 2, 1} {
		if candidate < 1 {
			candidate = 1
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func hardTruncate(messages []*schema.Message, budget int) ([]*schema.Message, int) {
	start := latestUserIndex(messages)
	if start < 0 {
		start = maxInt(len(messages)-1, 0)
	}
	kept := cloneMessages(messages[start:])
	for len(kept) > 1 && EstimateTokens(kept) > budget {
		// The current user message is always kept. Drop the oldest complete
		// assistant/tool group after it when the current turn itself is large.
		if kept[1].Role == schema.Assistant && len(kept[1].ToolCalls) > 0 {
			end := 2
			for end < len(kept) && kept[end].Role == schema.Tool {
				end++
			}
			kept = append(kept[:1], kept[end:]...)
			continue
		}
		kept = append(kept[:1], kept[2:]...)
	}
	return kept, start
}

func latestUserIndex(messages []*schema.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.User {
			return i
		}
	}
	return -1
}

func keepLatestUser(messages []*schema.Message) []*schema.Message {
	index := latestUserIndex(messages)
	if index < 0 {
		return cloneMessages(messages[len(messages)-1:])
	}
	return cloneMessages(messages[index : index+1])
}

func sanitizeToolMessages(messages []*schema.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))
	for i, message := range messages {
		if message == nil || message.Role != schema.Tool {
			if message != nil {
				result = append(result, message)
			}
			continue
		}
		if i == 0 || !hasMatchingToolCall(messages, i, message) {
			continue
		}
		result = append(result, message)
	}
	return result
}

func hasMatchingToolCall(messages []*schema.Message, toolIndex int, toolMessage *schema.Message) bool {
	for i := toolIndex - 1; i >= 0; i-- {
		message := messages[i]
		if message == nil {
			continue
		}
		if message.Role == schema.User || message.Role == schema.System {
			return false
		}
		if message.Role != schema.Assistant {
			continue
		}
		for _, call := range message.ToolCalls {
			if call.ID == toolMessage.ToolCallID {
				return true
			}
		}
	}
	return false
}

func cloneMessages(messages []*schema.Message) []*schema.Message {
	result := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		result = append(result, copyMessage(message))
	}
	return result
}

func copyMessage(message *schema.Message) *schema.Message {
	copyMessage := *message
	if message.ToolCalls != nil {
		copyMessage.ToolCalls = append([]schema.ToolCall(nil), message.ToolCalls...)
	}
	return &copyMessage
}

func EstimateTokens(messages []*schema.Message) int {
	total := 0
	for _, message := range messages {
		if message == nil {
			continue
		}
		total += estimateStringTokens(message.Content) + 4
		for _, call := range message.ToolCalls {
			total += estimateStringTokens(call.Function.Name) + estimateStringTokens(call.Function.Arguments)
		}
	}
	return total
}

func estimateStringTokens(value string) int {
	if value == "" {
		return 0
	}
	cjk := 0
	other := 0
	for _, r := range value {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			cjk++
		} else {
			other++
		}
	}
	return int(math.Ceil(float64(cjk)*1.5 + float64(other)/4))
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
