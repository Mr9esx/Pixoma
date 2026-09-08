package domain

import (
	"fmt"
	"strings"

	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
)

const (
	PlacementLocal  = "local"
	PlacementRemote = "remote"

	DriverSQLite   = "sqlite"
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"

	ProxyOff   = ""
	ProxyHTTP  = "http"
	ProxySOCKS = "socks5"

	// DefaultProxyPort is the proxy port used when no explicit port is set.
	DefaultProxyPort = 7897

	// MinMediaMaxBytes is the lower bound (1 KiB) enforced on MediaMaxBytes to
	// keep the value meaningful. Zero means "fall back to media.DefaultMaxBytes".
	MinMediaMaxBytes int64 = 1024

	DefaultUserAccessAlwaysAllowed = "always_allowed"
	DefaultUserAccessPaid          = "paid"
	DefaultUserAccessDenied        = "denied"
)

// Settings is the persisted platform configuration.
type Settings struct {
	Placement      string `json:"placement"`
	DBDriver       string `json:"db_driver"`
	DBDSN          string `json:"db_dsn"`
	BlobDriver     string `json:"blob_driver"`
	BlobRoot       string `json:"blob_root"`
	BlobEndpoint   string `json:"blob_endpoint"`
	BlobRegion     string `json:"blob_region"`
	BlobBucket     string `json:"blob_bucket"`
	BlobAccessKey  string `json:"blob_access_key,omitempty"`
	BlobSecretKey  string `json:"blob_secret_key,omitempty"`
	ComfyUIBaseURL string `json:"comfyui_base_url"`
	ClaimWaitMS    int    `json:"claim_wait_ms"`
	LeaseSeconds   int    `json:"lease_seconds"`
	ProxyKind      string `json:"proxy_kind,omitempty"`
	ProxyHost      string `json:"proxy_host,omitempty"`
	ProxyPort      int    `json:"proxy_port,omitempty"`
	// MediaMaxBytes caps a single preview media upload (bytes). 0 means
	// "use media.DefaultMaxBytes". Must be >= MinMediaMaxBytes when set.
	MediaMaxBytes int64 `json:"media_max_bytes,omitempty"`
	// AllowSelfRegistration toggles public console-account registration.
	AllowSelfRegistration bool `json:"allow_self_registration"`
	// DefaultUserAccess is applied only when creating a new channel user.
	DefaultUserAccess string `json:"default_user_access"`
}

// Validate checks local/remote vs blob legality.
func (s Settings) Validate() error {
	p := strings.TrimSpace(s.Placement)
	if p == "" {
		p = PlacementLocal
	}
	b := strings.TrimSpace(s.BlobDriver)
	if b == "" {
		b = botconfig.BlobDriverLocalFS
	}
	d := strings.TrimSpace(s.DBDriver)
	if d == "" {
		d = DriverSQLite
	}
	switch d {
	case DriverSQLite, DriverMySQL, DriverPostgres:
	default:
		return fmt.Errorf("settings: unknown db driver %q", d)
	}
	if strings.TrimSpace(s.DBDSN) == "" {
		return fmt.Errorf("settings: empty database dsn")
	}
	switch p {
	case PlacementLocal:
		switch b {
		case botconfig.BlobDriverLocalFS, botconfig.BlobDriverSharedFS,
			botconfig.BlobDriverS3, botconfig.BlobDriverTOS:
		default:
			return fmt.Errorf("settings: unknown blob.driver %q", b)
		}
		if (b == botconfig.BlobDriverLocalFS || b == botconfig.BlobDriverSharedFS) &&
			strings.TrimSpace(s.BlobRoot) == "" {
			return fmt.Errorf("settings: %s requires blob root", b)
		}
	case PlacementRemote:
		if b == botconfig.BlobDriverLocalFS {
			return fmt.Errorf("settings: remote deployment cannot use blob.driver=localfs")
		}
		if b != botconfig.BlobDriverS3 && b != botconfig.BlobDriverTOS && b != botconfig.BlobDriverSharedFS {
			return fmt.Errorf("settings: remote requires blob.driver=s3, tos or sharedfs, got %q", b)
		}
	default:
		return fmt.Errorf("settings: unknown placement %q", p)
	}
	if s.MediaMaxBytes < 0 {
		return fmt.Errorf("settings: media_max_bytes must be non-negative")
	}
	if s.MediaMaxBytes > 0 && s.MediaMaxBytes < MinMediaMaxBytes {
		return fmt.Errorf("settings: media_max_bytes must be at least %d bytes", MinMediaMaxBytes)
	}
	return validateProxy(s)
}

// NormalizeDefaultUserAccess maps unknown values to denied.
func NormalizeDefaultUserAccess(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case DefaultUserAccessAlwaysAllowed:
		return DefaultUserAccessAlwaysAllowed
	case DefaultUserAccessPaid:
		return DefaultUserAccessPaid
	default:
		return DefaultUserAccessDenied
	}
}

func validateProxy(s Settings) error {
	kind := strings.ToLower(strings.TrimSpace(s.ProxyKind))
	switch kind {
	case ProxyOff, "off", "none":
		return nil
	case ProxyHTTP, ProxySOCKS, "socks":
	default:
		return fmt.Errorf("settings: unknown proxy kind %q", s.ProxyKind)
	}
	if strings.TrimSpace(s.ProxyHost) == "" {
		return fmt.Errorf("settings: proxy host required")
	}
	if port := s.effectiveProxyPort(); port < 1 || port > 65535 {
		return fmt.Errorf("settings: invalid proxy port")
	}
	return nil
}

// effectiveProxyPort is the proxy port in use, falling back to the default
// 7897 when no explicit port is configured.
func (s Settings) effectiveProxyPort() int {
	if s.ProxyPort < 1 {
		return DefaultProxyPort
	}
	return s.ProxyPort
}

// EffectiveProxyPort returns the proxy port that will actually be used.
func (s Settings) EffectiveProxyPort() int {
	return s.effectiveProxyPort()
}

// ProxyURL is the process proxy URL, or empty when proxy is off.
func (s Settings) ProxyURL() string {
	kind := strings.ToLower(strings.TrimSpace(s.ProxyKind))
	host := strings.TrimSpace(s.ProxyHost)
	if host == "" {
		return ""
	}
	port := s.effectiveProxyPort()
	switch kind {
	case ProxyHTTP:
		return fmt.Sprintf("http://%s:%d", host, port)
	case ProxySOCKS, "socks":
		return fmt.Sprintf("socks5://%s:%d", host, port)
	default:
		return ""
	}
}
