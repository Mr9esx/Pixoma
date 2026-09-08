package adminhost

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/httpapi/setup"
)

func TestPermissionsForRole_Matrix(t *testing.T) {
	if !PermissionsForRole("admin")[PermAccountManage] || !PermissionsForRole("admin")[PermAccountDelete] {
		t.Fatal("admin must hold account permissions")
	}
	for _, r := range []string{"operator", "viewer", ""} {
		if PermissionsForRole(r)[PermAccountManage] || PermissionsForRole(r)[PermAccountDelete] {
			t.Fatalf("role %q must not hold account permissions", r)
		}
	}
}

func TestRequirePermission_Unauthenticated(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	rec := httptest.NewRecorder()
	RequirePermission(PermAccountManage)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequirePermission_ViewerForbidden(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(setup.WithAccount(req.Context(), setup.AccountSession{AccountID: "a", Role: "viewer"}))
	rec := httptest.NewRecorder()
	RequirePermission(PermAccountManage)(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequirePermission_AuthenticatedNoAccountForbidden(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(setup.WithAccount(req.Context(), setup.AccountSession{AccountID: "", Role: ""}))
	rec := httptest.NewRecorder()
	RequirePermission(PermAccountManage)(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequirePermission_AdminAllowed(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(setup.WithAccount(req.Context(), setup.AccountSession{AccountID: "a", Role: "admin"}))
	rec := httptest.NewRecorder()
	RequirePermission(PermAccountManage)(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
