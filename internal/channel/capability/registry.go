package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Registry stores registered capabilities by id.
type Registry struct {
	mu   sync.RWMutex
	caps map[string]Capability
	comp map[string]*jsonschema.Schema
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{caps: map[string]Capability{}, comp: map[string]*jsonschema.Schema{}}
}

// Register adds a capability; duplicate ids are rejected.
func (r *Registry) Register(c Capability) error {
	if c == nil || c.ID() == "" {
		return fmt.Errorf("capability: id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	compiler := jsonschema.NewCompiler()
	var doc any
	if err := json.Unmarshal(c.ParamsSchema(), &doc); err != nil {
		return fmt.Errorf("capability %q: parse schema: %w", c.ID(), err)
	}
	if err := compiler.AddResource("capability://"+c.ID(), doc); err != nil {
		return fmt.Errorf("capability %q: add schema: %w", c.ID(), err)
	}
	schema, err := compiler.Compile("capability://" + c.ID())
	if err != nil {
		return fmt.Errorf("capability %q: compile schema: %w", c.ID(), err)
	}
	if _, ok := r.caps[c.ID()]; ok {
		return fmt.Errorf("capability: duplicate id %q", c.ID())
	}
	r.caps[c.ID()] = c
	r.comp[c.ID()] = schema
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
	r.mu.RLock()
	c, ok := r.caps[inv.CapabilityID]
	schema := r.comp[inv.CapabilityID]
	r.mu.RUnlock()
	if !ok {
		return protocol.Result{}, fmt.Errorf("capability: unknown capability %q", inv.CapabilityID)
	}
	if err := schema.Validate(inv.Params); err != nil {
		return protocol.Result{}, fmt.Errorf("capability %q: invalid params: %w", inv.CapabilityID, err)
	}
	return c.Invoke(ctx, inv.Account, inv.Nav, sharedkernel.ChatID(inv.ChatID), inv.Params)
}
