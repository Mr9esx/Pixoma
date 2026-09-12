package mcp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCP_StreamableHTTP_InitializeAndListTools(t *testing.T) {
	srv := httptest.NewServer(pixmcp.NewHandler(pixmcp.Deps{}))
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()

	ir := cs.InitializeResult()
	if ir == nil || ir.Capabilities == nil {
		t.Fatal("initialize capabilities missing")
	}
	caps := ir.Capabilities
	if caps.Tools == nil {
		t.Fatal("missing tools capability")
	}
	if caps.Resources == nil {
		t.Fatal("missing resources capability")
	}
	if caps.Prompts == nil {
		t.Fatal("missing prompts capability")
	}
	if caps.Logging == nil {
		t.Fatal("missing logging capability")
	}
	if caps.Completions == nil {
		t.Fatal("missing completions capability")
	}
	if caps.Experimental == nil || caps.Experimental["elicitation"] == nil {
		t.Fatal("missing elicitation capability")
	}

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range res.Tools {
		got[tool.Name] = true
		if tool.Name == "create_case" || tool.Name == "patch_case" {
			t.Fatalf("must not expose case write tool %q", tool.Name)
		}
	}
	for _, name := range []string{"list_workflows", "get_workflow", "run_workflow", "list_tasks", "get_task"} {
		if !got[name] {
			t.Fatalf("missing tool %q in %v", name, got)
		}
	}
}

func TestMCP_LegacySSE_Initialize(t *testing.T) {
	srv := httptest.NewServer(pixmcp.NewHandler(pixmcp.Deps{}))
	defer srv.Close()

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, &mcp.SSEClientTransport{Endpoint: srv.URL + "/sse"}, nil)
	if err != nil {
		t.Fatalf("sse connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(res.Tools) == 0 {
		t.Fatal("expected tools")
	}
}

func TestMCP_StreamableAllowsFiveMiBBody(t *testing.T) {
	h := pixmcp.NewHandler(pixmcp.Deps{})
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"` +
		strings.Repeat("a", 5<<20) +
		`","version":"1"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("5MiB initialize rejected as 413")
	}
}
