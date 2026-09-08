// Package application implements admin channel lifecycle cleanup.
package application

import (
	"context"
	"time"

	"gorm.io/gorm"

	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	mencardpersist "github.com/Mr9esx/Pixoma/internal/menus/infrastructure/persistence"
	convdomain "github.com/Mr9esx/Pixoma/internal/sessions/domain"
	sesspersist "github.com/Mr9esx/Pixoma/internal/sessions/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

// DeleteWithCleanup returns a function that deletes the channel row,
// terminates its active sessions (collecting/confirming → exited) and removes
// channel-scoped menu/card rows in one transaction. It returns the chats of
// terminated sessions so the caller can notify users after commit.
func DeleteWithCleanup(db *gorm.DB) func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error) {
	return func(ctx context.Context, channelID string) ([]sharedkernel.ChatID, error) {
		var chats []sharedkernel.ChatID
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("id = ?", channelID).Delete(&channelpersist.ChannelRow{}).Error; err != nil {
				return err
			}
			var rows []sesspersist.SessionRow
			if err := tx.Where("channel_id = ? AND status IN ?", channelID, []string{
				string(convdomain.StatusCollecting),
				string(convdomain.StatusConfirming),
			}).Find(&rows).Error; err != nil {
				return err
			}
			for _, row := range rows {
				chats = append(chats, sharedkernel.ChatID(sharedkernel.FormatChatID(sharedkernel.ChannelAddr{
					ChannelID:      row.ChannelID,
					ExternalChatID: row.ChatExternalID,
				})))
			}
			if len(rows) > 0 {
				if err := tx.Model(&sesspersist.SessionRow{}).
					Where("channel_id = ? AND status IN ?", channelID, []string{
						string(convdomain.StatusCollecting),
						string(convdomain.StatusConfirming),
					}).
					Updates(map[string]any{
						"status":     string(convdomain.StatusExited),
						"updated_at": time.Now().UTC(),
					}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("channel_id = ?", channelID).Delete(&mencardpersist.MainMenuRow{}).Error; err != nil {
				return err
			}
			return tx.Where("channel_id = ?", channelID).Delete(&mencardpersist.CardRow{}).Error
		})
		return chats, err
	}
}
