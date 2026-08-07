package static

import (
	"context"
	"fmt"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type Registry struct {
	items []instance.Instance
}

func New(items ...instance.Instance) *Registry {
	out := make([]instance.Instance, len(items))
	copy(out, items)
	return &Registry{items: out}
}

func (r *Registry) ListHealthy(_ context.Context, filter instance.CapabilityFilter) ([]instance.Instance, error) {
	if len(filter.AnyOf) == 0 {
		out := make([]instance.Instance, len(r.items))
		copy(out, r.items)
		return out, nil
	}
	need := map[string]struct{}{}
	for _, c := range filter.AnyOf {
		need[c] = struct{}{}
	}
	var out []instance.Instance
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

func (r *Registry) Get(_ context.Context, id sharedkernel.InstanceID) (*instance.Instance, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			cp := r.items[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("instance: %s not found", id)
}

var _ instance.Registry = (*Registry)(nil)
