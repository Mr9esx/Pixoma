package einoagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	platformdb "github.com/Mr9esx/Pixoma/internal/platform/db"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type installSkillEventCapture struct {
	events []string
}

func TestParseGitHubSkillMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "YAML 与正文", content: "---\nname: sample-skill\ndescription: >-\n  Use this skill for samples.\n---\n# Steps", valid: true},
		{name: "CRLF 换行", content: "---\r\nname: sample-skill\r\ndescription: Use this skill for samples.\r\n---\r\n# Steps", valid: true},
		{name: "缺少元数据", content: "# Steps"},
		{name: "缺少结束标记", content: "---\nname: sample-skill\ndescription: Use this skill for samples.\n# Steps"},
		{name: "YAML 格式无效", content: "---\nname: [invalid\n---\n# Steps"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, err := parseGitHubSkillMetadata(test.content)
			if (err == nil) != test.valid {
				t.Fatalf("元数据 = %#v，错误 = %v", metadata, err)
			}
		})
	}
}

func (s *installSkillEventCapture) Emit(_ context.Context, eventType string, _ any) error {
	s.events = append(s.events, eventType)
	return nil
}

func TestParseGitHubSkillURL(t *testing.T) {
	for _, test := range []struct {
		name string
		url  string
		want githubSkillSource
	}{
		{
			name: "Skill file",
			url:  "https://github.com/MiniMax-AI/MiniMax-H3/blob/main/skills/h3-prompt-writing/SKILL.md",
			want: githubSkillSource{Owner: "MiniMax-AI", Repo: "MiniMax-H3", Ref: "main", Directory: "skills/h3-prompt-writing"},
		},
		{
			name: "Skill directory",
			url:  "https://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing",
			want: githubSkillSource{Owner: "MiniMax-AI", Repo: "MiniMax-H3", Ref: "main", Directory: "skills/h3-prompt-writing"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseGitHubSkillURL(test.url)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("source = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseGitHubSkillURLRejectsUnsupportedURLs(t *testing.T) {
	for _, rawURL := range []string{
		"http://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing",
		"https://github.com.attacker.example/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing",
		"https://github.com/MiniMax-AI/MiniMax-H3/blob/main/README.md",
		"https://github.com/MiniMax-AI/MiniMax-H3/tree/main/../private",
		"https://github.com/MiniMax-AI/MiniMax-H3/tree/main/skills/h3-prompt-writing?plain=1",
	} {
		if _, err := parseGitHubSkillURL(rawURL); err == nil {
			t.Errorf("parseGitHubSkillURL(%q) succeeded", rawURL)
		}
	}
}

func TestBuildGitHubSkillPackageKeepsReferencePaths(t *testing.T) {
	got, err := buildGitHubSkillPackage(map[string]string{
		"SKILL.md":                   "---\nname: h3-prompt-writing\ndescription: Write H3 video prompts\n---\n# H3 Prompt Writing\nFollow the references.",
		"references/base-en.txt":     "base mode prompt structure",
		"references/ref-en.txt":      "reference prompt structure",
		"references/nested/extra.md": "additional guidance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "h3-prompt-writing" || got.Description != "Write H3 video prompts" {
		t.Fatalf("metadata = %#v", got)
	}
	for _, want := range []string{
		"[SKILL.md]",
		"[references/base-en.txt]",
		"[references/ref-en.txt]",
		"[references/nested/extra.md]",
		"base mode prompt structure",
		"reference prompt structure",
	} {
		if !strings.Contains(got.Prompt, want) {
			t.Errorf("Prompt does not contain %q: %s", want, got.Prompt)
		}
	}
}

func TestGitHubSkillReferences(t *testing.T) {
	got := githubSkillReferences("Use `references/base-en.txt` and [the guide](references/guide.md). Do not load references/run.py.")
	if len(got) != 2 || got[0] != "references/base-en.txt" || got[1] != "references/guide.md" {
		t.Fatalf("references = %#v", got)
	}
}

func TestBuildGitHubSkillPackageRejectsInvalidMetadata(t *testing.T) {
	for _, skillFile := range []string{
		"# Missing metadata",
		"---\nname: h3-prompt-writing\n---\nMissing description",
		"---\nname: [invalid\ndescription: prompt\n---\nInvalid YAML",
	} {
		if _, err := buildGitHubSkillPackage(map[string]string{"SKILL.md": skillFile}); err == nil {
			t.Errorf("buildGitHubSkillPackage(%q) succeeded", skillFile)
		}
	}
	if _, err := buildGitHubSkillPackage(map[string]string{"README.md": "No Skill manifest"}); err == nil {
		t.Fatal("package without SKILL.md succeeded")
	}
}

func TestInstallPublicGitHubSkillLinkWhenConfigured(t *testing.T) {
	rawURL := os.Getenv("PIXOMA_GITHUB_SKILL_TEST_URL")
	if rawURL == "" {
		t.Skip("PIXOMA_GITHUB_SKILL_TEST_URL 未设置")
	}
	gdb, err := platformdb.Open(platformdb.Options{DSN: fmt.Sprintf("file:github_skill_import_%d?mode=memory&cache=shared", time.Now().UnixNano())})
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
	if err := platformdb.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	service := &studioapp.CapabilityConfigService{Repo: repo, IDs: func() string { return "skill-imported" }, Now: func() time.Time { return time.Now().UTC() }}
	sink := &installSkillEventCapture{}
	tool, err := newInstallSkillTool("account-a", service, sink, "zh")
	if err != nil {
		t.Fatal(err)
	}
	arguments, err := json.Marshal(map[string]string{"github_url": rawURL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.InvokableRun(context.Background(), string(arguments))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "h3-prompt-writing") {
		t.Fatalf("install result = %q", result)
	}
	listed, err := service.ListSkills(context.Background(), "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != "skill-imported" || !listed[0].Enabled {
		t.Fatalf("saved account Skill = %#v", listed)
	}
	detail, err := service.GetSkill(context.Background(), "account-a", listed[0].ID)
	if err != nil || !strings.Contains(detail.Prompt, "[references/base-en.txt]") || !strings.Contains(detail.Prompt, "[references/ref-en.txt]") {
		t.Fatalf("saved account Skill detail = %#v, err = %v", detail, err)
	}
	if len(sink.events) != 4 || sink.events[0] != studioapp.EventToolCallStart || sink.events[1] != studioapp.EventToolCallArgs || sink.events[2] != studioapp.EventToolCallResult || sink.events[3] != studioapp.EventToolCallEnd {
		t.Fatalf("tool events = %#v", sink.events)
	}
}
