// Package topicadmin implements admin dispatch-topic lifecycle operations
// that span topic/case/edge/task repositories in a single transaction.
package topicadmin

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	casepersist "github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	sesspersist "github.com/mr9esx/comfyui_tgbot/internal/conversation/infrastructure/persistence"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/notify"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	topicpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	taskpersist "github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// ErrNeedsAck is returned when the topic is referenced by cases or edges and
// the caller did not confirm cleanup.
var ErrNeedsAck = errors.New("topic delete needs ack for references")

// ErrDefaultProtected is returned when deleting the system default topic.
var ErrDefaultProtected = errors.New("default topic cannot be deleted")

// DeleteSummary reports what the cleanup did.
type DeleteSummary struct {
	RemovedCaseRules int `json:"removed_case_rules"`
	RemovedEdgeSubs  int `json:"removed_edge_subscriptions"`
	FailedTasks      int `json:"failed_tasks"`
}

// Service deletes a dispatch topic with cleanup in one DB transaction.
type Service struct {
	db     *gorm.DB
	notify notify.Publisher
	now    func() time.Time
}

// NewService constructs a delete service over the shared database handle.
func NewService(db *gorm.DB, n notify.Publisher) *Service {
	return &Service{db: db, notify: n, now: func() time.Time { return time.Now().UTC() }}
}

// DeleteTopic removes case routing rules targeting the topic, unsubscribes
// edges, fails queued tasks (topic_deleted) and deletes the topic row
// atomically. Notifications are sent best-effort after commit.
func (s *Service) DeleteTopic(ctx context.Context, key string, ack bool) (DeleteSummary, error) {
	var summary DeleteSummary
	var taskNotifies []sharedkernel.UserNotify

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		topicRepo := topicpersist.NewTopicRepository(tx)
		caseRepo := casepersist.NewGormRepository(tx)
		edgeRepo := instpersist.NewEdgeRepository(tx)
		taskRepo := taskpersist.NewTaskRepository(tx)
		sessRepo := sesspersist.NewSessionRepository(tx)

		if _, err := topicRepo.Get(ctx, key); err != nil {
			return err
		}
		if key == topic.DefaultKey {
			return ErrDefaultProtected
		}

		cases, err := caseRepo.List(ctx, catalogdomain.ListQuery{})
		if err != nil {
			return err
		}
		edges, err := edgeRepo.List(ctx)
		if err != nil {
			return err
		}

		var caseRefs, edgeRefs int
		for _, c := range cases {
			if c.Document.Routing == nil {
				continue
			}
			for _, rule := range c.Document.Routing.Rules {
				if rule.Topic == key {
					caseRefs++
				}
			}
		}
		for _, e := range edges {
			if containsTopic(e.SubscribeTopics, key) {
				edgeRefs++
			}
		}
		if caseRefs+edgeRefs > 0 && !ack {
			return ErrNeedsAck
		}

		now := s.now()
		for _, c := range cases {
			if c.Document.Routing == nil {
				continue
			}
			rules := c.Document.Routing.Rules
			kept := rules[:0]
			for _, rule := range rules {
				if rule.Topic != key {
					kept = append(kept, rule)
				}
			}
			if len(kept) == len(rules) {
				continue
			}
			summary.RemovedCaseRules += len(rules) - len(kept)
			c.Document.Routing.Rules = kept
			if err := caseRepo.Save(ctx, c); err != nil {
				return err
			}
		}
		for _, e := range edges {
			if !containsTopic(e.SubscribeTopics, key) {
				continue
			}
			topics := removeTopic(e.SubscribeTopics, key)
			if err := edgeRepo.UpdateSubscribeTopics(ctx, e.ID, topics); err != nil {
				return err
			}
			summary.RemovedEdgeSubs++
		}

		queued, err := taskRepo.ListByTopic(ctx, key, runtimedomain.ListByTopicQuery{Status: sharedkernel.TaskQueued})
		if err != nil {
			return err
		}
		for _, t := range queued {
			if err := t.MarkFailed(sharedkernel.TaskErrorTopicDeleted, sharedkernel.TopicDeletedMessage, now); err != nil {
				continue
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
		return topicRepo.Delete(ctx, key)
	})
	if err != nil {
		return DeleteSummary{}, err
	}

	for _, n := range taskNotifies {
		if n.ChatID == "" {
			continue
		}
		if err := s.notify.Publish(ctx, n); err != nil {
			slog.Warn("topic delete: task notify failed", "task", n.TaskID, "err", err)
		}
	}
	return summary, nil
}

func containsTopic(topics []string, key string) bool {
	for _, t := range topics {
		if t == key {
			return true
		}
	}
	return false
}

func removeTopic(topics []string, key string) []string {
	var out []string
	for _, t := range topics {
		if t != key {
			out = append(out, t)
		}
	}
	return out
}
