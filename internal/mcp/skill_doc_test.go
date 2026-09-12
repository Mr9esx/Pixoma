package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func TestPixomaMCPSkill_ProductAttachment(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "skills", "pixoma-mcp", "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "name: pixoma-mcp") {
		t.Fatal("missing name")
	}
	if !strings.Contains(text, "Use when") {
		t.Fatal("description must be Use when only")
	}
	descEnd := strings.Index(text, "---\n\n")
	if descEnd < 0 {
		t.Fatal("missing frontmatter close")
	}
	desc := text[:descEnd]
	for _, banned := range []string{"list_workflows", "get_workflow", "run_workflow", "get_task"} {
		if strings.Contains(desc, banned) {
			t.Fatalf("description leaked flow %q", banned)
		}
	}
	if !strings.Contains(text, "错误") || !strings.Contains(text, "连接器") {
		t.Fatal("missing error table / connector")
	}
	for _, need := range []string{"list_workflows", "get_workflow", "run_workflow", "get_task", "pixoma://task/"} {
		if !strings.Contains(text, need) {
			t.Fatalf("missing call chain %q", need)
		}
	}
	if !strings.Contains(text, "task_id") {
		t.Fatal("run_workflow must say it returns task_id")
	}
	if strings.Contains(text, "签发") || strings.Contains(text, "粘贴 Bearer") {
		t.Fatal("must not teach bearer")
	}
	devSkill := filepath.Join(root, ".agents", "skills", "pixoma-mcp")
	if st, err := os.Stat(devSkill); err == nil && st.IsDir() {
		t.Fatal("must not install into .agents/skills")
	}
}

func TestPixomaMCPSkill_InstallPromptInREADME(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "把 https://pixoma.miaoplus.com/skills/pixoma-mcp/SKILL.md 装进本机 skills 目录；用户的聊天产品需要能配置 MCP 连接器。仓库内同源文件：docs/skills/pixoma-mcp/SKILL.md。"
	if !strings.Contains(string(raw), want) {
		t.Fatalf("README missing install prompt")
	}
	if strings.Contains(string(raw), "Bearer "+want) {
		t.Fatal("README must not embed bearer with prompt")
	}
}
