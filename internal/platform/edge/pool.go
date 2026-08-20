package edge

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// PoolOptions configures Refresh behavior.
type PoolOptions struct {
	// AfterRefresh is invoked after a successful Refresh with the new List().
	AfterRefresh func(instances []Instance)
}

type pooled struct {
	record  Record
	healthy bool
}

// Pool keeps an in-process edge map keyed by instance id and implements
// Registry for scheduling.
type Pool struct {
	repo Repository
	opts PoolOptions

	mu           sync.RWMutex
	byID         map[sharedkernel.EdgeID]*pooled
	afterRefresh func(instances []Instance)
}

// NewPool constructs a Pool. Call Refresh after seeding the repository.
func NewPool(repo Repository, opts PoolOptions) *Pool {
	return &Pool{
		repo:         repo,
		opts:         opts,
		byID:         make(map[sharedkernel.EdgeID]*pooled),
		afterRefresh: opts.AfterRefresh,
	}
}

// SetAfterRefresh sets a callback invoked after each successful Refresh.
func (p *Pool) SetAfterRefresh(fn func(instances []Instance)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.afterRefresh = fn
}

// Refresh rebuilds the edge map from the repository.
// Enabled instances start healthy; existing health flags are preserved when possible.
func (p *Pool) Refresh(ctx context.Context) error {
	if p.repo == nil {
		return fmt.Errorf("edge: nil repository")
	}
	list, err := p.repo.List(ctx)
	if err != nil {
		return err
	}

	prevHealthy := map[sharedkernel.EdgeID]bool{}
	p.mu.RLock()
	for id, e := range p.byID {
		prevHealthy[id] = e.healthy
	}
	p.mu.RUnlock()

	next := make(map[sharedkernel.EdgeID]*pooled, len(list))
	for _, rec := range list {
		if rec == nil || !rec.Enabled {
			continue
		}
		healthy := true
		if was, ok := prevHealthy[rec.ID]; ok {
			healthy = was
		}
		cp := *rec
		if cp.Capabilities == nil {
			cp.Capabilities = []string{}
		} else {
			cp.Capabilities = append([]string(nil), cp.Capabilities...)
		}
		next[rec.ID] = &pooled{
			record:  cp,
			healthy: healthy,
		}
	}

	p.mu.Lock()
	p.byID = next
	cb := p.afterRefresh
	p.mu.Unlock()
	if cb != nil {
		cb(p.List())
	}
	return nil
}

// SetHealthy updates the in-memory health flag used by ListHealthy.
func (p *Pool) SetHealthy(id sharedkernel.EdgeID, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, exists := p.byID[id]; exists && e != nil {
		e.healthy = ok
	}
}

// ListHealthy returns enabled + healthy instances matching the capability filter.
// Results are sorted by ID for stable round-robin scheduling.
func (p *Pool) ListHealthy(_ context.Context, filter CapabilityFilter) ([]Instance, error) {
	return p.list(filter, true)
}

// ListEnabled returns enabled instances matching the filter, ignoring the healthy flag.
func (p *Pool) ListEnabled(_ context.Context, filter CapabilityFilter) ([]Instance, error) {
	return p.list(filter, false)
}

func (p *Pool) list(filter CapabilityFilter, requireHealthy bool) ([]Instance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var out []Instance
	for _, e := range p.byID {
		if e == nil || !e.record.Enabled {
			continue
		}
		if requireHealthy && !e.healthy {
			continue
		}
		if !matchCapabilities(e.record.Capabilities, filter) {
			continue
		}
		out = append(out, Instance{
			ID:            e.record.ID,
			DispatchTopic: sharedkernel.TopicDispatch(e.record.ID),
			Capabilities:  append([]string(nil), e.record.Capabilities...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// List returns scheduling views for all currently pooled (enabled) instances.
func (p *Pool) List() []Instance {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]Instance, 0, len(p.byID))
	for _, e := range p.byID {
		if e == nil {
			continue
		}
		out = append(out, Instance{
			ID:            e.record.ID,
			DispatchTopic: sharedkernel.TopicDispatch(e.record.ID),
			Capabilities:  append([]string(nil), e.record.Capabilities...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns a scheduling view for one instance (enabled or not).
func (p *Pool) Get(_ context.Context, id sharedkernel.EdgeID) (*Instance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.byID[id]
	if !ok || e == nil {
		return nil, fmt.Errorf("edge: %s not found", id)
	}
	cp := Instance{
		ID:            e.record.ID,
		DispatchTopic: sharedkernel.TopicDispatch(e.record.ID),
		Capabilities:  append([]string(nil), e.record.Capabilities...),
	}
	return &cp, nil
}

func matchCapabilities(have []string, filter CapabilityFilter) bool {
	if len(filter.AnyOf) == 0 {
		return true
	}
	need := map[string]struct{}{}
	for _, c := range filter.AnyOf {
		need[c] = struct{}{}
	}
	for _, c := range have {
		if _, ok := need[c]; ok {
			return true
		}
	}
	return false
}

var _ Registry = (*Pool)(nil)
