package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestApprovalSequenceContinuesAfterTwoHundredEvents(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	run, err := domain.NewRun("run-long-approval", "session-long-approval", "account-a", "message-long-approval", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := run.WaitForApproval(now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	approval, err := domain.NewApproval("approval-long", run.ID, run.SessionID, run.AccountID, "tool-long", "asset.create", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateApproval(ctx, approval); err != nil {
		t.Fatal(err)
	}
	for sequence := uint64(1); sequence <= 205; sequence++ {
		if err := repo.AppendEvent(ctx, &domain.Event{
			ID: fmt.Sprintf("event-long-%d", sequence), RunID: run.ID,
			SessionID: run.SessionID, AccountID: run.AccountID, Sequence: sequence,
			Type: "CUSTOM", Payload: []byte(`{}`), CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	service := &studioapp.ApprovalService{Repo: repo, Queue: &queueSpy{}}
	if err := service.Resolve(ctx, studioapp.ResolveApprovalInput{
		AccountID: run.AccountID, ApprovalID: approval.ID, Approved: true,
	}); err != nil {
		t.Fatal(err)
	}
	events, err := repo.ListEventsAfter(ctx, run.AccountID, run.ID, 205, 10)
	if err != nil || len(events) != 1 || events[0].Sequence != 206 || events[0].Type != studioapp.EventApprovalResolved {
		t.Fatalf("events after 205 = (%#v, %v)", events, err)
	}
}

func TestResolveApprovalWithoutCheckpointInterruptsLegacyRun(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	run, err := domain.NewRun("legacy-waiting", "session-legacy", "account-a", "message-legacy", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := run.WaitForApproval(now); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	approval, err := domain.NewApproval("legacy-approval", run.ID, run.SessionID, run.AccountID, "tool-1", "asset.create", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateApproval(ctx, approval); err != nil {
		t.Fatal(err)
	}
	queue := &queueSpy{}
	service := &studioapp.ApprovalService{Repo: repo, Queue: queue, Checkpoints: repo.Checkpoints()}
	if err := service.Resolve(ctx, studioapp.ResolveApprovalInput{AccountID: "account-a", ApprovalID: approval.ID, Approved: true}); err == nil {
		t.Fatal("approval without Eino checkpoint was accepted")
	}
	updated, err := repo.GetRun(ctx, run.AccountID, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.RunFailed || updated.ErrorCode != "checkpoint_missing" {
		t.Fatalf("legacy run status = %#v", updated)
	}
	if len(queue.items) != 0 {
		t.Fatalf("legacy run was requeued: %#v", queue.items)
	}
}

func TestApprovalPausesAndResumesRun(t *testing.T) {
	repo := openRepository(t)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ids := &idSequence{}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: studioapp.NewMockEngine(), IDs: ids.Next})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Queue: runner}

	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{
		AccountID: "account-a", Text: "生成一张分镜预览图",
	})
	if err != nil {
		t.Fatal(err)
	}
	waitRunStatus(t, repo, result.Run.ID, domain.RunWaitingApproval)
	approvals, err := repo.ListApprovals(context.Background(), "account-a", result.Run.ID)
	if err != nil || len(approvals) != 1 || approvals[0].Status != domain.ApprovalPending {
		t.Fatalf("approvals = (%#v, %v)", approvals, err)
	}

	approvalService := &studioapp.ApprovalService{Repo: repo, Queue: runner}
	if err := approvalService.Resolve(context.Background(), studioapp.ResolveApprovalInput{
		AccountID: "account-a", ApprovalID: approvals[0].ID, Approved: true,
	}); err != nil {
		t.Fatalf("Resolve(approved) error = %v", err)
	}
	waitRunStatus(t, repo, result.Run.ID, domain.RunSucceeded)
	resolved, err := repo.GetApproval(context.Background(), "account-a", approvals[0].ID)
	if err != nil || resolved.Status != domain.ApprovalApproved {
		t.Fatalf("resolved approval = (%#v, %v)", resolved, err)
	}
	assets, err := repo.ListSessionAssets(context.Background(), "account-a", result.Session.ID, 10)
	if err != nil || len(assets) != 2 {
		t.Fatalf("assets after approval = (%#v, %v)", assets, err)
	}
}

func TestRejectedApprovalCancelsRun(t *testing.T) {
	repo := openRepository(t)
	blobs, _ := localfs.New(t.TempDir())
	ids := &idSequence{}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: studioapp.NewMockEngine(), IDs: ids.Next})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	t.Cleanup(runner.Close)
	service := &studioapp.Service{Repo: repo, IDs: ids.Next, Queue: runner}
	result, err := service.SendMessage(context.Background(), studioapp.SendMessageInput{AccountID: "account-a", Text: "生成分镜"})
	if err != nil {
		t.Fatal(err)
	}
	waitRunStatus(t, repo, result.Run.ID, domain.RunWaitingApproval)
	approvals, _ := repo.ListApprovals(context.Background(), "account-a", result.Run.ID)
	approvalService := &studioapp.ApprovalService{Repo: repo, Queue: runner, Now: time.Now}
	if err := approvalService.Resolve(context.Background(), studioapp.ResolveApprovalInput{
		AccountID: "account-a", ApprovalID: approvals[0].ID, Approved: false,
	}); err != nil {
		t.Fatalf("Resolve(rejected) error = %v", err)
	}
	waitRunStatus(t, repo, result.Run.ID, domain.RunCancelled)
}
