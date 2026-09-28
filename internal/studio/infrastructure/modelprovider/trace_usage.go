package modelprovider

import (
	"encoding/json"
	"fmt"
)

type normalizedTraceUsage struct {
	reported   bool
	input      int
	output     int
	cacheRead  *int
	cacheWrite *int
	reasoning  *int
}

func traceUsage(body []byte) (normalizedTraceUsage, error) {
	if len(body) == 0 {
		return normalizedTraceUsage{}, nil
	}
	var response struct {
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return normalizedTraceUsage{}, fmt.Errorf("model provider: decode usage: %w", err)
	}
	if len(response.Usage) == 0 || string(response.Usage) == "null" {
		return normalizedTraceUsage{}, nil
	}
	var value struct {
		PromptTokens             *int `json:"prompt_tokens"`
		CompletionTokens         *int `json:"completion_tokens"`
		InputTokens              *int `json:"input_tokens"`
		OutputTokens             *int `json:"output_tokens"`
		PromptCacheHitTokens     *int `json:"prompt_cache_hit_tokens"`
		PromptCacheMissTokens    *int `json:"prompt_cache_miss_tokens"`
		CacheReadInputTokens     *int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens *int `json:"cache_creation_input_tokens"`
		PromptTokensDetails      struct {
			CachedTokens *int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		InputTokensDetails struct {
			CachedTokens *int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens *int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
		OutputTokensDetails struct {
			ReasoningTokens *int `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	}
	if err := json.Unmarshal(response.Usage, &value); err != nil {
		return normalizedTraceUsage{}, fmt.Errorf("model provider: decode usage fields: %w", err)
	}
	usage := normalizedTraceUsage{reported: true}
	switch {
	case value.PromptTokens != nil && value.CompletionTokens != nil:
		usage.input, usage.output = *value.PromptTokens, *value.CompletionTokens
		usage.cacheRead = value.PromptTokensDetails.CachedTokens
		if value.PromptCacheHitTokens != nil {
			usage.cacheRead = value.PromptCacheHitTokens
		}
		usage.reasoning = value.CompletionTokensDetails.ReasoningTokens
	case value.InputTokens != nil && value.OutputTokens != nil:
		usage.input, usage.output = *value.InputTokens, *value.OutputTokens
		usage.cacheRead = value.InputTokensDetails.CachedTokens
		usage.reasoning = value.OutputTokensDetails.ReasoningTokens
		if value.CacheReadInputTokens != nil || value.CacheCreationInputTokens != nil {
			if value.CacheReadInputTokens != nil {
				usage.input += *value.CacheReadInputTokens
				usage.cacheRead = value.CacheReadInputTokens
			}
			if value.CacheCreationInputTokens != nil {
				usage.input += *value.CacheCreationInputTokens
				usage.cacheWrite = value.CacheCreationInputTokens
			}
		}
	default:
		return normalizedTraceUsage{}, fmt.Errorf("model provider: incomplete usage fields")
	}
	if usage.input < 0 || usage.output < 0 || usage.cacheRead != nil && (*usage.cacheRead < 0 || *usage.cacheRead > usage.input) || usage.cacheWrite != nil && (*usage.cacheWrite < 0 || *usage.cacheWrite > usage.input) {
		return normalizedTraceUsage{}, fmt.Errorf("model provider: invalid usage fields")
	}
	if usage.cacheRead != nil && usage.cacheWrite != nil && *usage.cacheRead+*usage.cacheWrite > usage.input {
		return normalizedTraceUsage{}, fmt.Errorf("model provider: cache usage exceeds input tokens")
	}
	return usage, nil
}
