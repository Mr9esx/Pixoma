package capability

import (
	"context"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	texttpl "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

func TestFormatMyTasksEmpty(t *testing.T) {
	got := FormatMyTasks(nil, nil, time.Now(), shanghai())
	want := "我的任务\n\n当前任务\n当前没有排队中的任务\n\n最近任务\n"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestFormatMyTasksCurrentAndRecent(t *testing.T) {
	loc := shanghai()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, loc)
	runningStart := now.Add(-3*time.Minute - 12*time.Second)
	queuedAt := now.Add(-65 * time.Second)
	doneAt := time.Date(2026, 8, 30, 7, 50, 0, 0, loc)
	done := runtimedomain.NewPending("1120186", "s1", 10, "pfx", doneAt.Add(-time.Minute))
	done.Status = sharedkernel.TaskSucceeded
	done.CompletedAt = doneAt
	done.ChatID = "tg-default:1"

	running := runtimedomain.NewPending("r1", "s1", 10, "pfx", queuedAt)
	running.Status = sharedkernel.TaskRunning
	running.StartedAt = runningStart
	running.ChatID = "tg-default:1"

	queued := runtimedomain.NewPending("q1", "s1", 11, "pfx", queuedAt)
	queued.Status = sharedkernel.TaskQueued
	queued.ChatID = "tg-default:1"

	names := map[sharedkernel.CaseID]string{10: "后入高潮痉挛", 11: "口交"}
	got := FormatMyTasks([]*runtimedomain.Task{done, running, queued}, names, now, loc)

	if !strings.Contains(got, "我的任务\n\n当前任务\n") {
		t.Fatalf("missing current header: %q", got)
	}
	if !strings.Contains(got, "#r1 · 后入高潮痉挛 · 运行中 · 3 分 12 秒") {
		t.Fatalf("running line: %q", got)
	}
	if !strings.Contains(got, "#q1 · 口交 · 排队中 · 1 分 05 秒") {
		t.Fatalf("queued line: %q", got)
	}
	if !strings.Contains(got, "最近任务\n#1120186 · 后入高潮痉挛 · 成功 · 08-30 07:50") {
		t.Fatalf("recent line: %q", got)
	}
	if strings.Contains(got, "当前没有排队中的任务") {
		t.Fatalf("empty current copy leaked: %q", got)
	}
}

func TestFormatMyTasksRecentCapAndNoRecentSection(t *testing.T) {
	loc := shanghai()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, loc)
	var tasks []*runtimedomain.Task
	for i := 0; i < 10; i++ {
		id := sharedkernel.TaskID("d" + string(rune('a'+i)))
		doneAt := now.Add(-time.Duration(i) * time.Hour)
		t0 := runtimedomain.NewPending(id, "s", 1, "pfx", doneAt)
		t0.Status = sharedkernel.TaskSucceeded
		t0.CompletedAt = doneAt
		tasks = append(tasks, t0)
	}
	got := FormatMyTasks(tasks, map[sharedkernel.CaseID]string{1: "口交"}, now, loc)
	if strings.Count(got, " · 成功 · ") != 8 {
		t.Fatalf("recent cap: %q", got)
	}

	emptyRecent := FormatMyTasks([]*runtimedomain.Task{
		func() *runtimedomain.Task {
			q := runtimedomain.NewPending("q", "s", 1, "pfx", now)
			q.Status = sharedkernel.TaskPending
			return q
		}(),
	}, map[sharedkernel.CaseID]string{1: "口交"}, now, loc)
	if !strings.Contains(emptyRecent, "最近任务\n") {
		t.Fatalf("recent header missing: %q", emptyRecent)
	}
	if strings.Contains(emptyRecent, "✅ #") {
		t.Fatalf("recent lines leaked: %q", emptyRecent)
	}
}

func TestListTasksRegistryEmptyParams(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(ListTasks{}); err != nil {
		t.Fatal(err)
	}
	res, err := r.Invoke(context.Background(), protocol.CapabilityInvoke{
		CapabilityID: "list_tasks",
		Params:       map[string]any{},
		ChatID:       "tg-default:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "当前没有排队中的任务") {
		t.Fatalf("text=%q", res.Text)
	}
}

func TestListTasksInvoke(t *testing.T) {
	loc := shanghai()
	now := time.Date(2026, 8, 30, 8, 0, 0, 0, loc)
	tasks := runtimedomain.NewMemoryTaskRepository()
	done := runtimedomain.NewPending("1120186", "s1", 10, "pfx", now.Add(-10*time.Minute))
	done.Status = sharedkernel.TaskSucceeded
	done.CompletedAt = now.Add(-10 * time.Minute)
	done.ChatID = "tg-default:1"
	if err := tasks.Create(context.Background(), done); err != nil {
		t.Fatal(err)
	}
	cap := ListTasks{
		Tasks: tasks,
		Cases: stubCases{names: map[sharedkernel.CaseID]string{10: "后入高潮痉挛"}},
		Now:   func() time.Time { return now },
		Loc:   loc,
	}
	res, err := cap.Invoke(context.Background(), protocol.AccountCtx{}, protocol.Nav{}, "tg-default:1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "#1120186 · 后入高潮痉挛 · 成功 · 08-30 07:50") {
		t.Fatalf("text=%q", res.Text)
	}
}

type stubTexts struct {
	tpl string
}

func (s stubTexts) Render(_ context.Context, _, key string, vars map[string]string) string {
	if key == texttpl.KeyListTasks && s.tpl != "" {
		return texttpl.Render(s.tpl, vars)
	}
	return texttpl.Render(texttpl.Default(key), vars)
}

func TestListTasksUsesTextTemplates(t *testing.T) {
	cap := ListTasks{Texts: stubTexts{tpl: "任务清单\n\n{{ current }}{{ recent }}"}}
	res, err := cap.Invoke(context.Background(), protocol.AccountCtx{ChannelID: "tg-default"}, protocol.Nav{}, "tg-default:1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "任务清单\n\n当前没有排队中的任务" {
		t.Fatalf("text=%q", res.Text)
	}
}

func shanghai() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
	return loc
}

type stubCases struct {
	names map[sharedkernel.CaseID]string
}

func (s stubCases) Get(_ context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error) {
	name, ok := s.names[id]
	if !ok {
		return nil, catalogdomain.ErrNotFound
	}
	return &catalogdomain.Case{
		Document: catalogdomain.CaseDocument{ID: id, Name: name},
		Enabled:  true,
	}, nil
}

func (stubCases) Save(context.Context, *catalogdomain.Case) error { return nil }
func (stubCases) Create(context.Context, *catalogdomain.Case) error {
	return nil
}
func (stubCases) List(context.Context, catalogdomain.ListQuery) ([]*catalogdomain.Case, error) {
	return nil, nil
}
func (stubCases) Disable(context.Context, sharedkernel.CaseID) error { return nil }
func (stubCases) Enable(context.Context, sharedkernel.CaseID) error  { return nil }
func (stubCases) Delete(context.Context, sharedkernel.CaseID) error  { return nil }
