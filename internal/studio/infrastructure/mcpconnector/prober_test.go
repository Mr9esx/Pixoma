package mcpconnector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/mcpconnector"
)

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
}
