package static

import (
	"context"
	"fmt"

	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

type Registry struct {
	items []edge.Instance
}

func New(items ...edge.Instance) *Registry {
	out := make([]edge.Instance, len(items))
	copy(out, items)
	return &Registry{items: out}
}

func (r *Registry) ListHealthy(_ context.Context, filter edge.CapabilityFilter) ([]edge.Instance, error) {
	return r.list(filter)
}

func (r *Registry) ListEnabled(_ context.Context, filter edge.CapabilityFilter) ([]edge.Instance, error) {
	return r.list(filter)
}

func (r *Registry) list(filter edge.CapabilityFilter) ([]edge.Instance, error) {
	if len(filter.AnyOf) == 0 {
		out := make([]edge.Instance, len(r.items))
		copy(out, r.items)
		return out, nil
	}
	need := map[string]struct{}{}
	for _, c := range filter.AnyOf {
		need[c] = struct{}{}
	}
	var out []edge.Instance
	for _, it := range r.items {
		for _, c := range it.Capabilities {
			if _, ok := need[c]; ok {
				out = append(out, it)
				break
			}
		}
	}
	return out, nil
}

func (r *Registry) Get(_ context.Context, id sharedkernel.EdgeID) (*edge.Instance, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			cp := r.items[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("edge: %s not found", id)
}

var _ edge.Registry = (*Registry)(nil)
