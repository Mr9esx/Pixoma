package runtime

import (
	"context"
	"log/slog"
	"sync"
	"time"
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

// Assembler reconciles channel snapshots with running adapters (hot reload).
type Assembler struct {
	Store    SnapshotStore
	Factory  AdapterFactory
	Interval time.Duration // watch interval; default 5s

	mu      sync.Mutex
	adapters map[string]*managedAdapter
}

func (a *Assembler) interval() time.Duration {
	if a.Interval > 0 {
		return a.Interval
	}
	return 5 * time.Second
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
			return a.StopAll(context.Background())
		case <-ticker.C:
		}
	}
}

// StopAll stops every running adapter.
func (a *Assembler) StopAll(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var firstErr error
	for id, ma := range a.adapters {
		if ma.adapter != nil && (ma.state == stateRunning || ma.state == stateStarting) {
			if err := ma.adapter.Stop(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
			ma.state = stateAbsent
		}
		delete(a.adapters, id)
	}
	return firstErr
}

func (a *Assembler) reconcile(ctx context.Context) error {
	snapshots, err := a.Store.ListChannels(ctx)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.adapters == nil {
		a.adapters = map[string]*managedAdapter{}
	}

	seen := map[string]struct{}{}
	for _, snap := range snapshots {
		seen[snap.ID] = struct{}{}
		ma, ok := a.adapters[snap.ID]
		if !ok {
			a.adapters[snap.ID] = &managedAdapter{snapshot: snap, state: stateAbsent}
			ma = a.adapters[snap.ID]
		}
		credChanged := ma.snapshot.CredentialHash != snap.CredentialHash
		ma.snapshot = snap
		switch {
		case !snap.Enabled:
			a.stopLocked(ctx, ma)
		case credChanged || ma.state == stateError || ma.state == stateAbsent:
			a.restartLocked(ctx, ma)
		}
	}
	for id, ma := range a.adapters {
		if _, ok := seen[id]; !ok {
			a.stopLocked(ctx, ma)
			delete(a.adapters, id)
		}
	}
	return nil
}

func (a *Assembler) restartLocked(ctx context.Context, ma *managedAdapter) {
	if ma.adapter != nil && (ma.state == stateRunning || ma.state == stateStarting) {
		if err := ma.adapter.Stop(ctx); err != nil {
			slog.Error("channel adapter stop", "err", err, "channel", ma.snapshot.ID)
		}
	}
	ad, err := a.Factory.Create(ma.snapshot.Platform, ma.snapshot.Credential)
	if err != nil {
		ma.state = stateError
		ma.lastErr = err
		slog.Error("channel adapter create", "err", err, "channel", ma.snapshot.ID)
		return
	}
	ma.adapter = ad
	ma.state = stateStarting
	if err := ad.Start(ctx); err != nil {
		ma.state = stateError
		ma.lastErr = err
		slog.Error("channel adapter start", "err", err, "channel", ma.snapshot.ID)
		return
	}
	ma.state = stateRunning
	ma.lastErr = nil
	slog.Info("channel adapter started", "channel", ma.snapshot.ID)
}

func (a *Assembler) stopLocked(ctx context.Context, ma *managedAdapter) {
	if ma.adapter == nil || (ma.state != stateRunning && ma.state != stateStarting) {
		ma.state = stateAbsent
		return
	}
	if err := ma.adapter.Stop(ctx); err != nil {
		slog.Error("channel adapter stop", "err", err, "channel", ma.snapshot.ID)
	}
	ma.state = stateAbsent
}
