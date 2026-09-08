package domain

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// NewID returns a random instance id. Operators never type this.
func NewID() (sharedkernel.EdgeID, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("edge: id: %w", err)
	}
	return sharedkernel.EdgeID("node-" + hex.EncodeToString(b[:])), nil
}

// MintToken returns a new plaintext AGENT_TOKEN.
func MintToken() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("edge: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// EncryptToken stores an agent token at rest.
func EncryptToken(key []byte, plain string) (string, error) {
	enc, err := settingsdomain.EncryptString(key, plain)
	if err != nil {
		return "", fmt.Errorf("edge: encrypt token: %w", err)
	}
	return enc, nil
}

// DecryptToken returns the plaintext agent token.
func DecryptToken(key []byte, enc string) (string, error) {
	plain, err := settingsdomain.DecryptString(key, enc)
	if err != nil {
		return "", fmt.Errorf("edge: decrypt token: %w", err)
	}
	return plain, nil
}

// VerifyAgentToken checks a bearer token against the instance row.
func VerifyAgentToken(
	ctx context.Context,
	repo Repository,
	key []byte,
	id sharedkernel.EdgeID,
	bearer string,
) bool {
	if repo == nil || id == "" || strings.TrimSpace(bearer) == "" {
		return false
	}
	rec, err := repo.Get(ctx, id)
	if err != nil || rec == nil || !rec.Enabled {
		return false
	}
	plain, err := DecryptToken(key, rec.AgentTokenEnc)
	if err != nil || plain == "" {
		return false
	}
	got := strings.TrimSpace(bearer)
	return subtle.ConstantTimeCompare([]byte(plain), []byte(got)) == 1
}

// EnsureAgentTokens mints a token for every instance that does not have one.
func EnsureAgentTokens(ctx context.Context, repo Repository, key []byte) error {
	if repo == nil {
		return fmt.Errorf("edge: nil repository")
	}
	list, err := repo.List(ctx)
	if err != nil {
		return err
	}
	for _, rec := range list {
		if rec == nil || strings.TrimSpace(rec.AgentTokenEnc) != "" {
			continue
		}
		plain, err := MintToken()
		if err != nil {
			return err
		}
		enc, err := EncryptToken(key, plain)
		if err != nil {
			return err
		}
		if err := repo.UpdateAgentTokenEnc(ctx, rec.ID, enc); err != nil {
			return err
		}
		rec.AgentTokenEnc = enc
	}
	return nil
}

// PlainAgentToken decrypts the token for an instance (local spawn / admin).
func PlainAgentToken(
	ctx context.Context,
	repo Repository,
	key []byte,
	id sharedkernel.EdgeID,
) (string, error) {
	rec, err := repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	plain, err := DecryptToken(key, rec.AgentTokenEnc)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(plain) == "" {
		return "", fmt.Errorf("edge: empty agent token")
	}
	return plain, nil
}
