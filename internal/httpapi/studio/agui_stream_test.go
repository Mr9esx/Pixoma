package studio_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// approvalDuringStatusRead 在 pump 检查运行状态之前写入最后几条事件并进入等待批准，
// 复现「运行结束与事件落库同时发生」的交错。
type approvalDuringStatusRead struct {
	domain.Repository
	write func() error
	reads int
	fired bool
}

func (r *approvalDuringStatusRead) GetRun(ctx context.Context, accountID, runID string) (*domain.Run, error) {
	r.reads++
	// 第一次读取来自 attach 校验，第二次来自 pump 的状态检查。
	if r.reads == 2 {
		r.fired = true
		if err := r.write(); err != nil {
			return nil, err
		}
	}
	return r.Repository.GetRun(ctx, accountID, runID)
}

func TestStudioAGUIStreamClosesTheMessageBeforeTheInterrupt(t *testing.T) {
	handler, _ := newHandler(t)
	repo := handler.Repo
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 16, 30, 0, 0, time.UTC)
	const (
		accountID = "account-a"
		sessionID = "session-race"
		runID     = "run-race"
		messageID = "assistant-race"
	)

	session, err := domain.NewSession(sessionID, accountID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun(runID, sessionID, accountID, "message-race", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	appendEvent := func(sequence uint64, eventType, payload string) error {
		return repo.AppendEvent(ctx, &domain.Event{
			ID: fmt.Sprintf("event-race-%d", sequence), RunID: runID, SessionID: sessionID,
			AccountID: accountID, Sequence: sequence, Type: eventType,
			Payload: []byte(payload), CreatedAt: now.Add(time.Duration(sequence) * time.Second),
		})
	}
	if err := appendEvent(1, studioapp.EventTextMessageStart, `{"message_id":"`+messageID+`","role":"assistant"}`); err != nil {
		t.Fatal(err)
	}
	if err := appendEvent(2, studioapp.EventTextMessageContent, `{"message_id":"`+messageID+`","delta":"正在准备文件"}`); err != nil {
		t.Fatal(err)
	}

	writer := &approvalDuringStatusRead{Repository: repo, write: func() error {
		if err := appendEvent(3, studioapp.EventTextMessageEnd, `{"message_id":"`+messageID+`","content":"正在准备文件"}`); err != nil {
			return err
		}
		if err := appendEvent(4, studioapp.EventApprovalRequired, `{"approval_id":"approval-race","tool_call_id":"tool-race","action":"asset.create_text.race","description":"创建资产「大纲.md」"}`); err != nil {
			return err
		}
		approval, err := domain.NewApproval("approval-race", runID, sessionID, accountID, "tool-race", "asset.create_text.race", "创建资产「大纲.md」", now)
		if err != nil {
			return err
		}
		if err := repo.CreateApproval(ctx, approval); err != nil {
			return err
		}
		waiting, err := repo.GetRun(ctx, accountID, runID)
		if err != nil {
			return err
		}
		if err := waiting.WaitForApproval(now); err != nil {
			return err
		}
		return repo.UpdateRun(ctx, waiting)
	}}
	handler.Repo = writer

	router := chi.NewRouter()
	handler.Mount(router)
	stream := request(t, router, http.MethodPost, "/agui", map[string]any{
		"threadId": sessionID, "runId": "browser-run-race",
		"attachRunId": runID, "afterSequence": 0, "messages": []any{},
	}, accountID)
	if stream.Code != http.StatusOK {
		t.Fatalf("stream status = %d %s", stream.Code, stream.Body.String())
	}
	if !writer.fired {
		t.Fatal("运行状态检查之前没有写入事件，测试没有复现并发写入")
	}

	events := decodeAGUIStream(t, stream.Body.String())
	messageEnd := indexOfAGUIEvent(events, "TEXT_MESSAGE_END")
	runFinished := indexOfAGUIEvent(events, "RUN_FINISHED")
	if messageEnd < 0 || runFinished < 0 {
		t.Fatalf("流缺少消息结束或运行结束：%s", stream.Body.String())
	}
	if messageEnd > runFinished {
		t.Fatalf("消息结束在运行结束之后（%d > %d）：%s", messageEnd, runFinished, stream.Body.String())
	}
	outcome, _ := events[runFinished]["outcome"].(map[string]any)
	if outcome["type"] != "interrupt" {
		t.Fatalf("运行结束结果 = %#v", outcome)
	}
	interrupts, _ := outcome["interrupts"].([]any)
	if len(interrupts) != 1 {
		t.Fatalf("中断列表 = %#v", outcome)
	}
	first, _ := interrupts[0].(map[string]any)
	if first["message"] != "创建资产「大纲.md」" {
		t.Fatalf("中断文案 = %#v", first["message"])
	}
}

func decodeAGUIStream(t *testing.T, body string) []map[string]any {
	t.Helper()
	events := make([]map[string]any, 0)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if line == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("解析事件 %q：%v", line, err)
		}
		events = append(events, event)
	}
	return events
}

func indexOfAGUIEvent(events []map[string]any, eventType string) int {
	for index, event := range events {
		if event["type"] == eventType {
			return index
		}
	}
	return -1
}
