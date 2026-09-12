package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"

	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
)

var ErrTokenNotFound = errors.New("mcp token not found")

type TokenRecord struct {
	UserID      string
	TokenHash   string
	TokenCipher string
}

type TokenStore interface {
	Put(ctx context.Context, rec TokenRecord) error
	GetByHash(ctx context.Context, hash string) (TokenRecord, error)
	GetByUserID(ctx context.Context, userID string) (TokenRecord, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func EncryptToken(key []byte, plain string) (string, error) {
	return settingsdomain.EncryptString(key, plain)
}

func DecryptToken(key []byte, cipher string) (string, error) {
	return settingsdomain.DecryptString(key, cipher)
}

type MemoryTokenStore struct {
	mu     sync.Mutex
	byHash map[string]TokenRecord
	byUser map[string]TokenRecord
}

func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		byHash: map[string]TokenRecord{},
		byUser: map[string]TokenRecord{},
	}
}

func (s *MemoryTokenStore) Put(_ context.Context, rec TokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.byUser[rec.UserID]; ok {
		delete(s.byHash, old.TokenHash)
	}
	s.byHash[rec.TokenHash] = rec
	s.byUser[rec.UserID] = rec
	return nil
}

func (s *MemoryTokenStore) GetByHash(_ context.Context, hash string) (TokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.byHash[hash]
	if !ok {
		return TokenRecord{}, ErrTokenNotFound
	}
	return rec, nil
}

func (s *MemoryTokenStore) GetByUserID(_ context.Context, userID string) (TokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.byUser[userID]
	if !ok {
		return TokenRecord{}, ErrTokenNotFound
	}
	return rec, nil
}

func (s *MemoryTokenStore) DeleteByUserID(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.byUser[userID]; ok {
		delete(s.byHash, old.TokenHash)
		delete(s.byUser, userID)
	}
	return nil
}
