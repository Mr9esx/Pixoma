package adminusers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	consoledomain "github.com/mr9esx/comfyui_tgbot/internal/consoleuser/domain"
)

// Handler serves console user administration under /api/v1/adminusers.
// All routes require PermAccountManage (admin only); enforced by the caller.
type Handler struct {
	Repo consoledomain.Repository
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Patch("/{id}", h.patch)
	r.Delete("/{id}", h.delete)
}

// consoleUserDTO excludes password material by construction.
type consoleUserDTO struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	Email              string    `json:"email"`
	Nickname           string    `json:"nickname"`
	AvatarURL          string    `json:"avatar_url"`
	Role               string    `json:"role"`
	Enabled            bool      `json:"enabled"`
	MustChangePassword bool      `json:"must_change_password"`
	LastLoginAt        time.Time `json:"last_login_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func toDTO(u *consoledomain.ConsoleUser) consoleUserDTO {
	return consoleUserDTO{
		ID:                 u.ID,
		Username:           u.Username,
		Email:              u.Email,
		Nickname:           u.Nickname,
		AvatarURL:          u.AvatarURL,
		Role:               u.Role,
		Enabled:            u.Enabled,
		MustChangePassword: u.MustChangePassword,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := consoledomain.ListQuery{Q: r.URL.Query().Get("q")}
	if v := r.URL.Query().Get("limit"); v != "" {
		var n int
		if _, err := scanInt(v, &n); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid limit")
			return
		}
		q.Limit = n
	}
	users, err := h.Repo.List(r.Context(), q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]consoleUserDTO, 0, len(users))
	for _, u := range users {
		if u != nil {
			out = append(out, toDTO(u))
		}
	}
	writeJSON(w, http.StatusOK, out)
}

type createRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body createRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.TrimSpace(body.Email)
	if body.Username == "" {
		writeErr(w, http.StatusBadRequest, "账号名不能为空")
		return
	}
	if len(body.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if body.Email != "" && !validEmail(body.Email) {
		writeErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	u := &consoledomain.ConsoleUser{
		Username:     body.Username,
		Email:        body.Email,
		Nickname:     body.Nickname,
		Role:         consoledomain.RoleViewer,
		Enabled:      true,
		PasswordHash: string(hash),
	}
	if err := h.Repo.Create(r.Context(), u); err != nil {
		writeCreateError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDTO(u))
}

type patchRequest struct {
	Email      *string `json:"email"`
	Nickname   *string `json:"nickname"`
	Role       *string `json:"role"`
	Enabled    *bool   `json:"enabled"`
	Password   *string `json:"password"`
	Reactivate *bool   `json:"reactivate"`
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		writeUserErr(w, err)
		return
	}
	var body patchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Email != nil && (*body.Email != "" && !validEmail(*body.Email)) {
		writeErr(w, http.StatusBadRequest, "invalid email")
		return
	}
	if body.Role != nil && !validRole(*body.Role) {
		writeErr(w, http.StatusBadRequest, "invalid role")
		return
	}
	// Refuse to disable or demote the last enabled Admin.
	if u.Role == consoledomain.RoleAdmin &&
		((body.Enabled != nil && !*body.Enabled) || (body.Role != nil && *body.Role != consoledomain.RoleAdmin)) {
		n, err := h.Repo.CountAdmins(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "count admins failed")
			return
		}
		if n <= 1 {
			writeErr(w, http.StatusConflict, "cannot remove the last admin")
			return
		}
	}
	if body.Email != nil {
		u.Email = strings.TrimSpace(*body.Email)
	}
	if body.Nickname != nil {
		u.Nickname = *body.Nickname
	}
	if body.Role != nil {
		u.Role = *body.Role
	}
	if body.Enabled != nil {
		u.Enabled = *body.Enabled
	}
	if body.Password != nil && len(*body.Password) >= 8 {
		hash, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		u.PasswordHash = string(hash)
		u.MustChangePassword = true
	}
	if err := h.Repo.Update(r.Context(), u); err != nil {
		writeUpdateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(u))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		writeUserErr(w, err)
		return
	}
	if u.Role == consoledomain.RoleAdmin {
		n, err := h.Repo.CountAdmins(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "count admins failed")
			return
		}
		if n <= 1 {
			writeErr(w, http.StatusConflict, "cannot delete the last admin")
			return
		}
	}
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, "delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func validRole(role string) bool {
	switch role {
	case consoledomain.RoleAdmin, consoledomain.RoleOperator, consoledomain.RoleViewer:
		return true
	}
	return false
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func writeUserErr(w http.ResponseWriter, err error) {
	if errors.Is(err, consoledomain.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, "load failed")
}

func writeCreateError(w http.ResponseWriter, err error) {
	if errors.Is(err, consoledomain.ErrDuplicate) {
		writeErr(w, http.StatusConflict, "username or email already taken")
		return
	}
	writeErr(w, http.StatusInternalServerError, "create failed")
}

func writeUpdateError(w http.ResponseWriter, err error) {
	if errors.Is(err, consoledomain.ErrDuplicate) {
		writeErr(w, http.StatusConflict, "username or email already taken")
		return
	}
	writeErr(w, http.StatusInternalServerError, "update failed")
}

func scanInt(s string, n *int) (int, error) {
	i := 0
	for _, c := range strings.TrimSpace(s) {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid int")
		}
		i = i*10 + int(c-'0')
	}
	*n = i
	return i, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
