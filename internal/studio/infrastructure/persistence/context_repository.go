package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/modelprovider"
	"github.com/google/uuid"
)

type ContextRequestRow struct {
	AttemptID           string                      `gorm:"primaryKey;size:128" json:"attempt_id"`
	SessionID           string                      `gorm:"size:64;not null;index:idx_studio_context_session_time,priority:2" json:"session_id"`
	AccountID           string                      `gorm:"size:64;not null;index:idx_studio_context_session_time,priority:1" json:"-"`
	RunID               string                      `gorm:"size:64;not null;index" json:"run_id"`
	TurnID              string                      `gorm:"size:64;index" json:"turn_id"`
	TurnNumber          int64                       `json:"turn_number"`
	StepNumber          int64                       `json:"step_number"`
	Purpose             string                      `gorm:"size:32" json:"purpose"`
	Protocol            string                      `gorm:"size:64" json:"protocol"`
	Model               string                      `gorm:"size:128" json:"model"`
	Preview             string                      `gorm:"size:512" json:"preview"`
	Status              string                      `gorm:"size:32" json:"status"`
	StartedAt           time.Time                   `gorm:"index:idx_studio_context_session_time,priority:3" json:"started_at"`
	FirstTokenAt        *time.Time                  `json:"first_token_at,omitempty"`
	EndedAt             *time.Time                  `json:"ended_at,omitempty"`
	ContextWindowTokens int                         `json:"context_window_tokens"`
	MaxInputTokens      int                         `json:"max_input_tokens"`
	MaxOutputTokens     int                         `json:"max_output_tokens"`
	InputTokens         *int                        `json:"input_tokens"`
	OutputTokens        *int                        `json:"output_tokens"`
	CacheReadTokens     *int                        `json:"cache_read_tokens"`
	CacheWriteTokens    *int                        `json:"cache_write_tokens"`
	ReasoningTokens     *int                        `json:"reasoning_tokens"`
	PartsJSON           []byte                      `gorm:"type:blob" json:"-"`
	RequestEventID      string                      `gorm:"size:64" json:"-"`
	Parts               []modelprovider.ContextPart `gorm:"-" json:"parts"`
	ProjectedTokens     *int                        `gorm:"-" json:"projected_tokens,omitempty"`
}

func (ContextRequestRow) TableName() string { return "studio_context_requests" }

type ContextEventRow struct {
	ID                    string    `gorm:"primaryKey;size:64" json:"id"`
	SessionID             string    `gorm:"size:64;not null;index:idx_studio_context_events_session_time,priority:2" json:"-"`
	AccountID             string    `gorm:"size:64;not null;index:idx_studio_context_events_session_time,priority:1" json:"-"`
	RunID                 string    `gorm:"size:64" json:"run_id"`
	TurnNumber            int64     `json:"turn_number"`
	StepNumber            int64     `json:"step_number"`
	Kind                  string    `gorm:"size:48;not null" json:"kind"`
	Source                string    `gorm:"size:128" json:"source"`
	Detail                string    `gorm:"size:512" json:"detail"`
	DeltaTokensEstimated  *int      `json:"delta_tokens_estimated"`
	BeforeTokensEstimated *int      `json:"before_tokens_estimated"`
	AfterTokensEstimated  *int      `json:"after_tokens_estimated"`
	CreatedAt             time.Time `gorm:"index:idx_studio_context_events_session_time,priority:3" json:"created_at"`
}

func (ContextEventRow) TableName() string { return "studio_context_events" }

type contextEventPayload struct {
	AttemptID           string                      `json:"attempt_id"`
	TurnID              string                      `json:"turn_id"`
	Purpose             string                      `json:"purpose"`
	Protocol            string                      `json:"protocol"`
	Model               string                      `json:"model"`
	Preview             string                      `json:"context_preview"`
	At                  time.Time                   `json:"at"`
	ContextWindowTokens int                         `json:"context_window_tokens"`
	MaxInputTokens      int                         `json:"max_input_tokens"`
	MaxOutputTokens     int                         `json:"max_output_tokens"`
	InputTokens         *int                        `json:"input_tokens"`
	OutputTokens        *int                        `json:"output_tokens"`
	CacheReadTokens     *int                        `json:"cache_read_tokens"`
	CacheWriteTokens    *int                        `json:"cache_write_tokens"`
	ReasoningTokens     *int                        `json:"reasoning_tokens"`
	ContextParts        []modelprovider.ContextPart `json:"context_parts"`
	RunContext          string                      `json:"run_context"`
	RequestBody         json.RawMessage             `json:"request_body"`
}

func projectContextRequestEvent(tx *gorm.DB, event *domain.Event) error {
	switch event.Type {
	case "CONTEXT_INJECTED", "CONTEXT_COMPACTED", "CONTEXT_PRUNED":
		var payload struct {
			Source                string `json:"source"`
			Detail                string `json:"detail"`
			DeltaTokensEstimated  *int   `json:"delta_tokens_estimated"`
			BeforeTokensEstimated *int   `json:"before_tokens_estimated"`
			AfterTokensEstimated  *int   `json:"after_tokens_estimated"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		turnNumber, stepNumber, err := contextPosition(tx, event)
		if err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ContextEventRow{
			ID: event.ID, SessionID: event.SessionID, AccountID: event.AccountID, RunID: event.RunID,
			TurnNumber: turnNumber, StepNumber: stepNumber,
			Kind: event.Type, Source: payload.Source, Detail: payload.Detail,
			DeltaTokensEstimated: payload.DeltaTokensEstimated, BeforeTokensEstimated: payload.BeforeTokensEstimated,
			AfterTokensEstimated: payload.AfterTokensEstimated, CreatedAt: event.CreatedAt,
		}).Error
	case "MODEL_REQUEST_STARTED", "MODEL_FIRST_TOKEN", "MODEL_REQUEST_FINISHED", "MODEL_REQUEST_FAILED":
	default:
		return nil
	}
	var payload contextEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("studio: decode context event: %w", err)
	}
	if payload.AttemptID == "" {
		return fmt.Errorf("studio: context event has no attempt ID")
	}
	if payload.At.IsZero() {
		payload.At = event.CreatedAt
	}
	key := ContextRequestRow{AttemptID: payload.AttemptID, SessionID: event.SessionID, AccountID: event.AccountID, RunID: event.RunID, StartedAt: payload.At}
	switch event.Type {
	case "MODEL_REQUEST_STARTED":
		parts := payload.ContextParts
		if len(parts) == 0 && len(payload.RequestBody) > 0 {
			var err error
			parts, err = modelprovider.AnalyzeRequest(payload.RequestBody, payload.RunContext)
			if err != nil {
				return err
			}
		}
		key.Purpose, key.Protocol, key.Model, key.Preview, key.Status, key.TurnID = payload.Purpose, payload.Protocol, payload.Model, payload.Preview, "running", payload.TurnID
		if key.Preview == "" && len(payload.RequestBody) > 0 {
			fullParts, err := modelprovider.AnalyzeRequest(payload.RequestBody, payload.RunContext)
			if err != nil {
				return err
			}
			for index := len(fullParts) - 1; index >= 0; index-- {
				if fullParts[index].Category == "user_message" && fullParts[index].Content != "" {
					preview := []rune(fullParts[index].Content)
					if len(preview) > 100 {
						preview = preview[:100]
					}
					key.Preview = string(preview)
					break
				}
			}
		}
		for index := range parts {
			parts[index].Content = ""
		}
		encoded, err := json.Marshal(parts)
		if err != nil {
			return err
		}
		if key.TurnID == "" {
			var run RunRow
			if err := tx.Select("trigger_message_id").Where("id = ? AND account_id = ?", event.RunID, event.AccountID).First(&run).Error; err != nil {
				return err
			}
			key.TurnID = run.TriggerMessageID
		}
		turnNumber, stepNumber, err := contextPosition(tx, event)
		if err != nil {
			return err
		}
		key.TurnNumber = turnNumber
		if key.Purpose == "agent" {
			key.StepNumber = stepNumber
		}
		key.ContextWindowTokens, key.MaxInputTokens, key.MaxOutputTokens = payload.ContextWindowTokens, payload.MaxInputTokens, payload.MaxOutputTokens
		key.PartsJSON, key.RequestEventID = encoded, event.ID
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&key).Error
	case "MODEL_FIRST_TOKEN":
		return tx.Model(&ContextRequestRow{}).Where("attempt_id = ? AND account_id = ?", key.AttemptID, key.AccountID).Update("first_token_at", payload.At).Error
	default:
		status := "finished"
		if event.Type == "MODEL_REQUEST_FAILED" {
			status = "failed"
		}
		updates := map[string]any{"ended_at": payload.At, "status": status,
			"input_tokens": payload.InputTokens, "output_tokens": payload.OutputTokens,
			"cache_read_tokens": payload.CacheReadTokens, "cache_write_tokens": payload.CacheWriteTokens, "reasoning_tokens": payload.ReasoningTokens}
		result := tx.Model(&ContextRequestRow{}).Where("attempt_id = ? AND account_id = ?", key.AttemptID, key.AccountID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
		key.Status, key.EndedAt = status, &payload.At
		key.InputTokens, key.OutputTokens = payload.InputTokens, payload.OutputTokens
		key.CacheReadTokens, key.CacheWriteTokens, key.ReasoningTokens = payload.CacheReadTokens, payload.CacheWriteTokens, payload.ReasoningTokens
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&key).Error
	}
}

func contextPosition(tx *gorm.DB, event *domain.Event) (int64, int64, error) {
	var run RunRow
	if err := tx.Select("created_at, trigger_message_id").Where("id = ? AND account_id = ?", event.RunID, event.AccountID).First(&run).Error; err != nil {
		return 0, 0, err
	}
	var turnNumber, stepNumber int64
	if err := tx.Model(&RunRow{}).Where("account_id = ? AND session_id = ? AND created_at <= ?", event.AccountID, event.SessionID, run.CreatedAt).
		Distinct("trigger_message_id").Count(&turnNumber).Error; err != nil {
		return 0, 0, err
	}
	if err := tx.Model(&ContextRequestRow{}).Where("account_id = ? AND session_id = ? AND turn_id = ? AND purpose = ? AND started_at < ?", event.AccountID, event.SessionID, run.TriggerMessageID, "agent", event.CreatedAt).
		Count(&stepNumber).Error; err != nil {
		return 0, 0, err
	}
	return turnNumber, stepNumber + 1, nil
}

func appendSessionContextChange(tx *gorm.DB, session *domain.Session, kind, previous, current string) error {
	return tx.Create(&ContextEventRow{
		ID: uuid.NewString(), SessionID: session.ID, AccountID: session.AccountID,
		Kind: kind, Source: "session_config", Detail: previous + " → " + current, CreatedAt: session.UpdatedAt,
	}).Error
}

func (r *GormRepository) BackfillSessionContext(ctx context.Context, accountID, sessionID string) error {
	var requests, started, contextEvents, sourceEvents int64
	if err := r.db.WithContext(ctx).Model(&ContextRequestRow{}).Where("account_id = ? AND session_id = ?", accountID, sessionID).Count(&requests).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&EventRow{}).Where("account_id = ? AND session_id = ? AND type = ?", accountID, sessionID, "MODEL_REQUEST_STARTED").Count(&started).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&ContextEventRow{}).Where("account_id = ? AND session_id = ? AND run_id <> ''", accountID, sessionID).Count(&contextEvents).Error; err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Model(&EventRow{}).Where("account_id = ? AND session_id = ? AND type IN ?", accountID, sessionID,
		[]string{"CONTEXT_INJECTED", "CONTEXT_COMPACTED", "CONTEXT_PRUNED"}).Count(&sourceEvents).Error; err != nil {
		return err
	}
	if requests >= started && contextEvents >= sourceEvents {
		return nil
	}
	var rows []EventRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ? AND type IN ?", accountID, sessionID,
		[]string{"MODEL_REQUEST_STARTED", "MODEL_FIRST_TOKEN", "MODEL_REQUEST_FINISHED", "MODEL_REQUEST_FAILED", "CONTEXT_INJECTED", "CONTEXT_COMPACTED", "CONTEXT_PRUNED"}).
		Order("created_at ASC, run_id ASC, sequence ASC").Find(&rows).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := projectContextRequestEvent(tx, eventFromRow(row)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepository) ListContextRequests(ctx context.Context, accountID, sessionID string, limit, offset int) ([]ContextRequestRow, error) {
	rows := make([]ContextRequestRow, 0)
	err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("started_at DESC, attempt_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for index := range rows {
		if err := json.Unmarshal(rows[index].PartsJSON, &rows[index].Parts); err != nil && len(rows[index].PartsJSON) > 0 {
			return nil, err
		}
	}
	return rows, err
}

func (r *GormRepository) GetContextRequest(ctx context.Context, accountID, sessionID, attemptID string) (*ContextRequestRow, error) {
	var row ContextRequestRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ? AND attempt_id = ?", accountID, sessionID, attemptID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var event EventRow
	if err := r.db.WithContext(ctx).Where("id = ? AND account_id = ? AND session_id = ?", row.RequestEventID, accountID, sessionID).First(&event).Error; err != nil {
		return nil, err
	}
	var payload contextEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil, err
	}
	var err error
	row.Parts, err = modelprovider.AnalyzeRequest(payload.RequestBody, payload.RunContext)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

type ContextOverview struct {
	Turns       int64              `json:"turns"`
	Steps       int64              `json:"steps"`
	ToolCalls   int64              `json:"tool_calls"`
	Injections  int64              `json:"injections"`
	Compactions int64              `json:"compactions"`
	Prunes      int64              `json:"prunes"`
	Tokens      ContextTokenTotals `json:"tokens"`
	Timing      ContextTiming      `json:"timing"`
	Current     *ContextRequestRow `json:"current"`
}

func (r *GormRepository) projectCurrentContext(ctx context.Context, accountID, sessionID string, row *ContextRequestRow, includeContent bool) error {
	if row == nil || row.RequestEventID == "" {
		return nil
	}
	var start EventRow
	if err := r.db.WithContext(ctx).Select("id, run_id, sequence, created_at").Where("id = ? AND account_id = ? AND session_id = ?", row.RequestEventID, accountID, sessionID).First(&start).Error; err != nil {
		return err
	}
	var tail []EventRow
	if err := r.db.WithContext(ctx).Model(&EventRow{}).Select("id, type, payload, sequence").
		Where("account_id = ? AND session_id = ? AND run_id = ? AND sequence > ? AND type IN ?", accountID, sessionID, start.RunID, start.Sequence,
			[]string{"TOOL_CALL_START", "TOOL_CALL_RESULT", "TEXT_MESSAGE_END", "CONTEXT_INJECTED", "CONTEXT_PRUNED", "CONTEXT_COMPACTED"}).
		Order("sequence ASC").Find(&tail).Error; err != nil {
		return err
	}
	projected := 0
	for _, part := range row.Parts {
		projected += part.EstimatedTokens
	}
	if row.InputTokens != nil {
		projected = *row.InputTokens
	}
	toolNames := map[string]string{}
	for _, event := range tail {
		var payload struct {
			ToolCallID           string `json:"tool_call_id"`
			ToolName             string `json:"tool_name"`
			Content              string `json:"content"`
			Source               string `json:"source"`
			Detail               string `json:"detail"`
			DeltaTokensEstimated *int   `json:"delta_tokens_estimated"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		if event.Type == "TOOL_CALL_START" {
			toolNames[payload.ToolCallID] = payload.ToolName
			continue
		}
		if event.Type == "CONTEXT_COMPACTED" || event.Type == "CONTEXT_PRUNED" {
			if payload.DeltaTokensEstimated != nil {
				projected += *payload.DeltaTokensEstimated
			}
			continue
		}
		category, source, label := "", "", ""
		switch event.Type {
		case "TEXT_MESSAGE_END":
			category, source, label = "assistant_message", "assistant", "助手消息"
		case "TOOL_CALL_RESULT":
			if toolNames[payload.ToolCallID] == "load_skill" {
				continue
			}
			category, source, label = "tool_result", toolNames[payload.ToolCallID], toolNames[payload.ToolCallID]
		case "CONTEXT_INJECTED":
			if payload.Source != "skill" {
				continue
			}
			category, source, label = "skill_injection", payload.Source, payload.Detail
		}
		if payload.Content == "" {
			continue
		}
		tokens := modelprovider.EstimateContextText(payload.Content)
		projected += tokens
		content := payload.Content
		if !includeContent {
			content = ""
		}
		row.Parts = append(row.Parts, modelprovider.ContextPart{
			ID: "tail/" + event.ID, Category: category, Source: source, Label: label,
			Content: content, EstimatedTokens: tokens,
		})
	}
	var messages []MessageRow
	if err := r.db.WithContext(ctx).Model(&MessageRow{}).Select("id, content_json").
		Where("account_id = ? AND session_id = ? AND role = ? AND created_at > ?", accountID, sessionID, string(domain.MessageRoleUser), start.CreatedAt).
		Order("created_at ASC, id ASC").Find(&messages).Error; err != nil {
		return err
	}
	for _, message := range messages {
		var content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(message.ContentJSON, &content); err != nil {
			return err
		}
		for index, part := range content {
			if part.Type != "text" || part.Text == "" {
				continue
			}
			tokens := modelprovider.EstimateContextText(part.Text)
			projected += tokens
			text := part.Text
			if !includeContent {
				text = ""
			}
			row.Parts = append(row.Parts, modelprovider.ContextPart{
				ID: fmt.Sprintf("pending/%s/%d", message.ID, index), Category: "user_message", Source: "pending_user", Label: "用户消息",
				Content: text, EstimatedTokens: tokens,
			})
		}
	}
	if projected < 0 {
		projected = 0
	}
	row.ProjectedTokens = &projected
	var session SessionRow
	if err := r.db.WithContext(ctx).Select("model_config_id").Where("id = ? AND account_id = ?", sessionID, accountID).First(&session).Error; err != nil {
		return err
	}
	if session.ModelConfigID != "" {
		var config ModelConfigRow
		if err := r.db.WithContext(ctx).Select("model, context_window_tokens, max_input_tokens, max_output_tokens").
			Where("id = ? AND account_id = ?", session.ModelConfigID, accountID).First(&config).Error; err != nil {
			return err
		}
		row.Model = config.Model
		row.ContextWindowTokens, row.MaxInputTokens, row.MaxOutputTokens = config.ContextWindowTokens, config.MaxInputTokens, config.MaxOutputTokens
	}
	return nil
}

func (r *GormRepository) GetCurrentContextRequest(ctx context.Context, accountID, sessionID string) (*ContextRequestRow, error) {
	var row ContextRequestRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ? AND purpose = ?", accountID, sessionID, "agent").
		Order("started_at DESC, attempt_id DESC").First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	detail, err := r.GetContextRequest(ctx, accountID, sessionID, row.AttemptID)
	if err != nil {
		return nil, err
	}
	if err := r.projectCurrentContext(ctx, accountID, sessionID, detail, true); err != nil {
		return nil, err
	}
	return detail, nil
}

type ContextTokenTotals struct {
	Input           int64 `json:"input"`
	Output          int64 `json:"output"`
	CacheRead       int64 `json:"cache_read"`
	CacheWrite      int64 `json:"cache_write"`
	Uncached        int64 `json:"uncached"`
	Reasoning       int64 `json:"reasoning"`
	Missing         int64 `json:"missing_requests"`
	CacheKnown      int64 `json:"cache_known_requests"`
	CacheKnownInput int64 `json:"cache_known_input"`
}

type ContextTiming struct {
	ActiveMS     int64 `json:"active_ms"`
	ModelWaitMS  int64 `json:"model_wait_ms"`
	GenerationMS int64 `json:"generation_ms"`
	ModelOtherMS int64 `json:"model_other_ms"`
	ToolsMS      int64 `json:"tools_ms"`
	OtherMS      int64 `json:"other_ms"`
}

type contextInterval struct{ start, end time.Time }

func intervalMillis(intervals []contextInterval) int64 {
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].start.Before(intervals[j].start) })
	start, end := intervals[0].start, intervals[0].end
	var total int64
	for _, current := range intervals[1:] {
		if current.start.After(end) {
			if end.After(start) {
				total += end.Sub(start).Milliseconds()
			}
			start, end = current.start, current.end
		} else if current.end.After(end) {
			end = current.end
		}
	}
	if end.After(start) {
		total += end.Sub(start).Milliseconds()
	}
	return total
}

func (r *GormRepository) GetContextOverview(ctx context.Context, accountID, sessionID string) (*ContextOverview, error) {
	view := &ContextOverview{}
	if err := r.db.WithContext(ctx).Model(&RunRow{}).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Distinct("trigger_message_id").Count(&view.Turns).Error; err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Model(&ContextRequestRow{}).Where("account_id = ? AND session_id = ?", accountID, sessionID)
	requestRows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer requestRows.Close()
	for requestRows.Next() {
		var row ContextRequestRow
		if err := r.db.ScanRows(requestRows, &row); err != nil {
			return nil, err
		}
		if row.Purpose == "agent" {
			view.Steps++
		}
		if row.Purpose == "agent" && (view.Current == nil || row.StartedAt.After(view.Current.StartedAt)) {
			current := row
			if len(row.PartsJSON) > 0 {
				if err := json.Unmarshal(row.PartsJSON, &current.Parts); err != nil {
					return nil, err
				}
			}
			view.Current = &current
		}
		if row.InputTokens == nil || row.OutputTokens == nil {
			view.Tokens.Missing++
		} else {
			view.Tokens.Input += int64(*row.InputTokens)
			view.Tokens.Output += int64(*row.OutputTokens)
		}
		if row.CacheReadTokens != nil {
			view.Tokens.CacheRead += int64(*row.CacheReadTokens)
		}
		if row.CacheWriteTokens != nil {
			view.Tokens.CacheWrite += int64(*row.CacheWriteTokens)
		}
		if row.ReasoningTokens != nil {
			view.Tokens.Reasoning += int64(*row.ReasoningTokens)
		}
		if row.InputTokens != nil && row.CacheReadTokens != nil {
			write := 0
			if row.CacheWriteTokens != nil {
				write = *row.CacheWriteTokens
			}
			view.Tokens.Uncached += int64(*row.InputTokens - *row.CacheReadTokens - write)
			view.Tokens.CacheKnown++
			view.Tokens.CacheKnownInput += int64(*row.InputTokens)
		}
		if row.EndedAt != nil {
			if row.FirstTokenAt != nil && !row.FirstTokenAt.Before(row.StartedAt) && !row.FirstTokenAt.After(*row.EndedAt) {
				view.Timing.ModelWaitMS += row.FirstTokenAt.Sub(row.StartedAt).Milliseconds()
				view.Timing.GenerationMS += row.EndedAt.Sub(*row.FirstTokenAt).Milliseconds()
			} else {
				view.Timing.ModelOtherMS += row.EndedAt.Sub(row.StartedAt).Milliseconds()
			}
		}
	}
	if err := requestRows.Err(); err != nil {
		return nil, err
	}
	if err := requestRows.Close(); err != nil {
		return nil, err
	}
	var runRows []RunRow
	if err := r.db.WithContext(ctx).Select("id, started_at, completed_at").Where("account_id = ? AND session_id = ?", accountID, sessionID).Find(&runRows).Error; err != nil {
		return nil, err
	}
	active := make([]contextInterval, 0, len(runRows))
	for _, run := range runRows {
		if !run.StartedAt.IsZero() {
			end := run.CompletedAt
			if end.IsZero() {
				end = time.Now().UTC()
			}
			active = append(active, contextInterval{run.StartedAt, end})
		}
	}
	view.Timing.ActiveMS = intervalMillis(active)
	var events []EventRow
	if err := r.db.WithContext(ctx).Model(&EventRow{}).
		Select("id, run_id, type, COALESCE(summary_payload, payload) AS summary_payload, created_at").
		Where("account_id = ? AND session_id = ? AND type IN ?", accountID, sessionID,
			[]string{"TOOL_CALL_START", "TOOL_CALL_END", "CONTEXT_INJECTED", "CONTEXT_COMPACTED", "CONTEXT_PRUNED", "APPROVAL_REQUIRED", "APPROVAL_RESOLVED", "CLARIFICATION_REQUIRED", "CLARIFICATION_ANSWERED", "CLARIFICATION_SKIPPED"}).
		Order("created_at ASC, id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	toolStarts := map[string]time.Time{}
	pauseStarts := map[string]time.Time{}
	toolIntervals := make([]contextInterval, 0)
	pauseIntervals := make([]contextInterval, 0)
	for _, event := range events {
		switch event.Type {
		case "APPROVAL_REQUIRED", "CLARIFICATION_REQUIRED":
			pauseStarts[event.RunID] = event.CreatedAt
		case "APPROVAL_RESOLVED", "CLARIFICATION_ANSWERED", "CLARIFICATION_SKIPPED":
			if start, ok := pauseStarts[event.RunID]; ok {
				pauseIntervals = append(pauseIntervals, contextInterval{start, event.CreatedAt})
				delete(pauseStarts, event.RunID)
			}
		case "CONTEXT_INJECTED":
			view.Injections++
		case "CONTEXT_COMPACTED":
			view.Compactions++
		case "CONTEXT_PRUNED":
			view.Prunes++
		case "TOOL_CALL_START", "TOOL_CALL_END":
			var payload struct {
				ToolCallID string `json:"tool_call_id"`
			}
			if err := json.Unmarshal(event.SummaryPayload, &payload); err != nil {
				return nil, err
			}
			key := event.RunID + ":" + payload.ToolCallID
			if event.Type == "TOOL_CALL_START" {
				view.ToolCalls++
				toolStarts[key] = event.CreatedAt
			} else if start, ok := toolStarts[key]; ok {
				toolIntervals = append(toolIntervals, contextInterval{start, event.CreatedAt})
				delete(toolStarts, key)
			}
		}
	}
	for _, start := range pauseStarts {
		pauseIntervals = append(pauseIntervals, contextInterval{start, time.Now().UTC()})
	}
	view.Timing.ActiveMS -= intervalMillis(pauseIntervals)
	if view.Timing.ActiveMS < 0 {
		view.Timing.ActiveMS = 0
	}
	view.Timing.ToolsMS = intervalMillis(toolIntervals)
	view.Timing.OtherMS = view.Timing.ActiveMS - view.Timing.ModelWaitMS - view.Timing.GenerationMS - view.Timing.ModelOtherMS - view.Timing.ToolsMS
	if view.Timing.OtherMS < 0 {
		view.Timing.OtherMS = 0
	}
	if err := r.projectCurrentContext(ctx, accountID, sessionID, view.Current, false); err != nil {
		return nil, err
	}
	return view, nil
}

func (r *GormRepository) ListContextEvents(ctx context.Context, accountID, sessionID, kind string, limit, offset int) ([]ContextEventRow, error) {
	rows := make([]ContextEventRow, 0)
	query := r.db.WithContext(ctx).Model(&ContextEventRow{}).Where("account_id = ? AND session_id = ?", accountID, sessionID)
	if kind != "" {
		query = query.Where("kind = ?", kind)
	}
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
