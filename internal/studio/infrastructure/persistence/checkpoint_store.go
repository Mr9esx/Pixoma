package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// CheckpointStore keeps Eino's opaque resume state in the same database as the
// Studio run. A new process can therefore resume an approved tool call without
// replaying the preceding model response and tools.
type CheckpointStore struct{ db *gorm.DB }

func (r *GormRepository) Checkpoints() *CheckpointStore {
	return &CheckpointStore{db: r.db}
}

func (s *CheckpointStore) Get(ctx context.Context, runID string) ([]byte, bool, error) {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" {
		return nil, false, fmt.Errorf("%w: checkpoint store and run id are required", domain.ErrInvalid)
	}
	var row CheckpointRow
	err := s.db.WithContext(ctx).Where("run_id = ?", runID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return append([]byte(nil), row.Data...), true, nil
}

func (s *CheckpointStore) Set(ctx context.Context, runID string, data []byte) error {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" || len(data) == 0 {
		return fmt.Errorf("%w: checkpoint store, run id and data are required", domain.ErrInvalid)
	}
	row := CheckpointRow{RunID: runID, Data: append([]byte(nil), data...), UpdatedAt: time.Now().UTC()}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at"}),
	}).Create(&row).Error
}

func (s *CheckpointStore) Delete(ctx context.Context, runID string) error {
	if s == nil || s.db == nil || strings.TrimSpace(runID) == "" {
		return fmt.Errorf("%w: checkpoint store and run id are required", domain.ErrInvalid)
	}
	return s.db.WithContext(ctx).Where("run_id = ?", runID).Delete(&CheckpointRow{}).Error
}

var _ adk.CheckPointStore = (*CheckpointStore)(nil)
