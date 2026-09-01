package persistence

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type SessionAdminProjection struct {
	db *gorm.DB
}

func NewSessionAdminProjection(db *gorm.DB) *SessionAdminProjection {
	return &SessionAdminProjection{db: db}
}

type sessionAdminRow struct {
	SessionRow `gorm:"embedded"`

	AdminChannelName    sql.NullString
	AdminUserID         sql.NullString
	AdminUserInternalID sql.NullString
	AdminUserChannelID  sql.NullString
	AdminUserExternalID sql.NullString
	AdminUserUsername   sql.NullString
	AdminUserFirstName  sql.NullString
	AdminUserLastName   sql.NullString
}

func (r *SessionAdminProjection) baseQuery(ctx context.Context, q domain.ListQuery) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("sessions").
		Select(`sessions.*,
			channels.name AS admin_channel_name,
			channel_users.id AS admin_user_internal_id,
			identities.channel_id AS admin_user_channel_id,
			identities.external_user_id AS admin_user_external_id,
			channel_users.username AS admin_user_username,
			channel_users.first_name AS admin_user_first_name,
			channel_users.last_name AS admin_user_last_name`).
		Joins("LEFT JOIN channels ON channels.id = sessions.channel_id").
		Joins("LEFT JOIN channel_users ON channel_users.id = sessions.user_id").
		Joins("LEFT JOIN channel_user_external_identities AS identities ON identities.user_id = sessions.user_id AND identities.channel_id = sessions.channel_id")

	if q.UserID != "" {
		query = query.Where("sessions.user_id = ?", q.UserID)
	}
	if q.ChatID != nil {
		addr, err := sharedkernel.ParseChatID(string(*q.ChatID))
		if err != nil {
			return query.Where("1 = 0")
		}
		query = query.Where("sessions.channel_id = ? AND sessions.chat_external_id = ?", addr.ChannelID, addr.ExternalChatID)
	}
	if q.Status != "" {
		query = query.Where("sessions.status = ?", string(q.Status))
	}
	if q.CaseID != 0 {
		query = query.Where("sessions.case_id = ?", uint64(q.CaseID))
	}
	if q.ChannelID != "" {
		query = query.Where("sessions.channel_id = ?", q.ChannelID)
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		query = query.Where("sessions.id LIKE ? OR sessions.case_id LIKE ? OR sessions.user_id LIKE ?", like, like, like)
	}
	if q.CreatedFrom != nil {
		query = query.Where("sessions.created_at >= ?", *q.CreatedFrom)
	}
	if q.CreatedTo != nil {
		query = query.Where("sessions.created_at <= ?", *q.CreatedTo)
	}
	if q.Limit > 0 {
		query = query.Limit(q.Limit)
	}
	if q.Offset > 0 {
		query = query.Offset(q.Offset)
	}
	return query
}

func (r *SessionAdminProjection) toContext(row sessionAdminRow) (*domain.SessionAdminContext, error) {
	session, err := fromRow(row.SessionRow)
	if err != nil {
		return nil, err
	}
	context := &domain.SessionAdminContext{Session: session, ChannelName: row.AdminChannelName.String}
	if row.AdminUserInternalID.Valid && row.AdminUserInternalID.String != "" {
		context.User = &domain.SessionAdminUser{
			ID:             row.AdminUserInternalID.String,
			ChannelID:      row.AdminUserChannelID.String,
			ExternalUserID: row.AdminUserExternalID.String,
			Username:       row.AdminUserUsername.String,
			FirstName:      row.AdminUserFirstName.String,
			LastName:       row.AdminUserLastName.String,
		}
	}
	return context, nil
}

func (r *SessionAdminProjection) List(ctx context.Context, q domain.ListQuery) ([]*domain.SessionAdminContext, error) {
	var rows []sessionAdminRow
	if err := r.baseQuery(ctx, q).Order("sessions.updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.SessionAdminContext, 0, len(rows))
	for _, row := range rows {
		context, err := r.toContext(row)
		if err != nil {
			return nil, err
		}
		out = append(out, context)
	}
	return out, nil
}

func (r *SessionAdminProjection) Get(ctx context.Context, id sharedkernel.SessionID) (*domain.SessionAdminContext, error) {
	q := domain.ListQuery{Limit: 1}
	var row sessionAdminRow
	if err := r.baseQuery(ctx, q).Where("sessions.id = ?", string(id)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.toContext(row)
}
