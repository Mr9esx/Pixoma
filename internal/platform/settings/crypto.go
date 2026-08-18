package settings

import (
	"github.com/mr9esx/comfyui_tgbot/internal/platform/crypto"
)

// EncryptString AES-GCM encrypts a secret for at-rest storage.
func EncryptString(key []byte, plain string) (string, error) {
	return crypto.Encrypt(key, plain)
}

// DecryptString reverses EncryptString.
func DecryptString(key []byte, cipherText string) (string, error) {
	return crypto.Decrypt(key, cipherText)
}
