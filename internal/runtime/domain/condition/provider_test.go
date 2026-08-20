package condition

import (
	"context"
	"errors"
	"testing"
)

func TestUserProvider_IsPremium(t *testing.T) {
	seen := ""
	p := &UserProvider{Lookup: func(_ context.Context, userID string) (*bool, error) {
		seen = userID
		trueVal := true
		return &trueVal, nil
	}}
	ctx := WithUserID(context.Background(), "u1")
	v, err := p.Value(ctx, "user.is_premium")
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if v != true || seen != "u1" {
		t.Fatalf("value=%v seen=%q", v, seen)
	}
}

func TestUserProvider_MissingAndError(t *testing.T) {
	p := &UserProvider{Lookup: func(_ context.Context, _ string) (*bool, error) { return nil, nil }}
	if _, err := p.Value(context.Background(), "user.is_premium"); !errors.Is(err, ErrAttributeMissing) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := p.Value(WithUserID(context.Background(), "u2"), "user.is_premium"); !errors.Is(err, ErrAttributeMissing) {
		t.Fatalf("nil bool: %v", err)
	}
	boom := errors.New("db down")
	p2 := &UserProvider{Lookup: func(context.Context, string) (*bool, error) { return nil, boom }}
	if _, err := p2.Value(WithUserID(context.Background(), "u3"), "user.is_premium"); !errors.Is(err, boom) {
		t.Fatalf("expected propagated error, got %v", err)
	}
	if _, err := p.Value(WithUserID(context.Background(), "u4"), "user.level"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("unknown field: %v", err)
	}
}

func TestCaseProvider_CategoryAndTags(t *testing.T) {
	p := &CaseProvider{Lookup: func(_ context.Context, _ string) (string, []string, error) {
		return "image", []string{"fast"}, nil
	}}
	ctx := WithCaseID(context.Background(), "c1")
	cat, err := p.Value(ctx, "case.category")
	if err != nil || cat != "image" {
		t.Fatalf("category: %v %v", cat, err)
	}
	tags, err := p.Value(ctx, "case.tags")
	if err != nil || len(tags.([]string)) != 1 || tags.([]string)[0] != "fast" {
		t.Fatalf("tags: %v %v", tags, err)
	}

	empty := &CaseProvider{Lookup: func(context.Context, string) (string, []string, error) {
		return "", nil, nil
	}}
	if _, err := empty.Value(WithCaseID(context.Background(), "c2"), "case.category"); !errors.Is(err, ErrAttributeMissing) {
		t.Fatalf("empty category should be missing: %v", err)
	}
	if _, err := empty.Value(context.Background(), "case.tags"); !errors.Is(err, ErrAttributeMissing) {
		t.Fatalf("missing case id should be missing: %v", err)
	}
}

func TestRegistry_AttributesAndProviderFor(t *testing.T) {
	reg := NewRegistry()
	userP := &UserProvider{Lookup: func(context.Context, string) (*bool, error) { return nil, nil }}
	caseP := &CaseProvider{Lookup: func(context.Context, string) (string, []string, error) { return "", nil, nil }}
	reg.Register(userP)
	reg.Register(caseP)

	attrs := reg.Attributes()
	if len(attrs) != 3 {
		t.Fatalf("attributes = %d", len(attrs))
	}
	if attrs[0].Key != "user.is_premium" || attrs[1].Key != "case.category" || attrs[2].Key != "case.tags" {
		t.Fatalf("attribute order: %+v", attrs)
	}
	if _, err := reg.ProviderFor("user.is_premium"); err != nil {
		t.Fatalf("provider for: %v", err)
	}
	if _, err := reg.ProviderFor("user.level"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := reg.ProviderFor("nonsense"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("no namespace: %v", err)
	}
}
