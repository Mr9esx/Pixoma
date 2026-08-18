// Package crypto provides AES-GCM encryption for secrets at rest.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encrypt AES-GCM encrypts plain for at-rest storage. Empty input yields empty output.
func Encrypt(key []byte, plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("crypto: enc key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.RawStdEncoding.EncodeToString(out), nil
}

// Decrypt reverses Encrypt.
func Decrypt(key []byte, cipherText string) (string, error) {
	if cipherText == "" {
		return "", nil
	}
	if len(key) != 32 {
		return "", fmt.Errorf("crypto: enc key must be 32 bytes")
	}
	raw, err := base64.RawStdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("crypto: decode cipher: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("crypto: cipher too short")
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return string(plain), nil
}
