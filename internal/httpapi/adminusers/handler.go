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

	consoledomain "github.com/Mr9esx/Pixoma/internal/adminusers/domain"
	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/response"
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
			response.Fail(w, apierr.ErrAdminUserListInvalidLimit, "invalid limit")
			return
		}
		q.Limit = n
	}
	users, err := h.Repo.List(r.Context(), q)
	if err != nil {
		response.Fail(w, apierr.ErrAdminUserListListFailed, "list failed")
		return
	}
	out := make([]consoleUserDTO, 0, len(users))
	for _, u := range users {
		if u != nil {
			out = append(out, toDTO(u))
		}
	}
	response.OKStatus(w, http.StatusOK, out)
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
		response.Fail(w, apierr.ErrAdminUserCreateInvalidJSON, "invalid json")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.TrimSpace(body.Email)
	if body.Username == "" {
		response.Fail(w, apierr.ErrAdminUserCreateAccountNameRequired, "账号名不能为空")
		return
	}
	if len(body.Password) < 8 {
		response.Fail(w, apierr.ErrAdminUserCreatePasswordTooShort, "password must be at least 8 characters")
		return
	}
	if body.Email != "" && !validEmail(body.Email) {
		response.Fail(w, apierr.ErrAdminUserCreateInvalidEmail, "invalid email")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Fail(w, apierr.ErrAdminUserCreateHashPasswordFailed, "failed to hash password")
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
	response.OKStatus(w, http.StatusCreated, toDTO(u))
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
		response.Fail(w, apierr.ErrAdminUserCreateInvalidJSON, "invalid json")
		return
	}
	if body.Email != nil && (*body.Email != "" && !validEmail(*body.Email)) {
		response.Fail(w, apierr.ErrAdminUserCreateInvalidEmail, "invalid email")
		return
	}
	if body.Role != nil && !validRole(*body.Role) {
		response.Fail(w, apierr.ErrAdminUserUpdateInvalidRole, "invalid role")
		return
	}
	// Refuse to disable or demote the last enabled Admin.
	if u.Role == consoledomain.RoleAdmin &&
		((body.Enabled != nil && !*body.Enabled) || (body.Role != nil && *body.Role != consoledomain.RoleAdmin)) {
		n, err := h.Repo.CountAdmins(r.Context())
		if err != nil {
			response.Fail(w, apierr.ErrAdminUserUpdateCountAdminsFailed, "count admins failed")
			return
		}
		if n <= 1 {
			response.Fail(w, apierr.ErrAdminUserUpdateLastAdmin, "cannot remove the last admin")
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
			response.Fail(w, apierr.ErrAdminUserCreateHashPasswordFailed, "failed to hash password")
			return
		}
		u.PasswordHash = string(hash)
		u.MustChangePassword = true
	}
	if err := h.Repo.Update(r.Context(), u); err != nil {
		writeUpdateError(w, err)
		return
	}
	response.OKStatus(w, http.StatusOK, toDTO(u))
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
			response.Fail(w, apierr.ErrAdminUserUpdateCountAdminsFailed, "count admins failed")
			return
		}
		if n <= 1 {
			response.Fail(w, apierr.ErrAdminUserDeleteLastAdmin, "cannot delete the last admin")
			return
		}
	}
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		response.Fail(w, apierr.ErrAdminUserDeleteDeleteFailed, "delete failed")
		return
	}
	response.OKStatus(w, http.StatusOK, map[string]bool{"ok": true})
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
		response.Fail(w, apierr.ErrAdminUserNotFound, "user not found")
		return
	}
	response.Fail(w, apierr.ErrAdminUserLoadFailed, "load failed")
}

func writeCreateError(w http.ResponseWriter, err error) {
	if errors.Is(err, consoledomain.ErrDuplicate) {
		response.Fail(w, apierr.ErrAdminUserAlreadyTaken, "username or email already taken")
		return
	}
	response.Fail(w, apierr.ErrAdminUserCreateFailed, "create failed")
}

func writeUpdateError(w http.ResponseWriter, err error) {
	if errors.Is(err, consoledomain.ErrDuplicate) {
		response.Fail(w, apierr.ErrAdminUserAlreadyTaken, "username or email already taken")
		return
	}
	response.Fail(w, apierr.ErrAdminUserUpdateFailed, "update failed")
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
