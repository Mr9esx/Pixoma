package settings

import (
	"fmt"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
)

const (
	PlacementLocal  = "local"
	PlacementRemote = "remote"

	DriverSQLite   = "sqlite"
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
)

// Settings is the persisted platform configuration (no queue.driver / runtime_mode).
type Settings struct {
	Placement         string `json:"placement"`
	DBDriver          string `json:"db_driver"`
	DBDSN             string `json:"db_dsn"`
	BlobDriver        string `json:"blob_driver"`
	BlobRoot          string `json:"blob_root"`
	BlobEndpoint      string `json:"blob_endpoint"`
	BlobRegion        string `json:"blob_region"`
	BlobBucket        string `json:"blob_bucket"`
	BlobAccessKey     string `json:"blob_access_key,omitempty"`
	BlobSecretKey     string `json:"blob_secret_key,omitempty"`
	ComfyMock         bool   `json:"comfy_mock"`
	ComfyUIBaseURL    string `json:"comfyui_base_url"`
	DefaultInstanceID string `json:"default_instance_id"`
	AutoSpawnEdge     bool   `json:"auto_spawn_edge"`
	ClaimWaitMS       int    `json:"claim_wait_ms"`
	LeaseSeconds      int    `json:"lease_seconds"`
	TelegramBotToken  string `json:"telegram_bot_token,omitempty"`
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
		case botconfig.BlobDriverLocalFS, botconfig.BlobDriverS3, botconfig.BlobDriverTOS:
		default:
			return fmt.Errorf("settings: unknown blob.driver %q", b)
		}
		if b == botconfig.BlobDriverLocalFS && strings.TrimSpace(s.BlobRoot) == "" {
			return fmt.Errorf("settings: localfs requires blob root")
		}
	case PlacementRemote:
		if b == botconfig.BlobDriverLocalFS {
			return fmt.Errorf("settings: remote deployment cannot use blob.driver=localfs")
		}
		if b != botconfig.BlobDriverS3 && b != botconfig.BlobDriverTOS {
			return fmt.Errorf("settings: remote requires blob.driver=s3 or tos, got %q", b)
		}
	default:
		return fmt.Errorf("settings: unknown placement %q", p)
	}
	return nil
}
