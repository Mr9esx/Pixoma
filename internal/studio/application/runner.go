package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type RunStore interface {
	GetRun(ctx context.Context, accountID, runID string) (*domain.Run, error)
	UpdateRun(ctx context.Context, run *domain.Run) error
	ListRecoverableRuns(ctx context.Context, limit int) ([]*domain.Run, error)
	ListRecoverableRunsAfter(ctx context.Context, afterID string, limit int) ([]*domain.Run, error)
}

type Executor interface {
	Execute(ctx context.Context, run *domain.Run) error
}

type RunnerOptions struct {
	Workers   int
	QueueSize int
	Now       func() time.Time
}

type BackgroundRunner struct {
	store    RunStore
	executor Executor
	now      func() time.Time
	ctx      context.Context
	cancel   context.CancelFunc
	queue    chan RunRef
	wg       sync.WaitGroup
	close    sync.Once
	mu       sync.Mutex
	running  map[string]runningRun
}

type runningRun struct {
	accountID string
	cancel    context.CancelFunc
}

func NewBackgroundRunner(store RunStore, executor Executor, options RunnerOptions) *BackgroundRunner {
	workers := options.Workers
	if workers <= 0 {
		workers = 2
	}
	queueSize := options.QueueSize
	if queueSize <= 0 {
		queueSize = 128
	}
	now := options.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	ctx, cancel := context.WithCancel(context.Background())
	runner := &BackgroundRunner{
		store: store, executor: executor, now: now,
		ctx: ctx, cancel: cancel, queue: make(chan RunRef, queueSize), running: map[string]runningRun{},
	}
	for range workers {
		runner.wg.Add(1)
		go runner.worker()
	}
	return runner
}

func (r *BackgroundRunner) Enqueue(ref RunRef) error {
	if r == nil || r.store == nil || r.executor == nil {
		return fmt.Errorf("studio: background runner is not configured")
	}
	if ref.AccountID == "" || ref.RunID == "" {
		return fmt.Errorf("%w: run reference is required", domain.ErrInvalid)
	}
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	case r.queue <- ref:
		slog.Debug("Studio 运行已排队", "run_id", ref.RunID)
		return nil
	}
}

func (r *BackgroundRunner) Cancel(ctx context.Context, accountID, runID string) error {
	run, err := r.store.GetRun(ctx, accountID, runID)
	if err != nil {
		return err
	}
	if run.Status.Terminal() {
		return domain.ErrInvalidTransition
	}
	if err := run.Cancel(r.now()); err != nil {
		return err
	}
	if err := r.store.UpdateRun(ctx, run); err != nil {
		return err
	}
	r.mu.Lock()
	active := r.running[runID]
	r.mu.Unlock()
	if active.cancel != nil {
		active.cancel()
	}
	slog.Info("Studio 运行已取消", "run_id", runID)
	return nil
}

func (r *BackgroundRunner) CancelAccount(accountID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, active := range r.running {
		if active.accountID == accountID {
			active.cancel()
		}
	}
}

// Recover only requeues work that never started. A running Eino run without a
// safe checkpoint cannot be replayed after process restart: doing so could
// repeat model calls and side-effecting tools.
func (r *BackgroundRunner) Recover(ctx context.Context) (int, error) {
	if r == nil || r.store == nil {
		return 0, fmt.Errorf("studio: background runner is not configured")
	}
	count := 0
	afterID := ""
	for {
		runs, err := r.store.ListRecoverableRunsAfter(ctx, afterID, 200)
		if err != nil {
			return count, err
		}
		for _, run := range runs {
			afterID = run.ID
			if run.Status == domain.RunRunning {
				if err := run.Fail("interrupted_on_restart", "运行因服务重启中断，请手动重试", r.now()); err != nil {
					return count, err
				}
				if err := r.store.UpdateRun(ctx, run); err != nil {
					return count, err
				}
				slog.Warn("Studio 运行因服务重启中断", "run_id", run.ID)
				continue
			}
			if err := r.Enqueue(RunRef{AccountID: run.AccountID, RunID: run.ID}); err != nil {
				return count, err
			}
			count++
		}
		if len(runs) < 200 {
			return count, nil
		}
	}
}

func (r *BackgroundRunner) Close() {
	if r == nil {
		return
	}
	r.close.Do(func() {
		r.cancel()
		r.wg.Wait()
	})
}

func (r *BackgroundRunner) worker() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		case ref := <-r.queue:
			r.execute(ref)
		}
	}
}

func (r *BackgroundRunner) execute(ref RunRef) {
	run, err := r.store.GetRun(r.ctx, ref.AccountID, ref.RunID)
	if err != nil {
		slog.Error("读取 Studio 运行失败", "run_id", ref.RunID, "err", err)
		return
	}
	if run.Status != domain.RunQueued && run.Status != domain.RunRunning {
		slog.Debug("跳过 Studio 运行", "run_id", ref.RunID, "status", run.Status)
		return
	}
	if run.Status == domain.RunQueued {
		if err := run.Start(r.now()); err != nil {
			slog.Error("启动 Studio 运行失败", "run_id", run.ID, "err", err)
			return
		}
		if err := r.store.UpdateRun(r.ctx, run); err != nil {
			slog.Error("保存 Studio 运行开始状态失败", "run_id", run.ID, "err", err)
			return
		}
	}
	started := time.Now()
	slog.Info("Studio 运行已开始", "run_id", run.ID, "session_id", run.SessionID)
	runCtx, cancel := context.WithCancel(r.ctx)
	r.mu.Lock()
	r.running[run.ID] = runningRun{accountID: run.AccountID, cancel: cancel}
	r.mu.Unlock()
	err = r.executor.Execute(runCtx, run)
	cancel()
	r.mu.Lock()
	delete(r.running, run.ID)
	r.mu.Unlock()

	fresh, getErr := r.store.GetRun(context.WithoutCancel(r.ctx), ref.AccountID, ref.RunID)
	if getErr != nil {
		slog.Error("读取 Studio 运行结束状态失败", "run_id", run.ID, "err", getErr)
		return
	}
	if fresh.Status.Terminal() || fresh.Status == domain.RunWaitingApproval || fresh.Status == domain.RunWaitingClarification {
		slog.Debug("Studio 运行已暂停或结束", "run_id", run.ID, "status", fresh.Status)
		return
	}
	if err == nil {
		if succeedErr := fresh.Succeed(r.now()); succeedErr != nil {
			slog.Error("完成 Studio 运行失败", "run_id", run.ID, "err", succeedErr)
			return
		}
		if updateErr := r.store.UpdateRun(context.WithoutCancel(r.ctx), fresh); updateErr != nil {
			slog.Error("保存 Studio 运行完成状态失败", "run_id", run.ID, "err", updateErr)
			return
		}
		slog.Info("Studio 运行成功", "run_id", run.ID, "duration", time.Since(started))
		return
	}
	if errors.Is(err, context.Canceled) && r.ctx.Err() != nil {
		// Process shutdown leaves the run non-terminal so startup recovery can
		// enqueue it again instead of recording an infrastructure failure.
		return
	}
	slog.Error("Studio 运行失败", "run_id", run.ID, "err", err, "duration", time.Since(started))
	if failErr := fresh.Fail("execution_failed", err.Error(), r.now()); failErr != nil {
		slog.Error("设置 Studio 运行失败状态时出错", "run_id", run.ID, "err", failErr)
		return
	}
	if updateErr := r.store.UpdateRun(context.WithoutCancel(r.ctx), fresh); updateErr != nil {
		slog.Error("保存 Studio 运行失败状态时出错", "run_id", run.ID, "err", updateErr)
	}
}

var _ RunQueue = (*BackgroundRunner)(nil)
