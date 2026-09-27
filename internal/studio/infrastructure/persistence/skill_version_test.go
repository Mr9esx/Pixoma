package persistence_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type legacySkillRow struct {
	ID          string `gorm:"primaryKey;size:64"`
	AccountID   string
	Name        string
	Description string
	Prompt      string
	FilesJSON   []byte
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (legacySkillRow) TableName() string { return "studio_skills" }

func TestMigrateSkillVersionsPreservesExistingContent(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := gdb.AutoMigrate(&legacySkillRow{}); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(24 * time.Hour)
	files := []domain.SkillFile{{Path: "SKILL.md", Content: "Legacy content"}}
	filesJSON, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&legacySkillRow{
		ID: "legacy-skill", AccountID: "account-a", Name: "legacy", Description: "Legacy skill",
		Prompt: "Legacy content", FilesJSON: filesJSON, Enabled: true, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	if err := persistence.MigrateSkillVersions(context.Background(), gdb); err != nil {
		t.Fatal(err)
	}
	if err := persistence.MigrateSkillVersions(context.Background(), gdb); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	repo := persistence.NewGormRepository(gdb)
	current, err := repo.GetSkill(context.Background(), "account-a", "legacy-skill")
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != "1.0.0" || current.Prompt != "Legacy content" {
		t.Fatalf("current = %#v", current)
	}
	versions, err := repo.ListSkillVersions(context.Background(), "account-a", "legacy-skill")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].Version != "1.0.0" {
		t.Fatalf("versions = %#v", versions)
	}
	old, err := repo.GetSkillVersion(context.Background(), "account-a", "legacy-skill", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if old.Prompt != "Legacy content" || len(old.Files) != 1 || old.Files[0].Content != "Legacy content" {
		t.Fatalf("snapshot = %#v", old)
	}
}

func TestSkillVersionSaveRollsBackWhenVersionAlreadyExists(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	skill, err := domain.NewSkill("skill-1", "account-a", "storyboard", "Make storyboards", "Original prompt", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&persistence.SkillVersionRow{
		SkillID: skill.ID, AccountID: skill.AccountID, Version: "1.0.1",
		Name: skill.Name, Description: skill.Description, Prompt: "Reserved version", CreatedAt: now.Add(time.Second),
	}).Error; err != nil {
		t.Fatal(err)
	}
	skill.Version = "1.0.1"
	skill.Prompt = "Changed prompt"
	skill.UpdatedAt = now.Add(time.Second)
	if err := repo.UpdateSkillIfUnchanged(ctx, skill, now, "1.0.0"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("save error = %v", err)
	}
	current, err := repo.GetSkill(ctx, skill.AccountID, skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version != "1.0.0" || current.Prompt != "Original prompt" {
		t.Fatalf("current = %#v", current)
	}
}
