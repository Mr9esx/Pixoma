package studio

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/response"
)

func (h *Handler) contextSession(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	accountID, ok := accountID(w, r)
	if !ok {
		return "", "", false
	}
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := h.Repo.GetSession(r.Context(), accountID, sessionID); err != nil {
		failFromError(w, err)
		return "", "", false
	}
	if err := h.Context.BackfillSessionContext(r.Context(), accountID, sessionID); err != nil {
		failFromError(w, err)
		return "", "", false
	}
	return accountID, sessionID, true
}

func (h *Handler) getSessionContext(w http.ResponseWriter, r *http.Request) {
	accountID, sessionID, ok := h.contextSession(w, r)
	if !ok {
		return
	}
	view, err := h.Context.GetContextOverview(r.Context(), accountID, sessionID)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, view)
}

func (h *Handler) getCurrentContextRequest(w http.ResponseWriter, r *http.Request) {
	accountID, sessionID, ok := h.contextSession(w, r)
	if !ok {
		return
	}
	row, err := h.Context.GetCurrentContextRequest(r.Context(), accountID, sessionID)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, row)
}

func (h *Handler) listContextRequests(w http.ResponseWriter, r *http.Request) {
	accountID, sessionID, ok := h.contextSession(w, r)
	if !ok {
		return
	}
	limit := min(max(queryInt(r, "limit", 100), 1), 200)
	offset := max(queryInt(r, "offset", 0), 0)
	rows, err := h.Context.ListContextRequests(r.Context(), accountID, sessionID, limit+1, offset)
	if err != nil {
		failFromError(w, err)
		return
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	response.OK(w, map[string]any{"requests": rows, "has_more": hasMore, "next_offset": offset + len(rows)})
}

func (h *Handler) getContextRequest(w http.ResponseWriter, r *http.Request) {
	accountID, sessionID, ok := h.contextSession(w, r)
	if !ok {
		return
	}
	row, err := h.Context.GetContextRequest(r.Context(), accountID, sessionID, chi.URLParam(r, "attemptID"))
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, row)
}

func (h *Handler) listContextEvents(w http.ResponseWriter, r *http.Request) {
	accountID, sessionID, ok := h.contextSession(w, r)
	if !ok {
		return
	}
	limit := min(max(queryInt(r, "limit", 100), 1), 200)
	offset := max(queryInt(r, "offset", 0), 0)
	rows, err := h.Context.ListContextEvents(r.Context(), accountID, sessionID, r.URL.Query().Get("kind"), limit+1, offset)
	if err != nil {
		failFromError(w, err)
		return
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	response.OK(w, map[string]any{"events": rows, "has_more": hasMore, "next_offset": offset + len(rows)})
}
