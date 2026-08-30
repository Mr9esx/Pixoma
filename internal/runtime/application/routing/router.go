package routing

import (
	"context"
	"errors"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
)

// ErrNoMatch means routing is explicit but no rule selected a topic.
var ErrNoMatch = errors.New("no matching routing rule")

// Resolve picks the delivery topic for a task: rules are evaluated in order and
// the first match wins; no rules or no match returns ErrNoMatch.
// Provider errors are propagated so the caller can keep the task pending.
func Resolve(cfg *domain.RoutingConfig, ctx context.Context, reg *condition.Registry) (string, error) {
	if cfg == nil || len(cfg.Rules) == 0 {
		return "", ErrNoMatch
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
	return "", ErrNoMatch
}
