package studio_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type ids struct {
	mu sync.Mutex
	n  int
}

func (i *ids) next() string {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.n++
	return fmt.Sprintf("id-%d", i.n)
}

func newHandler(t *testing.T) (*studioapi.Handler, *studioapp.BackgroundRunner) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sequence := &ids{}
	executor := studioapp.NewAgentExecutor(studioapp.AgentExecutorOptions{Repo: repo, Blob: blobs, Engine: studioapp.NewMockEngine(), IDs: sequence.next})
	runner := studioapp.NewBackgroundRunner(repo, executor, studioapp.RunnerOptions{Workers: 1})
	service := &studioapp.Service{Repo: repo, IDs: sequence.next, Queue: runner}
	return &studioapi.Handler{
		Repo: repo, Service: service, Runner: runner,
		Approvals: &studioapp.ApprovalService{Repo: repo, Queue: runner, IDs: sequence.next},
		Blob:      blobs,
	}, runner
}

func TestStudioConversationAPICompletesMockWorkflow(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	response := request(t, router, http.MethodPost, "/messages", map[string]any{
		"text":            "为雨夜侦探生成漫画分镜",
		"permission_mode": domain.PermissionFullAccess,
	}, "account-a")
	if response.Code != http.StatusAccepted {
		t.Fatalf("POST /messages status=%d body=%s", response.Code, response.Body.String())
	}
	var created struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Session.ID == "" || created.Run.ID == "" {
		t.Fatalf("created = %#v", created)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response = request(t, router, http.MethodGet, "/runs/"+created.Run.ID, nil, "account-a")
		var run struct {
			Status domain.RunStatus `json:"status"`
		}
		_ = json.Unmarshal(response.Body.Bytes(), &run)
		if run.Status == domain.RunSucceeded {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	response = request(t, router, http.MethodGet, "/sessions/"+created.Session.ID, nil, "account-a")
	if response.Code != http.StatusOK {
		t.Fatalf("GET session status=%d body=%s", response.Code, response.Body.String())
	}
	var detail struct {
		Messages []json.RawMessage `json:"messages"`
		Assets   []json.RawMessage `json:"assets"`
		Flow     struct {
			Nodes []json.RawMessage `json:"nodes"`
			Edges []json.RawMessage `json:"edges"`
		} `json:"flow"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Messages) < 2 || len(detail.Assets) != 2 || len(detail.Flow.Nodes) != 4 || len(detail.Flow.Edges) != 3 {
		t.Fatalf("detail counts: messages=%d assets=%d nodes=%d edges=%d body=%s", len(detail.Messages), len(detail.Assets), len(detail.Flow.Nodes), len(detail.Flow.Edges), response.Body.String())
	}
}

func TestCreateStudioSessionAPI(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	response := request(t, router, http.MethodPost, "/sessions", map[string]any{}, "account-a")
	if response.Code != http.StatusCreated {
		t.Fatalf("POST /sessions status=%d body=%s", response.Code, response.Body.String())
	}
	var session struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	if session.ID == "" || session.Title != domain.DefaultSessionTitle {
		t.Fatalf("session = %#v", session)
	}
}

func TestStudioAPIRejectsCrossAccountRead(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)
	created := request(t, router, http.MethodPost, "/messages", map[string]any{"text": "hello", "permission_mode": domain.PermissionFullAccess}, "account-a")
	var payload struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &payload)

	response := request(t, router, http.MethodGet, "/sessions/"+payload.Session.ID, nil, "account-b")
	if response.Code != http.StatusNotFound {
		t.Fatalf("cross-account status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestStudioEventsResumeAfterCursor(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)
	created := request(t, router, http.MethodPost, "/messages", map[string]any{"text": "hello", "permission_mode": domain.PermissionFullAccess}, "account-a")
	var payload struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &payload)
	time.Sleep(100 * time.Millisecond)

	response := request(t, router, http.MethodGet, "/runs/"+payload.Run.ID+"/events?after=2", nil, "account-a")
	if response.Code != http.StatusOK {
		t.Fatalf("events status=%d body=%s", response.Code, response.Body.String())
	}
	var events []struct {
		Sequence uint64 `json:"sequence"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &events); err != nil {
		t.Fatal(err)
	}
	if len(events) == 0 || events[0].Sequence <= 2 {
		t.Fatalf("events = %#v", events)
	}
}

func TestStudioAGUIStreamsStandardEvents(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	sessionResponse := request(t, router, http.MethodPost, "/sessions", map[string]any{}, "account-a")
	var session struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(sessionResponse.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}

	response := request(t, router, http.MethodPost, "/agui", map[string]any{
		"threadId": session.ID,
		"runId":    "browser-run-1",
		"messages": []map[string]any{{
			"id": "user-message-1", "role": "user", "content": "为雨夜侦探生成漫画分镜",
		}},
		"forwardedProps": map[string]any{
			"runConfig": map[string]any{"permissionMode": domain.PermissionFullAccess},
		},
	}, "account-a")
	if response.Code != http.StatusOK {
		t.Fatalf("POST /agui status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		`"type":"RUN_STARTED"`,
		`"threadId":"` + session.ID + `"`,
		`"type":"TEXT_MESSAGE_CONTENT"`,
		`"messageId":`,
		`"type":"TOOL_CALL_START"`,
		`"toolCallName":"分镜生成"`,
		`"type":"RUN_FINISHED"`,
		`"outcome":{"type":"success"}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("SSE body does not contain %q:\n%s", want, body)
		}
	}
}

func request(t *testing.T, handler http.Handler, method, path string, body any, accountID string) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(setupapi.WithAccount(req.Context(), setupapi.AccountSession{AccountID: accountID, Username: accountID, Role: "admin"}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
