package mcp

import (
	"net/http"

	"github.com/Mr9esx/Pixoma/internal/packaging/botapp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// 16 MiB fits an 8 MiB input blob plus JSON/base64 overhead.
const maxStreamableBodyBytes = 16 << 20

// Deps wires MCP tools to the bot facade. Identity is the fallback when the
// request context has none (tests). Production sets identity via WithIdentity.
type Deps struct {
	Facade   *botapp.Facade
	Identity Identity
	Resolver *Resolver
}

// NewHandler serves Streamable HTTP at /mcp and legacy HTTP+SSE at /sse.
func NewHandler(deps Deps) http.Handler {
	h := &httpBackend{deps: deps, cache: &mcpsdk.SchemaCache{}}
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpsdk.NewStreamableHTTPHandler(h.server, &mcpsdk.StreamableHTTPOptions{
		MaxRequestBodyBytes: maxStreamableBodyBytes,
	}))
	mux.Handle("/sse", bindSSESessionUser(mcpsdk.NewSSEHandler(h.server, nil)))
	var out http.Handler = mux
	if deps.Resolver != nil {
		out = RequireBearer(deps.Resolver)(mux)
	}
	return out
}

type httpBackend struct {
	deps  Deps
	cache *mcpsdk.SchemaCache
}

func (h *httpBackend) server(r *http.Request) *mcpsdk.Server {
	ident, ok := IdentityFrom(r.Context())
	if !ok {
		ident = h.deps.Identity
	}
	return newServer(h.deps, ident, h.cache)
}
