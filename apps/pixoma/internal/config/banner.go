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
	b.WriteString("Pixoma 已启动\n")
	b.WriteString("后台地址：")
	b.WriteString(in.ListenURL)
	b.WriteByte('\n')
	if in.Username != "" {
		b.WriteString("管理员账号：")
		b.WriteString(in.Username)
		b.WriteByte('\n')
	}
	if in.Password != "" {
		b.WriteString("初始密码：")
		b.WriteString(in.Password)
		b.WriteString("\n(首次登录后改密)\n")
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
