package adminhost

import (
	"net/http"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
)

// Permission is a fine-grained capability an account's role must grant to call
// a restricted endpoint.
type Permission string

const (
	PermAccountManage Permission = "account.manage"
	PermAccountDelete Permission = "account.delete"
)

// PermissionsForRole maps a console role to its granted permissions. Roles are
// pre-set combinations; extra granular combinations can be added here later.
func PermissionsForRole(role string) map[Permission]bool {
	switch role {
	case "admin":
		return map[Permission]bool{PermAccountManage: true, PermAccountDelete: true}
	default:
		// operator / viewer inherit no account-management or delete permissions.
		return map[Permission]bool{}
	}
}

// RequirePermission guards a handler by the authenticated account's role. It is
// expected to run after setup.Gate, which populates the account context.
func RequirePermission(p Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acct, ok := setup.AccountFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// 已认证但缺少账号身份（旧 bootstrap 会话）或角色未授予权限时，
			// 返回 403 而不是 401，避免被前端当作“登录失效”而触发登出跳转。
			if acct.AccountID == "" || !PermissionsForRole(acct.Role)[p] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
