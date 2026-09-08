package presence

import (
	"sync"
	"time"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// OnlineWindow is how long a report/touch keeps an Edge worker online.
const OnlineWindow = 15 * time.Second

// Snapshot is the admin-facing view of one instance's presence.
type Snapshot struct {
	ID           string `json:"id"`
	EdgeOnline   bool   `json:"edge_online"`
	ComfyRunning bool   `json:"comfy_running"`
}

type record struct {
	lastSeen     time.Time
	comfyRunning bool
}

// Store keeps last_seen and last reported comfy_running in process memory.
type Store struct {
	Now func() time.Time

	mu      sync.Mutex
	records map[sharedkernel.EdgeID]record
}

func NewStore() *Store {
	return &Store{records: map[sharedkernel.EdgeID]record{}}
}

func (s *Store) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

// Report records an Edge presence heartbeat including local Comfy status.
func (s *Store) Report(id sharedkernel.EdgeID, comfyRunning bool) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = record{lastSeen: s.now(), comfyRunning: comfyRunning}
}

// Touch refreshes last_seen without changing the last comfy_running value.
func (s *Store) Touch(id sharedkernel.EdgeID) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.records[id]
	rec.lastSeen = s.now()
	s.records[id] = rec
}

// Remove drops the in-memory record for an edge (admin delete cleanup).
func (s *Store) Remove(id sharedkernel.EdgeID) {
	if s == nil || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, id)
}

// Snapshot returns the derived online/comfy flags. Stale reports look offline
// and Comfy-down so the UI never keeps a stale "running" tag.
func (s *Store) Snapshot(id sharedkernel.EdgeID) Snapshot {
	view := Snapshot{ID: string(id)}
	if s == nil || id == "" {
		return view
	}
	s.mu.Lock()
	rec, ok := s.records[id]
	s.mu.Unlock()
	if !ok {
		return view
	}
	online := s.now().Sub(rec.lastSeen) < OnlineWindow
	view.EdgeOnline = online
	view.ComfyRunning = online && rec.comfyRunning
	return view
}
