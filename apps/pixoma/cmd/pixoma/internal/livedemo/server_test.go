package livedemo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func startServer(t *testing.T) *httptest.Server {
	t.Helper()
	server, err := New(context.Background(), "127.0.0.1:0", "http://127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	raw := httptest.NewServer(server.Handler())
	t.Cleanup(func() {
		_ = server.Shutdown(context.Background())
		raw.Close()
	})
	return raw
}

func login(t *testing.T, server *httptest.Server) string {
	t.Helper()
	code, body := requestJSON(t, server, http.MethodPost, "/api/v1/setup/login", "", `{"username":"admin","password":"123456"}`)
	if code != http.StatusOK {
		t.Fatalf("login: code=%d body=%v", code, body)
	}
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatal("login token is empty")
	}
	return token
}

func getJSON(t *testing.T, server *httptest.Server, token, path string) (int, map[string]any) {
	t.Helper()
	return requestJSON(t, server, http.MethodGet, path, token, "")
}

func TestDemoServerServesFullAdminDataset(t *testing.T) {
	server := startServer(t)
	token := login(t, server)

	expectations := []struct {
		path     string
		contains string
	}{
		{"/api/v1/edges", "演示节点 A"},
		{"/api/v1/cases", "产品图精修"},
		{"/api/v1/channels", "演示 Telegram"},
		{"/api/v1/topics", "渲染队列"},
		{"/api/v1/tasks", "task-demo-1"},
		{"/api/v1/sessions", "session-demo-1"},
		{"/api/v1/users", "@alice"},
		{"/api/v1/adminusers", "演示管理员"},
		{"/api/v1/stats/tasks/daily?from=2026-08-31&to=2026-08-31", "\"processed\":128"},
		{"/api/v1/routing/attributes", "attributes"},
		{"/api/v1/channels/channel-demo/text-templates", "welcome"},
	}
	for _, expected := range expectations {
		code, body := getJSON(t, server, token, expected.path)
		if code != http.StatusOK {
			t.Fatalf("%s: code=%d body=%v", expected.path, code, body)
		}
		raw, _ := json.Marshal(body)
		if !strings.Contains(string(raw), expected.contains) {
			t.Fatalf("%s: expected %q in %s", expected.path, expected.contains, string(raw))
		}
	}
}

func TestDemoServerReadonlyBoundary(t *testing.T) {
	server := startServer(t)
	token := login(t, server)

	code, body := requestJSON(t, server, http.MethodPost, "/api/v1/cases", token, `{}`)
	if code != http.StatusForbidden || body["code"] != "readonly_live_demo" {
		t.Fatalf("create case: code=%d body=%v", code, body)
	}
	code, body = requestJSON(t, server, http.MethodPut, "/api/v1/setup/settings", token, `{}`)
	if code != http.StatusForbidden || body["code"] != "readonly_live_demo" {
		t.Fatalf("settings: code=%d body=%v", code, body)
	}
	code, body = requestJSON(t, server, http.MethodPost, "/api/v1/edges/edge-demo-1/rotate-token", token, `{}`)
	if code != http.StatusForbidden || body["code"] != "readonly_live_demo" {
		t.Fatalf("rotate token: code=%d body=%v", code, body)
	}
	code, body = requestJSON(t, server, http.MethodPut, "/api/v1/channels/channel-demo/text-templates", token, `{"templates":{}}`)
	if code != http.StatusForbidden || body["code"] != "readonly_live_demo" {
		t.Fatalf("channel text: code=%d body=%v", code, body)
	}
}

func TestDemoServerDoesNotPersistDataDirectoryFiles(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	server := startServer(t)

	code, body := getJSON(t, server, "", "/api/v1/setup/status")
	if code != http.StatusOK || body["live_demo"] != true {
		t.Fatalf("status: code=%d body=%v", code, body)
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("demo mode created persistent files: %v", names)
	}
	for _, forbidden := range []string{"bootstrap.db", "app.db", "blob", "sessions.json"} {
		if _, err := os.Stat(filepath.Join(dataDir, forbidden)); !os.IsNotExist(err) {
			t.Fatalf("forbidden demo file exists: %s", forbidden)
		}
	}
}

func TestDemoServerServesFrontendShell(t *testing.T) {
	server := startServer(t)

	res, err := server.Client().Get(server.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("frontend shell: code=%d", res.StatusCode)
	}
	var body bytes.Buffer
	if _, err := body.ReadFrom(res.Body); err != nil {
		t.Fatal(err)
	}
	if contentType := res.Header.Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("frontend shell content type: %q", contentType)
	}
}
