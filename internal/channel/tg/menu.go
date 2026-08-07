package tg

// Main menu labels (ReplyKeyboard), matching product mock.
const (
	BtnImage       = "🔞 图片"
	BtnVideo       = "🔞 视频"
	BtnVideoUndress = "🎬 视频脱衣"
	BtnHotTemplates = "🔥 热门模版"
	BtnRecharge    = "💰 充值积分"
	BtnCheckIn     = "📅 签到"
	BtnProfile     = "👤 我的"
	BtnInvite      = "🤝 邀请赚钱"
	BtnHelp        = "🆘 帮助"
)

func MainMenuRows() [][]string {
	return [][]string{
		{BtnImage, BtnVideo, BtnVideoUndress},
		{BtnHotTemplates, BtnRecharge, BtnCheckIn},
		{BtnProfile, BtnInvite, BtnHelp},
	}
}

// Callback data prefixes (Telegram limit 64 bytes).
const (
	CBCasePreview = "cp:" // cp:<case_id>
	CBCaseStart   = "cs:" // cs:<case_id>
	CBConfirm     = "cf"
	CBExit        = "ex"
	CBSkip        = "sk"
	CBMenu        = "mn"
	CBImgList     = "il"
)
