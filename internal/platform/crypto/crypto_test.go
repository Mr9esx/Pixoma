package crypto

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	ct, err := Encrypt(key, "supersecret")
	if err != nil {
		t.Fatal(err)
	}
	if ct == "supersecret" {
		t.Fatal("ciphertext must not be plaintext")
	}
	got, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if got != "supersecret" {
		t.Fatalf("roundtrip: %q", got)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	key := make([]byte, 32)
	ct, err := Encrypt(key, "secret")
	if err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 32)
	wrong[0] = 1
	if _, err := Decrypt(wrong, ct); err == nil {
		t.Fatal("want error for wrong key")
	}
}

func TestEmptyRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	ct, err := Encrypt(key, "")
	if err != nil || ct != "" {
		t.Fatalf("empty: %q %v", ct, err)
	}
	got, err := Decrypt(key, "")
	if err != nil || got != "" {
		t.Fatalf("empty: %q %v", got, err)
	}
}
