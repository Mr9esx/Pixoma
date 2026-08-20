package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
)

// ValidateRouting validates the optional Case routing config: each rule must
// reference an existing enabled topic and a condition valid per the protocol.
func ValidateRouting(ctx context.Context, r *domain.RoutingConfig, topics topic.Repository, reg *condition.Registry) error {
	if r == nil || len(r.Rules) == 0 {
		return nil
	}
	if topics == nil || reg == nil {
		return fmt.Errorf("routing: topics registry or condition registry not configured")
	}
	for i, rule := range r.Rules {
		ruleTopic := strings.TrimSpace(rule.Topic)
		if ruleTopic == "" {
			return fmt.Errorf("routing.rules[%d]: topic required", i)
		}
		got, err := topics.Get(ctx, ruleTopic)
		if err != nil || got == nil || !got.Enabled {
			return fmt.Errorf("routing.rules[%d]: unknown or disabled topic %q", i, ruleTopic)
		}
		parsed, err := condition.ParseRule(rule.When)
		if err != nil {
			return fmt.Errorf("routing.rules[%d]: %w", i, err)
		}
		if err := condition.ValidateRule(parsed, reg); err != nil {
			return fmt.Errorf("routing.rules[%d]: %w", i, err)
		}
	}
	return nil
}
