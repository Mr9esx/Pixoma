package tg

import (
	"strconv"
	"sync"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
)

// invokeStore maps short callback tokens to pending capability invokes.
// TG callbacks are limited to 64 bytes, so params travel via token.
type invokeStore struct {
	mu      sync.Mutex
	seq     uint64
	invokes map[string]protocol.CapabilityInvoke
}

func newInvokeStore() *invokeStore {
	return &invokeStore{invokes: map[string]protocol.CapabilityInvoke{}}
}

func (s *invokeStore) put(inv protocol.CapabilityInvoke) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	token := strconv.FormatUint(s.seq, 36)
	s.invokes[token] = inv
	return token
}

func (s *invokeStore) get(token string) (protocol.CapabilityInvoke, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invokes[token]
	if ok {
		delete(s.invokes, token)
	}
	return inv, ok
}
