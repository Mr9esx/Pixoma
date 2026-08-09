package domain

import "time"

// DefaultSeedTree returns the in-memory menu that matches the legacy MainMenuRows layout.
func DefaultSeedTree() MenuTree {
	return MenuTree{
		ID:    DocumentIDDefault,
		BotID: BotIDDefault,
		Items: []MenuNode{
			{ID: "btn-image", Label: "🖼 图片", Row: 0, Col: 0, Enabled: true, Kind: KindFolder},
			{ID: "btn-video", Label: "🎬 视频", Row: 0, Col: 1, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-recharge", Label: "💰 充值积分", Row: 1, Col: 0, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-checkin", Label: "📅 签到", Row: 1, Col: 1, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-profile", Label: "👤 个人中心", Row: 2, Col: 0, Enabled: true, Kind: KindPlaceholder},
			{ID: "btn-help", Label: "🆘 帮助", Row: 2, Col: 1, Enabled: true, Kind: KindPlaceholder},
		},
		UpdatedAt: time.Time{},
	}
}
