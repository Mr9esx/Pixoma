package tg

import (
	"github.com/Mr9esx/Pixoma/internal/channels/conversation"
)

func newInvokeStore() *conversation.ActionStore {
	return conversation.NewActionStore()
}
