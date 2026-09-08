package condition

import (
	"errors"
	"testing"
)

func TestRegistry_AttributesAndProviderFor(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubProvider{ns: "user", attrs: []AttributeDescriptor{
		{Key: "user.level", Context: "user", Label: "Level", Schema: map[string]any{"type": "number"}},
	}})

	attrs := reg.Attributes()
	if len(attrs) != 1 {
		t.Fatalf("attributes = %d", len(attrs))
	}
	if attrs[0].Key != "user.level" {
		t.Fatalf("attribute order: %+v", attrs)
	}
	if _, err := reg.ProviderFor("user.level"); err != nil {
		t.Fatalf("provider for existing field: %v", err)
	}
	if _, err := reg.ProviderFor("user.unknown"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("unknown field: %v", err)
	}
	if _, err := reg.ProviderFor("nonsense"); !errors.Is(err, ErrUnknownField) {
		t.Fatalf("no namespace: %v", err)
	}
}
