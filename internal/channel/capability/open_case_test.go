package capability

import (
	"context"
	"testing"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/botapp"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type fakeCaseService struct{}

func (fakeCaseService) StartCase(_ context.Context, cmd botapp.StartCaseCmd) (*botapp.SessionView, error) {
	return &botapp.SessionView{
		Status: convdomain.StatusCollecting,
		Index:  0,
		Keys:   []string{"prompt"},
		CaseID: cmd.CaseID,
	}, nil
}
func (fakeCaseService) GetCase(_ context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error) {
	return &catalogdomain.Case{
		Enabled: true,
		Document: catalogdomain.CaseDocument{
			ID: id, Name: "图片 B",
			Description: "示例模板", Preview: "返回一张示例图",
		},
	}, nil
}
func (fakeCaseService) GetSession(context.Context, sharedkernel.ChatID) (*botapp.SessionView, error) {
	return nil, nil
}
func (fakeCaseService) SkipInput(context.Context, sharedkernel.ChatID) (*botapp.SessionView, error) {
	return nil, nil
}
func (fakeCaseService) SubmitInput(context.Context, sharedkernel.ChatID, convdomain.DraftValue) (*botapp.SessionView, error) {
	return nil, nil
}
func (fakeCaseService) ExitSession(context.Context, sharedkernel.ChatID) error { return nil }
func (fakeCaseService) ConfirmRun(context.Context, botapp.ConfirmRunCmd) (*botapp.ConfirmRunResult, error) {
	return nil, nil
}

func TestOpenCase_ListAndPreviewFlow(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(OpenCase{App: fakeCaseService{}}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// 非法 step 被 schema 拒绝
	if _, err := r.Invoke(ctx, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "bogus", "case_ids": []any{"1"}},
	}); err == nil {
		t.Fatal("invalid step must be rejected by schema")
	}

	// list → options
	res, err := r.Invoke(ctx, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"case_ids": []any{"1"}},
		ChatID:       "tg-default:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Options) != 1 || res.Options[0].Label != "图片 B · ¥15" {
		t.Fatalf("list options=%+v", res.Options)
	}

	// preview → options 开始/返回
	res, err = r.Invoke(ctx, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "preview", "case_id": "1", "case_ids": []any{"1"}},
		ChatID:       "tg-default:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Options) != 2 || res.Options[0].Label != "▶ 开始 Case" {
		t.Fatalf("preview options=%+v", res.Options)
	}

	// start → prompt 提示 + 跳过/退出
	res, err = r.Invoke(ctx, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "start", "case_id": "1"},
		ChatID:       "tg-default:1",
		Account:      protocol.AccountCtx{InternalUserID: "u1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Options) != 2 || res.Options[0].Label != "跳过" {
		t.Fatalf("start options=%+v", res.Options)
	}
}
