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

func TestResolve_FirstMatchWins(t *testing.T) {
	reg := condition.NewRegistry()
	reg.Register(&condition.CaseProvider{Lookup: func(context.Context, string) (string, []string, error) {
		return "image", nil, nil
	}})
	ctx := condition.WithCaseID(context.Background(), "c1")

	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"case.category","op":"eq","value":"image"}`), Topic: "gpu-a"},
		{When: json.RawMessage(`{"field":"case.category","op":"eq","value":"video"}`), Topic: "gpu-b"},
	}}
	got, err := routing.Resolve(cfg, ctx, reg)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "gpu-a" {
		t.Fatalf("topic = %q, want gpu-a", got)
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
	reg.Register(&condition.CaseProvider{Lookup: func(context.Context, string) (string, []string, error) {
		return "", nil, nil // missing
	}})
	ctx := condition.WithCaseID(context.Background(), "c1")

	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"case.category","op":"eq","value":"image"}`), Topic: "gpu-a"},
	}}
	_, err := routing.Resolve(cfg, ctx, reg)
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
	reg.Register(&condition.CaseProvider{Lookup: func(context.Context, string) (string, []string, error) {
		return "", nil, boom
	}})
	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"case.category","op":"eq","value":"image"}`), Topic: "gpu-a"},
	}}
	_, err := routing.Resolve(cfg, condition.WithCaseID(context.Background(), "c1"), reg)
	if !errors.Is(err, boom) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestResolve_RuleOrderMatters(t *testing.T) {
	reg := condition.NewRegistry()
	reg.Register(&condition.CaseProvider{Lookup: func(context.Context, string) (string, []string, error) {
		return "image", nil, nil
	}})
	cfg := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"case.category","op":"eq","value":"image"}`), Topic: "first"},
		{When: json.RawMessage(`{"field":"case.category","op":"eq","value":"image"}`), Topic: "second"},
	}}
	got, _ := routing.Resolve(cfg, condition.WithCaseID(context.Background(), "c1"), reg)
	if got != "first" {
		t.Fatalf("topic = %q, want first", got)
	}
}
