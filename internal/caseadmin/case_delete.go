// Package caseadmin implements admin case lifecycle operations that span
// multiple repositories in a single transaction.
package caseadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	mcpersist "github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNeedsAck is returned when the case is referenced by menu/card entries
// and the caller did not confirm removal with ack_references.
var ErrNeedsAck = errors.New("case delete needs ack for references")

const sessionTerminatedMessage = "该工作流已被管理员删除，当前会话已结束。"

// RemovedPlacement is a menu/card entry that referenced the deleted case.
type RemovedPlacement struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name,omitempty"`
	ItemID      string `json:"item_id"`
	Label       string `json:"label"`
	Kind        string `json:"kind"`
}

// DeleteSummary reports what the cleanup did.
type DeleteSummary struct {
	RemovedPlacements  []RemovedPlacement `json:"removed_placements"`
	FailedTasks        int                `json:"failed_tasks"`
	TerminatedSessions int                `json:"terminated_sessions"`
}

// Service deletes a case with cleanup in one DB transaction.
type Service struct {
	db     *gorm.DB
	notify notify.Publisher
	now    func() time.Time
}

// NewService constructs a delete service over the shared database handle.
func NewService(db *gorm.DB, n notify.Publisher) *Service {
	return &Service{db: db, notify: n, now: func() time.Time { return time.Now().UTC() }}
}

// DeleteCase fails pending tasks, terminates active sessions, unlinks
// menu/card references and deletes the case row atomically. Notifications are
// sent best-effort after commit.
func (s *Service) DeleteCase(ctx context.Context, id sharedkernel.CaseID, ack bool) (DeleteSummary, error) {
	var summary DeleteSummary
	var taskNotifies []sharedkernel.UserNotify
	var terminatedChats []sharedkernel.ChatID

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		caseRepo := casepersist.NewGormRepository(tx)
		taskRepo := taskpersist.NewTaskRepository(tx)
		sessRepo := sesspersist.NewSessionRepository(tx)
		menuRepo := mcpersist.NewGormCardRepository(tx)

		if _, err := caseRepo.Get(ctx, id); err != nil {
			return err
		}
		idStr := fmt.Sprintf("%d", uint64(id))
		existing, err := menuRepo.WorkflowPlacements(ctx, idStr)
		if err != nil {
			return err
		}
		if len(existing) > 0 && !ack {
			return ErrNeedsAck
		}

		now := s.now()
		pending, err := taskRepo.List(ctx, runtimedomain.AdminListQuery{CaseID: id, Status: sharedkernel.TaskPending})
		if err != nil {
			return err
		}
		for _, t := range pending {
			if err := t.MarkFailed(sharedkernel.TaskErrorCaseDeleted, sharedkernel.CaseDeletedMessage, now); err != nil {
				return err
			}
			if err := taskRepo.Update(ctx, t); err != nil {
				return err
			}
			summary.FailedTasks++
			chatID := t.ChatID
			if chatID == "" {
				if sess, err := sessRepo.GetByID(ctx, t.SessionID); err == nil && sess != nil {
					chatID = sess.ChatID
				}
			}
			taskNotifies = append(taskNotifies, sharedkernel.UserNotify{
				ChatID:   chatID,
				TaskID:   t.ID,
				Kind:     "task_failed",
				ErrorMsg: t.ErrorMessage,
			})
		}

		sessions, err := sessRepo.ListActiveByCase(ctx, id)
		if err != nil {
			return err
		}
		for _, sess := range sessions {
			if err := sess.Exit(now); err != nil {
				return err
			}
			if err := sessRepo.Save(ctx, sess); err != nil {
				return err
			}
			summary.TerminatedSessions++
			terminatedChats = append(terminatedChats, sess.ChatID)
		}

		placements, err := menuRepo.RemoveWorkflowReferences(ctx, idStr)
		if err != nil {
			return err
		}
		for _, p := range placements {
			summary.RemovedPlacements = append(summary.RemovedPlacements, RemovedPlacement{
				ChannelID: p.ChannelID, ChannelName: p.ChannelName,
				ItemID: p.ItemID, Label: p.Label, Kind: p.Kind,
			})
		}
		return caseRepo.Delete(ctx, id)
	})
	if err != nil {
		return DeleteSummary{}, err
	}

	for _, n := range taskNotifies {
		if n.ChatID == "" {
			continue
		}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("case delete: task notify failed", "task", n.TaskID, "err", err)
		}
	}
	for _, chatID := range terminatedChats {
		n := sharedkernel.UserNotify{ChatID: chatID, Kind: "session_terminated", ErrorMsg: sessionTerminatedMessage}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("case delete: session notify failed", "chat", chatID, "err", err)
		}
	}
	return summary, nil
}
