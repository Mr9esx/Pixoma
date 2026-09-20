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
	if count != 2 {
		t.Fatalf("Recover() count = %d, want 2", count)
	}
	waitRunStatusForAccount(t, repo, "account-a", queued.ID, domain.RunSucceeded)
	waitRunStatusForAccount(t, repo, "account-b", running.ID, domain.RunSucceeded)
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
