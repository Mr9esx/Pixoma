package instance

import (
	"context"
	"fmt"
	"sync"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/comfyui"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// PoolOptions configures client construction when Refresh runs.
type PoolOptions struct {
	Mock bool
}

type pooled struct {
	record  Record
	client  comfyui.Client
	healthy bool
}

// Pool keeps an in-process Comfy client map keyed by instance id and
// implements Registry for scheduling.
type Pool struct {
	repo Repository
	opts PoolOptions

	mu   sync.RWMutex
	byID map[sharedkernel.InstanceID]*pooled
}

// NewPool constructs a Pool. Call Refresh after seeding the repository.
func NewPool(repo Repository, opts PoolOptions) *Pool {
	return &Pool{
		repo: repo,
		opts: opts,
		byID: make(map[sharedkernel.InstanceID]*pooled),
	}
}

// Refresh rebuilds the client map from the repository.
// Enabled instances start healthy; existing health flags are preserved when possible.
func (p *Pool) Refresh(ctx context.Context) error {
	if p.repo == nil {
		return fmt.Errorf("instance: nil repository")
	}
	list, err := p.repo.List(ctx)
	if err != nil {
		return err
	}

	prevHealthy := map[sharedkernel.InstanceID]bool{}
	p.mu.RLock()
	for id, e := range p.byID {
		prevHealthy[id] = e.healthy
	}
	p.mu.RUnlock()

	next := make(map[sharedkernel.InstanceID]*pooled, len(list))
	for _, rec := range list {
		if rec == nil || !rec.Enabled {
			continue
		}
		cli, err := comfyui.NewClient(comfyui.Options{
			Mock:    p.opts.Mock,
			BaseURL: rec.BaseURL,
		})
		if err != nil {
			return fmt.Errorf("instance: client %s: %w", rec.ID, err)
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
			client:  cli,
			healthy: healthy,
		}
	}

	p.mu.Lock()
	p.byID = next
	p.mu.Unlock()
	return nil
}

// Client returns the Comfy client for an instance id.
func (p *Pool) Client(id sharedkernel.InstanceID) (comfyui.Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.byID[id]
	if !ok || e == nil || e.client == nil {
		return nil, fmt.Errorf("instance: client %s not found", id)
	}
	return e.client, nil
}

// SetHealthy updates the in-memory health flag used by ListHealthy.
func (p *Pool) SetHealthy(id sharedkernel.InstanceID, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e, exists := p.byID[id]; exists && e != nil {
		e.healthy = ok
	}
}

// ListHealthy returns enabled + healthy instances matching the capability filter.
func (p *Pool) ListHealthy(_ context.Context, filter CapabilityFilter) ([]Instance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var out []Instance
	for _, e := range p.byID {
		if e == nil || !e.record.Enabled || !e.healthy {
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
	return out, nil
}

// Get returns a scheduling view for one instance (enabled or not).
func (p *Pool) Get(_ context.Context, id sharedkernel.InstanceID) (*Instance, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.byID[id]
	if !ok || e == nil {
		return nil, fmt.Errorf("instance: %s not found", id)
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
