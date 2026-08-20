package condition

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrAttributeMissing is returned by providers when the attribute has no value
// in the evaluation context (distinct from a provider error).
var ErrAttributeMissing = errors.New("condition: attribute missing")

// ErrUnknownField is returned when a rule references an unregistered field.
var ErrUnknownField = errors.New("condition: unknown field")

// AttributeDescriptor describes one evaluable attribute and its JSON Schema.
type AttributeDescriptor struct {
	Key     string         // e.g. "user.is_premium"
	Context string         // "user" | "case" | "input"
	Label   string         // UI label
	Schema  map[string]any // JSON Schema subset: type/enum/description
}

// Provider resolves attribute values for one namespace.
type Provider interface {
	Namespace() string
	ListAttributes() []AttributeDescriptor
	Value(ctx context.Context, field string) (any, error)
}

// Registry resolves providers by namespace and exposes the attribute catalog.
type Registry struct {
	providers map[string]Provider
	order     []string
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

// Register adds a provider under its namespace.
func (r *Registry) Register(p Provider) {
	if r.providers == nil {
		r.providers = map[string]Provider{}
	}
	ns := p.Namespace()
	if _, ok := r.providers[ns]; !ok {
		r.order = append(r.order, ns)
	}
	r.providers[ns] = p
}

// ProviderFor resolves the provider owning field (namespace.attr).
func (r *Registry) ProviderFor(field string) (Provider, error) {
	ns, _, ok := strings.Cut(field, ".")
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownField, field)
	}
	p, ok := r.providers[ns]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownField, field)
	}
	for _, d := range p.ListAttributes() {
		if d.Key == field {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownField, field)
}

// Attributes returns the full attribute catalog in registration order.
func (r *Registry) Attributes() []AttributeDescriptor {
	out := make([]AttributeDescriptor, 0)
	for _, ns := range r.order {
		out = append(out, r.providers[ns].ListAttributes()...)
	}
	return out
}
