package sharedkernel

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	CaseID    uint64
	TaskID    string
	SessionID string
	ChatID    string
	EdgeID    string
)

// ParseCaseID parses a decimal string into a CaseID.
func ParseCaseID(s string) (CaseID, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return CaseID(n), nil
}

// ChannelAddr identifies a conversation on a specific channel.
type ChannelAddr struct {
	ChannelID      string
	ExternalChatID string
}

// FormatChatID renders a ChannelAddr as "channelID:externalChatID" (e.g. "tg:123").
func FormatChatID(addr ChannelAddr) string {
	return addr.ChannelID + ":" + addr.ExternalChatID
}

// ParseChatID parses a "channelID:externalChatID" string into a ChannelAddr.
func ParseChatID(s string) (ChannelAddr, error) {
	channelID, externalChatID, ok := strings.Cut(s, ":")
	if !ok || channelID == "" || externalChatID == "" {
		return ChannelAddr{}, fmt.Errorf("invalid chat id %q", s)
	}
	return ChannelAddr{ChannelID: channelID, ExternalChatID: externalChatID}, nil
}

type BlobRef struct {
	Bucket string `json:"bucket,omitempty"`
	Key    string `json:"key"`
	MIME   string `json:"mime,omitempty"`
	Size   int64  `json:"size,omitempty"`
}

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskQueued    TaskStatus = "queued"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)
