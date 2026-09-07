package routing_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/routing"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain/condition"
)

type stubProvider struct {
	val func(ctx context.Context, field string) (any, error)
}

func (s *stubProvider) Namespace() string { return "user" }
func (s *stubProvider) ListAttributes() []condition.AttributeDescriptor {
	return []condition.AttributeDescriptor{{
		Key:     "user.level",
		Context: "user",
		Label:   "Level",
		Schema:  map[string]any{"type": "string", "enum": []any{"image", "video"}},
	}}
}
func (s *stubProvider) Value(ctx context.Context, field string) (any, error) {
	if s.val == nil {
		return nil, condition.ErrAttributeMissing
	}
	return s.val(ctx, field)
}

func TestResolve_FirstMatchWins(t *testing.T) {
	reg := condition.NewRegistry()
	reg.Register(&stubProvider{val: func(context.Context, string) (any, error) {
		return "image", nil
	}})

	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"video"}`), Topic: "slow-gpu"},
	}}
	got, err := routing.Resolve(cfg, context.Background(), reg)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "fast-gpu" {
		t.Fatalf("topic = %q, want fast-gpu", got)
	}
}

func TestResolve_AlwaysMatches(t *testing.T) {
	reg := condition.NewRegistry()
	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"always":true}`), Topic: "default"},
	}}
	got, err := routing.Resolve(cfg, context.Background(), reg)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "default" {
		t.Fatalf("topic = %q, want default", got)
	}
}

func TestResolve_NoMatchReturnsErrNoMatch(t *testing.T) {
	reg := condition.NewRegistry()
	reg.Register(&stubProvider{val: func(context.Context, string) (any, error) {
		return "", nil // but "" is not in enum; still a value
	}})

	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
	}}
	_, err := routing.Resolve(cfg, context.Background(), reg)
	// provided value "poster" != "image" so no match
	if !errors.Is(err, routing.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolve_EmptyRulesReturnsErrNoMatch(t *testing.T) {
	reg := condition.NewRegistry()
	_, err := routing.Resolve(&domain.RoutingConfig{}, context.Background(), reg)
	if !errors.Is(err, routing.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolve_NilRoutingReturnsErrNoMatch(t *testing.T) {
	reg := condition.NewRegistry()
	_, err := routing.Resolve(nil, context.Background(), reg)
	if !errors.Is(err, routing.ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

func TestResolve_ProviderErrorPropagates(t *testing.T) {
	boom := errors.New("provider boom")
	reg := condition.NewRegistry()
	reg.Register(&stubProvider{val: func(context.Context, string) (any, error) {
		return nil, boom
	}})
	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
	}}
	_, err := routing.Resolve(cfg, context.Background(), reg)
	if !errors.Is(err, boom) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestResolve_RuleOrderMatters(t *testing.T) {
	reg := condition.NewRegistry()
	reg.Register(&stubProvider{val: func(context.Context, string) (any, error) {
		return "image", nil
	}})
	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "first"},
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "second"},
	}}
	got, _ := routing.Resolve(cfg, context.Background(), reg)
	if got != "first" {
		t.Fatalf("topic = %q, want first", got)
	}
}
