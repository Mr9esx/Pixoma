package capability

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
)

// Registry stores registered capabilities by id.
type Registry struct {
	mu   sync.RWMutex
	caps map[string]Capability
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{caps: map[string]Capability{}}
}

// Register adds a capability; duplicate ids are rejected.
func (r *Registry) Register(c Capability) error {
	if c == nil || c.ID() == "" {
		return fmt.Errorf("capability: id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.caps[c.ID()]; ok {
		return fmt.Errorf("capability: duplicate id %q", c.ID())
	}
	r.caps[c.ID()] = c
	return nil
}

// Get returns the capability by id.
func (r *Registry) Get(id string) (Capability, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.caps[id]
	return c, ok
}

// List returns all registered capabilities sorted by id.
func (r *Registry) List() []Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Capability, 0, len(r.caps))
	for _, c := range r.caps {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Invoke validates the capability exists and dispatches to it.
func (r *Registry) Invoke(ctx context.Context, inv protocol.CapabilityInvoke) (protocol.Result, error) {
	c, ok := r.Get(inv.CapabilityID)
	if !ok {
		return protocol.Result{}, fmt.Errorf("capability: unknown capability %q", inv.CapabilityID)
	}
	return c.Invoke(ctx, inv.Account, inv.Nav, inv.Params)
}
