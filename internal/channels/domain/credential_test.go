package domain

import "testing"

func TestCredentialRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	ct, err := EncryptCredential(key, Credential{BotToken: "123456:ABC"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecryptCredential(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if got.BotToken != "123456:ABC" {
		t.Fatalf("token=%q", got.BotToken)
	}
}

func TestDecryptCredentialWrongKey(t *testing.T) {
	key := make([]byte, 32)
	ct, err := EncryptCredential(key, Credential{BotToken: "x"})
	if err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 32)
	wrong[0] = 7
	if _, err := DecryptCredential(wrong, ct); err == nil {
		t.Fatal("want error")
	}
}

func TestMaskedToken(t *testing.T) {
	cases := map[string]string{
		"1234567890":    "1234****7890",
		"abc":           "****",
		"1234567890123": "1234****0123",
		"":              "****",
	}
	for in, want := range cases {
		if got := MaskedToken(in); got != want {
			t.Fatalf("MaskedToken(%q)=%q want %q", in, got, want)
		}
	}
}
