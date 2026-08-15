package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mr9esx/comfyui_tgbot/apps/pixoma/internal/app"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/validation"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	agentapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/agent"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/adminhost"
	casesapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/cases"
	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/comfyinstances"
	sessionsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/sessions"
	tasksapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tasks"
	tgmenuapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/tgmenu"
	usersapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/users"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/appboot"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/blob/factory"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/botconfig"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/instance"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/instance/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	tgmenuapp "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/application"
	tgmenupersist "github.com/mr9esx/comfyui_tgbot/internal/tgmenu/infrastructure/persistence"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("pixoma failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dataDir := envOr("DATA_DIR", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	bootPath := filepath.Join(dataDir, "bootstrap.db")
	boot, creds, err := bootstrap.Open(bootPath)
	if err != nil {
		return err
	}
	defer boot.Close()

	agentTok, minted, err := boot.EnsureAgentToken()
	if err != nil {
		return err
	}
	tokenFile := filepath.Join(dataDir, "agent.token")
	if minted {
		if err := app.WriteAgentTokenFile(tokenFile, agentTok); err != nil {
			return err
		}
		slog.Info("minted agent token", "path", tokenFile)
	} else {
		agentTok, err = app.ReadAgentTokenFile(tokenFile)
		if err != nil {
			return fmt.Errorf("agent token hash exists but %s unreadable: %w (delete bootstrap to remint)", tokenFile, err)
		}
	}

	addr := envOr("HTTP_ADDR", "127.0.0.1:8080")
	listenURL := envOr("PUBLIC_URL", "http://"+addr)
	fmt.Print(app.StartupBanner(app.BannerInput{
		ListenURL: listenURL,
		Username:  creds.Username,
		Password:  creds.Password,
	}))

	appDSN := filepath.Join(dataDir, "app.db")
	blobRoot := filepath.Join(dataDir, "blob")
	comfyMock := envBool("COMFY_MOCK", true)

	gdb, cleanup, err := appboot.Bootstrap(ctx, appboot.Options{
		DSN:              appDSN,
		MigrateInstances: true,
		Models: []any{
			&casepersist.CaseRow{},
			&userpersist.UserRow{},
			&sesspersist.SessionRow{},
			&taskpersist.TaskRow{},
			&tgmenupersist.MenuHeaderRow{},
			&tgmenupersist.MenuItemRow{},
			&tgmenupersist.MenuItemCaseRow{},
		},
		Seed: &instance.SeedConfig{
			DefaultInstanceID: "local",
			ComfyUIBaseURL:    envOr("COMFYUI_BASE_URL", "http://127.0.0.1:8188"),
			ComfyMock:         comfyMock,
		},
	})
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	blobStore, err := factory.New(botconfig.BlobDriverLocalFS, blobRoot)
	if err != nil {
		return err
	}

	instRepo := instpersist.NewInstanceRepository(gdb)
	pool := instance.NewPool(instRepo, instance.PoolOptions{Mock: comfyMock})
	if err := pool.Refresh(ctx); err != nil {
		return err
	}
	caseRepo := casepersist.NewGormRepository(gdb)
	userRepo := userpersist.NewUserRepository(gdb)
	sessionRepo := sesspersist.NewSessionRepository(gdb)
	taskRepo := taskpersist.NewTaskRepository(gdb)
	menuStore := tgmenupersist.NewGormRepository(gdb)
	menuSvc := &tgmenuapp.Service{
		Store:            menuStore,
		Cases:            tgmenuapp.CatalogCaseChecker{Repo: caseRepo},
		ListImageCaseIDs: tgmenuapp.CatalogImageCaseIDs(caseRepo),
	}

	snap := &actuator.CaseSnapshot{
		Tasks: taskRepo,
		Cases: caseRepo,
		Blob:  blobStore,
	}
	orch := orchestrator.New(taskRepo, pool, nil, notify.Nop{})
	orch.Sessions = sessionRepo
	orch.Prep = snap
	orch.Now = func() time.Time { return time.Now().UTC() }

	adminH := adminhost.NewHandler(adminhost.Options{
		Instances: &comfyinstances.Handler{Repo: instRepo, Pool: pool, Tasks: taskRepo, Mock: comfyMock},
		Cases:     &casesapi.Handler{Repo: caseRepo, Validate: validation.New().ValidateDocument},
		Users:     &usersapi.Handler{Repo: userRepo},
		Sessions:  &sessionsapi.Handler{Repo: sessionRepo},
		Tasks:     &tasksapi.Handler{Tasks: taskRepo, Cancel: orch},
		TGMenu:    &tgmenuapi.Handler{Svc: menuSvc},
	})

	agentH := &agentapi.Handler{
		Token:  agentTok,
		Tasks:  taskRepo,
		Status: orch,
		Lease:  90 * time.Second,
	}

	r := chi.NewRouter()
	r.Mount("/", adminH)
	r.Route("/agent/v1", agentH.Mount)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: r, ReadHeaderTimeout: 5 * time.Second}

	var edgeCmd = (*os.Process)(nil)
	if envBool("EDGE_AUTO_SPAWN", true) {
		cmd := app.EdgeCommand(app.EdgeSpawnConfig{
			Binary:          app.ResolveEdgeBinary(),
			ControlPlaneURL: listenURL,
			AgentToken:      agentTok,
			InstanceID:      "local",
			BlobDriver:      botconfig.BlobDriverLocalFS,
			BlobRoot:        blobRoot,
			ComfyMock:       comfyMock,
		})
		if err := cmd.Start(); err != nil {
			slog.Warn("edge auto-spawn failed (install pixoma-edge-agent or set EDGE_AGENT_BIN)", "err", err)
		} else {
			edgeCmd = cmd.Process
			slog.Info("spawned local edge", "pid", cmd.Process.Pid, "bin", cmd.Path)
			go func() { _ = cmd.Wait() }()
		}
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("pixoma listening", "addr", addr, "data_dir", dataDir, "comfy_mock", comfyMock)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		if edgeCmd != nil {
			_ = edgeCmd.Signal(syscall.SIGTERM)
			_, _ = edgeCmd.Wait()
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if edgeCmd != nil {
			_ = edgeCmd.Signal(syscall.SIGTERM)
		}
		return err
	}
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}