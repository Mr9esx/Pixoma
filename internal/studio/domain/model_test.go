package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewSessionNormalizesDefaults(t *testing.T) {
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	session, err := NewSession("session-1", "account-1", now)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	if session.Title != DefaultSessionTitle {
		t.Fatalf("Title = %q, want %q", session.Title, DefaultSessionTitle)
	}
	if session.PermissionMode != PermissionRequestApproval {
		t.Fatalf("PermissionMode = %q, want %q", session.PermissionMode, PermissionRequestApproval)
	}
	if session.Status != SessionActive {
		t.Fatalf("Status = %q, want %q", session.Status, SessionActive)
	}
	if !session.CreatedAt.Equal(now) || !session.UpdatedAt.Equal(now) {
		t.Fatalf("timestamps = (%v, %v), want %v", session.CreatedAt, session.UpdatedAt, now)
	}
}

func TestNewSessionRejectsMissingOwnership(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name      string
		id        string
		accountID string
	}{
		{name: "missing session id", accountID: "account-1"},
		{name: "missing account id", id: "session-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSession(tc.id, tc.accountID, now)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestSessionContextSummaryKeepsMessageBoundary(t *testing.T) {
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	session, err := NewSession("session-1", "account-1", now)
	if err != nil {
		t.Fatal(err)
	}
	updatedAt := now.Add(time.Minute)
	if err := session.UpdateContextSummary("用户要做一部悬疑漫画", "message-7", updatedAt); err != nil {
		t.Fatal(err)
	}
	if session.ContextSummary != "用户要做一部悬疑漫画" || session.ContextSummaryThroughMessageID != "message-7" || !session.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("summary state = %#v", session)
	}
	if err := session.UpdateContextSummary("", "message-8", updatedAt); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty summary error = %v, want ErrInvalid", err)
	}
}

func TestRunLifecycle(t *testing.T) {
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	run, err := NewRun("run-1", "session-1", "account-1", "message-1", now)
	if err != nil {
		t.Fatalf("NewRun() error = %v", err)
	}
	if run.Status != RunQueued {
		t.Fatalf("initial status = %q, want %q", run.Status, RunQueued)
	}

	startedAt := now.Add(time.Second)
	if err := run.Start(startedAt); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if run.Status != RunRunning || !run.StartedAt.Equal(startedAt) {
		t.Fatalf("after Start = (%q, %v)", run.Status, run.StartedAt)
	}

	waitingAt := startedAt.Add(time.Second)
	if err := run.WaitForApproval(waitingAt); err != nil {
		t.Fatalf("WaitForApproval() error = %v", err)
	}
	if run.Status != RunWaitingApproval {
		t.Fatalf("after WaitForApproval status = %q", run.Status)
	}

	resumedAt := waitingAt.Add(time.Second)
	if err := run.Resume(resumedAt); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}

	completedAt := resumedAt.Add(time.Second)
	if err := run.Succeed(completedAt); err != nil {
		t.Fatalf("Succeed() error = %v", err)
	}
	if run.Status != RunSucceeded || !run.CompletedAt.Equal(completedAt) {
		t.Fatalf("after Succeed = (%q, %v)", run.Status, run.CompletedAt)
	}
	if err := run.Start(completedAt.Add(time.Second)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Start() after completion error = %v, want ErrInvalidTransition", err)
	}
}

func TestApprovalCanOnlyBeResolvedOnce(t *testing.T) {
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	approval, err := NewApproval("approval-1", "run-1", "session-1", "account-1", "tool-call-1", "workflow.execute", now)
	if err != nil {
		t.Fatalf("NewApproval() error = %v", err)
	}
	if approval.Status != ApprovalPending {
		t.Fatalf("status = %q, want %q", approval.Status, ApprovalPending)
	}
	if err := approval.Approve("account-1", now.Add(time.Second)); err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if approval.Status != ApprovalApproved || approval.ResolvedBy != "account-1" {
		t.Fatalf("approval not resolved: %#v", approval)
	}
	if err := approval.Reject("account-1", now.Add(2*time.Second)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Reject() after approval error = %v, want ErrInvalidTransition", err)
	}
}

func TestAssetVersionsAreAppendOnly(t *testing.T) {
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	asset, err := NewAsset("asset-1", "session-1", "account-1", "故事大纲.md", AssetDocument, AssetOriginAgent, now)
	if err != nil {
		t.Fatalf("NewAsset() error = %v", err)
	}
	first, err := asset.AppendVersion("version-1", "text/markdown", "blob/story-v1.md", 128, now)
	if err != nil {
		t.Fatalf("AppendVersion(v1) error = %v", err)
	}
	second, err := asset.AppendVersion("version-2", "text/markdown", "blob/story-v2.md", 256, now.Add(time.Second))
	if err != nil {
		t.Fatalf("AppendVersion(v2) error = %v", err)
	}
	if first.Version != 1 || second.Version != 2 || asset.CurrentVersion != 2 {
		t.Fatalf("versions = (%d, %d), current = %d", first.Version, second.Version, asset.CurrentVersion)
	}
	if first.BlobKey != "blob/story-v1.md" {
		t.Fatalf("first version mutated: %#v", first)
	}
}

func TestFlowNodeValidationAndOrdering(t *testing.T) {
	now := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	node, err := NewFlowNode("node-1", "session-1", "account-1", FlowNodeAsset, "角色三视图", 20, now)
	if err != nil {
		t.Fatalf("NewFlowNode() error = %v", err)
	}
	if node.SortOrder != 20 || node.Type != FlowNodeAsset {
		t.Fatalf("node = %#v", node)
	}
	_, err = NewFlowNode("node-2", "session-1", "account-1", FlowNodeType("trace"), "Trace", 30, now)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid node type error = %v, want ErrInvalid", err)
	}
}
