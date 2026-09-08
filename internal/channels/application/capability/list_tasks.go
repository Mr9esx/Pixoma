package capability

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	texttpl "github.com/Mr9esx/Pixoma/internal/channels/domain/templates"
	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
)

const (
	myTasksFetchLimit   = 64
	myTasksRecentCap    = 8
	myTasksCurrentEmpty = "✅ 当前没有排队中的任务"
)

// CaseLookup resolves a Case name for task list lines.
type CaseLookup interface {
	Get(ctx context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error)
}

// ListTasks is the platform capability that dumps the chat's current and recent tasks.
type ListTasks struct {
	Tasks runtimedomain.TaskRepository
	Cases CaseLookup
	Texts texttpl.Renderer
	Now   func() time.Time
	Loc   *time.Location
}

func (ListTasks) ID() string          { return "list_tasks" }
func (ListTasks) DisplayName() string { return "我的任务" }

func (ListTasks) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (ListTasks) Render(_ string, override map[string]any) (protocol.RenderDecl, error) {
	return MergeRender(protocol.RenderDecl{
		Entry:  "root",
		Config: map[string]any{"columns": 2},
	}, override), nil
}

func (l ListTasks) Invoke(ctx context.Context, acct protocol.AccountCtx, _ protocol.Nav, chatID sharedkernel.ChatID, _ map[string]any) (protocol.Result, error) {
	now := time.Now()
	if l.Now != nil {
		now = l.Now()
	}
	var tasks []*runtimedomain.Task
	if l.Tasks != nil && chatID != "" {
		listed, err := l.Tasks.ListByChat(ctx, chatID, myTasksFetchLimit)
		if err != nil {
			return protocol.Result{}, fmt.Errorf("list_tasks: %w", err)
		}
		tasks = listed
	}
	names := l.caseNames(ctx, tasks)
	current, recent := myTaskVars(tasks, names, now, l.Loc)
	return protocol.Result{Text: l.renderCopy(ctx, acct.ChannelID, current, recent)}, nil
}

func (l ListTasks) renderCopy(ctx context.Context, channelID, current, recent string) string {
	vars := map[string]string{"current": current, "recent": recent}
	if l.Texts == nil {
		return texttpl.Render(texttpl.Default(texttpl.KeyListTasks), vars)
	}
	return l.Texts.Render(ctx, channelID, texttpl.KeyListTasks, vars)
}

func (l ListTasks) caseNames(ctx context.Context, tasks []*runtimedomain.Task) map[sharedkernel.CaseID]string {
	names := map[sharedkernel.CaseID]string{}
	if l.Cases == nil {
		return names
	}
	seen := map[sharedkernel.CaseID]struct{}{}
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if _, ok := seen[t.CaseID]; ok {
			continue
		}
		seen[t.CaseID] = struct{}{}
		c, err := l.Cases.Get(ctx, t.CaseID)
		if err != nil || c == nil {
			continue
		}
		if c.Document.Name != "" {
			names[t.CaseID] = c.Document.Name
		}
	}
	return names
}

func FormatMyTasks(tasks []*runtimedomain.Task, names map[sharedkernel.CaseID]string, now time.Time, loc *time.Location) string {
	current, recent := myTaskVars(tasks, names, now, loc)
	return texttpl.Render(texttpl.Default(texttpl.KeyListTasks), map[string]string{
		"current": current,
		"recent":  recent,
	})
}

func myTaskVars(tasks []*runtimedomain.Task, names map[sharedkernel.CaseID]string, now time.Time, loc *time.Location) (current, recent string) {
	if loc == nil {
		var err error
		loc, err = time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
	}
	var currentTasks, recentTasks []*runtimedomain.Task
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if isInFlight(t.Status) {
			currentTasks = append(currentTasks, t)
			continue
		}
		recentTasks = append(recentTasks, t)
	}
	sort.Slice(currentTasks, func(i, j int) bool {
		return currentTasks[i].CreatedAt.After(currentTasks[j].CreatedAt)
	})
	sort.Slice(recentTasks, func(i, j int) bool {
		return recentTime(recentTasks[i]).After(recentTime(recentTasks[j]))
	})
	if len(recentTasks) > myTasksRecentCap {
		recentTasks = recentTasks[:myTasksRecentCap]
	}
	if len(currentTasks) == 0 {
		current = myTasksCurrentEmpty
	} else {
		current = joinTaskLines(currentTasks, names, now, loc, true)
	}
	recent = joinTaskLines(recentTasks, names, now, loc, false)
	return current, recent
}

func joinTaskLines(tasks []*runtimedomain.Task, names map[sharedkernel.CaseID]string, now time.Time, loc *time.Location, inFlight bool) string {
	var b strings.Builder
	for i, t := range tasks {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(formatTaskLine(t, names, now, loc, inFlight))
	}
	return b.String()
}

func isInFlight(st sharedkernel.TaskStatus) bool {
	switch st {
	case sharedkernel.TaskPending, sharedkernel.TaskQueued, sharedkernel.TaskRunning:
		return true
	default:
		return false
	}
}

func recentTime(t *runtimedomain.Task) time.Time {
	if !t.CompletedAt.IsZero() {
		return t.CompletedAt
	}
	return t.CreatedAt
}

func formatTaskLine(t *runtimedomain.Task, names map[sharedkernel.CaseID]string, now time.Time, loc *time.Location, inFlight bool) string {
	name := names[t.CaseID]
	if name == "" {
		name = fmt.Sprintf("%d", t.CaseID)
	}
	mark, status, extra := taskLineMeta(t, now, loc, inFlight)
	return fmt.Sprintf("%s #%s · %s · %s · %s", mark, t.ID, name, status, extra)
}

func taskLineMeta(t *runtimedomain.Task, now time.Time, loc *time.Location, inFlight bool) (mark, status, extra string) {
	switch t.Status {
	case sharedkernel.TaskRunning:
		start := t.StartedAt
		if start.IsZero() {
			start = t.CreatedAt
		}
		return "⏳", "执行中", "已跑 " + formatElapsed(now.Sub(start))
	case sharedkernel.TaskQueued, sharedkernel.TaskPending:
		return "🕒", "排队中", "等了 " + formatElapsed(now.Sub(t.CreatedAt))
	case sharedkernel.TaskSucceeded:
		return "✅", "已完成", formatClock(t, loc)
	case sharedkernel.TaskFailed:
		return "❌", "失败", formatClock(t, loc)
	case sharedkernel.TaskCancelled:
		return "➖", "已取消", formatClock(t, loc)
	default:
		if inFlight {
			return "🕒", string(t.Status), "等了 " + formatElapsed(now.Sub(t.CreatedAt))
		}
		return "➖", string(t.Status), formatClock(t, loc)
	}
}

func formatClock(t *runtimedomain.Task, loc *time.Location) string {
	at := recentTime(t)
	return at.In(loc).Format("01-02 15:04")
}

func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Round(time.Second).Seconds())
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%d小时%d分", h, m)
	case m > 0:
		return fmt.Sprintf("%d分%02d秒", m, s)
	default:
		return fmt.Sprintf("%d秒", s)
	}
}
