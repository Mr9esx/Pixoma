package einoagent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/Mr9esx/Pixoma/internal/studio/application/contextcompaction"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type skillReadResult struct {
	Path          string     `json:"path"`
	Offset        int        `json:"offset"`
	Content       string     `json:"content"`
	ReadableFiles []string   `json:"readable_files"`
	Next          *skillArgs `json:"next"`
	NextFiles     *skillArgs `json:"next_files"`
}

func decodeSkillRead(t *testing.T, output string, maxTokens int) skillReadResult {
	t.Helper()
	if tokens := contextcompaction.EstimateTokens([]*schema.Message{{Role: schema.Tool, Content: output}}); tokens > maxTokens {
		t.Fatalf("Tool output uses %d tokens, budget is %d", tokens, maxTokens)
	}
	var result skillReadResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestReadSkillManifestAndReferenceByPath(t *testing.T) {
	skill := domain.RunSkill{
		ID: "skill-a", Name: "storyboard", Prompt: "拼接内容不可返回",
		Files: []domain.SkillFile{
			{Path: "SKILL.md", Content: "# 操作说明\n参见 references/guide.md。"},
			{Path: "references", Directory: true},
			{Path: "references/guide.md", Content: "逐格注明景别。"},
			{Path: "assets/logo.png", Content: "aGVsbG8=", Binary: true},
		},
	}
	output, err := renderSkillRead(skill, skillArgs{SkillID: skill.ID}, 300)
	if err != nil {
		t.Fatal(err)
	}
	manifest := decodeSkillRead(t, output, 300)
	if manifest.Path != "SKILL.md" || !strings.Contains(manifest.Content, "操作说明") || strings.Contains(output, "拼接内容不可返回") || strings.Contains(output, "逐格注明景别") {
		t.Fatalf("manifest = %#v", manifest)
	}
	if len(manifest.ReadableFiles) != 2 || manifest.ReadableFiles[0] != "SKILL.md" || manifest.ReadableFiles[1] != "references/guide.md" {
		t.Fatalf("readable files = %#v", manifest.ReadableFiles)
	}
	output, err = renderSkillRead(skill, skillArgs{SkillID: skill.ID, Path: "references/guide.md"}, 300)
	if err != nil {
		t.Fatal(err)
	}
	reference := decodeSkillRead(t, output, 300)
	if reference.Content != "逐格注明景别。" || reference.Next != nil {
		t.Fatalf("reference = %#v", reference)
	}
	if _, err := renderSkillRead(skill, skillArgs{SkillID: skill.ID, Path: "assets/logo.png"}, 300); err == nil || !strings.Contains(err.Error(), "二进制") {
		t.Fatalf("binary read error = %v", err)
	}
	if _, err := renderSkillRead(skill, skillArgs{SkillID: skill.ID, Path: "references"}, 300); err == nil || !strings.Contains(err.Error(), "文件夹") {
		t.Fatalf("directory read error = %v", err)
	}
	output, err = renderSkillRead(skill, skillArgs{SkillID: skill.ID, Path: "references/guide.md", Offset: len([]rune("逐格注明景别。"))}, 300)
	if err != nil {
		t.Fatal(err)
	}
	atEnd := decodeSkillRead(t, output, 300)
	if atEnd.Content != "" || atEnd.Next != nil {
		t.Fatalf("end of file = %#v", atEnd)
	}
}

func TestReadSkillPaginatesLongSingleLineWithinTokenBudget(t *testing.T) {
	const maxTokens = 160
	content := strings.Repeat("界", (256<<10)/3)
	skill := domain.RunSkill{ID: "skill-a", Name: "long", Files: []domain.SkillFile{{Path: "SKILL.md", Content: content}}}
	firstOutput, err := renderSkillRead(skill, skillArgs{SkillID: skill.ID}, maxTokens)
	if err != nil {
		t.Fatal(err)
	}
	first := decodeSkillRead(t, firstOutput, maxTokens)
	if first.Next == nil || first.Next.Path != "SKILL.md" || first.Next.Offset != len([]rune(first.Content)) || first.Content == "" {
		t.Fatalf("first page = %#v", first)
	}
	secondOutput, err := renderSkillRead(skill, *first.Next, maxTokens)
	if err != nil {
		t.Fatal(err)
	}
	second := decodeSkillRead(t, secondOutput, maxTokens)
	if second.Offset != first.Next.Offset || second.Content == "" || !strings.HasPrefix(content, first.Content+second.Content) {
		t.Fatalf("second page = %#v", second)
	}
	lastOutput, err := renderSkillRead(skill, skillArgs{SkillID: skill.ID, Path: "SKILL.md", Offset: len([]rune(content)) - 30}, maxTokens)
	if err != nil {
		t.Fatal(err)
	}
	last := decodeSkillRead(t, lastOutput, maxTokens)
	if last.Content != strings.Repeat("界", 30) || last.Next != nil {
		t.Fatalf("last page = %#v", last)
	}
}

func TestReadSkillPaginatesFileIndexAndKeepsLegacyPrompt(t *testing.T) {
	files := []domain.SkillFile{{Path: "SKILL.md", Content: "# 操作说明"}}
	for i := 0; i < 50; i++ {
		files = append(files, domain.SkillFile{Path: fmt.Sprintf("references/long-reference-name-%03d.md", i), Content: "资料"})
	}
	skill := domain.RunSkill{ID: "skill-a", Name: "many", Files: files}
	output, err := renderSkillRead(skill, skillArgs{SkillID: skill.ID}, 160)
	if err != nil {
		t.Fatal(err)
	}
	first := decodeSkillRead(t, output, 160)
	if len(first.ReadableFiles) == 0 || first.NextFiles == nil || first.NextFiles.Path != "." {
		t.Fatalf("first file list = %#v", first)
	}
	output, err = renderSkillRead(skill, *first.NextFiles, 160)
	if err != nil {
		t.Fatal(err)
	}
	second := decodeSkillRead(t, output, 160)
	if second.Path != "." || len(second.ReadableFiles) == 0 || second.ReadableFiles[0] != fmt.Sprintf("references/long-reference-name-%03d.md", first.NextFiles.Offset-1) {
		t.Fatalf("next file list = %#v", second)
	}
	output, err = renderSkillRead(skill, skillArgs{SkillID: skill.ID, Path: ".", Offset: len(files)}, 160)
	if err != nil {
		t.Fatal(err)
	}
	end := decodeSkillRead(t, output, 160)
	if len(end.ReadableFiles) != 0 || end.Next != nil {
		t.Fatalf("end of file list = %#v", end)
	}
	legacy := domain.RunSkill{ID: "legacy", Name: "旧 Skill", Prompt: "直接执行说明"}
	output, err = renderSkillRead(legacy, skillArgs{SkillID: legacy.ID}, 160)
	if err != nil {
		t.Fatal(err)
	}
	result := decodeSkillRead(t, output, 160)
	if result.Content != "直接执行说明" || result.Path != "SKILL.md" {
		t.Fatalf("legacy result = %#v", result)
	}
}
