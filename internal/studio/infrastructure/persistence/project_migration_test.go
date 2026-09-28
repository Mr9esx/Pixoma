package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/appboot"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type previousStudioSessionRow struct {
	ID                             string `gorm:"primaryKey;size:64"`
	AccountID                      string `gorm:"size:64;not null"`
	Title                          string `gorm:"size:256;not null"`
	PermissionMode                 string `gorm:"size:32;not null"`
	ModelConfigID                  string `gorm:"size:64"`
	ContextSummary                 string `gorm:"type:text"`
	ContextSummaryThroughMessageID string `gorm:"size:64"`
	Status                         string `gorm:"size:32;not null"`
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

func (previousStudioSessionRow) TableName() string { return "studio_sessions" }

func TestProjectMigrationKeepsExistingSessionsInRecent(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&previousStudioSessionRow{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	legacy := previousStudioSessionRow{ID: "existing", AccountID: "account-a", Title: "已有对话", PermissionMode: "request_approval", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := gdb.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	recent := ""
	sessions, err := persistence.NewGormRepository(gdb).ListSessions(context.Background(), "account-a", domain.SessionListQuery{ProjectID: &recent, Limit: 30})
	if err != nil || len(sessions) != 1 || sessions[0].ID != "existing" {
		t.Fatalf("recent after migration = %v, %v", sessions, err)
	}
}

func TestProjectMigrationRepairsNullableProjectIDs(t *testing.T) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.AutoMigrate(&previousStudioSessionRow{}); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("ALTER TABLE studio_sessions ADD COLUMN project_id TEXT").Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	legacy := previousStudioSessionRow{ID: "existing", AccountID: "account-a", Title: "已有对话", PermissionMode: "request_approval", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := gdb.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	assigned := legacy
	assigned.ID = "assigned"
	if err := gdb.Create(&assigned).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Table("studio_sessions").Where("id = ?", assigned.ID).UpdateColumn("project_id", "project-1").Error; err != nil {
		t.Fatal(err)
	}
	migrated, cleanup, err := appboot.Bootstrap(context.Background(), appboot.Options{
		DSN:           dsn,
		Models:        persistence.Models(),
		BeforeMigrate: persistence.MigrateSessionProjectIDs,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cleanup() })
	recent := ""
	sessions, err := persistence.NewGormRepository(migrated).ListSessions(context.Background(), "account-a", domain.SessionListQuery{ProjectID: &recent, Limit: 30})
	if err != nil || len(sessions) != 1 || sessions[0].ID != "existing" {
		t.Fatalf("recent after migration = %v, %v", sessions, err)
	}
	projectID := "project-1"
	sessions, err = persistence.NewGormRepository(migrated).ListSessions(context.Background(), "account-a", domain.SessionListQuery{ProjectID: &projectID, Limit: 5})
	if err != nil || len(sessions) != 1 || sessions[0].ID != "assigned" {
		t.Fatalf("project after migration = %v, %v", sessions, err)
	}
	_, cleanupSecond, err := appboot.Bootstrap(context.Background(), appboot.Options{DSN: dsn, Models: persistence.Models(), BeforeMigrate: persistence.MigrateSessionProjectIDs})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cleanupSecond() })
}
