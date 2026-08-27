package setup_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/setup"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/bootstrap"
)

func TestGate_MediaReachableWithSetupSessionBeforeInit(t *testing.T) {
	boot, creds, err := bootstrap.Open(filepath.Join(t.TempDir(), "bootstrap.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()

	sess := setup.NewSessions("")
	token, err := sess.Issue(creds.Username, false)
	if err != nil {
		t.Fatal(err)
	}

	gate := &setup.Gate{Boot: boot, Sessions: sess}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	for _, tc := range []struct {
		name  string
		path  string
		token string
		want  int
	}{{
		name:  "media upload allowed with setup session",
		path:  "/api/v1/media/",
		token: token,
		want:  http.StatusOK,
	}, {
		name: "media refused without a session",
		path: "/api/v1/media/",
		want: http.StatusUnauthorized,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(""))
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			gate.Middleware(next).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("code = %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
			if tc.want == http.StatusOK && !called {
				t.Fatal("expected next handler to be called")
			}
		})
	}
}
