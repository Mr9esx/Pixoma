package cases

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Handler serves Case admin CRUD under /api/v1/cases.
type Handler struct {
	Repo     domain.Repository
	Validate func(domain.CaseDocument) error
}

// Mount registers chi routes on r (caller mounts under /api/v1/cases).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Patch("/{id}", h.patch)
	r.Post("/{id}/disable", h.disable)
	r.Post("/{id}/enable", h.enable)
}

type caseDTO struct {
	domain.CaseDocument
	Enabled bool `json:"enabled"`
}

type writeBody struct {
	domain.CaseDocument
	Enabled *bool `json:"enabled"`
}

func toDTO(c *domain.Case) caseDTO {
	return caseDTO{CaseDocument: c.Document, Enabled: c.Enabled}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := parseListQuery(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := h.Repo.List(r.Context(), q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]caseDTO, 0, len(list))
	for _, c := range list {
		if c == nil {
			continue
		}
		out = append(out, toDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body writeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.validate(body.CaseDocument); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	c := &domain.Case{Document: body.CaseDocument, Enabled: enabled}
	if err := h.Repo.Create(r.Context(), c); errors.Is(err, domain.ErrAlreadyExists) {
		writeErr(w, http.StatusConflict, "case already exists")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), body.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toDTO(got))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.CaseID(chi.URLParam(r, "id"))
	c, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "case not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(c))
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.CaseID(chi.URLParam(r, "id"))
	existing, err := h.Repo.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "case not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var body writeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	doc := body.CaseDocument
	doc.ID = id
	if err := h.validate(doc); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	existing.Document = doc
	if body.Enabled != nil {
		existing.Enabled = *body.Enabled
	}
	if err := h.Repo.Save(r.Context(), existing); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(got))
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.CaseID(chi.URLParam(r, "id"))
	if err := h.Repo.Disable(r.Context(), id); errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "case not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(got))
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	id := sharedkernel.CaseID(chi.URLParam(r, "id"))
	if err := h.Repo.Enable(r.Context(), id); errors.Is(err, domain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "case not found")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	got, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDTO(got))
}

func (h *Handler) validate(doc domain.CaseDocument) error {
	if h.Validate == nil {
		return nil
	}
	return h.Validate(doc)
}

func parseListQuery(r *http.Request) (domain.ListQuery, error) {
	q := domain.ListQuery{
		Q:        r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		Tag:      r.URL.Query().Get("tag"),
		MenuKey:  r.URL.Query().Get("menu_key"),
	}
	if v := r.URL.Query().Get("enabled"); v != "" {
		b, err := parseBoolQuery(v)
		if err != nil {
			return q, err
		}
		q.Enabled = &b
	}
	if v := r.URL.Query().Get("created_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, errors.New("invalid created_from")
		}
		q.CreatedFrom = &t
	}
	if v := r.URL.Query().Get("created_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return q, errors.New("invalid created_to")
		}
		q.CreatedTo = &t
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return q, errors.New("invalid limit")
		}
		q.Limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return q, errors.New("invalid offset")
		}
		q.Offset = n
	}
	return q, nil
}

func parseBoolQuery(v string) (bool, error) {
	switch v {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, errors.New("invalid enabled")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
