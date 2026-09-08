package templates_test

import (
	"context"
	"testing"

	templates "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	textpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
)

func TestRender_InterpolatesVariables(t *testing.T) {
	got := templates.Render("task={{ task_id }} done", map[string]string{"task_id": "abc-123"})
	if got != "task=abc-123 done" {
		t.Fatalf("render: got %q", got)
	}
}

func TestRender_KeepsUnknownVariables(t *testing.T) {
	got := templates.Render("task={{ missing }}", map[string]string{"task_id": "x"})
	if got != "task={{ missing }}" {
		t.Fatalf("unknown var should stay: got %q", got)
	}
}

func openStore(t *testing.T, name string) (*textpersist.Store, context.Context) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dbc, _ := gdb.DB(); _ = dbc.Close() })
	st, err := textpersist.NewStore(gdb)
	if err != nil {
		t.Fatal(err)
	}
	return st, context.Background()
}

func TestDefaultReturnsBuiltins(t *testing.T) {
	want := "✅ 工作流完成\ntask={{ task_id }}"
	if got := templates.Default(templates.KeyWorkflowDone); got != want {
		t.Fatalf("default workflow_done: got %q want %q", got, want)
	}
	if got := templates.Default("unknown-key"); got != "" {
		t.Fatalf("unknown key should be empty: got %q", got)
	}
	if got := templates.Default(templates.KeyTaskFailed); got != "❌ 任务执行失败\ntask={{ task_id }}\n状态：{{ status }}\n{{ error_msg }}" {
		t.Fatalf("default task_failed: got %q", got)
	}
	if got := templates.Default(templates.KeyTaskCancelled); got != "任务已取消\ntask={{ task_id }}" {
		t.Fatalf("default task_cancelled: got %q", got)
	}
}

func TestSpecsExposeTerminalVariables(t *testing.T) {
	byKey := map[string]templates.Spec{}
	for _, s := range templates.Specs() {
		byKey[s.Key] = s
	}
	for _, k := range []string{templates.KeyTaskFailed, templates.KeyTaskCancelled, templates.KeyInputPrompt} {
		if _, ok := byKey[k]; !ok {
			t.Fatalf("specs missing key %q", k)
		}
	}
	if got := byKey[templates.KeyTaskFailed].Variables; len(got) != 3 {
		t.Fatalf("task_failed variables=%v", got)
	}
	if got := byKey[templates.KeyInputPrompt].Variables; len(got) != 3 {
		t.Fatalf("input_prompt variables=%v", got)
	}
}

func TestSpecsExposeGroupsAndInteractionKeys(t *testing.T) {
	byKey := map[string]templates.Spec{}
	for _, s := range templates.Specs() {
		byKey[s.Key] = s
	}

	type want struct {
		key    string
		group  string
		defalt string
	}
	for _, item := range []want{
		{key: templates.KeyWelcome, group: templates.GroupPlatform},
		{key: templates.KeyHelp, group: templates.GroupCommands},
		{key: templates.KeySelectTemplate, group: templates.GroupPlatform},
		{key: templates.KeyWorkflowDone, group: templates.GroupNotifications},
		{key: templates.KeyWorkflowDoneFollowp, group: templates.GroupNotifications},
		{key: templates.KeySubmitStarted, group: templates.GroupWorkflow},
		{key: templates.KeyConfirmRun, group: templates.GroupWorkflow},
		{key: templates.KeyInputPrompt, group: templates.GroupWorkflow},
		{key: templates.KeyExitDone, group: templates.GroupPlatform},
		{key: templates.KeyUnfinishedSession, group: templates.GroupPlatform},
		{key: templates.KeyMenuActionPlaceholder, group: templates.GroupPlatform},
		{key: templates.KeyMenuUpdated, group: templates.GroupPlatform},
		{key: templates.KeyTaskFailed, group: templates.GroupNotifications},
		{key: templates.KeyTaskCancelled, group: templates.GroupNotifications},
		{key: templates.KeyPreviewHintLabel, group: templates.GroupWorkflow, defalt: "预览说明："},
		{key: templates.KeyButtonStartCase, group: templates.GroupWorkflow, defalt: "▶ 开始 Case"},
		{key: templates.KeyInputInvalidNumber, group: templates.GroupWorkflow, defalt: "请输入合法数字，例如 42"},
		{key: templates.KeyInputInvalidBoolean, group: templates.GroupWorkflow, defalt: "请输入 true 或 false"},
		{key: templates.KeyButtonSkip, group: templates.GroupWorkflow, defalt: "跳过"},
		{key: templates.KeyButtonConfirmRun, group: templates.GroupWorkflow, defalt: "✅ 确认生成"},
		{key: templates.KeyButtonExit, group: templates.GroupWorkflow, defalt: "✕ 退出"},
		{key: templates.KeySessionTerminated, group: templates.GroupNotifications, defalt: "该工作流已被管理员删除，当前会话已结束。"},
		{key: templates.KeyListTasks, group: templates.GroupPlatform, defalt: "我的任务\n\n当前任务\n{{ current }}\n\n最近任务\n{{ recent }}"},
	} {
		spec, ok := byKey[item.key]
		if !ok {
			t.Fatalf("specs missing key %q", item.key)
		}
		if spec.Group != item.group {
			t.Fatalf("%s group=%q want %q", item.key, spec.Group, item.group)
		}
		if item.defalt != "" && spec.Default != item.defalt {
			t.Fatalf("%s default=%q want %q", item.key, spec.Default, item.defalt)
		}
	}
}

func TestStore_RenderUsesOverrideAndChannelPrecedence(t *testing.T) {
	st, ctx := openStore(t, "text_test")
	// built-in default before any override
	if got := st.Render(ctx, "ch", templates.KeyWorkflowDone, map[string]string{"task_id": "T1"}); got != "✅ 工作流完成\ntask=T1" {
		t.Fatalf("builtin render: got %q", got)
	}
	// platform default override applies to a channel without its own override
	if err := st.Save(ctx, templates.GlobalDefaultID, map[string]string{templates.KeyWorkflowDone: "全局完成 {{ task_id }}"}); err != nil {
		t.Fatal(err)
	}
	if got := st.Render(ctx, "ch", templates.KeyWorkflowDone, map[string]string{"task_id": "T9"}); got != "全局完成 T9" {
		t.Fatalf("global fallback render: got %q", got)
	}
	// channel override wins over the platform default
	if err := st.Save(ctx, "ch", map[string]string{templates.KeyWorkflowDone: "消息平台完成 {{ task_id }}"}); err != nil {
		t.Fatal(err)
	}
	if got := st.Render(ctx, "ch", templates.KeyWorkflowDone, map[string]string{"task_id": "T2"}); got != "消息平台完成 T2" {
		t.Fatalf("channel override render: got %q", got)
	}
	// reset channel -> falls back to platform default again
	if err := st.Reset(ctx, "ch", []string{templates.KeyWorkflowDone}); err != nil {
		t.Fatal(err)
	}
	if got := st.Render(ctx, "ch", templates.KeyWorkflowDone, map[string]string{"task_id": "T3"}); got != "全局完成 T3" {
		t.Fatalf("after reset render: got %q", got)
	}
}

func TestStore_SeedMaterializesEffectiveDefaults(t *testing.T) {
	st, ctx := openStore(t, "text_test2")
	if err := st.Save(ctx, templates.GlobalDefaultID, map[string]string{
		templates.KeyWorkflowDone: "全局完成 {{ task_id }}",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Seed(ctx, "ch-new"); err != nil {
		t.Fatal(err)
	}
	// seeded channel renders the global override
	if got := st.Render(ctx, "ch-new", templates.KeyWorkflowDone, map[string]string{"task_id": "T5"}); got != "全局完成 T5" {
		t.Fatalf("seeded override: got %q", got)
	}
	// a key without a global override falls back to built-in, still renderable
	if got := st.Render(ctx, "ch-new", templates.KeyWelcome, nil); got != "欢迎使用 Pixoma\n请选择功能：" {
		t.Fatalf("seeded builtin fallback: got %q", got)
	}
}

func TestStore_SaveIgnoresUnknownKeys(t *testing.T) {
	st, ctx := openStore(t, "text_test3")
	if err := st.Save(ctx, templates.GlobalDefaultID, map[string]string{"bogus": "x"}); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Load(ctx, templates.GlobalDefaultID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no stored rows, got %v", rows)
	}
}
