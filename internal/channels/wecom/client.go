// Package wecom isolates the Enterprise WeChat intelligent-bot protocol from
// Pixoma's channel and conversation layers.
package wecom

import (
	"context"

	aibot "github.com/seastart/wecom-aibot-go"
)

// ChatType identifies the delivery scope of an intelligent-bot message.
type ChatType string

const (
	Single ChatType = "single"
	Group  ChatType = "group"
)

// InboundMessage is the protocol-neutral form consumed by the adapter.
type InboundMessage struct {
	ReqID    string
	UserID   string
	ChatID   string
	ChatType ChatType
	Text     string
	Image    *Image
}

// Image identifies encrypted inbound media supplied by Enterprise WeChat.
type Image struct {
	URL    string
	AESKey string
	MIME   string
}

// Event is an intelligent-bot event frame.
type Event struct {
	ReqID    string
	Type     string
	EventKey string
	UserID   string
	ChatID   string
	ChatType ChatType
}

// Client is the only Enterprise WeChat protocol dependency used by adapters.
// The SDK implementation is kept behind this interface.
type Client interface {
	Run(ctx context.Context) error
	OnMessage(func(context.Context, InboundMessage))
	OnEvent(func(context.Context, Event))
}

// SDKClient adapts one aibot.Client connection to Pixoma's local protocol
// contract. No caller outside this package imports the SDK.
type SDKClient struct {
	client *aibot.Client
}

// NewSDKClient creates a single-connection Enterprise WeChat client.
func NewSDKClient(botID, secret, wsURL string) *SDKClient {
	return &SDKClient{client: aibot.NewClient(aibot.Config{BotID: botID, Secret: secret, Endpoint: wsURL})}
}

func (c *SDKClient) Run(ctx context.Context) error { return c.client.Run(ctx) }

func (c *SDKClient) OnMessage(handler func(context.Context, InboundMessage)) {
	if c == nil || c.client == nil || handler == nil {
		return
	}
	c.client.OnMessage(func(ctx context.Context, msg *aibot.Message) error {
		handler(ctx, mapAIBotMessage(msg))
		return nil
	})
}

func (c *SDKClient) OnEvent(handler func(context.Context, Event)) {
	if c == nil || c.client == nil || handler == nil {
		return
	}
	c.client.OnEvent(func(ctx context.Context, event *aibot.Event) error {
		handler(ctx, mapAIBotEvent(event))
		return nil
	})
}

// sdkMessage is an SDK-shaped boundary value. The concrete SDK wrapper maps
// its protocol structs into this private representation before application use.
type sdkMessage struct {
	ReqID      string
	ChatType   string
	ChatID     string
	FromUserID string
	Text       string
}

const sdkGroupChat = "group"

func mapSDKMessage(raw sdkMessage) InboundMessage {
	chatType := Single
	if raw.ChatType == sdkGroupChat {
		chatType = Group
	}
	return InboundMessage{
		ReqID: raw.ReqID, ChatType: chatType, ChatID: raw.ChatID, UserID: raw.FromUserID, Text: raw.Text,
	}
}

func mapAIBotMessage(msg *aibot.Message) InboundMessage {
	if msg == nil {
		return InboundMessage{}
	}
	raw := sdkMessage{ReqID: msg.ReqID, ChatType: msg.ChatType, ChatID: msg.ChatID}
	if msg.From != nil {
		raw.FromUserID = msg.From.UserID
	}
	if msg.Text != nil {
		raw.Text = msg.Text.Content
	}
	out := mapSDKMessage(raw)
	if msg.Image != nil {
		out.Image = &Image{URL: msg.Image.URL, AESKey: msg.Image.AESKey, MIME: msg.Image.MimeType}
	}
	return out
}

func mapAIBotEvent(event *aibot.Event) Event {
	if event == nil {
		return Event{}
	}
	out := Event{ReqID: event.ReqID, ChatID: event.ChatID, ChatType: mapChatType(event.ChatType)}
	if event.From != nil {
		out.UserID = event.From.UserID
	}
	if event.Event != nil {
		out.Type = event.Event.EventType
	}
	return out
}

func mapChatType(value string) ChatType {
	if value == sdkGroupChat {
		return Group
	}
	return Single
}
