package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/app"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge/static"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue/memory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestAgentTokenFile_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.token")
	if err := app.WriteAgentTokenFile(path, "plain-token"); err != nil {
		t.Fatal(err)
	}
	got, err := app.ReadAgentTokenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "plain-token" {
		t.Fatalf("got %q", got)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("token file too open: %v", st.Mode())
	}
}

func TestApplyBlobEnv_SetsS3Connection(t *testing.T) {
	for _, k := range []string{"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY", "S3_SECRET_KEY"} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	app.ApplyBlobEnv(settings.Settings{
		BlobDriver:    botconfig.BlobDriverS3,
		BlobEndpoint:  "http://minio:9000",
		BlobRegion:    "us-east-1",
		BlobBucket:    "from-wizard",
		BlobAccessKey: "ak",
		BlobSecretKey: "sk",
	})
	if os.Getenv("S3_ENDPOINT") != "http://minio:9000" {
		t.Fatalf("S3_ENDPOINT=%q", os.Getenv("S3_ENDPOINT"))
	}
	if os.Getenv("S3_BUCKET") != "from-wizard" {
		t.Fatalf("S3_BUCKET=%q", os.Getenv("S3_BUCKET"))
	}
	if os.Getenv("S3_ACCESS_KEY") != "ak" || os.Getenv("S3_SECRET_KEY") != "sk" {
		t.Fatal("s3 secrets not applied")
	}
}

func TestApplyHTTPProxy_FromSettingsWhenEnvEmpty(t *testing.T) {
	for _, k := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "NO_PROXY", "no_proxy", "ALL_PROXY", "all_proxy",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	app.ApplyHTTPProxy(settings.Settings{
		ProxyKind: settings.ProxyHTTP,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7897,
	})
	if os.Getenv("HTTPS_PROXY") != "http://127.0.0.1:7897" {
		t.Fatalf("HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
	if os.Getenv("HTTP_PROXY") != "http://127.0.0.1:7897" {
		t.Fatalf("HTTP_PROXY=%q", os.Getenv("HTTP_PROXY"))
	}
	if !strings.Contains(os.Getenv("NO_PROXY"), "127.0.0.1") {
		t.Fatalf("NO_PROXY=%q", os.Getenv("NO_PROXY"))
	}
}

func TestApplyHTTPProxy_Socks5(t *testing.T) {
	for _, k := range []string{
		"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy", "NO_PROXY", "no_proxy", "ALL_PROXY", "all_proxy",
	} {
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	app.ApplyHTTPProxy(settings.Settings{
		ProxyKind: settings.ProxySOCKS,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7897,
	})
	if os.Getenv("ALL_PROXY") != "socks5://127.0.0.1:7897" {
		t.Fatalf("ALL_PROXY=%q", os.Getenv("ALL_PROXY"))
	}
	if os.Getenv("HTTPS_PROXY") != "socks5://127.0.0.1:7897" {
		t.Fatalf("HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
}

func TestApplyHTTPProxy_KeepsExistingEnv(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:8888")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")
	t.Setenv("NO_PROXY", "")
	_ = os.Unsetenv("NO_PROXY")
	_ = os.Unsetenv("no_proxy")
	app.ApplyHTTPProxy(settings.Settings{
		ProxyKind: settings.ProxyHTTP,
		ProxyHost: "127.0.0.1",
		ProxyPort: 7897,
	})
	if os.Getenv("HTTPS_PROXY") != "http://127.0.0.1:8888" {
		t.Fatalf("HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
	if !strings.Contains(os.Getenv("NO_PROXY"), "192.168.0.0/16") {
		t.Fatalf("NO_PROXY=%q", os.Getenv("NO_PROXY"))
	}
}

func TestLoadVerifiedAgentToken_RejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	st, _, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tok, minted, err := st.EnsureAgentToken()
	if err != nil || !minted {
		t.Fatalf("minted=%v err=%v", minted, err)
	}
	path := filepath.Join(dir, "agent.token")
	if err := app.WriteAgentTokenFile(path, "not-the-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.LoadVerifiedAgentToken(st, path); err == nil {
		t.Fatal("expected mismatch error")
	}
	if err := app.WriteAgentTokenFile(path, tok); err != nil {
		t.Fatal(err)
	}
	got, err := app.LoadVerifiedAgentToken(st, path)
	if err != nil {
		t.Fatal(err)
	}
	if got != tok {
		t.Fatalf("got %q", got)
	}
}

func TestSubscribeTaskCreated_MakesClaimable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Unix(1000, 0).UTC()
	tasks := runtimedomain.NewMemoryTaskRepository()
	if err := tasks.Create(ctx, runtimedomain.NewPending("t-bus", "s", sharedkernel.CaseID(1), "in", now)); err != nil {
		t.Fatal(err)
	}
	reg := static.New(edge.Instance{ID: "local", SubscribeTopics: []string{"default"}})
	orch := orchestrator.New(tasks, reg, nil, notify.Nop{})
	orch.Now = func() time.Time { return now }
	orch.Prep = jobPrep{ref: sharedkernel.BlobRef{Key: "jobs/t-bus/job.json"}}
	bus := memory.New()
	if err := app.SubscribeTaskCreated(ctx, bus, orch); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(sharedkernel.TaskCreated{TaskID: "t-bus", CreatedAt: now})
	if err := bus.Publish(ctx, queue.Message{Topic: sharedkernel.TopicTaskCreated, Key: "t-bus", Payload: payload}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := tasks.Get(ctx, "t-bus")
		if err == nil && got.Status == sharedkernel.TaskQueued && got.JobRef.Key == "jobs/t-bus/job.json" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	got, _ := tasks.Get(ctx, "t-bus")
	t.Fatalf("task not claimable: %+v", got)
}

type jobPrep struct {
	ref sharedkernel.BlobRef
}

func (j jobPrep) PrepareJob(context.Context, sharedkernel.TaskID) (sharedkernel.BlobRef, error) {
	return j.ref, nil
}

func TestBootstrapBannerContract_FirstOpen(t *testing.T) {
	dir := t.TempDir()
	st, creds, err := bootstrap.Open(filepath.Join(dir, "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tok, minted, err := st.EnsureAgentToken()
	if err != nil || !minted || tok == "" {
		t.Fatalf("token minted=%v err=%v", minted, err)
	}
	if err := app.WriteAgentTokenFile(filepath.Join(dir, "agent.token"), tok); err != nil {
		t.Fatal(err)
	}
	msg := app.StartupBanner(app.BannerInput{
		ListenURL: "http://127.0.0.1:8080",
		Username:  creds.Username,
		Password:  creds.Password,
	})
	for _, want := range []string{"http://127.0.0.1:8080", creds.Username, creds.Password} {
		if !strings.Contains(msg, want) {
			t.Fatalf("banner missing %q:\n%s", want, msg)
		}
	}
	h := adminhost.NewHandler(adminhost.Options{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("healthz=%d %q", rec.Code, rec.Body.String())
	}
}
