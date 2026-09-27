package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestMessageTextProjectsInlineReferences(t *testing.T) {
	raw := json.RawMessage(`[{"type":"text","text":"用 / 和 "},{"type":"skill_ref","skill_id":"skill-1","name":"分镜草稿"},{"type":"text","text":" 处理 "},{"type":"asset_ref","asset_id":"asset-1","asset_version_id":"version-2","name":"角色设定图"}]`)
	text, err := messageText(raw)
	if err != nil {
		t.Fatal(err)
	}
	if want := "用 / 和 「分镜草稿」Skill 处理 「角色设定图」资产"; text != want {
		t.Fatalf("message text = %q, want %q", text, want)
	}
}

func TestMessageTextProjectsWorkflowReference(t *testing.T) {
	raw := json.RawMessage(`[{"type":"workflow_ref","workflow_id":"12","name":"角色三视图"},{"type":"text","text":"需要哪些输入？"}]`)
	text, err := messageText(raw)
	if err != nil {
		t.Fatal(err)
	}
	if text != "「角色三视图」工作流需要哪些输入？" {
		t.Fatalf("message text = %q", text)
	}
}

func TestTranscriptPreservesWorkflowReference(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	message := &domain.Message{
		ID: "message-1", RunID: "run-1", Role: domain.MessageRoleUser,
		ContentJSON: json.RawMessage(`[{"type":"workflow_ref","workflow_id":"12","name":"角色三视图"},{"type":"text","text":"需要哪些输入？"}]`),
		CreatedAt:   now,
	}
	run := &domain.Run{ID: "run-1", TriggerMessageID: message.ID, CreatedAt: now}
	transcript := ProjectSessionTranscript([]*domain.Message{message}, []*domain.Run{run}, nil)
	if len(transcript.Messages) != 1 || transcript.Messages[0].Content != "「角色三视图」工作流需要哪些输入？" {
		t.Fatalf("transcript messages = %#v", transcript.Messages)
	}
	if parts := transcript.Messages[0].Parts; len(parts) != 2 || parts[0].Type != "workflow_ref" || parts[0].WorkflowID != "12" {
		t.Fatalf("transcript parts = %#v", parts)
	}
}

func TestNormalizeMessageInputKeepsWorkflowReferenceInConversation(t *testing.T) {
	input := SendMessageInput{
		Text: "「角色三视图」工作流需要哪些输入？",
		Parts: []MessagePart{
			{Type: "workflow_ref", WorkflowID: "12", Name: "角色三视图"},
			{Type: "text", Text: "需要哪些输入？"},
		},
	}
	got, err := normalizeMessageInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Parts) != 2 || got.Parts[0].WorkflowID != "12" || len(got.SkillIDs) != 0 || len(got.SelectedAssets) != 0 {
		t.Fatalf("message input = %#v", got)
	}
}

func TestTranscriptPreservesInlineReferences(t *testing.T) {
	now := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	message := &domain.Message{
		ID: "message-1", RunID: "run-1", Role: domain.MessageRoleUser,
		ContentJSON: json.RawMessage(`[{"type":"text","text":"用 "},{"type":"skill_ref","skill_id":"skill-1","name":"分镜草稿"}]`),
		CreatedAt:   now,
	}
	run := &domain.Run{ID: "run-1", TriggerMessageID: message.ID, CreatedAt: now}
	transcript := ProjectSessionTranscript([]*domain.Message{message}, []*domain.Run{run}, nil)
	if len(transcript.Messages) != 1 {
		t.Fatalf("messages = %#v", transcript.Messages)
	}
	got := transcript.Messages[0]
	if got.Content != "用 「分镜草稿」Skill" || len(got.Parts) != 2 || got.Parts[1].Type != "skill_ref" || got.Parts[1].SkillID != "skill-1" {
		t.Fatalf("transcript message = %#v", got)
	}
}

func TestNormalizeMessageInputSelectsOnlyInlineReferences(t *testing.T) {
	input := SendMessageInput{
		Text: "用 / 和 「分镜草稿」Skill 处理 「角色设定图」资产",
		Parts: []MessagePart{
			{Type: "text", Text: "用 / 和 "},
			{Type: "skill_ref", SkillID: "skill-1", Name: "分镜草稿"},
			{Type: "text", Text: " 处理 "},
			{Type: "asset_ref", AssetID: "asset-1", AssetVersionID: "version-2", Name: "角色设定图"},
			{Type: "skill_ref", SkillID: "skill-1", Name: "分镜草稿"},
		},
	}
	input.Text += "「分镜草稿」Skill"
	got, err := normalizeMessageInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.SkillIDs) != 1 || got.SkillIDs[0] != "skill-1" {
		t.Fatalf("skill ids = %#v", got.SkillIDs)
	}
	if len(got.SelectedAssets) != 1 || got.SelectedAssets[0].AssetID != "asset-1" || got.SelectedAssets[0].AssetVersionID != "version-2" {
		t.Fatalf("selected assets = %#v", got.SelectedAssets)
	}
}

func TestNormalizeMessageInputRejectsTextAndReferenceMismatch(t *testing.T) {
	_, err := normalizeMessageInput(SendMessageInput{
		Text:  "普通文字",
		Parts: []MessagePart{{Type: "skill_ref", SkillID: "skill-1", Name: "分镜草稿"}},
	})
	if err == nil {
		t.Fatal("expected mismatched message to fail")
	}
}
