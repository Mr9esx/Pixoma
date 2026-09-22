package mcpconnector_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/mcpconnector"
)

func discoveredTool(name, description string) domain.MCPTool {
	return domain.MCPTool{
		Name: name, Description: description,
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
	}
}

func TestProberDiscoversStreamableHTTPToolsWithBearerCredential(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
		return nil, struct{}{}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer connector-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer endpoint.Close()

	tools, err := (mcpconnector.Prober{HTTPClient: endpoint.Client()}).Probe(context.Background(), endpoint.URL, "connector-secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "search_reference" || tools[0].Description != "Search reference material" {
		t.Fatalf("tools = %#v", tools)
	}
	require.True(t, json.Valid(tools[0].InputSchema))
	require.Contains(t, string(tools[0].InputSchema), `"type":"object"`)
}

func TestRuntimeToolsUsePersistedSnapshotsWithoutEndpointProbe(t *testing.T) {
	t.Parallel()
	tools, err := mcpconnector.NewRuntimeTools(context.Background(), []studioapp.ResolvedMCPConnector{{
		ID: "connector-01", Name: "Reference", URL: "http://127.0.0.1:1/mcp", Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
		Tools: []domain.MCPTool{discoveredTool("search_reference", "Persisted description")},
	}}, mcpconnector.ToolAccess{})
	require.NoError(t, err)
	require.Len(t, tools, 1)
	info, err := tools[0].Info(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Persisted description", info.Desc)
}

func TestRuntimeToolsInvokeDiscoveredMCPToolWithBearer(t *testing.T) {
	t.Parallel()
	server := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(_ context.Context, _ *mcp.CallToolRequest, input struct {
		Query string `json:"query" jsonschema:"Search query"`
	}) (*mcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"answer": "result for " + input.Query}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer connector-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer endpoint.Close()

	events := make([]string, 0, 2)
	tools, err := mcpconnector.NewRuntimeTools(context.Background(), []studioapp.ResolvedMCPConnector{{
		ID: "connector-01", Name: "Reference", URL: endpoint.URL, Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
		Tools: []domain.MCPTool{discoveredTool("search_reference", "Search reference material")},
	}}, mcpconnector.ToolAccess{Emit: func(_ context.Context, eventType string, _ any) error {
		events = append(events, eventType)
		return nil
	}})
	require.NoError(t, err)
	require.Len(t, tools, 1)

	info, err := tools[0].Info(context.Background())
	require.NoError(t, err)
	require.Equal(t, "mcp_connector_01_search_reference", info.Name)
	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok, "runtime tool must implement Eino's tool.InvokableTool")
	output, err := invokable.InvokableRun(context.Background(), `{"query":"rain"}`)
	require.NoError(t, err)
	require.Contains(t, output, "result for rain")
	require.Equal(t, []string{studioapp.EventToolCallStart, studioapp.EventToolCallArgs, studioapp.EventToolCallResult, studioapp.EventToolCallEnd}, events)
}

func TestRuntimeToolsRequestsApprovalBeforeInvokingProtectedTool(t *testing.T) {
	t.Parallel()
	remoteCalls := 0
	server := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
		remoteCalls++
		return nil, struct{}{}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	endpoint := httptest.NewServer(handler)
	defer endpoint.Close()

	approvalActions := make([]string, 0, 1)
	tools, err := mcpconnector.NewRuntimeTools(context.Background(), []studioapp.ResolvedMCPConnector{{
		ID: "connector-01", Name: "Reference", URL: endpoint.URL, Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
		Tools: []domain.MCPTool{discoveredTool("search_reference", "Search reference material")},
	}}, mcpconnector.ToolAccess{
		PermissionMode: domain.PermissionRequestApproval,
		RequestApproval: func(_ context.Context, toolCallID, action string) error {
			approvalActions = append(approvalActions, toolCallID+":"+action)
			return nil
		},
	})
	require.NoError(t, err)
	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok)

	_, err = invokable.InvokableRun(context.Background(), `{}`)
	require.Error(t, err)
	require.True(t, errors.Is(err, studioapp.ErrApprovalRequired))
	_, err = invokable.InvokableRun(context.Background(), `{}`)
	require.Error(t, err)
	require.Len(t, approvalActions, 2)
	first := strings.Split(approvalActions[0], ":")
	second := strings.Split(approvalActions[1], ":")
	require.Len(t, first, 2)
	require.Len(t, second, 2)
	require.Equal(t, first[1], second[1])
	require.Regexp(t, `^mcp\.connector-01\.search_reference\.[a-f0-9]{64}$`, first[1])
	require.True(t, strings.HasPrefix(first[0], first[1]+"."))
	require.True(t, strings.HasPrefix(second[0], second[1]+"."))
	require.NotEqual(t, first[0], second[0])
	require.Zero(t, remoteCalls)
}

func TestRuntimeToolsCloseToolEventAfterRemoteFailure(t *testing.T) {
	t.Parallel()
	server := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct{}, error) {
		return nil, struct{}{}, nil
	})
	endpoint := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	events := make([]string, 0, 2)
	tools, err := mcpconnector.NewRuntimeTools(context.Background(), []studioapp.ResolvedMCPConnector{{
		ID: "connector-01", Name: "Reference", URL: endpoint.URL, Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
		Tools: []domain.MCPTool{discoveredTool("search_reference", "Search reference material")},
	}}, mcpconnector.ToolAccess{Emit: func(_ context.Context, eventType string, _ any) error {
		events = append(events, eventType)
		return nil
	}})
	require.NoError(t, err)
	endpoint.Close()

	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok)
	_, err = invokable.InvokableRun(context.Background(), `{}`)
	require.Error(t, err)
	require.Equal(t, []string{studioapp.EventToolCallStart, studioapp.EventToolCallArgs, studioapp.EventToolCallResult, studioapp.EventToolCallEnd}, events)
}

func TestRuntimeToolsRedactConnectorCredentialFromToolResult(t *testing.T) {
	t.Parallel()
	server := mcp.NewServer(&mcp.Implementation{Name: "reference", Version: "1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "search_reference", Description: "Search reference material"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, map[string]string, error) {
		return nil, map[string]string{"answer": "connector-secret"}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer connector-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer endpoint.Close()
	tools, err := mcpconnector.NewRuntimeTools(context.Background(), []studioapp.ResolvedMCPConnector{{
		ID: "connector-01", Name: "Reference", URL: endpoint.URL, Credential: "connector-secret", Policy: domain.ConnectorPolicyAuto,
		Tools: []domain.MCPTool{discoveredTool("search_reference", "Search reference material")},
	}}, mcpconnector.ToolAccess{})
	require.NoError(t, err)
	invokable, ok := tools[0].(einotool.InvokableTool)
	require.True(t, ok)

	output, err := invokable.InvokableRun(context.Background(), `{}`)
	require.NoError(t, err)
	require.NotContains(t, output, "connector-secret")
	require.Contains(t, output, "[REDACTED]")
}
