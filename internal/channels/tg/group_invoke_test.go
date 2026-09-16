package tg

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	"github.com/Mr9esx/Pixoma/internal/channels/application/capability"
	"github.com/Mr9esx/Pixoma/internal/channels/protocol"
	"github.com/Mr9esx/Pixoma/internal/channels/tg/tginternal"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
)

type captureAccountCapability struct {
	account protocol.AccountCtx
	params  map[string]any
}

func (*captureAccountCapability) ID() string          { return "open_case" }
func (*captureAccountCapability) DisplayName() string { return "Open case" }
func (*captureAccountCapability) ParamsSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}
func (*captureAccountCapability) Render(string, map[string]any) (protocol.RenderDecl, error) {
	return protocol.RenderDecl{}, nil
}
func (c *captureAccountCapability) Invoke(_ context.Context, account protocol.AccountCtx, _ protocol.Nav, _, _ sharedkernel.ChatID, params map[string]any) (protocol.Result, error) {
	c.account = account
	c.params = params
	return protocol.Result{}, nil
}

func TestHandleTextInvokesEnabledGroupCaseByName(t *testing.T) {
	cap := &captureAccountCapability{}
	registry := capability.NewRegistry()
	if err := registry.Register(cap); err != nil {
		t.Fatal(err)
	}
	ad := New(noopOutbound{})
	ad.ChannelID = "tg-default"
	ad.Registry = registry
	ad.Cases = groupCaseRepository{cases: []*catalogdomain.Case{{Enabled: true, Document: catalogdomain.CaseDocument{ID: 7, Name: "Portrait"}}}}

	if err := ad.HandleText(context.Background(), "tg-default:-100123", "/run Portrait", "42"); err != nil {
		t.Fatal(err)
	}
	if cap.params["step"] != "preview" || cap.params["case_id"] != "7" {
		t.Fatalf("invoke params = %#v, want preview of case 7", cap.params)
	}
}

type groupCaseRepository struct{ cases []*catalogdomain.Case }

func (r groupCaseRepository) Save(context.Context, *catalogdomain.Case) error   { return nil }
func (r groupCaseRepository) Create(context.Context, *catalogdomain.Case) error { return nil }
func (r groupCaseRepository) Get(_ context.Context, id sharedkernel.CaseID) (*catalogdomain.Case, error) {
	for _, c := range r.cases {
		if c.Document.ID == id {
			return c, nil
		}
	}
	return nil, catalogdomain.ErrNotFound
}
func (r groupCaseRepository) List(context.Context, catalogdomain.ListQuery) ([]*catalogdomain.Case, error) {
	return r.cases, nil
}
func (r groupCaseRepository) Disable(context.Context, sharedkernel.CaseID) error { return nil }
func (r groupCaseRepository) Enable(context.Context, sharedkernel.CaseID) error  { return nil }
func (r groupCaseRepository) Delete(context.Context, sharedkernel.CaseID) error  { return nil }

func TestHandleTextUsesMessageAuthorForGroupAccount(t *testing.T) {
	cap := &captureAccountCapability{}
	registry := capability.NewRegistry()
	if err := registry.Register(cap); err != nil {
		t.Fatal(err)
	}

	ad := New(noopOutbound{})
	ad.ChannelID = "tg-default"
	ad.Registry = registry
	ad.Users = identityResolverFunc(func(_ context.Context, _ sharedkernel.ChannelAddr, in identitydomain.UpsertFrom) (string, error) {
		return "user-" + in.ExternalUserID, nil
	})

	if err := ad.HandleText(context.Background(), "tg-default:-100123", "/skip", "42"); err != nil {
		t.Fatal(err)
	}

	if cap.account.ExternalUserID != "42" {
		t.Fatalf("group account external user ID = %q, want message author %q", cap.account.ExternalUserID, "42")
	}
	if cap.account.InternalUserID != "user-42" {
		t.Fatalf("group account internal user ID = %q, want %q", cap.account.InternalUserID, "user-42")
	}
}

func TestAcceptInboundTextFiltersUnaddressedGroupMessages(t *testing.T) {
	tests := []struct {
		name       string
		chatType   string
		text       string
		replyToBot bool
		want       bool
	}{
		{name: "private text", chatType: "private", text: "hello", want: true},
		{name: "group chatter", chatType: "group", text: "hello", want: false},
		{name: "group command", chatType: "group", text: "/run demo", want: true},
		{name: "group mention", chatType: "supergroup", text: "@pixoma_bot demo", want: true},
		{name: "reply to bot", chatType: "group", text: "demo", replyToBot: true, want: true},
		{name: "channel", chatType: "channel", text: "/run demo", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := acceptInboundText(tt.chatType, tt.text, tt.replyToBot, "pixoma_bot"); got != tt.want {
				t.Fatalf("acceptInboundText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAcceptInboundMediaRequiresReplyToBotInGroup(t *testing.T) {
	if !acceptInboundMedia("private", false) {
		t.Fatal("private media must remain accepted")
	}
	if acceptInboundMedia("group", false) {
		t.Fatal("unaddressed group media must be ignored")
	}
	if !acceptInboundMedia("supergroup", true) {
		t.Fatal("a reply to bot in a group must be accepted")
	}
}

func TestSendMenuOmitsReplyKeyboardInGroup(t *testing.T) {
	b, server := tginternal.NewBot(t)
	messenger := &BotMessenger{Bot: b}
	if err := messenger.SendMenu(context.Background(), sharedkernel.ChannelAddr{ChannelID: "tg-default", ExternalChatID: "-100123"}, "群入口", nil); err != nil {
		t.Fatal(err)
	}
	if markup := server.LastReplyMarkup(); strings.Contains(strings.ToLower(markup), "keyboard") {
		t.Fatalf("group menu reply markup = %q, must not include ReplyKeyboard", markup)
	}
}

type noopOutbound struct{}

func (noopOutbound) SendText(context.Context, sharedkernel.ChannelAddr, string) error { return nil }
func (noopOutbound) SendMenu(context.Context, sharedkernel.ChannelAddr, string, []protocol.MenuEntry) error {
	return nil
}
func (noopOutbound) SendList(context.Context, sharedkernel.ChannelAddr, string, [][]protocol.Button) error {
	return nil
}
func (noopOutbound) SendMedia(context.Context, sharedkernel.ChannelAddr, sharedkernel.BlobRef, string, [][]protocol.Button) error {
	return nil
}
func (noopOutbound) SendMediaURL(context.Context, sharedkernel.ChannelAddr, string, string, string, [][]protocol.Button) error {
	return nil
}
func (noopOutbound) EditReplyMarkup(context.Context, sharedkernel.ChannelAddr, int, [][]protocol.Button) error {
	return nil
}
