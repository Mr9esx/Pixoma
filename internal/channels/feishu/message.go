package feishu

import (
	"encoding/json"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// inbound is a normalized Feishu message event suitable for protocol mapping.
type inbound struct {
	ChatID       string
	MessageID    string
	SenderUserID string
	ChatType     string
	IsGroup      bool
	IsAtBot      bool
	Text         string
	ImageKey     string
	MIME         string
}

type textContent struct {
	Text string `json:"text"`
}

type imageContent struct {
	ImageKey string `json:"image_key"`
}

// parseInbound translates a raw lark message event into the normalized shape.
// Group messages are only actionable when the bot is mentioned (not @all).
func parseInbound(e *larkim.P2MessageReceiveV1) *inbound {
	if e == nil || e.Event == nil || e.Event.Message == nil {
		return nil
	}
	msg := e.Event.Message
	in := &inbound{}
	if msg.ChatId != nil {
		in.ChatID = *msg.ChatId
	}
	if msg.MessageId != nil {
		in.MessageID = *msg.MessageId
	}
	if msg.ChatType != nil {
		in.ChatType = *msg.ChatType
	}
	chatType := ""
	if msg.ChatType != nil {
		chatType = *msg.ChatType
	}
	in.IsGroup = chatType == "group" || chatType == "topic_group"

	if s := e.Event.Sender; s != nil && s.SenderId != nil {
		if s.SenderId.UserId != nil && *s.SenderId.UserId != "" {
			in.SenderUserID = *s.SenderId.UserId
		} else if s.SenderId.OpenId != nil && *s.SenderId.OpenId != "" {
			in.SenderUserID = *s.SenderId.OpenId
		}
	}
	if in.ChatID == "" || in.SenderUserID == "" {
		return nil
	}
	if in.IsGroup {
		atBot := false
		for _, m := range msg.Mentions {
			if m == nil || m.Key == nil {
				continue
			}
			if *m.Key != "mention" {
				continue
			}
			atBot = true
			break
		}
		if !atBot {
			return nil
		}
		in.IsAtBot = true
	}

	msgType := ""
	if msg.MessageType != nil {
		msgType = *msg.MessageType
	}
	content := ""
	if msg.Content != nil {
		content = *msg.Content
	}
	switch msgType {
	case "text":
		var tc textContent
		if json.Unmarshal([]byte(content), &tc) == nil {
			in.Text = strings.TrimSpace(tc.Text)
		}
	case "image":
		var ic imageContent
		if json.Unmarshal([]byte(content), &ic) == nil {
			in.ImageKey = ic.ImageKey
		}
		in.MIME = "image/jpeg"
	}
	if in.Text == "" && in.ImageKey == "" {
		return nil
	}
	return in
}
