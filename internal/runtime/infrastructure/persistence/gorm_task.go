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
	ID           string    `gorm:"primaryKey;size:64"`
	SessionID    string    `gorm:"column:session_id;size:36;index;not null"`
	CaseID       string    `gorm:"column:case_id;size:128;not null"`
	Status       string    `gorm:"size:32;not null;index"`
	InstanceID   string    `gorm:"column:instance_id;size:128;index"`
	PromptID     string    `gorm:"column:prompt_id;size:128"`
	InputPrefix  string    `gorm:"column:input_prefix;size:512;not null"`
	OutputsJSON  string    `gorm:"column:outputs_json;type:text;not null"`
	ErrorCode    string    `gorm:"column:error_code;size:128"`
	ErrorMessage string    `gorm:"column:error_message;type:text"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
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
		"session_id":    row.SessionID,
		"case_id":       row.CaseID,
		"status":        row.Status,
		"instance_id":   row.InstanceID,
		"prompt_id":     row.PromptID,
		"input_prefix":  row.InputPrefix,
		"outputs_json":  row.OutputsJSON,
		"error_code":    row.ErrorCode,
		"error_message": row.ErrorMessage,
		"updated_at":    row.UpdatedAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (r *TaskRepository) ClaimQueued(ctx context.Context, id sharedkernel.TaskID, instanceID sharedkernel.InstanceID, now time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&TaskRow{}).
		Where("id = ? AND status = ?", string(id), string(sharedkernel.TaskPending)).
		Updates(map[string]any{
			"status":      string(sharedkernel.TaskQueued),
			"instance_id": string(instanceID),
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

func (r *TaskRepository) ListByChat(ctx context.Context, chatID sharedkernel.ChatID, limit int) ([]*domain.Task, error) {
	q := r.db.WithContext(ctx).Table("tasks").
		Joins("JOIN sessions ON tasks.session_id = sessions.id").
		Where("sessions.chat_id = ?", int64(chatID)).
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

func (r *TaskRepository) ListByInstance(ctx context.Context, instanceID sharedkernel.InstanceID, q domain.ListByInstanceQuery) ([]*domain.Task, error) {
	if instanceID == "" {
		return nil, nil
	}
	tx := r.db.WithContext(ctx).
		Where("instance_id = ? AND instance_id != ?", string(instanceID), "").
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
	joinChat := q.ChatID != 0
	col := func(name string) string {
		if joinChat {
			return "tasks." + name
		}
		return name
	}

	var tx *gorm.DB
	if joinChat {
		tx = r.db.WithContext(ctx).Table("tasks").
			Joins("JOIN sessions ON tasks.session_id = sessions.id").
			Where("sessions.chat_id = ?", int64(q.ChatID))
	} else {
		tx = r.db.WithContext(ctx).Model(&TaskRow{})
	}

	if q.Status != "" {
		tx = tx.Where(col("status")+" = ?", string(q.Status))
	}
	if q.InstanceID != "" {
		tx = tx.Where(col("instance_id")+" = ?", string(q.InstanceID))
	}
	if q.SessionID != "" {
		tx = tx.Where(col("session_id")+" = ?", string(q.SessionID))
	}
	if q.CaseID != "" {
		tx = tx.Where(col("case_id")+" = ?", string(q.CaseID))
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
	return &TaskRow{
		ID:           string(t.ID),
		SessionID:    string(t.SessionID),
		CaseID:       string(t.CaseID),
		Status:       string(t.Status),
		InstanceID:   string(t.InstanceID),
		PromptID:     t.PromptID,
		InputPrefix:  t.InputPrefix,
		OutputsJSON:  string(b),
		ErrorCode:    t.ErrorCode,
		ErrorMessage: t.ErrorMessage,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}, nil
}

func fromRow(row TaskRow) (*domain.Task, error) {
	var outputs []domain.OutputRef
	if row.OutputsJSON != "" {
		if err := json.Unmarshal([]byte(row.OutputsJSON), &outputs); err != nil {
			return nil, fmt.Errorf("runtime: decode outputs: %w", err)
		}
	}
	return &domain.Task{
		ID:           sharedkernel.TaskID(row.ID),
		SessionID:    sharedkernel.SessionID(row.SessionID),
		CaseID:       sharedkernel.CaseID(row.CaseID),
		Status:       sharedkernel.TaskStatus(row.Status),
		InstanceID:   sharedkernel.InstanceID(row.InstanceID),
		PromptID:     row.PromptID,
		InputPrefix:  row.InputPrefix,
		Outputs:      outputs,
		ErrorCode:    row.ErrorCode,
		ErrorMessage: row.ErrorMessage,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
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
