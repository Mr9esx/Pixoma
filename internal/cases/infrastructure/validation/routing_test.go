package validation_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	domain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/cases/infrastructure/validation"
	topicdomain "github.com/Mr9esx/Pixoma/internal/topics/domain"
	"github.com/Mr9esx/Pixoma/internal/tasks/domain/condition"
)

type fakeTopicRepo struct {
	topics map[string]bool // key -> enabled
}

type valStubProvider struct {
	val func(ctx context.Context, field string) (any, error)
}

func (s *valStubProvider) Namespace() string { return "user" }
func (s *valStubProvider) ListAttributes() []condition.AttributeDescriptor {
	return []condition.AttributeDescriptor{{
		Key:     "user.level",
		Context: "user",
		Label:   "Level",
		Schema:  map[string]any{"type": "string"},
	}}
}
func (s *valStubProvider) Value(ctx context.Context, field string) (any, error) {
	if s.val == nil {
		return nil, condition.ErrAttributeMissing
	}
	return s.val(ctx, field)
}

func (f *fakeTopicRepo) List(_ context.Context, _ *bool) ([]topicdomain.Topic, error) { return nil, nil }

func (f *fakeTopicRepo) Get(_ context.Context, key string) (*topicdomain.Topic, error) {
	enabled, ok := f.topics[key]
	if !ok {
		return nil, topicdomain.ErrTopicNotFound
	}
	return &topicdomain.Topic{Key: key, Enabled: enabled}, nil
}

func (f *fakeTopicRepo) Create(_ context.Context, _ topicdomain.Topic) error { return nil }
func (f *fakeTopicRepo) Update(_ context.Context, _ topicdomain.Topic) error { return nil }
func (f *fakeTopicRepo) Delete(_ context.Context, _ string) error      { return nil }

func TestValidateRouting(t *testing.T) {
	topics := &fakeTopicRepo{topics: map[string]bool{"fast-gpu": true, "disabled": false}}
	reg := condition.NewRegistry()
	reg.Register(&valStubProvider{})

	ok := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "fast-gpu"},
	}}
	if err := validation.ValidateRouting(context.Background(), ok, topics, reg); err != nil {
		t.Fatalf("valid routing rejected: %v", err)
	}

	always := &domain.RoutingConfig{Rules: []domain.RoutingRule{
		{When: json.RawMessage(`{"always":true}`), Topic: "fast-gpu"},
	}}
	if err := validation.ValidateRouting(context.Background(), always, topics, reg); err != nil {
		t.Fatalf("always routing rejected: %v", err)
	}

	empty := []struct {
		name string
		cfg  *domain.RoutingConfig
	}{
		{"nil routing", nil},
		{"empty routing", &domain.RoutingConfig{}},
	}
	for _, tc := range empty {
		if err := validation.ValidateRouting(context.Background(), tc.cfg, topics, reg); err == nil {
			t.Fatalf("%s: expected validation error", tc.name)
		}
	}

	bad := []struct {
		name string
		cfg  *domain.RoutingConfig
		want string
	}{
		{"unknown topic", &domain.RoutingConfig{Rules: []domain.RoutingRule{
			{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "nope"},
		}}, "nope"},
		{"disabled topic", &domain.RoutingConfig{Rules: []domain.RoutingRule{
			{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`), Topic: "disabled"},
		}}, "disabled"},
		{"unknown condition field", &domain.RoutingConfig{Rules: []domain.RoutingRule{
			{When: json.RawMessage(`{"field":"user.unknown","op":"eq","value":1}`), Topic: "fast-gpu"},
		}}, "user.unknown"},
		{"missing topic", &domain.RoutingConfig{Rules: []domain.RoutingRule{
			{When: json.RawMessage(`{"field":"user.level","op":"eq","value":"image"}`)},
		}}, "topic required"},
	}
	for _, tc := range bad {
		err := validation.ValidateRouting(context.Background(), tc.cfg, topics, reg)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: err=%v want contains %q", tc.name, err, tc.want)
		}
	}
}
