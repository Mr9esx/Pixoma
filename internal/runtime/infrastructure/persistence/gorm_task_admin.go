package persistence

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"

	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type TaskAdminProjection struct {
	db *gorm.DB
}

func NewTaskAdminProjection(db *gorm.DB) *TaskAdminProjection {
	return &TaskAdminProjection{db: db}
}

type taskAdminRow struct {
	TaskRow `gorm:"embedded"`

	AdminChannelID      sql.NullString
	AdminChannelName    sql.NullString
	AdminUserID         sql.NullString
	AdminUserInternalID sql.NullString
	AdminUserChannelID  sql.NullString
	AdminUserExternalID sql.NullString
	AdminUserUsername   sql.NullString
	AdminUserFirstName  sql.NullString
	AdminUserLastName   sql.NullString
}

func (r *TaskAdminProjection) baseQuery(ctx context.Context, q runtimedomain.AdminListQuery) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("tasks").
		Select(`tasks.*,
			sessions.channel_id AS admin_channel_id,
			channels.name AS admin_channel_name,
			sessions.user_id AS admin_user_id,
			channel_users.id AS admin_user_internal_id,
			identities.channel_id AS admin_user_channel_id,
			identities.external_user_id AS admin_user_external_id,
			channel_users.username AS admin_user_username,
			channel_users.first_name AS admin_user_first_name,
			channel_users.last_name AS admin_user_last_name`).
		Joins("LEFT JOIN sessions ON tasks.session_id = sessions.id").
		Joins("LEFT JOIN channel_users ON channel_users.id = sessions.user_id").
		Joins("LEFT JOIN channel_user_external_identities AS identities ON identities.user_id = sessions.user_id AND identities.channel_id = sessions.channel_id").
		Joins("LEFT JOIN channels ON channels.id = sessions.channel_id")

	if q.ChatID != "" {
		addr, err := sharedkernel.ParseChatID(string(q.ChatID))
		if err != nil {
			return query.Where("1 = 0")
		}
		query = query.Where("sessions.channel_id = ? AND sessions.chat_external_id = ?", addr.ChannelID, addr.ExternalChatID)
	}
	if q.ChannelID != "" {
		query = query.Where("sessions.channel_id = ?", q.ChannelID)
	}
	if q.Status != "" {
		query = query.Where("tasks.status = ?", string(q.Status))
	}
	if q.EdgeID != "" {
		query = query.Where("tasks.edge_id = ?", string(q.EdgeID))
	}
	if q.SessionID != "" {
		query = query.Where("tasks.session_id = ?", string(q.SessionID))
	}
	if q.CaseID != 0 {
		query = query.Where("tasks.case_id = ?", q.CaseID)
	}
	if q.DispatchTopic != "" {
		query = query.Where("tasks.dispatch_topic = ?", q.DispatchTopic)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		query = query.Where("tasks.id LIKE ? OR tasks.case_id LIKE ? OR tasks.session_id LIKE ?", like, like, like)
	}
	if q.CreatedFrom != nil {
		query = query.Where("tasks.created_at >= ?", *q.CreatedFrom)
	}
	if q.CreatedTo != nil {
		query = query.Where("tasks.created_at <= ?", *q.CreatedTo)
	}
	if q.Offset > 0 {
		query = query.Offset(q.Offset)
	}
	if q.Limit > 0 {
		query = query.Limit(q.Limit)
	}
	return query
}

func (r *TaskAdminProjection) list(ctx context.Context, q runtimedomain.AdminListQuery) ([]*runtimedomain.TaskAdminContext, error) {
	var rows []taskAdminRow
	if err := r.baseQuery(ctx, q).Order("tasks.created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*runtimedomain.TaskAdminContext, 0, len(rows))
	for _, row := range rows {
		tasks, err := rowsToTasks([]TaskRow{row.TaskRow})
		if err != nil {
			return nil, err
		}
		out = append(out, toAdminContext(tasks[0], row))
	}
	return out, nil
}

func (r *TaskAdminProjection) List(ctx context.Context, q runtimedomain.AdminListQuery) ([]*runtimedomain.TaskAdminContext, error) {
	return r.list(ctx, q)
}

func (r *TaskAdminProjection) Get(ctx context.Context, id sharedkernel.TaskID) (*runtimedomain.TaskAdminContext, error) {
	q := runtimedomain.AdminListQuery{Limit: 1}
	var row taskAdminRow
	if err := r.baseQuery(ctx, q).Where("tasks.id = ?", string(id)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, runtimedomain.ErrTaskNotFound
		}
		return nil, err
	}
	tasks, err := rowsToTasks([]TaskRow{row.TaskRow})
	if err != nil {
		return nil, err
	}
	return toAdminContext(tasks[0], row), nil
}

func toAdminContext(task *runtimedomain.Task, row taskAdminRow) *runtimedomain.TaskAdminContext {
	context := runtimedomain.NewTaskAdminContext(task)
	context.ChannelID = row.AdminChannelID.String
	context.ChannelName = row.AdminChannelName.String
	context.UserID = row.AdminUserID.String
	if row.AdminUserInternalID.Valid && row.AdminUserInternalID.String != "" {
		context.User = &runtimedomain.TaskAdminUser{
			ID:             row.AdminUserInternalID.String,
			ChannelID:      row.AdminUserChannelID.String,
			ExternalUserID: row.AdminUserExternalID.String,
			Username:       row.AdminUserUsername.String,
			FirstName:      row.AdminUserFirstName.String,
			LastName:       row.AdminUserLastName.String,
		}
	}
	return context
}
