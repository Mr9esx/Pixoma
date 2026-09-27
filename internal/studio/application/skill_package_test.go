package application_test

import (
	"strings"
	"testing"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestValidateSkillFilesFrontmatter(t *testing.T) {
	cases := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "YAML 与正文", content: "---\nname: sample-skill\ndescription: >-\n  Use this skill for samples.\n---\n# Steps\n\nUse the guide.", valid: true},
		{name: "CRLF 换行", content: "---\r\nname: sample-skill\r\ndescription: Use this skill for samples.\r\n---\r\n# Steps", valid: true},
		{name: "缺少元数据", content: "# Steps"},
		{name: "缺少结束标记", content: "---\nname: sample-skill\ndescription: Use this skill for samples.\n# Steps"},
		{name: "YAML 格式无效", content: "---\nname: [invalid\n---\n# Steps"},
		{name: "正文为空", content: "---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n  "},
		{name: "缺少说明", content: "---\nname: sample-skill\n---\n# Steps"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			metadata, _, err := studioapp.ValidateSkillFiles([]domain.SkillFile{{Path: "SKILL.md", Content: test.content}})
			if (err == nil) != test.valid {
				t.Fatalf("校验结果 = %#v，错误 = %v", metadata, err)
			}
			if test.valid && metadata.Name != "sample-skill" {
				t.Fatalf("元数据 = %#v", metadata)
			}
		})
	}
}

func TestValidateSkillFilesPathLimits(t *testing.T) {
	manifest := domain.SkillFile{Path: "SKILL.md", Content: "---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Steps"}
	for _, filePath := range []string{
		strings.Repeat("a", 512),
		strings.Repeat("中", 170),
		strings.Repeat("d/", 15) + "guide.md",
	} {
		_, _, err := studioapp.ValidateSkillFiles([]domain.SkillFile{manifest, {Path: filePath, Content: "guide"}})
		if err != nil {
			t.Fatalf("路径 %q 应通过校验：%v", filePath, err)
		}
	}
	for _, filePath := range []string{
		"references/name\x00.md",
		strings.Repeat("a", 513),
		strings.Repeat("中", 171),
		strings.Repeat("d/", 16) + "guide.md",
	} {
		_, _, err := studioapp.ValidateSkillFiles([]domain.SkillFile{manifest, {Path: filePath, Content: "guide"}})
		if err == nil {
			t.Fatalf("路径 %q 应被拒绝", filePath)
		}
	}
}
