package application_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type idSequence struct {
	mu sync.Mutex
	n  int
}

func (s *idSequence) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return fmt.Sprintf("id-%d", s.n)
}

type queueSpy struct {
	mu    sync.Mutex
	items []studioapp.RunRef
}

func (q *queueSpy) Enqueue(ref studioapp.RunRef) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, ref)
	return nil
}

func (q *queueSpy) last() (studioapp.RunRef, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return studioapp.RunRef{}, false
	}
	return q.items[len(q.items)-1], true
}

type staticTitleGenerator struct {
	title string
}

func (g staticTitleGenerator) GenerateTitle(context.Context, string) (string, error) {
	return g.title, nil
}

func openRepository(t *testing.T) *persistence.GormRepository {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return persistence.NewGormRepository(gdb)
}

func TestSendMessageCreatesSessionTurnAndAIGeneratedTitle(t *testing.T) {
	repo := openRepository(t)
	ids := &idSequence{}
	queue := &queueSpy{}
	now := time.Date(2026, 9, 21, 14, 0, 0, 0, time.UTC)
	service := &studioapp.Service{
		Repo:   repo,
		IDs:    ids.Next,
		Now:    func() time.Time { return now },
		Titles: staticTitleGenerator{title: "雨夜侦探的第一桩委托"},
		Queue:  queue,
	}

	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a",
		Text:      "帮我构思一个发生在雨夜的侦探漫画",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if result.Session.Title != "雨夜侦探的第一桩委托" {
		t.Fatalf("title = %q", result.Session.Title)
	}
	if result.Message.Role != domain.MessageRoleUser || result.Run.Status != domain.RunQueued {
		t.Fatalf("turn = %#v", result)
	}
	messages, err := repo.ListMessages(context.Background(), "account-a", result.Session.ID, 10)
	if err != nil || len(messages) != 1 {
		t.Fatalf("persisted messages = (%#v, %v)", messages, err)
	}
	ref, ok := queue.last()
	if !ok || ref.AccountID != "account-a" || ref.RunID != result.Run.ID {
		t.Fatalf("queued run = %#v, ok = %v", ref, ok)
	}
}

func TestIdempotentSendKeepsOneTurnAndRejectsConcurrentDifferentRequest(t *testing.T) {
	repo := openRepository(t)
	queue := &queueSpy{}
	service := &studioapp.Service{Repo: repo, IDs: (&idSequence{}).Next, Queue: queue}
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.SendMessage(ctx, studioapp.SendMessageInput{
		AccountID: "account-a", SessionID: session.ID, RequestID: "request-1", Text: "第一个问题",
	})
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := service.SendMessage(ctx, studioapp.SendMessageInput{
		AccountID: "account-a", SessionID: session.ID, RequestID: "request-1", Text: "第一个问题",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Run.ID != first.Run.ID || repeated.Message.ID != first.Message.ID {
		t.Fatalf("repeat created another turn: first=%s second=%s", first.Run.ID, repeated.Run.ID)
	}
	if _, err := service.SendMessage(ctx, studioapp.SendMessageInput{
		AccountID: "account-a", SessionID: session.ID, RequestID: "request-1", Text: "不同内容",
	}); err == nil {
		t.Fatal("same request id with different content was accepted")
	}
	if _, err := service.SendMessage(ctx, studioapp.SendMessageInput{
		AccountID: "account-a", SessionID: session.ID, RequestID: "request-2", Text: "第二个问题",
	}); err == nil {
		t.Fatal("concurrent different request was accepted")
	}
	messages, err := repo.ListMessages(ctx, "account-a", session.ID, 10)
	if err != nil || len(messages) != 1 {
		t.Fatalf("messages = (%d, %v), want one", len(messages), err)
	}
	if len(queue.items) != 1 {
		t.Fatalf("queue items = %d, want one", len(queue.items))
	}
}

func TestIdempotentSendConcurrentRetriesReturnOneRun(t *testing.T) {
	repo := openRepository(t)
	queue := &queueSpy{}
	service := &studioapp.Service{Repo: repo, IDs: (&idSequence{}).Next, Queue: queue}
	ctx := context.Background()
	session, err := service.CreateSession(ctx, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Rename("并发测试", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make([]*studioapp.SendMessageResult, 4)
	errors := make([]error, 4)
	start := make(chan struct{})
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			results[index], errors[index] = service.SendMessage(ctx, studioapp.SendMessageInput{
				AccountID: "account-a", SessionID: session.ID, RequestID: "same-request", Text: "问题",
			})
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errors {
		if err != nil {
			t.Fatalf("send[%d]: %v", i, err)
		}
		if results[i].Run.ID != results[0].Run.ID {
			t.Fatalf("send[%d] run = %s, want %s", i, results[i].Run.ID, results[0].Run.ID)
		}
	}
	if len(queue.items) != 1 {
		t.Fatalf("queue items = %d, want one", len(queue.items))
	}
}

func TestSendMessageSnapshotsExplicitlySelectedEnabledSkills(t *testing.T) {
	repo := openRepository(t)
	ids := &idSequence{}
	skill, err := domain.NewSkill(ids.Next(), "account-a", "漫画分镜", "拆分镜头", "先输出镜头表", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	skill.Enabled = true
	if err := repo.CreateSkill(context.Background(), skill); err != nil {
		t.Fatal(err)
	}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: time.Now, Queue: &queueSpy{}}

	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a", Text: "为角色设计分镜", SkillIDs: []string{skill.ID},
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if len(result.Run.SkillIDs) != 1 || result.Run.SkillIDs[0] != skill.ID {
		t.Fatalf("run skills = %#v", result.Run.SkillIDs)
	}
	persisted, err := repo.GetRun(context.Background(), "account-a", result.Run.ID)
	if err != nil || len(persisted.SkillIDs) != 1 || persisted.SkillIDs[0] != skill.ID {
		t.Fatalf("persisted run = %#v, err = %v", persisted, err)
	}
}

func TestSendMessageUsesSafeFallbackWhenTitleGenerationFails(t *testing.T) {
	repo := openRepository(t)
	ids := &idSequence{}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: time.Now, Queue: &queueSpy{}}
	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a",
		Text:      "这是一个很长很长的标题内容，需要被安全地截断而不是阻止用户发送消息",
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if count := len([]rune(result.Session.Title)); count == 0 || count > 20 {
		t.Fatalf("fallback title rune count = %d, title = %q", count, result.Session.Title)
	}
}

type blockingExecutor struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (e *blockingExecutor) Execute(ctx context.Context, _ *domain.Run) error {
	e.once.Do(func() { close(e.started) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.release:
		return nil
	}
}

func TestBackgroundRunnerOutlivesRequestContext(t *testing.T) {
	repo := openRepository(t)
	now := time.Now().UTC()
	run, _ := domain.NewRun("run-1", "session-1", "account-a", "message-1", now)
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	executor := &blockingExecutor{started: make(chan struct{}), release: make(chan struct{})}
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)

	requestCtx, cancelRequest := context.WithCancel(context.Background())
	if err := runner.Enqueue(studioapp.RunRef{AccountID: "account-a", RunID: run.ID}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	cancelRequest()
	_ = requestCtx
	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("background run did not start")
	}
	close(executor.release)
	waitRunStatus(t, repo, run.ID, domain.RunSucceeded)
}

func TestBackgroundRunnerRecoversPersistedRuns(t *testing.T) {
	repo := openRepository(t)
	now := time.Now().UTC()
	queued, _ := domain.NewRun("run-queued", "session-1", "account-a", "message-1", now)
	running, _ := domain.NewRun("run-running", "session-1", "account-b", "message-2", now)
	if err := running.Start(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(context.Background(), queued); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(context.Background(), running); err != nil {
		t.Fatal(err)
	}
	executor := &countingExecutor{}
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	count, err := runner.Recover(context.Background())
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("Recover() count = %d, want 1", count)
	}
	waitRunStatusForAccount(t, repo, "account-a", queued.ID, domain.RunSucceeded)
	waitRunStatusForAccount(t, repo, "account-b", running.ID, domain.RunFailed)
}

type countingExecutor struct{}

func (*countingExecutor) Execute(context.Context, *domain.Run) error { return nil }

func TestCancelAndRetryRun(t *testing.T) {
	repo := openRepository(t)
	now := time.Now().UTC()
	run, _ := domain.NewRun("run-1", "session-1", "account-a", "message-1", now)
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	executor := &blockingExecutor{started: make(chan struct{}), release: make(chan struct{})}
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	if err := runner.Enqueue(studioapp.RunRef{AccountID: "account-a", RunID: run.ID}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not start")
	}
	if err := runner.Cancel(context.Background(), "account-a", run.ID); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	waitRunStatus(t, repo, run.ID, domain.RunCancelled)

	ids := &idSequence{n: 100}
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Now: time.Now, Queue: runner}
	retried, err := service.RetryRun(context.Background(), "account-a", run.ID)
	if err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if retried.ID == run.ID || retried.TriggerMessageID != run.TriggerMessageID || retried.Status != domain.RunQueued {
		t.Fatalf("retried run = %#v", retried)
	}
}

func waitRunStatus(t *testing.T, repo *persistence.GormRepository, runID string, want domain.RunStatus) {
	waitRunStatusForAccount(t, repo, "account-a", runID, want)
}

func waitRunStatusForAccount(t *testing.T, repo *persistence.GormRepository, accountID, runID string, want domain.RunStatus) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		run, err := repo.GetRun(context.Background(), accountID, runID)
		if err == nil && run.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	run, err := repo.GetRun(context.Background(), accountID, runID)
	t.Fatalf("run status = %#v, error = %v, want %q", run, err, want)
}
