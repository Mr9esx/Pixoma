package livedemo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	auth := NewAuthStore()
	router := chi.NewRouter()
	router.Route("/api/v1/setup", func(api chi.Router) {
		Mount(api, auth)
	})
	router.Group(func(resource chi.Router) {
		resource.Use(auth.Handler)
		resource.Use(Readonly)
		resource.Get("/api/v1/cases", writeOK)
		resource.Post("/api/v1/cases", writeOK)
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func writeOK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func requestJSON(t *testing.T, server *httptest.Server, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var raw any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{}
	if array, ok := raw.([]any); ok {
		payload["data"] = array
	} else if object, ok := raw.(map[string]any); ok {
		payload = object
	}
	return res.StatusCode, payload
}

func requestJSONWithCookie(t *testing.T, server *httptest.Server, method, path, body, cookie string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "pixoma_session", Value: cookie})
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, payload
}

func loginWithCookie(t *testing.T, server *httptest.Server) (string, string, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/setup/login", bytes.NewBufferString(`{"username":"admin","password":"123456"}`))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", "", &statusError{code: res.StatusCode}
	}
	token, _ := payload["token"].(string)
	cookie := ""
	for _, candidate := range res.Cookies() {
		if candidate.Name == "pixoma_session" {
			cookie = candidate.Value
			break
		}
	}
	return token, cookie, nil
}

type statusError struct {
	code int
}

func (e *statusError) Error() string {
	return fmt.Sprintf("unexpected status: %d", e.code)
}

func TestSetupStatusAndLogin(t *testing.T) {
	server := newTestServer(t)

	code, body := requestJSON(t, server, http.MethodGet, "/api/v1/setup/status", "", "")
	if code != http.StatusOK || body["live_demo"] != true || body["authenticated"] != false {
		t.Fatalf("status: code=%d body=%v", code, body)
	}

	code, body = requestJSON(t, server, http.MethodPost, "/api/v1/setup/login", "", `{"username":"wrong","password":"wrong"}`)
	if code != http.StatusUnauthorized {
		t.Fatalf("invalid login: code=%d body=%v", code, body)
	}

	code, body = requestJSON(t, server, http.MethodPost, "/api/v1/setup/login", "", `{"username":"admin","password":"123456"}`)
	if code != http.StatusOK || body["ok"] != true || body["initialized"] != true {
		t.Fatalf("login: code=%d body=%v", code, body)
	}
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatal("login did not issue token")
	}
	token, cookie, err := loginWithCookie(t, server)
	if err != nil {
		t.Fatal(err)
	}
	if cookie == "" {
		t.Fatal("login did not set session cookie")
	}
	code, body = requestJSONWithCookie(t, server, http.MethodGet, "/api/v1/setup/status", "", cookie)
	if code != http.StatusOK || body["authenticated"] != true {
		t.Fatalf("cookie status: code=%d body=%v", code, body)
	}

	code, body = requestJSON(t, server, http.MethodGet, "/api/v1/setup/me", token, "")
	if code != http.StatusOK || body["live_demo"] != true || body["username"] != "admin" {
		t.Fatalf("me: code=%d body=%v", code, body)
	}

	code, body = requestJSON(t, server, http.MethodPost, "/api/v1/setup/logout", token, "")
	if code != http.StatusOK || body["ok"] != true {
		t.Fatalf("logout: code=%d body=%v", code, body)
	}
}

func TestReadonlyAndAuthentication(t *testing.T) {
	server := newTestServer(t)

	code, body := requestJSON(t, server, http.MethodGet, "/api/v1/cases", "", "")
	if code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: code=%d body=%v", code, body)
	}

	_, loginBody := requestJSON(t, server, http.MethodPost, "/api/v1/setup/login", "", `{"username":"admin","password":"123456"}`)
	token, _ := loginBody["token"].(string)

	code, body = requestJSON(t, server, http.MethodGet, "/api/v1/cases", token, "")
	if code != http.StatusOK || body["ok"] != true {
		t.Fatalf("read: code=%d body=%v", code, body)
	}

	code, body = requestJSON(t, server, http.MethodPost, "/api/v1/cases", token, `{"name":"new"}`)
	if code != http.StatusForbidden || body["code"] != "readonly_live_demo" {
		t.Fatalf("mutate: code=%d body=%v", code, body)
	}

	code, body = requestJSON(t, server, http.MethodPost, "/api/v1/setup/password", token, `{"new_password":"new-password"}`)
	if code != http.StatusForbidden || body["code"] != "readonly_live_demo" {
		t.Fatalf("password: code=%d body=%v", code, body)
	}
}
