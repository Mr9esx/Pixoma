package topic

import (
	"errors"
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
