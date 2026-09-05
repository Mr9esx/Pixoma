package condition

import (
	"context"
	"errors"
	"testing"
)

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
	caseP := &CaseProvider{Lookup: func(context.Context, string) (string, []string, error) { return "", nil, nil }}
	reg.Register(caseP)

	attrs := reg.Attributes()
	if len(attrs) != 2 {
		t.Fatalf("attributes = %d", len(attrs))
	}
	if attrs[0].Key != "case.category" || attrs[1].Key != "case.tags" {
		t.Fatalf("attribute order: %+v", attrs)
	}
	if _, err := reg.ProviderFor("user.level"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("unknown: %v", err)
	}
	if _, err := reg.ProviderFor("nonsense"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("no namespace: %v", err)
	}
}
