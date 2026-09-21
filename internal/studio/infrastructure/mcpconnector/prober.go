package mcpconnector

import (
	"context"
	"net/http"
	"strings"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

// Prober discovers Streamable HTTP MCP tools with a short-lived client
// session. The stored credential is forwarded only as a Bearer token and is
// never included in returned metadata or errors.
type Prober struct {
	HTTPClient *http.Client
}

func (p Prober) Probe(ctx context.Context, endpoint, credential string) ([]domain.MCPTool, error) {
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	client = cloneClientWithBearer(client, credential)
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "pixoma-studio", Version: "1.0"}, nil)
	session, err := mcpClient.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: endpoint, HTTPClient: client, MaxRetries: -1, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	tools := make([]domain.MCPTool, 0, len(result.Tools))
	for _, item := range result.Tools {
		if item == nil || strings.TrimSpace(item.Name) == "" {
			continue
		}
		tools = append(tools, domain.MCPTool{Name: item.Name, Description: item.Description})
	}
	return tools, nil
}

type bearerTransport struct {
	base       http.RoundTripper
	credential string
}

func (t bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	cloned := request.Clone(request.Context())
	if strings.TrimSpace(t.credential) != "" {
		cloned.Header.Set("Authorization", "Bearer "+t.credential)
	}
	return t.base.RoundTrip(cloned)
}

func cloneClientWithBearer(client *http.Client, credential string) *http.Client {
	cloned := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	cloned.Transport = bearerTransport{base: base, credential: credential}
	return &cloned
}
