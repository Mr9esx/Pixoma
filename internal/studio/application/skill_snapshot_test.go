package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func TestSnapshotSkillsIncludesNewEnabledSkillsOnNextRead(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:skill_snapshot_test?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, err := gdb.DB()
		if err != nil {
			t.Error(err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		id      string
		account string
		enabled bool
	}{
		{id: "skill-a", account: "account-a", enabled: true},
		{id: "skill-disabled", account: "account-a", enabled: false},
		{id: "skill-other-account", account: "account-b", enabled: true},
	} {
		skill, err := domain.NewSkill(item.id, item.account, item.id, item.id+" description", item.id+" prompt", now)
		if err != nil {
			t.Fatal(err)
		}
		skill.Enabled = item.enabled
		if err := repo.CreateSkill(ctx, skill); err != nil {
			t.Fatal(err)
		}
	}
	service := &Service{Repo: repo}
	first, err := service.snapshotSkills(ctx, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].ID != "skill-a" || first[0].Prompt != "skill-a prompt" {
		t.Fatalf("initial Skill snapshot = %#v", first)
	}
	if _, err := resolveSkillIDs([]string{"skill-disabled"}, first); err == nil {
		t.Fatal("disabled Skill was accepted")
	}
	newSkill, err := domain.NewSkill("skill-new", "account-a", "new", "new description", "new prompt", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	newSkill.Enabled = true
	if err := repo.CreateSkill(ctx, newSkill); err != nil {
		t.Fatal(err)
	}
	second, err := service.snapshotSkills(ctx, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 2 || second[1].ID != "skill-new" || second[1].Prompt != "new prompt" {
		t.Fatalf("next Skill snapshot = %#v", second)
	}
	if len(first) != 1 {
		t.Fatalf("previous Skill snapshot changed: %#v", first)
	}
	oldSkill, err := repo.GetSkill(ctx, "account-a", "skill-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetSkillEnabled(ctx, oldSkill.AccountID, oldSkill.ID, false); err != nil {
		t.Fatal(err)
	}
	third, err := service.snapshotSkills(ctx, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(third) != 1 || third[0].ID != "skill-new" {
		t.Fatalf("Skill snapshot after disabling = %#v", third)
	}
	selected, err := (&AgentExecutor{repo: repo}).selectedSkills(ctx, &domain.Run{
		AccountID: "account-a", SkillIDs: []string{"skill-a"}, SkillSnapshot: first,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].Prompt != "skill-a prompt" {
		t.Fatalf("selected Skill from prior Run snapshot = %#v", selected)
	}
}

func TestSkillSnapshotKeepsPackageFilesAcrossRunPersistence(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:skill_package_snapshot_test?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, dbErr := gdb.DB()
		if dbErr != nil {
			t.Error(dbErr)
			return
		}
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 9, 0, 0, 0, time.UTC)
	skill, err := domain.NewSkill("skill-package", "account-a", "storyboard", "编排镜头", "拼接内容", now)
	if err != nil {
		t.Fatal(err)
	}
	skill.Enabled = true
	skill.Files = []domain.SkillFile{
		{Path: "SKILL.md", Content: "---\nname: storyboard\ndescription: 编排镜头\n---\n\n先阅读参考资料。"},
		{Path: "references/guide.md", Content: "先列镜头景别。"},
		{Path: "assets/logo.png", Content: "aGVsbG8=", Binary: true},
	}
	if err := repo.CreateSkill(ctx, skill); err != nil {
		t.Fatal(err)
	}
	snapshot, err := (&Service{Repo: repo}).snapshotSkills(ctx, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "references/guide.md") || !strings.Contains(string(encoded), "assets/logo.png") || strings.Contains(string(encoded), "拼接内容") {
		t.Fatalf("Skill package files missing from snapshot: %s", encoded)
	}
	run, err := domain.NewRun("run-package", "session-package", "account-a", "message-package", now)
	if err != nil {
		t.Fatal(err)
	}
	run.SkillIDs = []string{skill.ID}
	run.SkillSnapshot = snapshot
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetRun(ctx, "account-a", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := (&AgentExecutor{repo: repo}).selectedSkills(ctx, stored)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || len(selected[0].Files) != 3 || selected[0].Files[1].Content != "先列镜头景别。" {
		t.Fatalf("selected Skill files = %#v", selected)
	}
}
