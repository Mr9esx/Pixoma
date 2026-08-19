package domain

import "time"

// DefaultSeedTree returns the in-memory default menu for a channel.
func DefaultSeedTree(channelID string) MenuTree {
	return MenuTree{
		ChannelID: channelID,
		Items: []MenuNode{
			{ID: "btn-image", Label: "🖼 图片", Order: 0, Enabled: true, CapabilityID: "open_case", Params: map[string]any{"case_ids": []any{}}},
			{ID: "btn-video", Label: "🎬 视频", Order: 1, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "🎬 视频：暂未开放"}},
			{ID: "btn-recharge", Label: "💰 充值积分", Order: 2, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "💰 充值积分：暂未开放"}},
			{ID: "btn-checkin", Label: "📅 签到", Order: 3, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "📅 签到：暂未开放"}},
			{ID: "btn-profile", Label: "👤 个人中心", Order: 4, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "👤 个人中心：暂未开放"}},
			{ID: "btn-help", Label: "🆘 帮助", Order: 5, Enabled: true, CapabilityID: "reply_text", Params: map[string]any{"text": "🆘 帮助：暂未开放"}},
		},
		UpdatedAt: time.Time{},
	}
}
