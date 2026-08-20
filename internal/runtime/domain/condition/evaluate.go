package condition

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// Evaluate returns whether the rule holds for the current context.
// A missing attribute evaluates to false (except for the exists operator);
// provider errors are propagated, never swallowed.
func Evaluate(ctx context.Context, rule Rule, reg *Registry) (bool, error) {
	if len(rule.And) > 0 {
		for _, sub := range rule.And {
			ok, err := Evaluate(ctx, sub, reg)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
	if len(rule.Or) > 0 {
		for _, sub := range rule.Or {
			ok, err := Evaluate(ctx, sub, reg)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	provider, err := reg.ProviderFor(rule.Field)
	if err != nil {
		return false, err
	}
	got, err := provider.Value(ctx, rule.Field)
	if err != nil {
		if errors.Is(err, ErrAttributeMissing) {
			return false, nil
		}
		return false, fmt.Errorf("evaluate %s: %w", rule.Field, err)
	}
	return compare(rule.Op, rule.Value, got), nil
}

func compare(op string, want, got any) bool {
	switch op {
	case "exists":
		return true
	case "eq":
		return equalValues(want, got)
	case "ne":
		return !equalValues(want, got)
	case "in":
		list, ok := want.([]any)
		if !ok {
			return false
		}
		for _, item := range list {
			if equalValues(item, got) {
				return true
			}
		}
		return false
	case "gt", "gte", "lt", "lte":
		w, okW := toFloat(want)
		g, okG := toFloat(got)
		if !okW || !okG {
			return false
		}
		switch op {
		case "gt":
			return g > w
		case "gte":
			return g >= w
		case "lt":
			return g < w
		default:
			return g <= w
		}
	}
	return false
}

func equalValues(a, b any) bool {
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af == bf
	}
	return reflect.DeepEqual(a, b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}
