package domain

import "time"

// DefaultSeed returns the in-memory menu that matches the legacy MainMenuRows layout.
func DefaultSeed() MenuDocument {
	return MenuDocument{
		ID: DocumentIDDefault,
		Items: []MenuItem{
			{ID: "btn-image", Label: "🖼 图片", Row: 0, Col: 0, Enabled: true, Action: ActionListCasesByTag, Tag: "image"},
			{ID: "btn-video", Label: "🎬 视频", Row: 0, Col: 1, Enabled: true, Action: ActionPlaceholder},
			{ID: "btn-recharge", Label: "💰 充值积分", Row: 1, Col: 0, Enabled: true, Action: ActionPlaceholder},
			{ID: "btn-checkin", Label: "📅 签到", Row: 1, Col: 1, Enabled: true, Action: ActionPlaceholder},
			{ID: "btn-profile", Label: "👤 个人中心", Row: 2, Col: 0, Enabled: true, Action: ActionPlaceholder},
			{ID: "btn-help", Label: "🆘 帮助", Row: 2, Col: 1, Enabled: true, Action: ActionPlaceholder},
		},
		UpdatedAt: time.Time{},
	}
}
