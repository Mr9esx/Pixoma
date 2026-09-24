//go:build integration

package studio_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/response"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/einoagent"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

const browserAccountID = "studio-browser-account"

type gatedMockEngine struct {
	release <-chan struct{}
}

func (e gatedMockEngine) Execute(ctx context.Context, request studioapp.AgentRequest, sink studioapp.AgentSink) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.release:
		return studioapp.NewMockEngine().Execute(ctx, request, sink)
	}
}

func TestStudioMockWorkflowBrowserJourney(t *testing.T) {
	if os.Getenv("PIXOMA_STUDIO_BROWSER_INTEGRATION") != "1" {
		t.Skip("set PIXOMA_STUDIO_BROWSER_INTEGRATION=1 to build and test the Studio browser journey")
	}
	frontendDir, distDir := buildBrowserFrontend(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	release := make(chan struct{})
	handler, runner := newHandlerWithEngine(t, gatedMockEngine{release: release})
	defer runner.Close()
	_, err := handler.Models.Create(context.Background(), studioapp.CreateModelConfigInput{
		AccountID:    browserAccountID,
		Name:         "Browser Test Model",
		Protocol:     domain.ModelProtocolOpenAIChat,
		BaseURL:      "http://127.0.0.1:1/v1",
		Model:        "test-only",
		APIKey:       "test-only",
		Enabled:      true,
		AgentEnabled: true,
		Default:      true,
		Limits: domain.ModelLimits{
			ContextWindowTokens: 8192,
			MaxInputTokens:      7000,
			MaxOutputTokens:     1024,
		},
		Capabilities: domain.ModelCapabilities{Tools: true},
	})
	if err != nil {
		t.Fatalf("create browser test model: %v", err)
	}
	server := newBrowserStudioServer(t, handler, distDir, release)
	defer server.Close()
	runBrowserScript(t, ctx, frontendDir, "scripts/studio-mock-workflow-smoke.mjs", server.URL, nil)
}

// This opt-in journey exercises the real Eino engine and the user's online
// provider, while keeping the database, Blob store and admin login isolated.
func TestStudioLiveBrowserJourney(t *testing.T) {
	if os.Getenv("PIXOMA_STUDIO_LIVE_BROWSER_INTEGRATION") != "1" {
		t.Skip("set PIXOMA_STUDIO_LIVE_BROWSER_INTEGRATION=1 and model credentials for the live Studio browser journey")
	}
	baseURL := os.Getenv("PIXOMA_STUDIO_MODEL_URL")
	modelID := os.Getenv("PIXOMA_STUDIO_MODEL_ID")
	apiKey := os.Getenv("PIXOMA_STUDIO_MODEL_API_KEY")
	if baseURL == "" || modelID == "" || apiKey == "" {
		t.Fatal("set PIXOMA_STUDIO_MODEL_URL, PIXOMA_STUDIO_MODEL_ID and PIXOMA_STUDIO_MODEL_API_KEY")
	}
	frontendDir, distDir := buildBrowserFrontend(t)
	ctx, cancel := context.WithTimeout(context.Background(), 210*time.Second)
	defer cancel()
	engine := &einoagent.Engine{Client: modelprovider.NewOpenAICompatibleClient(&http.Client{Timeout: 60 * time.Second})}
	handler, runner := newHandlerWithEngine(t, &studioapp.DispatchEngine{Online: engine})
	defer runner.Close()
	engine.Models = handler.Models
	engine.Capabilities = handler.Capabilities
	engine.Blob = handler.Blob
	repo, ok := handler.Repo.(*persistence.GormRepository)
	if !ok {
		t.Fatal("browser test needs a GORM Studio repository for Eino checkpoints")
	}
	engine.Checkpoints = repo.Checkpoints()
	_, err := handler.Models.Create(ctx, studioapp.CreateModelConfigInput{
		AccountID: browserAccountID, Name: "Live Browser Model", Protocol: domain.ModelProtocolOpenAIChat,
		BaseURL: baseURL, Model: modelID, APIKey: apiKey, Enabled: true, AgentEnabled: true, Default: true,
		Limits:       domain.ModelLimits{ContextWindowTokens: 8192, MaxInputTokens: 7000, MaxOutputTokens: 1024},
		Capabilities: domain.ModelCapabilities{Tools: true, Streaming: true},
	})
	if err != nil {
		t.Fatalf("create live browser model config: %v", err)
	}
	server := newBrowserStudioServer(t, handler, distDir, nil)
	defer server.Close()
	runBrowserScript(t, ctx, frontendDir, "scripts/studio-live-smoke.mjs", server.URL, []string{
		"PIXOMA_STUDIO_FRONTEND_URL=" + server.URL,
		"PIXOMA_STUDIO_SMOKE_USER=studio-browser",
		"PIXOMA_STUDIO_SMOKE_PASSWORD=test-only",
	})
}

func buildBrowserFrontend(t *testing.T) (string, string) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate browser integration test")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../.."))
	frontendDir := filepath.Join(repoRoot, "web/admin")
	distDir := filepath.Join(repoRoot, "web/admin/dist")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "pnpm", "build")
	build.Dir = frontendDir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build current Studio frontend: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		t.Fatalf("built Studio frontend has no index.html: %v", err)
	}
	return frontendDir, distDir
}

func newBrowserStudioServer(t *testing.T, handler *studioapi.Handler, distDir string, releaseMock chan struct{}) *httptest.Server {
	t.Helper()
	router := chi.NewRouter()
	if releaseMock != nil {
		var releaseOnce sync.Once
		router.Post("/__test__/release-mock-run", func(w http.ResponseWriter, _ *http.Request) {
			releaseOnce.Do(func() {
				close(releaseMock)
			})
			response.OK(w, map[string]any{"released": true})
		})
	}
	router.Get("/api/v1/setup/status", func(w http.ResponseWriter, _ *http.Request) {
		response.OK(w, map[string]any{
			"initialized": true, "authenticated": true, "must_change_password": false,
		})
	})
	router.Get("/api/v1/setup/me", func(w http.ResponseWriter, _ *http.Request) {
		response.OK(w, map[string]any{
			"username": "studio-browser", "nickname": "Studio Browser", "role": "admin",
		})
	})
	router.Post("/api/v1/setup/login", func(w http.ResponseWriter, _ *http.Request) {
		response.OK(w, map[string]any{"username": "studio-browser", "role": "admin"})
	})
	router.Route("/api/v1/studio", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				account := setupapi.AccountSession{AccountID: browserAccountID, Username: "studio-browser", Role: "admin"}
				next.ServeHTTP(w, req.WithContext(setupapi.WithAccount(req.Context(), account)))
			})
		})
		handler.Mount(r)
	})
	router.Handle("/assets/*", http.FileServer(http.Dir(distDir)))
	router.Get("/studio", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})
	return httptest.NewServer(router)
}

func runBrowserScript(t *testing.T, ctx context.Context, frontendDir, script, serverURL string, extraEnv []string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "node", script)
	cmd.Dir = frontendDir
	env := make([]string, 0, len(os.Environ())+len(extraEnv)+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "PIXOMA_STUDIO_MODEL_API_KEY=") {
			env = append(env, entry)
		}
	}
	cmd.Env = append(env, append([]string{"PIXOMA_STUDIO_MOCK_FRONTEND_URL=" + serverURL}, extraEnv...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser workflow journey failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}
	t.Log(strings.TrimSpace(string(output)))
}
