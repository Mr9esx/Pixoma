package studio_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	catalogdomain "github.com/Mr9esx/Pixoma/internal/cases/domain"
	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type ids struct {
	mu sync.Mutex
	n  int
}

type workflowCatalog struct{ cases []*catalogdomain.Case }

type modelConnectionTester struct{}

func (modelConnectionTester) Test(_ context.Context, _ domain.ResolvedModelConfig) error {
	return nil
}

func (c workflowCatalog) List(_ context.Context, _ catalogdomain.ListQuery) ([]*catalogdomain.Case, error) {
	return c.cases, nil
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
		Approvals:    &studioapp.ApprovalService{Repo: repo, Queue: runner, IDs: sequence.next},
		Models:       &studioapp.ModelConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: sequence.next, Tester: modelConnectionTester{}},
		Capabilities: &studioapp.CapabilityConfigService{Repo: repo, EncryptionKey: []byte(strings.Repeat("k", 32)), IDs: sequence.next, WorkflowCatalog: workflowCatalog{cases: []*catalogdomain.Case{{Document: catalogdomain.CaseDocument{ID: sharedkernel.CaseID(1), Name: "漫画生成", Description: "生成分镜"}, Enabled: true}}}},
		Blob:         blobs,
	}, runner
}

func TestStudioModelConnectionTestAPIIsAccountScoped(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	created := request(t, router, http.MethodPost, "/models", map[string]any{
		"name": "Ark DeepSeek", "protocol": "openai_chat_compatible", "base_url": "https://ark.example.test/v3",
		"model": "deepseek-v4-flash", "api_key": "temporary-secret", "enabled": true, "agent_enabled": true,
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("POST /models = %d %s", created.Code, created.Body.String())
	}
	var model struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &model); err != nil || model.ID == "" {
		t.Fatalf("created model = %s, err=%v", created.Body.String(), err)
	}

	tested := request(t, router, http.MethodPost, "/models/"+model.ID+"/test", nil, "account-a")
	if tested.Code != http.StatusOK || !strings.Contains(tested.Body.String(), `"success":true`) || strings.Contains(tested.Body.String(), "temporary-secret") {
		t.Fatalf("POST /models/{id}/test = %d %s", tested.Code, tested.Body.String())
	}
	foreign := request(t, router, http.MethodPost, "/models/"+model.ID+"/test", nil, "account-b")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign model test = %d %s", foreign.Code, foreign.Body.String())
	}
}

func TestStudioWorkflowAvailabilityAPIIsAccountScoped(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	list := request(t, router, http.MethodGet, "/workflows", nil, "account-a")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "\"agent_enabled\":false") {
		t.Fatalf("GET /workflows = %d %s", list.Code, list.Body.String())
	}
	updated := request(t, router, http.MethodPatch, "/workflows/1", map[string]any{"agent_enabled": true}, "account-a")
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), "\"agent_enabled\":true") {
		t.Fatalf("PATCH /workflows/1 = %d %s", updated.Code, updated.Body.String())
	}
	foreign := request(t, router, http.MethodGet, "/workflows", nil, "account-b")
	if foreign.Code != http.StatusOK || strings.Contains(foreign.Body.String(), "\"agent_enabled\":true") {
		t.Fatalf("GET /workflows as another account = %d %s", foreign.Code, foreign.Body.String())
	}
}

func TestStudioSkillConfigAPIIsAccountScoped(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	created := request(t, router, http.MethodPost, "/skills", map[string]any{
		"name": "漫画分镜", "description": "把故事拆成镜头", "prompt": "先输出镜头表", "enabled": true,
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("POST /skills status=%d body=%s", created.Code, created.Body.String())
	}
	if strings.Contains(created.Body.String(), "account-a") {
		t.Fatalf("skill response exposes ownership: %s", created.Body.String())
	}

	foreign := request(t, router, http.MethodGet, "/skills", nil, "account-b")
	if foreign.Code != http.StatusOK || strings.Contains(foreign.Body.String(), "漫画分镜") {
		t.Fatalf("GET /skills as another account = %d %s", foreign.Code, foreign.Body.String())
	}
}

func TestStudioCapabilityConfigAPIUpdatesEnabledState(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	skill := request(t, router, http.MethodPost, "/skills", map[string]any{
		"name": "漫画分镜", "prompt": "先输出镜头表", "enabled": true,
	}, "account-a")
	var createdSkill struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(skill.Body.Bytes(), &createdSkill); err != nil || createdSkill.ID == "" {
		t.Fatalf("created skill = %s, err=%v", skill.Body.String(), err)
	}
	updatedSkill := request(t, router, http.MethodPatch, "/skills/"+createdSkill.ID, map[string]any{
		"name": "漫画分镜", "prompt": "先输出镜头表", "enabled": false,
	}, "account-a")
	if updatedSkill.Code != http.StatusOK || !strings.Contains(updatedSkill.Body.String(), "\"enabled\":false") {
		t.Fatalf("PATCH skill = %d %s", updatedSkill.Code, updatedSkill.Body.String())
	}

	connector := request(t, router, http.MethodPost, "/connectors", map[string]any{
		"name": "Reference", "url": "https://mcp.example.com", "credential": "secret", "enabled": true, "policy": "approval",
	}, "account-a")
	var createdConnector struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(connector.Body.Bytes(), &createdConnector); err != nil || createdConnector.ID == "" {
		t.Fatalf("created connector = %s, err=%v", connector.Body.String(), err)
	}
	updatedConnector := request(t, router, http.MethodPatch, "/connectors/"+createdConnector.ID, map[string]any{
		"name": "Reference", "url": "https://mcp.example.com", "enabled": false, "policy": "forbidden",
	}, "account-a")
	if updatedConnector.Code != http.StatusOK || !strings.Contains(updatedConnector.Body.String(), "\"enabled\":false") || strings.Contains(updatedConnector.Body.String(), "secret") {
		t.Fatalf("PATCH connector = %d %s", updatedConnector.Code, updatedConnector.Body.String())
	}
}

func TestStudioConnectorConfigAPIStoresMaskedAccountScopedConfiguration(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	created := request(t, router, http.MethodPost, "/connectors", map[string]any{
		"name": "Reference tools", "url": "https://mcp.example.com", "credential": "secret-token", "enabled": true, "policy": "approval",
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("POST /connectors status=%d body=%s", created.Code, created.Body.String())
	}
	if strings.Contains(created.Body.String(), "secret-token") || !strings.Contains(created.Body.String(), "••••••••") {
		t.Fatalf("connector response must mask its credential: %s", created.Body.String())
	}

	foreign := request(t, router, http.MethodGet, "/connectors", nil, "account-b")
	if foreign.Code != http.StatusOK || strings.Contains(foreign.Body.String(), "Reference tools") {
		t.Fatalf("GET /connectors as another account = %d %s", foreign.Code, foreign.Body.String())
	}
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

func TestStudioAGUIStoresSelectedSkillIDsOnRun(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	skill, err := handler.Capabilities.CreateSkill(context.Background(), studioapp.CreateSkillInput{
		AccountID: "account-a", Name: "漫画分镜", Prompt: "先输出镜头表", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
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
		"threadId": session.ID, "runId": "browser-run-skills", "messages": []map[string]any{{"id": "user-message", "role": "user", "content": "写分镜"}},
		"forwardedProps": map[string]any{"runConfig": map[string]any{"permissionMode": domain.PermissionFullAccess, "selectedSkillIds": []string{skill.ID}}},
	}, "account-a")
	if response.Code != http.StatusOK {
		t.Fatalf("POST /agui status=%d body=%s", response.Code, response.Body.String())
	}
	var started struct {
		Metadata struct {
			StudioRunID string `json:"studioRunId"`
		} `json:"metadata"`
	}
	for _, chunk := range strings.Split(response.Body.String(), "\n\n") {
		if !strings.Contains(chunk, "RUN_STARTED") {
			continue
		}
		if index := strings.Index(chunk, "data: "); index >= 0 {
			_ = json.Unmarshal([]byte(chunk[index+6:]), &started)
		}
	}
	run, err := handler.Repo.GetRun(context.Background(), "account-a", started.Metadata.StudioRunID)
	if err != nil || len(run.SkillIDs) != 1 || run.SkillIDs[0] != skill.ID {
		t.Fatalf("run = %#v, err = %v", run, err)
	}
}

func TestStudioAGUIStoresSelectedAssetVersionOnRun(t *testing.T) {
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
	assetResponse := request(t, router, http.MethodPost, "/assets/text", map[string]any{
		"session_id": session.ID, "name": "故事.md", "content": "# 雨夜侦探",
	}, "account-a")
	var asset struct {
		ID       string `json:"id"`
		Versions []struct {
			ID string `json:"id"`
		} `json:"versions"`
	}
	if assetResponse.Code != http.StatusCreated || json.Unmarshal(assetResponse.Body.Bytes(), &asset) != nil || len(asset.Versions) != 1 {
		t.Fatalf("created asset = status=%d body=%s", assetResponse.Code, assetResponse.Body.String())
	}
	response := request(t, router, http.MethodPost, "/agui", map[string]any{
		"threadId": session.ID, "runId": "browser-run-assets", "messages": []map[string]any{{"id": "user-message", "role": "user", "content": "根据故事生成分镜"}},
		"forwardedProps": map[string]any{"runConfig": map[string]any{"permissionMode": domain.PermissionFullAccess, "selectedAssets": []map[string]string{{"assetId": asset.ID, "assetVersionId": asset.Versions[0].ID}}}},
	}, "account-a")
	if response.Code != http.StatusOK {
		t.Fatalf("POST /agui status=%d body=%s", response.Code, response.Body.String())
	}
	studioRunID := studioRunIDFromSSE(t, response.Body.String())
	run, err := handler.Repo.GetRun(context.Background(), "account-a", studioRunID)
	if err != nil || len(run.AssetReferences) != 1 || run.AssetReferences[0].AssetID != asset.ID || run.AssetReferences[0].AssetVersionID != asset.Versions[0].ID {
		t.Fatalf("run = %#v, err = %v", run, err)
	}
}

func studioRunIDFromSSE(t *testing.T, body string) string {
	t.Helper()
	var started struct {
		Metadata struct {
			StudioRunID string `json:"studioRunId"`
		} `json:"metadata"`
	}
	for _, chunk := range strings.Split(body, "\n\n") {
		if !strings.Contains(chunk, "RUN_STARTED") {
			continue
		}
		if index := strings.Index(chunk, "data: "); index >= 0 {
			_ = json.Unmarshal([]byte(chunk[index+6:]), &started)
		}
	}
	if started.Metadata.StudioRunID == "" {
		t.Fatalf("RUN_STARTED missing studio run id: %s", body)
	}
	return started.Metadata.StudioRunID
}

func TestStudioManualAssetAndFlowPositionAPIs(t *testing.T) {
	handler, runner := newHandler(t)
	t.Cleanup(runner.Close)
	router := chi.NewRouter()
	handler.Mount(router)

	created := request(t, router, http.MethodPost, "/messages", map[string]any{
		"text": "为雨夜侦探生成漫画分镜", "permission_mode": domain.PermissionFullAccess,
	}, "account-a")
	var turn struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &turn); err != nil {
		t.Fatal(err)
	}
	waitForRun(t, router, turn.Run.ID, "account-a")

	assetResponse := request(t, router, http.MethodPost, "/assets/text", map[string]any{
		"session_id": turn.Session.ID, "name": "角色设定.md", "content": "# 主角\n雨夜侦探。",
	}, "account-a")
	if assetResponse.Code != http.StatusCreated {
		t.Fatalf("POST manual asset status=%d body=%s", assetResponse.Code, assetResponse.Body.String())
	}
	var asset struct {
		ID     string             `json:"id"`
		Origin domain.AssetOrigin `json:"origin"`
	}
	if err := json.Unmarshal(assetResponse.Body.Bytes(), &asset); err != nil {
		t.Fatal(err)
	}
	if asset.ID == "" || asset.Origin != domain.AssetOriginUser {
		t.Fatalf("asset=%#v", asset)
	}
	updatedAsset := request(t, router, http.MethodPatch, "/assets/"+asset.ID+"/text", map[string]any{
		"content": "# 主角\n雨夜侦探，携带旧案卷宗。",
	}, "account-a")
	if updatedAsset.Code != http.StatusOK || !strings.Contains(updatedAsset.Body.String(), `"current_version":2`) {
		t.Fatalf("PATCH manual asset status=%d body=%s", updatedAsset.Code, updatedAsset.Body.String())
	}
	var updated struct {
		Versions []struct {
			ID         string `json:"id"`
			ContentURL string `json:"content_url"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(updatedAsset.Body.Bytes(), &updated); err != nil || len(updated.Versions) != 2 {
		t.Fatalf("updated asset = %s, err=%v", updatedAsset.Body.String(), err)
	}
	oldContent := request(t, router, http.MethodGet, strings.TrimPrefix(updated.Versions[0].ContentURL, "/api/v1/studio"), nil, "account-a")
	if oldContent.Code != http.StatusOK || !strings.Contains(oldContent.Body.String(), "雨夜侦探。") || strings.Contains(oldContent.Body.String(), "旧案卷宗") {
		t.Fatalf("old asset content status=%d body=%s", oldContent.Code, oldContent.Body.String())
	}
	saved := request(t, router, http.MethodPost, "/assets/"+asset.ID+"/save-to-library", nil, "account-a")
	if saved.Code != http.StatusNoContent {
		t.Fatalf("save asset status=%d body=%s", saved.Code, saved.Body.String())
	}
	thirdVersion := request(t, router, http.MethodPatch, "/assets/"+asset.ID+"/text", map[string]any{
		"content": "# 主角\n雨夜侦探，携带旧案卷宗，并决定重查旧案。",
	}, "account-a")
	if thirdVersion.Code != http.StatusOK || !strings.Contains(thirdVersion.Body.String(), `"current_version":3`) {
		t.Fatalf("PATCH third asset version status=%d body=%s", thirdVersion.Code, thirdVersion.Body.String())
	}
	library := request(t, router, http.MethodGet, "/library/assets", nil, "account-a")
	var libraryAssets []struct {
		CurrentVersion int `json:"current_version"`
		Versions       []struct {
			ContentURL string `json:"content_url"`
		} `json:"versions"`
	}
	if library.Code != http.StatusOK || json.Unmarshal(library.Body.Bytes(), &libraryAssets) != nil || len(libraryAssets) != 1 || libraryAssets[0].CurrentVersion != 2 || len(libraryAssets[0].Versions) != 1 {
		t.Fatalf("library asset = status=%d body=%s", library.Code, library.Body.String())
	}
	libraryContent := request(t, router, http.MethodGet, strings.TrimPrefix(libraryAssets[0].Versions[0].ContentURL, "/api/v1/studio"), nil, "account-a")
	if libraryContent.Code != http.StatusOK || !strings.Contains(libraryContent.Body.String(), "旧案卷宗") || strings.Contains(libraryContent.Body.String(), "重查旧案") {
		t.Fatalf("library content status=%d body=%s", libraryContent.Code, libraryContent.Body.String())
	}

	detail := request(t, router, http.MethodGet, "/sessions/"+turn.Session.ID, nil, "account-a")
	var payload struct {
		Flow struct {
			Nodes []struct {
				ID string `json:"id"`
			} `json:"nodes"`
		} `json:"flow"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Flow.Nodes) == 0 {
		t.Fatal("expected mock flow nodes")
	}
	flowResponse := request(t, router, http.MethodPatch, "/sessions/"+turn.Session.ID+"/flow", map[string]any{
		"nodes": []map[string]any{{"id": payload.Flow.Nodes[0].ID, "position": map[string]float64{"x": 480, "y": 240}, "sort_order": 99}},
	}, "account-a")
	if flowResponse.Code != http.StatusNoContent {
		t.Fatalf("PATCH flow status=%d body=%s", flowResponse.Code, flowResponse.Body.String())
	}

	detail = request(t, router, http.MethodGet, "/sessions/"+turn.Session.ID, nil, "account-a")
	if !strings.Contains(detail.Body.String(), `"x":480`) || !strings.Contains(detail.Body.String(), `"sort_order":99`) {
		t.Fatalf("flow was not persisted: %s", detail.Body.String())
	}

	createdNode := request(t, router, http.MethodPost, "/sessions/"+turn.Session.ID+"/flow/nodes", map[string]any{
		"type": "plan", "title": "确认本话色彩脚本", "body": "根据角色设定补充色彩与情绪。",
		"position": map[string]float64{"x": 720, "y": 240},
	}, "account-a")
	if createdNode.Code != http.StatusCreated {
		t.Fatalf("POST flow node status=%d body=%s", createdNode.Code, createdNode.Body.String())
	}
	var manualNode struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createdNode.Body.Bytes(), &manualNode); err != nil || manualNode.ID == "" {
		t.Fatalf("manual node = %#v, err=%v", manualNode, err)
	}

	createdEdge := request(t, router, http.MethodPost, "/sessions/"+turn.Session.ID+"/flow/edges", map[string]any{
		"source": payload.Flow.Nodes[0].ID, "target": manualNode.ID, "label": "下一步",
	}, "account-a")
	if createdEdge.Code != http.StatusCreated {
		t.Fatalf("POST flow edge status=%d body=%s", createdEdge.Code, createdEdge.Body.String())
	}

	removedNode := request(t, router, http.MethodDelete, "/sessions/"+turn.Session.ID+"/flow/nodes/"+manualNode.ID, nil, "account-a")
	if removedNode.Code != http.StatusNoContent {
		t.Fatalf("DELETE flow node status=%d body=%s", removedNode.Code, removedNode.Body.String())
	}
	detail = request(t, router, http.MethodGet, "/sessions/"+turn.Session.ID, nil, "account-a")
	if strings.Contains(detail.Body.String(), "确认本话色彩脚本") || strings.Contains(detail.Body.String(), "下一步") {
		t.Fatalf("deleted flow graph elements remain: %s", detail.Body.String())
	}
}

func waitForRun(t *testing.T, router http.Handler, runID, accountID string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		response := request(t, router, http.MethodGet, "/runs/"+runID, nil, accountID)
		var run struct {
			Status domain.RunStatus `json:"status"`
		}
		_ = json.Unmarshal(response.Body.Bytes(), &run)
		if run.Status == domain.RunSucceeded {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("mock run did not complete")
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
