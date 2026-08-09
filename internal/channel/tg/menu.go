package tg

// Main menu labels (ReplyKeyboard).
const (
	BtnImage    = "🖼 图片"
	BtnVideo    = "🎬 视频"
	BtnRecharge = "💰 充值积分"
	BtnCheckIn  = "📅 签到"
	BtnProfile  = "👤 个人中心"
	BtnHelp     = "🆘 帮助"
)

func MainMenuRows() [][]string {
	return [][]string{
		{BtnImage, BtnVideo},
		{BtnRecharge, BtnCheckIn},
		{BtnProfile, BtnHelp},
	}
}

// Callback data prefixes (Telegram limit 64 bytes).
const (
	CBCasePreview       = "cp:"  // cp:<case_id>
	CBCasePreviewFolder = "cpf:" // cpf:<folder_item_id>:<case_id>
	CBCaseStart    = "cs:" // cs:<case_id>
	CBConfirm      = "cf"
	CBExit         = "ex"
	CBSkip         = "sk"
	CBMenu         = "mn"
	CBImgList      = "il"
	CBContinue     = "ct"  // resume active session
	CBReplaceStart = "rs:" // abandon active + start:<case_id>
	CBMenuFolder   = "mf:" // mf:<menu_item_id>
	CBMenuBack     = "mb:" // mb:root | mb:<parent_item_id>
)
