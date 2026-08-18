package domain

import "time"

// DefaultSeedTree returns the in-memory default menu for a channel.
func DefaultSeedTree(channelID string) MenuTree {
	return MenuTree{
		ChannelID: channelID,
		Items: []MenuNode{
			{ID: "btn-image", Label: "🖼 图片", Order: 0, Enabled: true, Kind: KindFolder},
			{ID: "btn-video", Label: "🎬 视频", Order: 1, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-recharge", Label: "💰 充值积分", Order: 2, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-checkin", Label: "📅 签到", Order: 3, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-profile", Label: "👤 个人中心", Order: 4, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-help", Label: "🆘 帮助", Order: 5, Enabled: true, Kind: KindPlaceholder},
		},
		UpdatedAt: time.Time{},
	}
}
