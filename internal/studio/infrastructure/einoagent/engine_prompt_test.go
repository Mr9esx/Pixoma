package einoagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestToolPromptSectionUsesRegisteredTools(t *testing.T) {
	skillTool, err := newLoadSkillTool([]domain.RunSkill{{ID: "skill-a", Name: "分镜", Prompt: "输出镜头表"}}, nil, "zh")
	if err != nil {
		t.Fatal(err)
	}
	section, err := toolPromptSectionForLocale(context.Background(), []einotool.BaseTool{skillTool}, "zh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(section, "<tools>\n- load_skill: ") || !strings.Contains(section, "\n</tools>") {
		t.Fatalf("Tool section = %q", section)
	}
	if strings.Contains(section, "create_text_asset") || strings.Contains(section, "read_asset") {
		t.Fatalf("unregistered Tool appeared in section: %q", section)
	}
	if promptCopy("zh").toolGuidance["create_text_asset"] == "" || promptCopy("zh").toolGuidance["read_asset"] == "" {
		t.Fatal("built-in Tool guidance is missing")
	}
}

func TestStudioPromptUsesLocaleAndStableCatalogOrder(t *testing.T) {
	skills := []domain.RunSkill{
		{ID: "skill-b", Name: "B", Description: "second"},
		{ID: "skill-a", Name: "A", Description: "first"},
	}
	prompt, err := buildStudioPrompt(context.Background(), "en", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Reply in English by default") || !strings.Contains(prompt, "<skills>") {
		t.Fatalf("English prompt is incomplete: %q", prompt)
	}
	context := buildStudioRunContext("en", skills, nil)
	if !strings.Contains(context, "No workflows are currently available") || strings.Index(context, "<id>skill-a</id>") >= strings.Index(context, "<id>skill-b</id>") {
		t.Fatalf("Run context is incomplete or unstable: %q", context)
	}
}

func TestEnglishBuiltInToolMetadata(t *testing.T) {
	tool, err := newLoadSkillTool(nil, nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "Read SKILL.md") || !strings.Contains(string(encoded), "Skill ID in this Run") {
		t.Fatalf("English Tool metadata is incomplete: %s", encoded)
	}
}
