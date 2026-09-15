package conversation

import (
	"strconv"
	"sync"

	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
)

// ActionStore maps compact, one-shot callback tokens to capability invokes.
type ActionStore struct {
	mu      sync.Mutex
	seq     uint64
	actions map[string]protocol.CapabilityInvoke
}

func NewActionStore() *ActionStore {
	return &ActionStore{actions: make(map[string]protocol.CapabilityInvoke)}
}

func (s *ActionStore) Put(inv protocol.CapabilityInvoke) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	token := strconv.FormatUint(s.seq, 36)
	s.actions[token] = inv
	return token
}

func (s *ActionStore) Get(token string) (protocol.CapabilityInvoke, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.actions[token]
	return inv, ok
}

func (s *ActionStore) Take(token string) (protocol.CapabilityInvoke, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.actions[token]
	if ok {
		delete(s.actions, token)
	}
	return inv, ok
}

func (s *ActionStore) Consume(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.actions, token)
}

// BackStack records nested card origins per conversation.
type BackStack struct {
	mu     sync.Mutex
	stacks map[string][]string
}

// NotificationStore suppresses duplicate task notifications per channel
// process lifetime.
type NotificationStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewNotificationStore() *NotificationStore {
	return &NotificationStore{seen: make(map[string]struct{})}
}

// Mark returns true only the first time a notification key is observed.
func (s *NotificationStore) Mark(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[key]; ok {
		return false
	}
	s.seen[key] = struct{}{}
	return true
}

func NewBackStack() *BackStack { return &BackStack{stacks: make(map[string][]string)} }

func (s *BackStack) Push(chat, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stacks[chat] = append(s.stacks[chat], id)
}

func (s *BackStack) Pop(chat string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.stacks[chat]
	if len(items) == 0 {
		return "", false
	}
	last := items[len(items)-1]
	s.stacks[chat] = items[:len(items)-1]
	return last, true
}

func (s *BackStack) Top(chat string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.stacks[chat]
	if len(items) == 0 {
		return "", false
	}
	return items[len(items)-1], true
}

func (s *BackStack) Clear(chat string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.stacks, chat)
}
