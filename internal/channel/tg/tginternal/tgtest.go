// Package tginternal exposes a small test helper that spins up an httptest
// server emulating the Telegram Bot API, so tests can drive the real
// *bot.Bot through tg.BotMessenger without writing Outbound mocks.
package tginternal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
)

// Server wraps an httptest server that emulates the Telegram Bot API for
// tests that drive the real *bot.Bot through its send paths.
type Server struct {
	srv *httptest.Server
	mu  sync.Mutex

	sendMessage            int
	sendPhoto              int
	sendVideo              int
	sendAnimation          int
	sendDocument           int
	editMessageReplyMarkup int

	lastChatID  string
	lastCaption string
	lastMethod  string

	uploadNames      []string
	editMessageIDs   []int
	sendMessageTexts []string

	// lastInlineKeyboardText mirrors the most recent sendMessage's inline
	// keyboard buttons, captured from the reply_markup form field. Each row
	// is a slice of button texts. Use for asserting routing decisions that
	// don't depend on the full button payload.
	lastInlineKeyboardText [][]string
	// lastInlineKeyboardData mirrors the most recent inline_keyboard buttons'
	// callback_data values (parallel to lastInlineKeyboardText).
	lastInlineKeyboardData [][]string

	lastReplyMarkup string
}

// Close shuts the underlying httptest server down.
func (s *Server) Close() { s.srv.Close() }

func (s *Server) SendMessageCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendMessage
}

func (s *Server) SendPhotoCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendPhoto
}

func (s *Server) SendVideoCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendVideo
}

func (s *Server) SendAnimationCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendAnimation
}

func (s *Server) SendDocumentCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendDocument
}

func (s *Server) EditMessageReplyMarkupCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.editMessageReplyMarkup
}

func (s *Server) LastChatID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastChatID
}

func (s *Server) LastCaption() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastCaption
}

func (s *Server) LastMethod() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastMethod
}

// UploadNames returns the ordered list of multipart file names that
// BotMessenger sent on send* APIs. Tests use this to assert the blob key
// routed through to Telegram (photoUploadName(key) is the filename).
func (s *Server) UploadNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.uploadNames))
	copy(out, s.uploadNames)
	return out
}

// EditedMessageIDs returns the message_id values passed to
// editMessageReplyMarkup in arrival order.
func (s *Server) EditedMessageIDs() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int, len(s.editMessageIDs))
	copy(out, s.editMessageIDs)
	return out
}

// SendMessageTexts returns the text values of every sendMessage call in
// arrival order. Use this to disambiguate multiple text sends in one test
// (e.g. text payload + followup menu).
func (s *Server) SendMessageTexts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.sendMessageTexts))
	copy(out, s.sendMessageTexts)
	return out
}

// LastInlineKeyboardTexts returns the button texts of the most recent
// sendMessage's inline keyboard, in row order. Empty when no buttons.
func (s *Server) LastInlineKeyboardTexts() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastInlineKeyboardText
}

// LastInlineKeyboardData returns the callback_data values of the most recent
// sendMessage's inline keyboard, parallel to LastInlineKeyboardTexts.
func (s *Server) LastInlineKeyboardData() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastInlineKeyboardData
}

// LastReplyMarkup returns the raw reply_markup form value from the most
// recent sendPhoto / sendVideo / sendMessage call. Tests use this to verify
// that inline keyboards are forwarded when sending media.
func (s *Server) LastReplyMarkup() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastReplyMarkup
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastMethod = r.URL.Path
	var textVal string
	if err := r.ParseMultipartForm(1 << 20); err == nil && r.MultipartForm != nil {
		s.lastChatID = firstNonEmpty(r.MultipartForm.Value["chat_id"])
		textVal = firstNonEmpty(r.MultipartForm.Value["text"])
		s.lastCaption = firstNonEmpty(r.MultipartForm.Value["caption"])
	}
	if s.lastChatID == "" {
		s.lastChatID = r.FormValue("chat_id")
	}
	if textVal == "" {
		textVal = r.FormValue("text")
	}
	if s.lastCaption == "" {
		s.lastCaption = r.FormValue("caption")
	}
	// Decode inline_keyboard for any send* call that carries a reply_markup.
	if rm := r.FormValue("reply_markup"); rm != "" {
		s.lastInlineKeyboardText, s.lastInlineKeyboardData = decodeInlineKeyboard(rm)
	} else {
		s.lastInlineKeyboardText = nil
		s.lastInlineKeyboardData = nil
	}
	switch {
	case strings.HasSuffix(r.URL.Path, "/sendMessage"):
		s.sendMessage++
		s.sendMessageTexts = append(s.sendMessageTexts, textVal)
	case strings.HasSuffix(r.URL.Path, "/sendPhoto"):
		s.sendPhoto++
		s.recordUpload(r)
	case strings.HasSuffix(r.URL.Path, "/sendVideo"):
		s.sendVideo++
		s.recordUpload(r)
	case strings.HasSuffix(r.URL.Path, "/sendAnimation"):
		s.sendAnimation++
		s.recordUpload(r)
	case strings.HasSuffix(r.URL.Path, "/sendDocument"):
		s.sendDocument++
		s.recordUpload(r)
	case strings.HasSuffix(r.URL.Path, "/editMessageReplyMarkup"):
		s.editMessageReplyMarkup++
		if mid := r.FormValue("message_id"); mid != "" {
			if id, perr := strconv.Atoi(mid); perr == nil {
				s.editMessageIDs = append(s.editMessageIDs, id)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
}

func (s *Server) recordUpload(r *http.Request) {
	if r.MultipartForm == nil {
		return
	}
	for _, files := range r.MultipartForm.File {
		for _, fh := range files {
			s.uploadNames = append(s.uploadNames, fh.Filename)
		}
	}
}

func firstNonEmpty(vals []string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// NewBot returns a *bot.Bot wired to a local httptest server emulating the
// Telegram Bot API. The server is closed automatically when the test ends;
// callers can inspect calls via the returned *Server and wrap the bot in
// their own outbound (e.g. tg.BotMessenger) without depending on this
// package.
func NewBot(t *testing.T) (*bot.Bot, *Server) {
	t.Helper()
	s := &Server{}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(s.srv.Close)
	b, err := bot.New("test-token",
		bot.WithServerURL(s.srv.URL),
		bot.WithSkipGetMe(),
	)
	if err != nil {
		t.Fatalf("new tg bot: %v", err)
	}
	return b, s
}

// decodeInlineKeyboard parses a Telegram reply_markup JSON blob ({"inline_keyboard":
// [[{"text":..., "callback_data":...}, ...], ...]}) into parallel text and
// callback_data slices. Unknown / non-inline markup returns nil.
func decodeInlineKeyboard(raw string) ([][]string, [][]string) {
	var wrap struct {
		InlineKeyboard [][]struct {
			Text         string `json:"text"`
			CallbackData string `json:"callback_data"`
		} `json:"inline_keyboard"`
	}
	if err := json.Unmarshal([]byte(raw), &wrap); err != nil {
		return nil, nil
	}
	if len(wrap.InlineKeyboard) == 0 {
		return nil, nil
	}
	texts := make([][]string, len(wrap.InlineKeyboard))
	data := make([][]string, len(wrap.InlineKeyboard))
	for i, row := range wrap.InlineKeyboard {
		for _, b := range row {
			texts[i] = append(texts[i], b.Text)
			data[i] = append(data[i], b.CallbackData)
		}
	}
	return texts, data
}
