package topic

import (
	"errors"
	"strings"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
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
// empty or missing values resolve to the default topic, duplicates are removed,
// and order is preserved.
func NormalizeTopics(raw []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		key := strings.TrimSpace(t)
		if key == "" {
			key = DefaultKey
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	if len(out) == 0 {
		return []string{DefaultKey}
	}
	return out
}
