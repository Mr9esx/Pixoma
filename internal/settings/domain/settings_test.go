package domain_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/platform/botconfig"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	settingsdomain "github.com/Mr9esx/Pixoma/internal/settings/domain"
	settingsinfra "github.com/Mr9esx/Pixoma/internal/settings/infrastructure"
)

func TestValidate_RemoteRejectsLocalFS(t *testing.T) {
	s := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementRemote,
		BlobDriver: botconfig.BlobDriverLocalFS,
		DBDriver:   settingsdomain.DriverSQLite,
		DBDSN:      "data/app.db",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected remote+localfs to fail")
	}
	if !strings.Contains(err.Error(), "localfs") {
		t.Fatalf("error should mention localfs, got %v", err)
	}
}

func TestValidate_UnknownDBDriverRejected(t *testing.T) {
	s := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementLocal,
		DBDriver:   "oracle",
		DBDSN:      "host=127.0.0.1",
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   "data/blob",
	}
	err := s.Validate()
	if err == nil {
		t.Fatal("expected unknown db driver to fail")
	}
	if !strings.Contains(err.Error(), "unknown db driver") {
		t.Fatalf("error should mention unknown db driver, got %v", err)
	}
}

func TestValidate_LocalAllowsLocalFS(t *testing.T) {
	s := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementLocal,
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   "data/blob",
		DBDriver:   settingsdomain.DriverSQLite,
		DBDSN:      "data/app.db",
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStore_SQLiteRoundTripSecrets(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:settings_rt_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 3)
	}
	st, err := settingsinfra.NewStore(gdb, key)
	if err != nil {
		t.Fatal(err)
	}
	in := settingsdomain.Settings{
		Placement:      settingsdomain.PlacementLocal,
		DBDriver:       settingsdomain.DriverSQLite,
		DBDSN:          filepath.Join(t.TempDir(), "app.db"),
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       "data/blob",
		ComfyUIBaseURL: "http://127.0.0.1:8188",
		BlobAccessKey:  "ak-secret",
		BlobSecretKey:  "sk-secret",
		ProxyKind:      settingsdomain.ProxyHTTP,
		ProxyHost:      "127.0.0.1",
		ProxyPort:      7897,
	}
	if err := st.Save(in); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Placement != settingsdomain.PlacementLocal {
		t.Fatalf("placement=%q", got.Placement)
	}
	wipe := got
	wipe.BlobAccessKey = ""
	wipe.BlobSecretKey = ""
	if err := st.Save(wipe); err != nil {
		t.Fatal(err)
	}
	again, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if again.BlobAccessKey != "ak-secret" || again.BlobSecretKey != "sk-secret" {
		t.Fatalf("empty save wiped blob secrets: %+v", again)
	}
	if got.BlobAccessKey != "ak-secret" || got.BlobSecretKey != "sk-secret" {
		t.Fatalf("blob secrets mismatch: %+v", got)
	}
	if got.ProxyKind != settingsdomain.ProxyHTTP || got.ProxyHost != "127.0.0.1" || got.ProxyPort != 7897 {
		t.Fatalf("proxy mismatch: %+v", got)
	}
	if got.DefaultUserAccess != "" && got.DefaultUserAccess != "denied" {
		t.Fatalf("unset default user access=%q", got.DefaultUserAccess)
	}
	in.DefaultUserAccess = "always_allowed"
	if err := st.Save(in); err != nil {
		t.Fatal(err)
	}
	got, err = st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultUserAccess != "always_allowed" {
		t.Fatalf("default user access=%q", got.DefaultUserAccess)
	}
}

func TestSettings_ProxyURLAndValidate(t *testing.T) {
	ok := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementLocal,
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   "data/blob",
		DBDriver:   settingsdomain.DriverSQLite,
		DBDSN:      "data/app.db",
		ProxyKind:  settingsdomain.ProxyHTTP,
		ProxyHost:  "127.0.0.1",
		ProxyPort:  7897,
	}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := ok.ProxyURL(); got != "http://127.0.0.1:7897" {
		t.Fatalf("ProxyURL=%q", got)
	}
	ok.ProxyKind = settingsdomain.ProxySOCKS
	if got := ok.ProxyURL(); got != "socks5://127.0.0.1:7897" {
		t.Fatalf("socks ProxyURL=%q", got)
	}
	ok.ProxyKind = settingsdomain.ProxyHTTP
	ok.ProxyPort = 0
	if err := ok.Validate(); err != nil {
		t.Fatalf("zero port should default to 7897, got %v", err)
	}
	if got := ok.ProxyURL(); got != "http://127.0.0.1:7897" {
		t.Fatalf("ProxyURL with default port=%q, want http://127.0.0.1:7897", got)
	}
	if got := ok.EffectiveProxyPort(); got != settingsdomain.DefaultProxyPort {
		t.Fatalf("EffectiveProxyPort=%d, want %d", got, settingsdomain.DefaultProxyPort)
	}
}

func TestValidate_SharedFSRequiresRoot(t *testing.T) {
	s := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementLocal,
		DBDriver:   settingsdomain.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverSharedFS,
	}
	err := s.Validate()
	if err == nil || !strings.Contains(err.Error(), "blob root") {
		t.Fatalf("want blob root error, got %v", err)
	}
}

func TestValidate_RemoteAllowsSharedFS(t *testing.T) {
	s := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementRemote,
		DBDriver:   settingsdomain.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverSharedFS,
		BlobRoot:   "/mnt/pixoma-shared",
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("remote sharedfs should pass, got %v", err)
	}
}

// legacySettingsRow mirrors the pre-upgrade platform_settings schema (all
// current columns EXCEPT the newly added allow_self_registration), so the
// migration test targets exactly the ALTER ADD that happens on real upgrades.
type legacySettingsRow struct {
	ID               string `gorm:"primaryKey;size:32"`
	Placement        string `gorm:"size:32;not null"`
	DBDriver         string `gorm:"column:db_driver;size:32;not null"`
	DBDSN            string `gorm:"column:db_dsn;type:text;not null"`
	BlobDriver       string `gorm:"column:blob_driver;size:32;not null"`
	BlobRoot         string `gorm:"column:blob_root;type:text"`
	BlobEndpoint     string `gorm:"column:blob_endpoint;type:text"`
	BlobRegion       string `gorm:"column:blob_region;size:64"`
	BlobBucket       string `gorm:"column:blob_bucket;size:256"`
	BlobAccessCipher string `gorm:"column:blob_access_cipher;type:text"`
	BlobSecretCipher string `gorm:"column:blob_secret_cipher;type:text"`
	ComfyUIBaseURL   string `gorm:"column:comfyui_base_url;type:text"`
	ClaimWaitMS      int    `gorm:"column:claim_wait_ms"`
	LeaseSeconds     int    `gorm:"column:lease_seconds"`
	ProxyKind        string `gorm:"column:proxy_kind;size:16"`
	ProxyHost        string `gorm:"column:proxy_host;type:text"`
	ProxyPort        int    `gorm:"column:proxy_port"`
}

func (legacySettingsRow) TableName() string { return "platform_settings" }

func TestMigrate_ExistingTableAddsAllowSelfRegistration(t *testing.T) {
	// Reproduce upgrading a deployment whose platform_settings table already
	// exists and has a row: AutoMigrate must add the new allow_self_registration
	// column without failing on SQLite (NOT NULL column needs a default).
	gdb, err := db.Open(db.Options{DSN: "file:mig_settings?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	defer gdb.DB()

	if err := gdb.AutoMigrate(&legacySettingsRow{}); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&legacySettingsRow{
		ID: "singleton", Placement: "local", DBDriver: "sqlite", DBDSN: "data/app.db",
	}).Error; err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 32)
	st, err := settingsinfra.NewStore(gdb, key)
	if err != nil {
		t.Fatalf("migrate existing table: %v", err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatalf("load after migrate: %v", err)
	}
	if got.AllowSelfRegistration {
		t.Fatal("expected AllowSelfRegistration to default to false")
	}
	if got.DefaultUserAccess != "" && got.DefaultUserAccess != "denied" {
		t.Fatalf("expected DefaultUserAccess denied, got %q", got.DefaultUserAccess)
	}
}

func TestValidate_MediaMaxBytes(t *testing.T) {
	base := settingsdomain.Settings{
		Placement:  settingsdomain.PlacementLocal,
		DBDriver:   settingsdomain.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   "data/blob",
	}
	cases := []struct {
		name      string
		bytes     int64
		wantError string
	}{
		{"zero means default", 0, ""},
		{"default 25 MiB", 25 * 1024 * 1024, ""},
		{"hundred MiB ok", 100 * 1024 * 1024, ""},
		{"min boundary", settingsdomain.MinMediaMaxBytes, ""},
		{"below min rejected", settingsdomain.MinMediaMaxBytes - 1, "media_max_bytes"},
		{"negative rejected", -1, "non-negative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			cfg.MediaMaxBytes = tc.bytes
			err := cfg.Validate()
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("expected ok, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error should contain %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestStore_RoundTripMediaMaxBytes(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:settings_media_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	defer gdb.DB()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 5)
	}
	st, err := settingsinfra.NewStore(gdb, key)
	if err != nil {
		t.Fatal(err)
	}
	in := settingsdomain.Settings{
		Placement:      settingsdomain.PlacementLocal,
		DBDriver:       settingsdomain.DriverSQLite,
		DBDSN:          "data/app.db",
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       "data/blob",
		ComfyUIBaseURL: "http://127.0.0.1:8188",
		MediaMaxBytes:  50 * 1024 * 1024,
	}
	if err := st.Save(in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.MediaMaxBytes != in.MediaMaxBytes {
		t.Fatalf("MediaMaxBytes round-trip: want %d, got %d", in.MediaMaxBytes, got.MediaMaxBytes)
	}

	// Saving with zero should clear the cap back to "use default".
	in.MediaMaxBytes = 0
	if err := st.Save(in); err != nil {
		t.Fatalf("save zero: %v", err)
	}
	got, err = st.Load()
	if err != nil {
		t.Fatalf("load after zero: %v", err)
	}
	if got.MediaMaxBytes != 0 {
		t.Fatalf("MediaMaxBytes after clearing: want 0, got %d", got.MediaMaxBytes)
	}
}

type legacyMediaSettingsRow struct {
	ID          string `gorm:"primaryKey;size:32"`
	Placement   string `gorm:"size:32;not null"`
	DBDriver    string `gorm:"column:db_driver;size:32;not null"`
	DBDSN       string `gorm:"column:db_dsn;type:text;not null"`
	BlobDriver  string `gorm:"column:blob_driver;size:32;not null"`
	BlobRoot    string `gorm:"column:blob_root;type:text"`
	ComfyUIBase string `gorm:"column:comfyui_base_url;type:text"`
}

func (legacyMediaSettingsRow) TableName() string { return "platform_settings" }

func TestMigrate_ExistingTableAddsMediaMaxBytes(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:mig_media?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	defer gdb.DB()
	if err := gdb.AutoMigrate(&legacyMediaSettingsRow{}); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&legacyMediaSettingsRow{
		ID: "singleton", Placement: "local",
		DBDriver: "sqlite", DBDSN: "data/app.db",
		BlobDriver: "localfs", BlobRoot: "data/blob",
		ComfyUIBase: "http://127.0.0.1:8188",
	}).Error; err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	st, err := settingsinfra.NewStore(gdb, key)
	if err != nil {
		t.Fatalf("migrate existing table: %v", err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatalf("load after migrate: %v", err)
	}
	if got.MediaMaxBytes != 0 {
		t.Fatalf("expected MediaMaxBytes to default to 0, got %d", got.MediaMaxBytes)
	}
}
