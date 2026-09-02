package text_test

import (
	"context"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
)

func TestRender_InterpolatesVariables(t *testing.T) {
	got := text.Render("task={{ task_id }} done", map[string]string{"task_id": "abc-123"})
	if got != "task=abc-123 done" {
		t.Fatalf("render: got %q", got)
	}
}

func TestRender_KeepsUnknownVariables(t *testing.T) {
	got := text.Render("task={{ missing }}", map[string]string{"task_id": "x"})
	if got != "task={{ missing }}" {
		t.Fatalf("unknown var should stay: got %q", got)
	}
}

func openStore(t *testing.T, name string) (*text.Store, context.Context) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:" + name + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dbc, _ := gdb.DB(); _ = dbc.Close() })
	st, err := text.NewStore(gdb)
	if err != nil {
		t.Fatal(err)
	}
	return st, context.Background()
}

func TestDefaultReturnsBuiltins(t *testing.T) {
	want := "✅ 工作流完成\ntask={{ task_id }}"
	if got := text.Default(text.KeyWorkflowDone); got != want {
		t.Fatalf("default workflow_done: got %q want %q", got, want)
	}
	if got := text.Default("unknown-key"); got != "" {
		t.Fatalf("unknown key should be empty: got %q", got)
	}
	if got := text.Default(text.KeyTaskFailed); got != "❌ 任务执行失败\ntask={{ task_id }}\n状态：{{ status }}\n{{ error_msg }}" {
		t.Fatalf("default task_failed: got %q", got)
	}
	if got := text.Default(text.KeyTaskCancelled); got != "任务已取消\ntask={{ task_id }}" {
		t.Fatalf("default task_cancelled: got %q", got)
	}
}

func TestSpecsExposeTerminalVariables(t *testing.T) {
	byKey := map[string]text.Spec{}
	for _, s := range text.Specs() {
		byKey[s.Key] = s
	}
	for _, k := range []string{text.KeyTaskFailed, text.KeyTaskCancelled, text.KeyInputPrompt} {
		if _, ok := byKey[k]; !ok {
			t.Fatalf("specs missing key %q", k)
		}
	}
	if got := byKey[text.KeyTaskFailed].Variables; len(got) != 3 {
		t.Fatalf("task_failed variables=%v", got)
	}
	if got := byKey[text.KeyInputPrompt].Variables; len(got) != 3 {
		t.Fatalf("input_prompt variables=%v", got)
	}
}

func TestSpecsExposeGroupsAndInteractionKeys(t *testing.T) {
	byKey := map[string]text.Spec{}
	for _, s := range text.Specs() {
		byKey[s.Key] = s
	}

	type want struct {
		key    string
		group  string
		defalt string
	}
	for _, item := range []want{
		{key: text.KeyWelcome, group: text.GroupPlatform},
		{key: text.KeyHelp, group: text.GroupCommands},
		{key: text.KeySelectTemplate, group: text.GroupPlatform},
		{key: text.KeyWorkflowDone, group: text.GroupNotifications},
		{key: text.KeyWorkflowDoneFollowp, group: text.GroupNotifications},
		{key: text.KeySubmitStarted, group: text.GroupWorkflow},
		{key: text.KeyConfirmRun, group: text.GroupWorkflow},
		{key: text.KeyInputPrompt, group: text.GroupWorkflow},
		{key: text.KeyExitDone, group: text.GroupPlatform},
		{key: text.KeyUnfinishedSession, group: text.GroupPlatform},
		{key: text.KeyMenuActionPlaceholder, group: text.GroupPlatform},
		{key: text.KeyMenuUpdated, group: text.GroupPlatform},
		{key: text.KeyTaskFailed, group: text.GroupNotifications},
		{key: text.KeyTaskCancelled, group: text.GroupNotifications},
		{key: text.KeyPreviewHintLabel, group: text.GroupWorkflow, defalt: "预览说明："},
		{key: text.KeyButtonStartCase, group: text.GroupWorkflow, defalt: "▶ 开始 Case"},
		{key: text.KeyInputInvalidNumber, group: text.GroupWorkflow, defalt: "请输入合法数字，例如 42"},
		{key: text.KeyInputInvalidBoolean, group: text.GroupWorkflow, defalt: "请输入 true 或 false"},
		{key: text.KeyButtonSkip, group: text.GroupWorkflow, defalt: "跳过"},
		{key: text.KeyButtonConfirmRun, group: text.GroupWorkflow, defalt: "✅ 确认生成"},
		{key: text.KeyButtonExit, group: text.GroupWorkflow, defalt: "✕ 退出"},
		{key: text.KeySessionTerminated, group: text.GroupNotifications, defalt: "该工作流已被管理员删除，当前会话已结束。"},
		{key: text.KeyListTasks, group: text.GroupPlatform, defalt: "我的任务\n\n当前任务\n{{ current }}\n\n最近任务\n{{ recent }}"},
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
	if got := st.Render(ctx, "ch", text.KeyWorkflowDone, map[string]string{"task_id": "T1"}); got != "✅ 工作流完成\ntask=T1" {
		t.Fatalf("builtin render: got %q", got)
	}
	// platform default override applies to a channel without its own override
	if err := st.Save(ctx, text.GlobalDefaultID, map[string]string{text.KeyWorkflowDone: "全局完成 {{ task_id }}"}); err != nil {
		t.Fatal(err)
	}
	if got := st.Render(ctx, "ch", text.KeyWorkflowDone, map[string]string{"task_id": "T9"}); got != "全局完成 T9" {
		t.Fatalf("global fallback render: got %q", got)
	}
	// channel override wins over the platform default
	if err := st.Save(ctx, "ch", map[string]string{text.KeyWorkflowDone: "消息平台完成 {{ task_id }}"}); err != nil {
		t.Fatal(err)
	}
	if got := st.Render(ctx, "ch", text.KeyWorkflowDone, map[string]string{"task_id": "T2"}); got != "消息平台完成 T2" {
		t.Fatalf("channel override render: got %q", got)
	}
	// reset channel -> falls back to platform default again
	if err := st.Reset(ctx, "ch", []string{text.KeyWorkflowDone}); err != nil {
		t.Fatal(err)
	}
	if got := st.Render(ctx, "ch", text.KeyWorkflowDone, map[string]string{"task_id": "T3"}); got != "全局完成 T3" {
		t.Fatalf("after reset render: got %q", got)
	}
}

func TestStore_SeedMaterializesEffectiveDefaults(t *testing.T) {
	st, ctx := openStore(t, "text_test2")
	if err := st.Save(ctx, text.GlobalDefaultID, map[string]string{
		text.KeyWorkflowDone: "全局完成 {{ task_id }}",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Seed(ctx, "ch-new"); err != nil {
		t.Fatal(err)
	}
	// seeded channel renders the global override
	if got := st.Render(ctx, "ch-new", text.KeyWorkflowDone, map[string]string{"task_id": "T5"}); got != "全局完成 T5" {
		t.Fatalf("seeded override: got %q", got)
	}
	// a key without a global override falls back to built-in, still renderable
	if got := st.Render(ctx, "ch-new", text.KeyWelcome, nil); got != "欢迎使用 ComfyUI Bot\n请选择功能：" {
		t.Fatalf("seeded builtin fallback: got %q", got)
	}
}

func TestStore_SaveIgnoresUnknownKeys(t *testing.T) {
	st, ctx := openStore(t, "text_test3")
	if err := st.Save(ctx, text.GlobalDefaultID, map[string]string{"bogus": "x"}); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Load(ctx, text.GlobalDefaultID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no stored rows, got %v", rows)
	}
}
