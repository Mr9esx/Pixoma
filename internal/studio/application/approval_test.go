package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

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
