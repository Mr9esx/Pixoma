//go:build integration

package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// This test requires an already initialized, isolated Pixoma instance with an
// Agent-enabled model. It creates a real Session and makes billable model calls.
func TestStudioLiveAgentCreatesAssetOverWebSocket(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("PIXOMA_STUDIO_SMOKE_URL"), "/")
	username := os.Getenv("PIXOMA_STUDIO_SMOKE_USER")
	password := os.Getenv("PIXOMA_STUDIO_SMOKE_PASSWORD")
	modelID := os.Getenv("PIXOMA_STUDIO_SMOKE_MODEL_ID")
	if baseURL == "" || username == "" || password == "" || modelID == "" {
		t.Skip("set PIXOMA_STUDIO_SMOKE_URL, USER, PASSWORD and MODEL_ID to run against an isolated Pixoma instance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 10 * time.Second}
	var login struct {
		Token string `json:"token"`
	}
	studioSmokeRequest(t, ctx, client, baseURL, "", http.MethodPost, "/api/v1/setup/login", map[string]string{
		"username": username, "password": password,
	}, &login)
	require.NotEmpty(t, login.Token)
	var session struct {
		ID string `json:"id"`
	}
	studioSmokeRequest(t, ctx, client, baseURL, login.Token, http.MethodPost, "/api/v1/studio/sessions", map[string]any{}, &session)
	require.NotEmpty(t, session.ID)

	wsURL, err := url.Parse(baseURL)
	require.NoError(t, err)
	wsURL.Scheme = map[bool]string{true: "wss", false: "ws"}[wsURL.Scheme == "https"]
	wsURL.Path = "/api/v1/studio/agui/ws"
	conn, response, err := (&websocket.Dialer{HandshakeTimeout: 10 * time.Second}).DialContext(ctx, wsURL.String(), http.Header{
		"Authorization": []string{"Bearer " + login.Token},
	})
	if response != nil && response.Body != nil {
		defer response.Body.Close()
	}
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(80*time.Second)))
	require.NoError(t, conn.WriteJSON(map[string]any{
		"threadId": session.ID,
		"runId":    "studio-live-smoke",
		"messages": []map[string]any{{
			"id": "studio-live-user", "role": "user",
			"content": "请调用 create_text_asset 工具，在当前对话创建名为 story.md 的 Markdown 资产，内容以「# 雨夜侦探」开头，并简短告诉我已创建。",
		}},
		"forwardedProps": map[string]any{"runConfig": map[string]any{
			"modelConfigId": modelID, "permissionMode": "full_access",
		}},
	}))
	seenToolCall := false
	for {
		var event struct {
			Type string `json:"type"`
		}
		require.NoError(t, conn.ReadJSON(&event))
		if event.Type == "TOOL_CALL_START" {
			seenToolCall = true
		}
		if event.Type == "RUN_ERROR" {
			t.Fatal("Studio Agent run failed; inspect its trace in the isolated instance")
		}
		if event.Type == "RUN_FINISHED" {
			break
		}
	}
	require.True(t, seenToolCall, "the online Agent did not call the asset creation tool")

	var detail struct {
		Session struct {
			LatestRun struct {
				Status string `json:"status"`
			} `json:"latest_run"`
		} `json:"session"`
		Assets []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Versions []struct {
				ID         string `json:"id"`
				ContentURL string `json:"content_url"`
			} `json:"versions"`
		} `json:"assets"`
		Flow struct {
			Nodes []struct {
				AssetID        string `json:"asset_id"`
				AssetVersionID string `json:"asset_version_id"`
			} `json:"nodes"`
		} `json:"flow"`
	}
	studioSmokeRequest(t, ctx, client, baseURL, login.Token, http.MethodGet, "/api/v1/studio/sessions/"+session.ID, nil, &detail)
	require.Equal(t, "succeeded", detail.Session.LatestRun.Status)
	var assetID, versionID, contentURL string
	for _, asset := range detail.Assets {
		if asset.Name == "story.md" && len(asset.Versions) > 0 {
			assetID, versionID, contentURL = asset.ID, asset.Versions[0].ID, asset.Versions[0].ContentURL
			break
		}
	}
	require.NotEmpty(t, assetID, "the model-created Markdown asset was not persisted")
	pinned := false
	for _, node := range detail.Flow.Nodes {
		if node.AssetID == assetID && node.AssetVersionID == versionID {
			pinned = true
		}
	}
	require.True(t, pinned, "the asset version was not added to the Session Flow")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+contentURL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	contentResponse, err := client.Do(req)
	require.NoError(t, err)
	defer contentResponse.Body.Close()
	require.Equal(t, http.StatusOK, contentResponse.StatusCode)
	content, err := io.ReadAll(io.LimitReader(contentResponse.Body, 1<<20))
	require.NoError(t, err)
	require.Contains(t, string(content), "# 雨夜侦探")
}

func studioSmokeRequest(t *testing.T, ctx context.Context, client *http.Client, baseURL, token, method, path string, input, output any) {
	t.Helper()
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		require.NoError(t, err)
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	require.NoError(t, err)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	require.True(t, response.StatusCode >= 200 && response.StatusCode < 300, "%s %s returned HTTP %d", method, path, response.StatusCode)
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&envelope))
	require.NotEmpty(t, envelope.Data, fmt.Sprintf("%s %s returned no data", method, path))
	require.NoError(t, json.Unmarshal(envelope.Data, output))
}
