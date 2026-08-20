package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// TaskRow is the GORM model for the tasks table.
type TaskRow struct {
	ID            string    `gorm:"primaryKey;size:64"`
	SessionID     string    `gorm:"column:session_id;size:36;index;not null"`
	CaseID        uint64    `gorm:"column:case_id;not null"`
	Status        string    `gorm:"size:32;not null;index"`
	EdgeID        string    `gorm:"column:edge_id;size:128;index"`
	DispatchTopic string    `gorm:"column:dispatch_topic;size:64;index"`
	Attempts      int       `gorm:"column:attempts;not null;default:0"`
	RequeueAt     time.Time `gorm:"column:requeue_at"`
	PromptID      string    `gorm:"column:prompt_id;size:128"`
	InputPrefix   string    `gorm:"column:input_prefix;size:512;not null"`
	JobRefJSON    string    `gorm:"column:job_ref_json;type:text"`
	LeaseUntil    time.Time `gorm:"column:lease_until"`
	OutputsJSON   string    `gorm:"column:outputs_json;type:text;not null"`
	ErrorCode     string    `gorm:"column:error_code;size:128"`
	ErrorMessage  string    `gorm:"column:error_message;type:text"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (TaskRow) TableName() string { return "tasks" }

// TaskRepository is a GORM-backed domain.TaskRepository.
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository constructs a TaskRepository.
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *domain.Task) error {
	if t == nil {
		return fmt.Errorf("runtime: nil task")
	}
	row, err := toRow(t)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *TaskRepository) Get(ctx context.Context, id sharedkernel.TaskID) (*domain.Task, error) {
	var row TaskRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", string(id)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromRow(row)
}

func (r *TaskRepository) Update(ctx context.Context, t *domain.Task) error {
	if t == nil {
		return fmt.Errorf("runtime: nil task")
	}
	row, err := toRow(t)
	if err != nil {
		return err
	}
	res := r.db.WithContext(ctx).Model(&TaskRow{}).Where("id = ?", row.ID).Updates(map[string]any{
		"session_id":     row.SessionID,
		"case_id":        row.CaseID,
		"status":         row.Status,
		"edge_id":        row.EdgeID,
		"dispatch_topic": row.DispatchTopic,
		"attempts":       row.Attempts,
		"requeue_at":     row.RequeueAt,
		"prompt_id":      row.PromptID,
		"input_prefix":   row.InputPrefix,
		"job_ref_json":   row.JobRefJSON,
		"lease_until":    row.LeaseUntil,
		"outputs_json":   row.OutputsJSON,
		"error_code":     row.ErrorCode,
		"error_message":  row.ErrorMessage,
		"updated_at":     row.UpdatedAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepository) ClaimQueued(ctx context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, now time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&TaskRow{}).
		Where("id = ? AND status = ?", string(id), string(sharedkernel.TaskPending)).
		Updates(map[string]any{
			"status":     string(sharedkernel.TaskQueued),
			"edge_id":    string(edgeID),
			"updated_at": now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return true, nil
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&TaskRow{}).Where("id = ?", string(id)).Count(&n).Error; err != nil {
		return false, err
	}
	if n == 0 {
		return false, domain.ErrTaskNotFound
	}
	return false, nil
}

func (r *TaskRepository) PrepareForClaim(ctx context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, jobRef sharedkernel.BlobRef, now time.Time) (bool, error) {
	if edgeID == "" || jobRef.Key == "" {
		return false, domain.ErrInvalidTransition
	}
	raw, err := json.Marshal(jobRef)
	if err != nil {
		return false, fmt.Errorf("runtime: encode job_ref: %w", err)
	}
	res := r.db.WithContext(ctx).Model(&TaskRow{}).
		Where("id = ? AND status = ?", string(id), string(sharedkernel.TaskPending)).
		Updates(map[string]any{
			"status":       string(sharedkernel.TaskQueued),
			"edge_id":      string(edgeID),
			"job_ref_json": string(raw),
			"lease_until":  time.Time{},
			"updated_at":   now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return true, nil
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&TaskRow{}).Where("id = ?", string(id)).Count(&n).Error; err != nil {
		return false, err
	}
	if n == 0 {
		return false, domain.ErrTaskNotFound
	}
	return false, nil
}

func (r *TaskRepository) ClaimNextWithLease(ctx context.Context, edgeID sharedkernel.EdgeID, lease time.Duration, now time.Time) (*domain.Task, error) {
	if edgeID == "" || lease <= 0 {
		return nil, nil
	}
	var claimed *domain.Task
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for {
			var row TaskRow
			err := tx.Where(
				"status = ? AND edge_id = ? AND job_ref_json != '' AND job_ref_json IS NOT NULL",
				string(sharedkernel.TaskQueued), string(edgeID),
			).Order("created_at ASC").First(&row).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			leaseUntil := now.Add(lease)
			res := tx.Model(&TaskRow{}).
				Where("id = ? AND status = ?", row.ID, string(sharedkernel.TaskQueued)).
				Updates(map[string]any{
					"status":      string(sharedkernel.TaskRunning),
					"lease_until": leaseUntil,
					"updated_at":  now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				continue
			}
			row.Status = string(sharedkernel.TaskRunning)
			row.LeaseUntil = leaseUntil
			row.UpdatedAt = now
			t, err := fromRow(row)
			if err != nil {
				return err
			}
			claimed = t
			return nil
		}
	})
	return claimed, err
}

func (r *TaskRepository) HeartbeatLease(ctx context.Context, id sharedkernel.TaskID, edgeID sharedkernel.EdgeID, lease time.Duration, now time.Time) (bool, error) {
	if lease <= 0 {
		return false, nil
	}
	res := r.db.WithContext(ctx).Model(&TaskRow{}).
		Where("id = ? AND status = ? AND edge_id = ?", string(id), string(sharedkernel.TaskRunning), string(edgeID)).
		Updates(map[string]any{
			"lease_until": now.Add(lease),
			"updated_at":  now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return true, nil
	}
	var n int64
	if err := r.db.WithContext(ctx).Model(&TaskRow{}).Where("id = ?", string(id)).Count(&n).Error; err != nil {
		return false, err
	}
	if n == 0 {
		return false, domain.ErrTaskNotFound
	}
	return false, nil
}

func (r *TaskRepository) RequeueExpiredLeases(ctx context.Context, now time.Time) (int, error) {
	res := r.db.WithContext(ctx).Model(&TaskRow{}).
		Where("status = ? AND lease_until != ? AND lease_until < ?", string(sharedkernel.TaskRunning), time.Time{}, now).
		Updates(map[string]any{
			"status":      string(sharedkernel.TaskQueued),
			"lease_until": time.Time{},
			"updated_at":  now,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

func (r *TaskRepository) ListByChat(ctx context.Context, chatID sharedkernel.ChatID, limit int) ([]*domain.Task, error) {
	addr, err := sharedkernel.ParseChatID(string(chatID))
	if err != nil {
		return nil, err
	}
	q := r.db.WithContext(ctx).Table("tasks").
		Joins("JOIN sessions ON tasks.session_id = sessions.id").
		Where("sessions.channel_id = ? AND sessions.chat_external_id = ?", addr.ChannelID, addr.ExternalChatID).
		Order("tasks.created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []TaskRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rowsToTasks(rows)
}

func (r *TaskRepository) ListByStatus(ctx context.Context, st sharedkernel.TaskStatus, limit int) ([]*domain.Task, error) {
	q := r.db.WithContext(ctx).Where("status = ?", string(st)).Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []TaskRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rowsToTasks(rows)
}

func (r *TaskRepository) ListByInstance(ctx context.Context, edgeID sharedkernel.EdgeID, q domain.ListByInstanceQuery) ([]*domain.Task, error) {
	if edgeID == "" {
		return nil, nil
	}
	tx := r.db.WithContext(ctx).
		Where("edge_id = ? AND edge_id != ?", string(edgeID), "").
		Order("created_at ASC")
	if q.Status != "" {
		tx = tx.Where("status = ?", string(q.Status))
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}
	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	var rows []TaskRow
	if err := tx.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rowsToTasks(rows)
}

func (r *TaskRepository) List(ctx context.Context, q domain.AdminListQuery) ([]*domain.Task, error) {
	joinChat := q.ChatID != ""
	col := func(name string) string {
		if joinChat {
			return "tasks." + name
		}
		return name
	}

	var tx *gorm.DB
	if joinChat {
		addr, err := sharedkernel.ParseChatID(string(q.ChatID))
		if err != nil {
			return nil, err
		}
		tx = r.db.WithContext(ctx).Table("tasks").
			Joins("JOIN sessions ON tasks.session_id = sessions.id").
			Where("sessions.channel_id = ? AND sessions.chat_external_id = ?", addr.ChannelID, addr.ExternalChatID)
	} else {
		tx = r.db.WithContext(ctx).Model(&TaskRow{})
	}

	if q.Status != "" {
		tx = tx.Where(col("status")+" = ?", string(q.Status))
	}
	if q.EdgeID != "" {
		tx = tx.Where(col("edge_id")+" = ?", string(q.EdgeID))
	}
	if q.SessionID != "" {
		tx = tx.Where(col("session_id")+" = ?", string(q.SessionID))
	}
	if q.CaseID != 0 {
		tx = tx.Where(col("case_id")+" = ?", q.CaseID)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where(
			col("id")+" LIKE ? OR "+col("case_id")+" LIKE ? OR "+col("session_id")+" LIKE ?",
			like, like, like,
		)
	}
	if q.CreatedFrom != nil {
		tx = tx.Where(col("created_at")+" >= ?", *q.CreatedFrom)
	}
	if q.CreatedTo != nil {
		tx = tx.Where(col("created_at")+" <= ?", *q.CreatedTo)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}
	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}

	var rows []TaskRow
	if err := tx.Order(col("created_at") + " DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rowsToTasks(rows)
}

func toRow(t *domain.Task) (*TaskRow, error) {
	outputs := t.Outputs
	if outputs == nil {
		outputs = []domain.OutputRef{}
	}
	b, err := json.Marshal(outputs)
	if err != nil {
		return nil, fmt.Errorf("runtime: encode outputs: %w", err)
	}
	jobRefJSON := ""
	if t.JobRef.Key != "" {
		raw, err := json.Marshal(t.JobRef)
		if err != nil {
			return nil, fmt.Errorf("runtime: encode job_ref: %w", err)
		}
		jobRefJSON = string(raw)
	}
	return &TaskRow{
		ID:            string(t.ID),
		SessionID:     string(t.SessionID),
		CaseID:        uint64(t.CaseID),
		Status:        string(t.Status),
		EdgeID:        string(t.EdgeID),
		DispatchTopic: t.DispatchTopic,
		Attempts:      t.Attempts,
		RequeueAt:     t.RequeueAt,
		PromptID:      t.PromptID,
		InputPrefix:   t.InputPrefix,
		JobRefJSON:    jobRefJSON,
		LeaseUntil:    t.LeaseUntil,
		OutputsJSON:   string(b),
		ErrorCode:     t.ErrorCode,
		ErrorMessage:  t.ErrorMessage,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}, nil
}

func fromRow(row TaskRow) (*domain.Task, error) {
	var outputs []domain.OutputRef
	if row.OutputsJSON != "" {
		if err := json.Unmarshal([]byte(row.OutputsJSON), &outputs); err != nil {
			return nil, fmt.Errorf("runtime: decode outputs: %w", err)
		}
	}
	var jobRef sharedkernel.BlobRef
	if row.JobRefJSON != "" {
		if err := json.Unmarshal([]byte(row.JobRefJSON), &jobRef); err != nil {
			return nil, fmt.Errorf("runtime: decode job_ref: %w", err)
		}
	}
	return &domain.Task{
		ID:            sharedkernel.TaskID(row.ID),
		SessionID:     sharedkernel.SessionID(row.SessionID),
		CaseID:        sharedkernel.CaseID(row.CaseID),
		Status:        sharedkernel.TaskStatus(row.Status),
		EdgeID:        sharedkernel.EdgeID(row.EdgeID),
		DispatchTopic: row.DispatchTopic,
		Attempts:      row.Attempts,
		RequeueAt:     row.RequeueAt,
		PromptID:      row.PromptID,
		InputPrefix:   row.InputPrefix,
		JobRef:        jobRef,
		LeaseUntil:    row.LeaseUntil,
		Outputs:       outputs,
		ErrorCode:     row.ErrorCode,
		ErrorMessage:  row.ErrorMessage,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func rowsToTasks(rows []TaskRow) ([]*domain.Task, error) {
	out := make([]*domain.Task, 0, len(rows))
	for _, row := range rows {
		t, err := fromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}
