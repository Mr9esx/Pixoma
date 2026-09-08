package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// DefaultKey is the reserved default topic key created at startup.
const DefaultKey = "default"

// ErrTopicNotFound is returned when a topic key does not exist.
var ErrTopicNotFound = errors.New("topic: not found")

// ErrTopicConflict is returned when creating a topic with an existing key.
var ErrTopicConflict = errors.New("topic: key conflict")

// Topic is a logical delivery target for tasks.
type Topic struct {
	Key       string
	Name      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TopicID is the stable topic key.
type TopicID = sharedkernel.TopicKey

// NormalizeTopics maps a raw subscription list to the effective topic set:
// empty or missing entries are dropped, duplicates are removed, and order is
// preserved. An empty result means the edge has no topic binding and consumes
// no tasks (it must be bound to a task queue to receive work).
func NormalizeTopics(raw []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		key := strings.TrimSpace(t)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
