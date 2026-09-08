// Package application implements admin compute-node lifecycle operations that
// span edge/task/session repositories in a single transaction.
package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	instpersist "github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/platform/notify"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	runtimedomain "github.com/Mr9esx/Pixoma/internal/tasks/domain"
	taskpersist "github.com/Mr9esx/Pixoma/internal/tasks/infrastructure/persistence"
)

// ErrNeedsAck is returned when the edge has running tasks and the caller did
// not confirm marking them failed.
var ErrNeedsAck = errors.New("edge delete needs ack for running tasks")

// DeleteSummary reports what the cleanup did.
type DeleteSummary struct {
	FailedTasks int `json:"failed_tasks"`
}

// Service deletes a compute node with task cleanup in one DB transaction.
type Service struct {
	db     *gorm.DB
	notify notify.Publisher
	now    func() time.Time
}

// NewService constructs a delete service over the shared database handle.
func NewService(db *gorm.DB, n notify.Publisher) *Service {
	return &Service{db: db, notify: n, now: func() time.Time { return time.Now().UTC() }}
}

// DeleteEdge fails the edge's running tasks (edge_deleted), deletes the edge
// row, then notifies task owners best-effort after commit.
func (s *Service) DeleteEdge(ctx context.Context, id sharedkernel.EdgeID, ack bool) (DeleteSummary, error) {
	var summary DeleteSummary
	var taskNotifies []sharedkernel.UserNotify

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		edgeRepo := instpersist.NewEdgeRepository(tx)
		taskRepo := taskpersist.NewTaskRepository(tx)
		sessRepo := sesspersist.NewSessionRepository(tx)

		if _, err := edgeRepo.Get(ctx, id); err != nil {
			return err
		}
		running, err := taskRepo.List(ctx, runtimedomain.AdminListQuery{EdgeID: id, Status: sharedkernel.TaskRunning})
		if err != nil {
			return err
		}
		if len(running) > 0 && !ack {
			return ErrNeedsAck
		}

		now := s.now()
		for _, t := range running {
			if err := t.MarkFailed(sharedkernel.TaskErrorEdgeDeleted, sharedkernel.EdgeDeletedMessage, now); err != nil {
				continue // already terminal (e.g. success raced in before delete)
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
		return edgeRepo.Delete(ctx, id)
	})
	if err != nil {
		return DeleteSummary{}, err
	}

	for _, n := range taskNotifies {
		if n.ChatID == "" {
			continue
		}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("edge delete: task notify failed", "task", n.TaskID, "err", err)
		}
	}
	return summary, nil
}
