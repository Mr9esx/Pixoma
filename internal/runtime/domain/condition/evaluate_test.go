package condition

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestEvaluate_AndOrShortCircuit(t *testing.T) {
	calls := 0
	p := &countingProvider{val: func(_ context.Context, field string) (any, error) {
		calls++
		return "x", nil
	}}
	reg := NewRegistry()
	reg.Register(p)
	ctx := context.Background()

	andRule := Rule{And: []Rule{
		{Field: "count.f", Op: "eq", Value: "no-match"},
		{Field: "count.g", Op: "eq", Value: "x"},
	}}
	ok, err := Evaluate(ctx, andRule, reg)
	if err != nil || ok {
		t.Fatalf("and short circuit: ok=%v err=%v calls=%d", ok, err, calls)
	}
	if calls != 1 {
		t.Fatalf("expected 1 provider call, got %d", calls)
	}

	calls = 0
	orRule := Rule{Or: []Rule{
		{Field: "count.f", Op: "eq", Value: "x"},
		{Field: "count.g", Op: "eq", Value: "x"},
	}}
	ok, err = Evaluate(ctx, orRule, reg)
	if err != nil || !ok {
		t.Fatalf("or short circuit: ok=%v err=%v", ok, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 provider call for or, got %d", calls)
	}
}

func TestEvaluate_MissingAttribute(t *testing.T) {
	p := &countingProvider{val: func(context.Context, string) (any, error) { return nil, ErrAttributeMissing }}
	reg := NewRegistry()
	reg.Register(p)
	ctx := context.Background()

	ok, err := Evaluate(ctx, Rule{Field: "count.f", Op: "eq", Value: true}, reg)
	if err != nil || ok {
		t.Fatalf("missing eq: ok=%v err=%v", ok, err)
	}
	ok, err = Evaluate(ctx, Rule{Field: "count.f", Op: "exists"}, reg)
	if err != nil || ok {
		t.Fatalf("missing exists should be false: ok=%v err=%v", ok, err)
	}
}

func TestEvaluate_ProviderErrorPropagates(t *testing.T) {
	boom := errors.New("provider boom")
	p := &countingProvider{val: func(context.Context, string) (any, error) { return nil, boom }}
	reg := NewRegistry()
	reg.Register(p)
	_, err := Evaluate(context.Background(), Rule{Field: "count.f", Op: "eq", Value: true}, reg)
	if !errors.Is(err, boom) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestEvaluate_Always(t *testing.T) {
	got, err := Evaluate(context.Background(), Rule{Always: true}, NewRegistry())
	if err != nil {
		t.Fatalf("evaluate always: %v", err)
	}
	if !got {
		t.Fatal("always must evaluate true")
	}
}

func TestEvaluate_Comparisons(t *testing.T) {
	cases := []struct {
		rule Rule
		got  any
		want bool
	}{
		{Rule{Field: "count.f", Op: "eq", Value: true}, true, true},
		{Rule{Field: "count.f", Op: "eq", Value: true}, false, false},
		{Rule{Field: "count.f", Op: "ne", Value: 1}, 2, true},
		{Rule{Field: "count.f", Op: "gt", Value: 5}, 7, true},
		{Rule{Field: "count.f", Op: "gt", Value: 9}, 7, false},
		{Rule{Field: "count.f", Op: "gte", Value: 7}, 7, true},
		{Rule{Field: "count.f", Op: "lt", Value: 7}, 5, true},
		{Rule{Field: "count.f", Op: "lte", Value: 7}, 7, true},
		{Rule{Field: "count.f", Op: "in", Value: []any{"a", "b"}}, "b", true},
		{Rule{Field: "count.f", Op: "in", Value: []any{"a", "b"}}, "z", false},
		{Rule{Field: "count.f", Op: "exists"}, "anything", true},
	}
	for _, tc := range cases {
		p := &countingProvider{val: func(context.Context, string) (any, error) { return tc.got, nil }}
		reg := NewRegistry()
		reg.Register(p)
		got, err := Evaluate(context.Background(), tc.rule, reg)
		if err != nil {
			t.Fatalf("%+v: %v", tc.rule, err)
		}
		if got != tc.want {
			t.Fatalf("rule %+v with got=%v: want %v", tc.rule, tc.got, tc.want)
		}
	}
}

func TestEvaluate_NewProviderNoEngineChange(t *testing.T) {
	// A newly registered attribute must work through the existing engine.
	reg := NewRegistry()
	reg.Register(&stubProvider{ns: "user", attrs: []AttributeDescriptor{
		{Key: "user.region", Context: "user", Label: "Region", Schema: map[string]any{"type": "string", "enum": []any{"cn", "us"}}},
	}, val: func(_ context.Context, _ string) (any, error) { return "cn", nil }})

	raw := `{"field":"user.region","op":"eq","value":"cn"}`
	r, err := ParseRule([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := ValidateRule(r, reg); err != nil {
		t.Fatalf("validate new attribute: %v", err)
	}
	ok, err := Evaluate(context.Background(), r, reg)
	if err != nil || !ok {
		t.Fatalf("evaluate new attribute: ok=%v err=%v", ok, err)
	}
}

type countingProvider struct {
	val func(ctx context.Context, field string) (any, error)
}

func (c *countingProvider) Namespace() string { return "count" }

func (c *countingProvider) ListAttributes() []AttributeDescriptor {
	return []AttributeDescriptor{
		{Key: "count.f", Context: "user", Label: "F", Schema: map[string]any{"type": "string"}},
		{Key: "count.g", Context: "user", Label: "G", Schema: map[string]any{"type": "string"}},
	}
}

func (c *countingProvider) Value(ctx context.Context, field string) (any, error) {
	if c.val == nil {
		return nil, fmt.Errorf("no val")
	}
	return c.val(ctx, field)
}
