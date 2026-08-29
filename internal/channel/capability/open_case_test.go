package capability

import (
	"context"
	"strings"
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

type lockedCaseService struct{ fakeCaseService }

func (lockedCaseService) StartCase(context.Context, botapp.StartCaseCmd) (*botapp.SessionView, error) {
	return nil, convdomain.ErrSessionLocked
}

// mediaPreviewCaseService 返回带媒体预览的 Case，用于验证 preview 步骤以媒体引用投递。
type mediaPreviewCaseService struct{ fakeCaseService }

func (mediaPreviewCaseService) GetCase(_ context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error) {
	return &catalogdomain.Case{
		Enabled: true,
		Document: catalogdomain.CaseDocument{
			ID: id, Name: "图片 B",
			Description: "示例模板", Preview: "previews/abc.png",
		},
	}, nil
}

func TestOpenCase_PreviewMediaDelivery(t *testing.T) {
	ctx := context.Background()

	// 媒体预览：preview 返回媒体引用（blob key）供 runtime 发送，且不再拼接文本预览。
	r := NewRegistry()
	if err := r.Register(OpenCase{App: mediaPreviewCaseService{}}); err != nil {
		t.Fatal(err)
	}
	res, err := r.Invoke(ctx, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "preview", "case_id": "1"},
		ChatID:       "tg-default:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Media) != 1 {
		t.Fatalf("media delivery expected 1 MediaRef, got %+v", res.Media)
	}
	if res.Media[0].Key != "previews/abc.png" || res.Media[0].MIME != "image/png" {
		t.Fatalf("unexpected media ref: %+v", res.Media[0])
	}
	if strings.Contains(res.Text, "预览说明") {
		t.Fatalf("media preview must not include text preview hint: %q", res.Text)
	}
	if !strings.Contains(res.Text, "示例模板") {
		t.Fatalf("caption must include description: %q", res.Text)
	}

	// 旧文本回退：非 previews/ 前缀的 Preview 不走媒体，仍拼接文本预览说明。
	r2 := NewRegistry()
	if err := r2.Register(OpenCase{App: fakeCaseService{}}); err != nil {
		t.Fatal(err)
	}
	res2, err := r2.Invoke(ctx, protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "preview", "case_id": "1"},
		ChatID:       "tg-default:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Media) != 0 {
		t.Fatalf("legacy preview must not emit media, got %+v", res2.Media)
	}
	if !strings.Contains(res2.Text, "预览说明") {
		t.Fatalf("legacy preview must keep text hint: %q", res2.Text)
	}
}
func TestOpenCase_StartWithLockedSessionOffersExit(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(OpenCase{App: lockedCaseService{}}); err != nil {
		t.Fatal(err)
	}
	res, err := r.Invoke(context.Background(), protocol.CapabilityInvoke{
		CapabilityID: "open_case",
		Params:       map[string]any{"step": "start", "case_id": "1"},
		ChatID:       "tg-default:1",
		Account:      protocol.AccountCtx{InternalUserID: "u1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text == "" {
		t.Fatal("locked session must return a hint text")
	}
	if len(res.Options) != 1 || res.Options[0].Label != "✕ 退出" {
		t.Fatalf("locked options=%+v", res.Options)
	}
}
