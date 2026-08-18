package app

import (
	"fmt"
	"os"
	"os/exec"
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

// EdgeSpawnConfig configures the local edge child process.
type EdgeSpawnConfig struct {
	Binary          string
	ControlPlaneURL string
	AgentToken      string
	EdgeID          string
	BlobDriver      string
	BlobRoot        string
	ComfyMock       bool
	ExtraEnv        []string
}

// EdgeCommand builds an *exec.Cmd for pixoma-edge-agent (not started).
func EdgeCommand(cfg EdgeSpawnConfig) *exec.Cmd {
	bin := cfg.Binary
	if bin == "" {
		bin = "pixoma-edge-agent"
	}
	cmd := exec.Command(bin)
	mock := "false"
	if cfg.ComfyMock {
		mock = "true"
	}
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"CONTROL_PLANE_URL="+cfg.ControlPlaneURL,
		"AGENT_TOKEN="+cfg.AgentToken,
		"EDGE_ID="+cfg.EdgeID,
		"BLOB_DRIVER="+cfg.BlobDriver,
		"BLOB_LOCAL_ROOT="+cfg.BlobRoot,
		"COMFY_MOCK="+mock,
	)
	env = append(env, cfg.ExtraEnv...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
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

// ResolveEdgeBinary finds edge binary next to pixoma or on PATH.
func ResolveEdgeBinary() string {
	if v := strings.TrimSpace(os.Getenv("EDGE_AGENT_BIN")); v != "" {
		return v
	}
	self, err := os.Executable()
	if err == nil {
		cand := filepath.Join(filepath.Dir(self), "pixoma-edge-agent")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand
		}
	}
	if p, err := exec.LookPath("pixoma-edge-agent"); err == nil {
		return p
	}
	// Dev fallback: same module tree built as edge-agent.
	if p, err := exec.LookPath("edge-agent"); err == nil {
		return p
	}
	return "pixoma-edge-agent"
}
