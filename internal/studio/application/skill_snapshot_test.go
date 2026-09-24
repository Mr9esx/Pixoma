package application

import (
	"context"
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
	oldSkill.Enabled = false
	oldSkill.UpdatedAt = now.Add(2 * time.Second)
	if err := repo.UpdateSkill(ctx, oldSkill); err != nil {
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
