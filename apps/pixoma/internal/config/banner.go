package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BannerInput is logged once at process start.
type BannerInput struct {
	ListenURL string
	Username  string
	Password  string // empty → omit password line
}

// StartupBanner formats the zero-config welcome message.
func StartupBanner(in BannerInput) string {
	var b strings.Builder
	b.WriteString("Pixoma control plane ready\n")
	b.WriteString("Admin URL: ")
	b.WriteString(in.ListenURL)
	b.WriteByte('\n')
	if in.Username != "" {
		b.WriteString("Admin user: ")
		b.WriteString(in.Username)
		b.WriteByte('\n')
	}
	if in.Password != "" {
		b.WriteString("Admin password: ")
		b.WriteString(in.Password)
		b.WriteString("\n(change this password on first login)\n")
	}
	return b.String()
}

// WriteAgentTokenFile stores plaintext token for local Edge spawn (0600).
func WriteAgentTokenFile(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token), 0o600)
}

// ReadAgentTokenFile loads a previously written agent token.
func ReadAgentTokenFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	tok := strings.TrimSpace(string(b))
	if tok == "" {
		return "", fmt.Errorf("empty agent token file")
	}
	return tok, nil
}
