package routing

import (
	"context"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
)

// Resolve picks the delivery topic for a task: rules are evaluated in order and
// the first match wins; no rules or no match falls back to the default topic.
// Provider errors are propagated so the caller can keep the task pending.
func Resolve(cfg *domain.RoutingConfig, ctx context.Context, reg *condition.Registry) (string, error) {
	if cfg == nil || len(cfg.Rules) == 0 {
		return topic.DefaultKey, nil
	}
	for i, rule := range cfg.Rules {
		parsed, err := condition.ParseRule(rule.When)
		if err != nil {
			return "", fmt.Errorf("routing rule %d: %w", i, err)
		}
		ok, err := condition.Evaluate(ctx, parsed, reg)
		if err != nil {
			return "", fmt.Errorf("routing rule %d: %w", i, err)
		}
		if ok {
			return rule.Topic, nil
		}
	}
	return topic.DefaultKey, nil
}
