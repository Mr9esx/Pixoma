package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Adapter is a running channel adapter instance.
type Adapter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// AdapterFactory creates a channel adapter from a channel snapshot.
type AdapterFactory interface {
	Create(snap ChannelSnapshot) (Adapter, error)
}

// ChannelSnapshot is a channel row projection used for reconciliation.
type ChannelSnapshot struct {
	ID             string
	Platform       string
	Credential     string // decrypted credential (memory-resident for adapter creation)
	CredentialHash string
	Enabled        bool
	UpdatedAt      time.Time
}

// SnapshotStore lists channels for the assembler.
type SnapshotStore interface {
	ListChannels(ctx context.Context) ([]ChannelSnapshot, error)
}

func hashCredential(cred string) string {
	sum := sha256.Sum256([]byte(cred))
	return hex.EncodeToString(sum[:])
}
