package settings_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
)

func TestValidate_RemoteRejectsLocalFS(t *testing.T) {
	s := settings.Settings{
		Placement:  settings.PlacementRemote,
		BlobDriver: botconfig.BlobDriverLocalFS,
		DBDriver:   settings.DriverSQLite,
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
	s := settings.Settings{
		Placement:  settings.PlacementLocal,
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
	s := settings.Settings{
		Placement:  settings.PlacementLocal,
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   "data/blob",
		DBDriver:   settings.DriverSQLite,
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
	st, err := settings.NewStore(gdb, key)
	if err != nil {
		t.Fatal(err)
	}
	in := settings.Settings{
		Placement:      settings.PlacementLocal,
		DBDriver:       settings.DriverSQLite,
		DBDSN:          filepath.Join(t.TempDir(), "app.db"),
		BlobDriver:     botconfig.BlobDriverLocalFS,
		BlobRoot:       "data/blob",
		ComfyMock:      true,
		ComfyUIBaseURL: "http://127.0.0.1:8188",
		BlobAccessKey:  "ak-secret",
		BlobSecretKey:  "sk-secret",
		ProxyKind:      settings.ProxyHTTP,
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
	if got.Placement != settings.PlacementLocal {
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
	if got.ProxyKind != settings.ProxyHTTP || got.ProxyHost != "127.0.0.1" || got.ProxyPort != 7897 {
		t.Fatalf("proxy mismatch: %+v", got)
	}
	if !got.ComfyMock {
		t.Fatal("comfy_mock should round-trip")
	}
}

func TestSettings_ProxyURLAndValidate(t *testing.T) {
	ok := settings.Settings{
		Placement:  settings.PlacementLocal,
		BlobDriver: botconfig.BlobDriverLocalFS,
		BlobRoot:   "data/blob",
		DBDriver:   settings.DriverSQLite,
		DBDSN:      "data/app.db",
		ProxyKind:  settings.ProxyHTTP,
		ProxyHost:  "127.0.0.1",
		ProxyPort:  7897,
	}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := ok.ProxyURL(); got != "http://127.0.0.1:7897" {
		t.Fatalf("ProxyURL=%q", got)
	}
	ok.ProxyKind = settings.ProxySOCKS
	if got := ok.ProxyURL(); got != "socks5://127.0.0.1:7897" {
		t.Fatalf("socks ProxyURL=%q", got)
	}
	ok.ProxyKind = settings.ProxyHTTP
	ok.ProxyPort = 0
	if err := ok.Validate(); err == nil {
		t.Fatal("expected invalid port")
	}
}

func TestValidate_SharedFSRequiresRoot(t *testing.T) {
	s := settings.Settings{
		Placement:  settings.PlacementLocal,
		DBDriver:   settings.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverSharedFS,
	}
	err := s.Validate()
	if err == nil || !strings.Contains(err.Error(), "blob root") {
		t.Fatalf("want blob root error, got %v", err)
	}
}

func TestValidate_RemoteAllowsSharedFS(t *testing.T) {
	s := settings.Settings{
		Placement:  settings.PlacementRemote,
		DBDriver:   settings.DriverSQLite,
		DBDSN:      "data/app.db",
		BlobDriver: botconfig.BlobDriverSharedFS,
		BlobRoot:   "/mnt/pixoma-shared",
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("remote sharedfs should pass, got %v", err)
	}
}
