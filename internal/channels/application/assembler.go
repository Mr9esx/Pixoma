package application

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Mr9esx/Pixoma/internal/channels/domain"
)

type adapterState string

const (
	stateAbsent   adapterState = "absent"
	stateStarting adapterState = "starting"
	stateRunning  adapterState = "running"
	stateStopping adapterState = "stopping"
	stateError    adapterState = "error"
)

type managedAdapter struct {
	snapshot ChannelSnapshot
	adapter  Adapter
	state    adapterState
	lastErr  error
}

// AdapterStatus is the live state of a managed channel adapter.
type AdapterStatus struct {
	State   adapterState
	LastErr error
}

type startJob struct {
	id   string
	snap ChannelSnapshot
}

// Assembler reconciles channel snapshots with running adapters (hot reload).
type Assembler struct {
	Store    SnapshotStore
	Factory  AdapterFactory
	Interval time.Duration // watch interval; default 5s

	mu       sync.Mutex
	halted   bool
	adapters map[string]*managedAdapter
}

func (a *Assembler) interval() time.Duration {
	if a.Interval > 0 {
		return a.Interval
	}
	return 5 * time.Second
}

// Status returns the current state of every managed adapter keyed by channel ID.
func (a *Assembler) Status() map[string]AdapterStatus {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[string]AdapterStatus, len(a.adapters))
	for id, ma := range a.adapters {
		out[id] = AdapterStatus{State: ma.state, LastErr: ma.lastErr}
	}
	return out
}

// Run reconciles channels until ctx is cancelled.
func (a *Assembler) Run(ctx context.Context) error {
	ticker := time.NewTicker(a.interval())
	defer ticker.Stop()
	for {
		if err := a.reconcile(ctx); err != nil {
			slog.Error("channel assembler reconcile", "err", err)
		}
		select {
		case <-ctx.Done():
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return a.StopAll(stopCtx)
		case <-ticker.C:
		}
	}
}

// StopAll stops every running adapter.
func (a *Assembler) StopAll(ctx context.Context) error {
	a.mu.Lock()
	a.halted = true
	var toStop []Adapter
	for id, ma := range a.adapters {
		if ad := detachIfActive(ma); ad != nil {
			toStop = append(toStop, ad)
		}
		delete(a.adapters, id)
	}
	a.mu.Unlock()
	var firstErr error
	for _, ad := range toStop {
		if err := ad.Stop(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (a *Assembler) reconcile(ctx context.Context) error {
	snapshots, err := a.Store.ListChannels(ctx)
	if err != nil {
		return err
	}

	var starts []startJob
	var toStop []Adapter

	a.mu.Lock()
	if a.halted {
		a.mu.Unlock()
		return nil
	}
	if a.adapters == nil {
		a.adapters = map[string]*managedAdapter{}
	}
	seen := map[string]struct{}{}
	for _, snap := range snapshots {
		seen[snap.ID] = struct{}{}
		ma, ok := a.adapters[snap.ID]
		if !ok {
			ma = &managedAdapter{snapshot: snap, state: stateAbsent}
			a.adapters[snap.ID] = ma
		}
		credChanged := ma.snapshot.CredentialHash != snap.CredentialHash
		ma.snapshot = snap
		needsAdapter := snap.Enabled && snap.Platform != string(domain.PlatformMCP)
		switch {
		case !needsAdapter:
			if ad := detachIfActive(ma); ad != nil {
				toStop = append(toStop, ad)
			}
			ma.state = stateAbsent
			ma.lastErr = nil
		case credChanged || ma.state == stateError || ma.state == stateAbsent:
			if ad := detachIfActive(ma); ad != nil {
				toStop = append(toStop, ad)
			}
			ma.state = stateStarting
			ma.lastErr = nil
			starts = append(starts, startJob{id: snap.ID, snap: snap})
		}
	}
	for id, ma := range a.adapters {
		if _, ok := seen[id]; !ok {
			if ad := detachIfActive(ma); ad != nil {
				toStop = append(toStop, ad)
			}
			delete(a.adapters, id)
		}
	}
	a.mu.Unlock()

	for _, ad := range toStop {
		if err := ad.Stop(ctx); err != nil {
			slog.Error("channel adapter stop", "err", err)
		}
	}
	for _, job := range starts {
		a.startOne(ctx, job.id, job.snap)
	}
	return nil
}

func detachIfActive(ma *managedAdapter) Adapter {
	if ma == nil || ma.adapter == nil {
		return nil
	}
	if ma.state != stateRunning && ma.state != stateStarting {
		return nil
	}
	ad := ma.adapter
	ma.adapter = nil
	return ad
}

func (a *Assembler) startOne(ctx context.Context, id string, snap ChannelSnapshot) {
	ad, err := a.Factory.Create(snap)
	if err != nil {
		a.finishStart(id, snap, nil, err)
		slog.Error("channel adapter create", "err", err, "channel", id)
		return
	}
	if err := ad.Start(ctx); err != nil {
		a.finishStart(id, snap, nil, err)
		slog.Error("channel adapter start", "err", err, "channel", id)
		return
	}
	if a.finishStart(id, snap, ad, nil) {
		if stopErr := ad.Stop(ctx); stopErr != nil {
			slog.Error("channel adapter stop", "err", stopErr, "channel", id)
		}
		return
	}
	slog.Info("channel adapter started", "channel", id)
}

func (a *Assembler) finishStart(id string, snap ChannelSnapshot, ad Adapter, err error) (discard bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ma, ok := a.adapters[id]
	if a.halted || !ok || !ma.snapshot.Enabled || ma.snapshot.CredentialHash != snap.CredentialHash {
		return ad != nil
	}
	if err != nil {
		ma.state = stateError
		ma.lastErr = err
		ma.adapter = nil
		return false
	}
	ma.adapter = ad
	ma.state = stateRunning
	ma.lastErr = nil
	return false
}
