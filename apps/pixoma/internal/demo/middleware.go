package demo

import (
	"encoding/json"
	"net/http"
	"strings"
)

func Readonly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/setup/login", "/api/v1/setup/logout":
			next.ServeHTTP(w, r)
			return
		}
		if isSensitiveDemoPath(r.URL.Path) {
			writeReadonly(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
		default:
			writeReadonly(w, r)
		}
	})
}

func isSensitiveDemoPath(path string) bool {
	sensitive := []string{
		"/api/v1/setup/password",
		"/api/v1/setup/profile",
		"/api/v1/setup/database",
		"/api/v1/setup/draft",
		"/api/v1/setup/blob-test",
		"/api/v1/setup/finalize",
		"/api/v1/setup/settings",
		"/api/v1/auth/register",
		"/api/v1/auth/registration",
	}
	for _, prefix := range sensitive {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func writeReadonly(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusForbidden, "Live Demo 为只读模式，不能修改数据。", "readonly_live_demo")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, map[string]string{"error": message, "code": code})
}
