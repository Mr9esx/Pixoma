package tg

import (
	"fmt"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/channel/ports"
)

// TranslateCallback converts a Telegram callback data payload into a normalized Action.
func TranslateCallback(data string) (ports.Action, error) {
	switch {
	case data == CBMenu:
		return ports.Action{Type: ports.ActionOpenMenu}, nil
	case data == CBConfirm:
		return ports.Action{Type: ports.ActionConfirm}, nil
	case data == CBExit:
		return ports.Action{Type: ports.ActionExit}, nil
	case data == CBSkip:
		return ports.Action{Type: ports.ActionSkip}, nil
	case data == CBContinue:
		return ports.Action{Type: ports.ActionContinue}, nil
	case strings.HasPrefix(data, CBReplaceStart):
		id := strings.TrimPrefix(data, CBReplaceStart)
		if id == "" {
			return ports.Action{}, fmt.Errorf("tg callback: empty replace_start case")
		}
		return ports.Action{Type: ports.ActionReplaceStart, CaseID: id}, nil
	case strings.HasPrefix(data, CBCasePreviewFolder):
		rest := strings.TrimPrefix(data, CBCasePreviewFolder)
		folderID, caseID, ok := strings.Cut(rest, ":")
		if !ok || folderID == "" || caseID == "" {
			return ports.Action{}, fmt.Errorf("tg callback: invalid folder preview %q", data)
		}
		return ports.Action{Type: ports.ActionOpenCase, CaseID: caseID, BackRef: folderID}, nil
	case strings.HasPrefix(data, CBCasePreview):
		id := strings.TrimPrefix(data, CBCasePreview)
		if id == "" {
			return ports.Action{}, fmt.Errorf("tg callback: empty case preview")
		}
		return ports.Action{Type: ports.ActionOpenCase, CaseID: id}, nil
	case strings.HasPrefix(data, CBCaseStart):
		id := strings.TrimPrefix(data, CBCaseStart)
		if id == "" {
			return ports.Action{}, fmt.Errorf("tg callback: empty case start")
		}
		return ports.Action{Type: ports.ActionStartCase, CaseID: id}, nil
	case strings.HasPrefix(data, CBMenuFolder):
		id := strings.TrimPrefix(data, CBMenuFolder)
		if id == "" {
			return ports.Action{}, fmt.Errorf("tg callback: empty folder")
		}
		return ports.Action{Type: ports.ActionOpenFolder, MenuItemID: id}, nil
	case strings.HasPrefix(data, CBMenuBack):
		target := strings.TrimPrefix(data, CBMenuBack)
		if target == "root" {
			return ports.Action{Type: ports.ActionOpenMenu}, nil
		}
		if target == "" {
			return ports.Action{}, fmt.Errorf("tg callback: empty back target")
		}
		return ports.Action{Type: ports.ActionOpenFolder, MenuItemID: target}, nil
	default:
		return ports.Action{}, fmt.Errorf("tg callback: unknown data %q", data)
	}
}

// encodeAction renders a normalized Action back into a Telegram callback payload.
func encodeAction(a ports.Action) string {
	switch a.Type {
	case ports.ActionOpenMenu:
		return CBMenu
	case ports.ActionOpenFolder:
		return CBMenuFolder + a.MenuItemID
	case ports.ActionOpenCase:
		if a.BackRef != "" && a.BackRef != "root" {
			return CBCasePreviewFolder + a.BackRef + ":" + a.CaseID
		}
		return CBCasePreview + a.CaseID
	case ports.ActionStartCase:
		return CBCaseStart + a.CaseID
	case ports.ActionConfirm:
		return CBConfirm
	case ports.ActionExit:
		return CBExit
	case ports.ActionSkip:
		return CBSkip
	case ports.ActionContinue:
		return CBContinue
	case ports.ActionReplaceStart:
		return CBReplaceStart + a.CaseID
	default:
		return CBMenu
	}
}
