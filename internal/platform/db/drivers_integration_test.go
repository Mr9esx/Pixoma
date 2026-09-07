//go:build integration

package db_test

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	channelpersist "github.com/mr9esx/comfyui_tgbot/internal/channel/infrastructure/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	userpersist "github.com/mr9esx/comfyui_tgbot/internal/identity/infrastructure/persistence"
	mencardpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	statspersist "github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	taskdomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	runtimepersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestIntegration_FullMigrateAndRoundtrip(t *testing.T) {
	drivers := []struct {
		driver string
		env    string
	}{
		{db.DriverMySQL, "PIXOMA_MYSQL_DSN"},
		{db.DriverPostgres, "PIXOMA_POSTGRES_DSN"},
	}
	for _, tc := range drivers {
		t.Run(tc.driver, func(t *testing.T) {
			gdb := openEnvDB(t, tc.driver, tc.env)
			if err := db.AutoMigrate(gdb, allBusinessModels()...); err != nil {
				t.Fatalf("migrate all: %v", err)
			}
			exerciseCoreRoundtrip(t, gdb)
		})
	}
}

func TestIntegration_EnsureDatabaseCreatesMissingDB(t *testing.T) {
	drivers := []struct {
		driver string
		env    string
	}{
		{db.DriverMySQL, "PIXOMA_MYSQL_DSN"},
		{db.DriverPostgres, "PIXOMA_POSTGRES_DSN"},
	}
	for _, tc := range drivers {
		t.Run(tc.driver, func(t *testing.T) {
			base := strings.TrimSpace(os.Getenv(tc.env))
			if base == "" {
				t.Skipf("set %s to run", tc.env)
			}
			name := "pixoma_ensure_" + strconv.FormatInt(time.Now().UnixNano(), 36)
			dsn := withDatabaseName(tc.driver, base, name)
			t.Cleanup(func() {
				if err := db.DropDatabase(tc.driver, dsn); err != nil {
					t.Errorf("drop database %s: %v", name, err)
				}
			})
			if err := db.EnsureDatabase(tc.driver, dsn); err != nil {
				t.Fatalf("ensure database: %v", err)
			}
			gdb, err := db.Open(db.Options{Driver: tc.driver, DSN: dsn})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := gdb.DB()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = sqlDB.Close() }()
			if err := sqlDB.Ping(); err != nil {
				t.Fatal(err)
			}
			if err := db.AutoMigrate(gdb, &pingRow{}); err != nil {
				t.Fatalf("migrate after ensure: %v", err)
			}
		})
	}
}

func withDatabaseName(driver, dsn, name string) string {
	switch driver {
	case db.DriverMySQL:
		cfg, err := gomysql.ParseDSN(dsn)
		if err != nil {
			panic(err)
		}
		cfg.DBName = name
		return cfg.FormatDSN()
	case db.DriverPostgres:
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			panic(err)
		}
		cfg.Database = name
		return cfg.ConnString()
	default:
		return dsn
	}
}

type pingRow struct {
	ID string `gorm:"primaryKey;size:8"`
}

func (pingRow) TableName() string { return "pixoma_integration_ping" }

func openEnvDB(t *testing.T, driver, env string) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(env))
	if dsn == "" {
		t.Skipf("set %s to run", env)
	}
	gdb, err := db.Open(db.Options{Driver: driver, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, e := gdb.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return gdb
}

func allBusinessModels() []any {
	return []any{
		&casepersist.CaseRow{},
		&userpersist.UserRow{},
		&userpersist.UserExternalIdentityRow{},
		&sesspersist.SessionRow{},
		&runtimepersist.TaskRow{},
		&statspersist.DailyStatsRow{},
		&statspersist.EdgeDailyStatsRow{},
		&statspersist.ErrorDailyStatsRow{},
		&statspersist.CaseDailyStatsRow{},
		&topicpersist.TopicRow{},
		&channelpersist.ChannelRow{},
		&mencardpersist.MainMenuRow{},
		&mencardpersist.CardRow{},
		&instpersist.EdgeRow{},
		&instpersist.MetricsRow{},
	}
}

func exerciseCoreRoundtrip(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Unix(1000, 0).UTC()

	// settings roundtrip
	key := bytes.Repeat([]byte{7}, 32)
	st, err := settings.NewStore(gdb, key)
	if err != nil {
		t.Fatalf("settings store: %v", err)
	}
	want := settings.Settings{
		Placement:      settings.PlacementLocal,
		DBDriver:       settings.DriverSQLite,
		DBDSN:          "data/app.db",
		BlobDriver:     "localfs",
		BlobRoot:       "data/blob",
		ComfyUIBaseURL: "http://127.0.0.1:8188",
	}
	if err := st.Save(want); err != nil {
		t.Fatalf("settings save: %v", err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatalf("settings load: %v", err)
	}
	if got.BlobRoot != want.BlobRoot || got.ComfyUIBaseURL != want.ComfyUIBaseURL {
		t.Fatalf("settings roundtrip mismatch: %+v", got)
	}

	// topic roundtrip
	tRepo := topicpersist.NewTopicRepository(gdb)
	tRow := topic.Topic{
		Key:       "integration",
		Name:      "Integration Topic",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tRepo.Create(ctx, tRow); err != nil {
		t.Fatalf("topic create: %v", err)
	}
	tgot, err := tRepo.Get(ctx, "integration")
	if err != nil || tgot.Name != "Integration Topic" {
		t.Fatalf("topic get: %+v err=%v", tgot, err)
	}

	// case roundtrip
	cRepo := casepersist.NewGormRepository(gdb)
	c := &catalogdomain.Case{
		Document: catalogdomain.CaseDocument{
			ID:      sharedkernel.CaseID(1),
			Name:    "Integration Case",
			Inputs:  []catalogdomain.InputField{},
			Outputs: []catalogdomain.OutputField{},
			Bindings: catalogdomain.ComfyBindings{
				WorkflowJSON: map[string]any{},
				Inputs:       []catalogdomain.InputBinding{},
				Outputs:      []catalogdomain.OutputBinding{},
			},
			InputSchema: map[string]any{"type": "object"},
		},
		Enabled: true,
	}
	if err := cRepo.Create(ctx, c); err != nil {
		t.Fatalf("case create: %v", err)
	}
	cgot, err := cRepo.Get(ctx, sharedkernel.CaseID(1))
	if err != nil || cgot.Document.Name != "Integration Case" {
		t.Fatalf("case get: %+v err=%v", cgot, err)
	}

	// task roundtrip: explicit topic + claim (writes NULL timestamps on prepare)
	tasks := runtimepersist.NewTaskRepository(gdb)
	ref := sharedkernel.BlobRef{Key: "jobs/t-int/job.json"}
	task := taskdomain.NewPending("t-int", "s-int", sharedkernel.CaseID(1), "inputs/t-int", now)
	_ = task.PrepareForTopic("integration", ref, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatalf("task create: %v", err)
	}
	claimed, err := tasks.ClaimNextWithLease(ctx, "gpu-int", []string{"integration"}, 90*time.Second, now)
	if err != nil || claimed == nil || claimed.ID != "t-int" {
		t.Fatalf("claim: %+v err=%v", claimed, err)
	}
	gotTask, err := tasks.Get(ctx, "t-int")
	if err != nil || gotTask.Status != sharedkernel.TaskRunning {
		t.Fatalf("task get: %+v err=%v", gotTask, err)
	}

	// stats upsert (clause.OnConflict) exercised twice on the same day
	statsRepo := statspersist.NewGormStatsRepository(gdb, 24*time.Hour, time.UTC)
	for i := 0; i < 2; i++ {
		if err := statsRepo.AddTerminal(ctx, taskstats.AddTerminalInput{
			EdgeID:          "gpu-int",
			Status:          taskstats.StatusSucceeded,
			CompletedAt:     now,
			CreatedAt:       now.Add(-time.Minute),
			CaseID:          1,
			QueueDurationMS: 10,
			ExecDurationMS:  20,
		}); err != nil {
			t.Fatalf("stats add %d: %v", i, err)
		}
	}

}
