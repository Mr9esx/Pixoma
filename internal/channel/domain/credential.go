package domain

import (
	"encoding/json"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/crypto"
)

// Credential is the platform-specific credential payload (JSON-encrypted at rest).
type Credential struct {
	BotToken string `json:"bot_token,omitempty"`
}

// EncryptCredential serializes and encrypts a credential with the given key.
func EncryptCredential(key []byte, cred Credential) (string, error) {
	raw, err := json.Marshal(cred)
	if err != nil {
		return "", err
	}
	return crypto.Encrypt(key, string(raw))
}

// DecryptCredential decrypts and parses a stored credential.
func DecryptCredential(key []byte, cipherText string) (Credential, error) {
	plain, err := crypto.Decrypt(key, cipherText)
	if err != nil {
		return Credential{}, err
	}
	var cred Credential
	if plain == "" {
		return cred, nil
	}
	if err := json.Unmarshal([]byte(plain), &cred); err != nil {
		return Credential{}, err
	}
	return cred, nil
}

// MaskedToken renders a token for display: first 4 + "****" + last 4, or "****" when too short.
func MaskedToken(token string) string {
	if len(token) < 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
