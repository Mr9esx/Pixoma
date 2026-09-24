package einoagent

import (
	"context"
	"strings"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestToolPromptSectionUsesRegisteredTools(t *testing.T) {
	skillTool, err := newLoadSkillTool([]domain.RunSkill{{ID: "skill-a", Name: "分镜", Prompt: "输出镜头表"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	section, err := toolPromptSection(context.Background(), []einotool.BaseTool{skillTool})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(section, "<tools>\n- load_skill: ") || !strings.Contains(section, "\n</tools>") {
		t.Fatalf("Tool section = %q", section)
	}
	if strings.Contains(section, "create_text_asset") || strings.Contains(section, "read_asset") {
		t.Fatalf("unregistered Tool appeared in section: %q", section)
	}
	if builtInToolGuidance("create_text_asset") == "" || builtInToolGuidance("read_asset") == "" {
		t.Fatal("built-in Tool guidance is missing")
	}
}
