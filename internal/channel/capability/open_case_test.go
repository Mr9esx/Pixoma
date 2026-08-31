package capability

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	catalogdomain "github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/channel/protocol"
	texttpl "github.com/mr9esx/comfyui_tgbot/internal/channel/text"
	convdomain "github.com/mr9esx/comfyui_tgbot/internal/conversation/domain"
	identitydomain "github.com/mr9esx/comfyui_tgbot/internal/identity/domain"
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

type unauthorizedCaseService struct{ fakeCaseService }

func (unauthorizedCaseService) StartCase(context.Context, botapp.StartCaseCmd) (*botapp.SessionView, error) {
	return nil, fmt.Errorf("start must be blocked before StartCase")
}

type accessUserRepository map[string]identitydomain.UserAccess

func (repo accessUserRepository) UpsertByChannelExternal(context.Context, identitydomain.UpsertFrom) (*identitydomain.User, error) {
	return nil, errors.New("not used")
}

func (repo accessUserRepository) GetByID(_ context.Context, id string) (*identitydomain.User, error) {
	access, ok := repo[id]
	if !ok {
		return nil, identitydomain.ErrNotFound
	}
	return &identitydomain.User{ID: id, Access: access}, nil
}

func (repo accessUserRepository) SetAccess(_ context.Context, id string, access identitydomain.UserAccess) (*identitydomain.User, error) {
	return &identitydomain.User{ID: id, Access: access}, nil
}

func (repo accessUserRepository) List(context.Context, identitydomain.ListQuery) ([]*identitydomain.User, error) {
	return nil, errors.New("not used")
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

type configuredCaseService struct{ fakeCaseService }

func (configuredCaseService) GetCase(_ context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error) {
	return &catalogdomain.Case{
		Enabled: true,
		Document: catalogdomain.CaseDocument{
			ID:   id,
			Name: "图片 B",
			Inputs: []catalogdomain.InputField{
				{Key: "count", Type: "number"},
			},
		},
	}, nil
}

func (configuredCaseService) GetSession(_ context.Context, _ sharedkernel.ChatID) (*botapp.SessionView, error) {
	return &botapp.SessionView{
		Status: convdomain.StatusCollecting,
		Index:  0,
		Keys:   []string{"count"},
		CaseID: 1,
	}, nil
}

func (configuredCaseService) ConfirmRun(context.Context, botapp.ConfirmRunCmd) (*botapp.ConfirmRunResult, error) {
	return &botapp.ConfirmRunResult{TaskID: "123"}, nil
}

type mappedRenderer struct{}

func (mappedRenderer) Render(_ context.Context, _ string, key string, vars map[string]string) string {
	if key == texttpl.KeySubmitStarted {
		return texttpl.Render(key+"|task={{ task_id }}", vars)
	}
	return texttpl.Render(key+"|custom", vars)
}

func TestOpenCase_ConfigurableWorkflowStages(t *testing.T) {
	ctx := context.Background()
	app := configuredCaseService{fakeCaseService{}}
	capability := OpenCase{
		App: app, Texts: mappedRenderer{},
		Users: accessUserRepository{"": identitydomain.UserAccessAlwaysAllowed},
	}

	preview, err := capability.Invoke(ctx, protocol.AccountCtx{ChannelID: "tg-custom"}, protocol.Nav{}, "tg-custom:1", map[string]any{"step": "preview", "case_id": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.Text, "preview_hint_label|custom") ||
		!strings.Contains(preview.Text, "preview_mock_hint|custom") {
		t.Fatalf("preview copy not configurable: %q", preview.Text)
	}
	if preview.Options[0].Label != "button_start_case|custom" {
		t.Fatalf("start button not configurable: %+v", preview.Options)
	}

	number, err := capability.submitText(ctx, "tg-custom", "tg-custom:1", map[string]any{"text": "bad"})
	if err != nil {
		t.Fatal(err)
	}
	if number.Text != "input_invalid_number|custom" {
		t.Fatalf("number error not configurable: %q", number.Text)
	}

	input, err := capability.start(ctx, "tg-custom", protocol.AccountCtx{InternalUserID: "u1"}, "tg-custom:1", map[string]any{"case_id": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if input.Text != "input_prompt|custom" {
		t.Fatalf("input prompt not configurable: %q", input.Text)
	}
	if input.Options[0].Label != "button_skip|custom" || input.Options[1].Label != "button_exit|custom" {
		t.Fatalf("input buttons not configurable: %+v", input.Options)
	}

	confirm, err := renderSession(ctx, "tg-custom", capability, &botapp.SessionView{Status: convdomain.StatusConfirming})
	if err != nil {
		t.Fatal(err)
	}
	if confirm.Text != "confirm_run|custom" {
		t.Fatalf("confirm copy not configurable: %q", confirm.Text)
	}
	if confirm.Options[0].Label != "button_confirm_run|custom" || confirm.Options[1].Label != "button_exit|custom" {
		t.Fatalf("confirm buttons not configurable: %+v", confirm.Options)
	}

	submit, err := capability.confirm(ctx, "tg-custom", "tg-custom:1")
	if err != nil {
		t.Fatal(err)
	}
	if submit.Text != "submit_started|task=123" {
		t.Fatalf("submitted copy not configurable: %q", submit.Text)
	}
}

func TestOpenCase_PreviewMediaDelivery(t *testing.T) {
	ctx := context.Background()

	// 媒体预览：preview 返回媒体引用（blob key）供 runtime 发送，且不再拼接文本预览。
	r := NewRegistry()
	if err := r.Register(OpenCase{
		App:   mediaPreviewCaseService{},
		Users: accessUserRepository{"": identitydomain.UserAccessAlwaysAllowed},
	}); err != nil {
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
	if err := r2.Register(OpenCase{
		App:   fakeCaseService{},
		Users: accessUserRepository{"": identitydomain.UserAccessAlwaysAllowed},
	}); err != nil {
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
	if err := r.Register(OpenCase{
		App:   lockedCaseService{},
		Users: accessUserRepository{"u1": identitydomain.UserAccessAlwaysAllowed},
	}); err != nil {
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

func TestOpenCase_DeniedOrPaidUserCannotStartCase(t *testing.T) {
	ctx := context.Background()
	capability := OpenCase{
		App:   unauthorizedCaseService{},
		Texts: mappedRenderer{},
	}
	for _, access := range []identitydomain.UserAccess{
		identitydomain.UserAccessDenied,
		identitydomain.UserAccessPaid,
	} {
		capability.Users = accessUserRepository{"u1": access}
		res, err := capability.Invoke(ctx, protocol.AccountCtx{InternalUserID: "u1"}, protocol.Nav{}, "tg-default:1", map[string]any{
			"step": "start", "case_id": "1",
		})
		if err != nil {
			t.Fatalf("%s invoke: %v", access, err)
		}
		if res.Text != capability.renderText(ctx, "tg-default", texttpl.KeyAccessDenied, nil) {
			t.Fatalf("%s text=%q", access, res.Text)
		}
	}
}

func TestOpenCase_AlwaysAllowedUserCanStartCase(t *testing.T) {
	ctx := context.Background()
	capability := OpenCase{
		App:   fakeCaseService{},
		Users: accessUserRepository{"u1": identitydomain.UserAccessAlwaysAllowed},
	}
	res, err := capability.Invoke(ctx, protocol.AccountCtx{InternalUserID: "u1"}, protocol.Nav{}, "tg-default:1", map[string]any{
		"step": "start", "case_id": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text == "" {
		t.Fatal("allowed user did not enter workflow")
	}
}
