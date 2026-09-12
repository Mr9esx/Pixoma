package mcp_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	channeldomain "github.com/Mr9esx/Pixoma/internal/channels/domain"
	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCP_RequiresBearer(t *testing.T) {
	ctx := context.Background()
	tokens := pixmcp.NewMemoryTokenStore()
	users := &authUsers{byID: map[string]*identitydomain.User{
		"user-1": {
			ID: "user-1", ChannelID: "ch-mcp", ExternalUserID: "ext-1",
			Access: identitydomain.UserAccessAlwaysAllowed,
		},
	}}
	channels := &authChannels{byID: map[string]channeldomain.Channel{
		"ch-mcp": {ID: "ch-mcp", Platform: string(channeldomain.PlatformMCP), Enabled: true},
	}}
	key := make([]byte, 32)
	plain := "mcp-plain-token-1"
	enc, err := pixmcp.EncryptToken(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := tokens.Put(ctx, pixmcp.TokenRecord{
		UserID: "user-1", TokenHash: pixmcp.HashToken(plain), TokenCipher: enc,
	}); err != nil {
		t.Fatal(err)
	}
	h := pixmcp.NewHandler(pixmcp.Deps{
		Resolver: pixmcp.NewResolver(tokens, users, channels, key),
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}")))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no bearer status=%d", rec.Code)
	}

	cookieReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	cookieReq.AddCookie(&http.Cookie{Name: "pixoma_session", Value: "admin-cookie"})
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, cookieReq)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie only status=%d", rec.Code)
	}

	bad := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	bad.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, bad)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong bearer status=%d", rec.Code)
	}

	srv := httptest.NewServer(h)
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: srv.URL + "/mcp",
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req = req.Clone(req.Context())
			req.Header.Set("Authorization", "Bearer "+plain)
			return http.DefaultTransport.RoundTrip(req)
		})},
	}, nil)
	if err != nil {
		t.Fatalf("valid bearer connect: %v", err)
	}
	defer cs.Close()
	if _, err := cs.ListTools(ctx, nil); err != nil {
		t.Fatalf("list tools: %v", err)
	}

	ch := channels.byID["ch-mcp"]
	ch.Enabled = false
	channels.byID["ch-mcp"] = ch
	disabled := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	disabled.Header.Set("Authorization", "Bearer "+plain)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, disabled)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled channel status=%d", rec.Code)
	}
}

func TestMCP_StreamableSessionRejectsOtherBearer(t *testing.T) {
	ctx := context.Background()
	tokens := pixmcp.NewMemoryTokenStore()
	users := &authUsers{byID: map[string]*identitydomain.User{
		"user-1": {
			ID: "user-1", ChannelID: "ch-mcp", ExternalUserID: "ext-1",
			Access: identitydomain.UserAccessAlwaysAllowed,
		},
		"user-2": {
			ID: "user-2", ChannelID: "ch-mcp", ExternalUserID: "ext-2",
			Access: identitydomain.UserAccessAlwaysAllowed,
		},
	}}
	channels := &authChannels{byID: map[string]channeldomain.Channel{
		"ch-mcp": {ID: "ch-mcp", Platform: string(channeldomain.PlatformMCP), Enabled: true},
	}}
	key := make([]byte, 32)
	plain1 := "mcp-session-owner"
	plain2 := "mcp-session-stranger"
	mustStoreMCPToken(t, tokens, key, "user-1", plain1)
	mustStoreMCPToken(t, tokens, key, "user-2", plain2)

	srv := httptest.NewServer(pixmcp.NewHandler(pixmcp.Deps{
		Resolver: pixmcp.NewResolver(tokens, users, channels, key),
	}))
	defer srv.Close()

	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	ownerInit, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/mcp", strings.NewReader(initBody))
	if err != nil {
		t.Fatal(err)
	}
	setMCPJSONHeaders(ownerInit, plain1)
	ownerResp, err := http.DefaultClient.Do(ownerInit)
	if err != nil {
		t.Fatal(err)
	}
	defer ownerResp.Body.Close()
	if ownerResp.StatusCode != http.StatusOK {
		t.Fatalf("initialize status=%d", ownerResp.StatusCode)
	}
	sessionID := ownerResp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("missing Mcp-Session-Id")
	}

	pingBody := `{"jsonrpc":"2.0","id":2,"method":"ping"}`
	strangerPing, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/mcp", strings.NewReader(pingBody))
	if err != nil {
		t.Fatal(err)
	}
	setMCPJSONHeaders(strangerPing, plain2)
	strangerPing.Header.Set("Mcp-Session-Id", sessionID)
	strangerResp, err := http.DefaultClient.Do(strangerPing)
	if err != nil {
		t.Fatal(err)
	}
	defer strangerResp.Body.Close()
	if strangerResp.StatusCode != http.StatusForbidden {
		t.Fatalf("stranger session status=%d want 403", strangerResp.StatusCode)
	}

	ownerPing, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/mcp", strings.NewReader(pingBody))
	if err != nil {
		t.Fatal(err)
	}
	setMCPJSONHeaders(ownerPing, plain1)
	ownerPing.Header.Set("Mcp-Session-Id", sessionID)
	okResp, err := http.DefaultClient.Do(ownerPing)
	if err != nil {
		t.Fatal(err)
	}
	defer okResp.Body.Close()
	if okResp.StatusCode != http.StatusOK {
		t.Fatalf("owner session status=%d", okResp.StatusCode)
	}
}

func TestMCP_LegacySSESessionRejectsOtherBearer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tokens := pixmcp.NewMemoryTokenStore()
	users := &authUsers{byID: map[string]*identitydomain.User{
		"user-1": {
			ID: "user-1", ChannelID: "ch-mcp", ExternalUserID: "ext-1",
			Access: identitydomain.UserAccessAlwaysAllowed,
		},
		"user-2": {
			ID: "user-2", ChannelID: "ch-mcp", ExternalUserID: "ext-2",
			Access: identitydomain.UserAccessAlwaysAllowed,
		},
	}}
	channels := &authChannels{byID: map[string]channeldomain.Channel{
		"ch-mcp": {ID: "ch-mcp", Platform: string(channeldomain.PlatformMCP), Enabled: true},
	}}
	key := make([]byte, 32)
	plain1 := "mcp-sse-owner"
	plain2 := "mcp-sse-stranger"
	mustStoreMCPToken(t, tokens, key, "user-1", plain1)
	mustStoreMCPToken(t, tokens, key, "user-2", plain2)

	srv := httptest.NewServer(pixmcp.NewHandler(pixmcp.Deps{
		Resolver: pixmcp.NewResolver(tokens, users, channels, key),
	}))
	defer srv.Close()

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/sse", nil)
	if err != nil {
		t.Fatal(err)
	}
	getReq.Header.Set("Accept", "text/event-stream")
	getReq.Header.Set("Authorization", "Bearer "+plain1)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("sse get status=%d", getResp.StatusCode)
	}
	sessionID := readSSESessionID(t, getResp.Body)
	if sessionID == "" {
		t.Fatal("missing sse sessionid")
	}

	pingBody := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	stranger, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/sse?sessionid="+sessionID, strings.NewReader(pingBody))
	if err != nil {
		t.Fatal(err)
	}
	stranger.Header.Set("Content-Type", "application/json")
	stranger.Header.Set("Authorization", "Bearer "+plain2)
	strangerResp, err := http.DefaultClient.Do(stranger)
	if err != nil {
		t.Fatal(err)
	}
	defer strangerResp.Body.Close()
	if strangerResp.StatusCode != http.StatusForbidden {
		t.Fatalf("stranger sse session status=%d want 403", strangerResp.StatusCode)
	}

	owner, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/sse?sessionid="+sessionID, strings.NewReader(pingBody))
	if err != nil {
		t.Fatal(err)
	}
	owner.Header.Set("Content-Type", "application/json")
	owner.Header.Set("Authorization", "Bearer "+plain1)
	ownerResp, err := http.DefaultClient.Do(owner)
	if err != nil {
		t.Fatal(err)
	}
	defer ownerResp.Body.Close()
	if ownerResp.StatusCode != http.StatusAccepted {
		t.Fatalf("owner sse session status=%d", ownerResp.StatusCode)
	}
}

func mustStoreMCPToken(t *testing.T, tokens pixmcp.TokenStore, key []byte, userID, plain string) {
	t.Helper()
	enc, err := pixmcp.EncryptToken(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := tokens.Put(context.Background(), pixmcp.TokenRecord{
		UserID: userID, TokenHash: pixmcp.HashToken(plain), TokenCipher: enc,
	}); err != nil {
		t.Fatal(err)
	}
}

func setMCPJSONHeaders(req *http.Request, bearer string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+bearer)
}

func readSSESessionID(t *testing.T, r io.Reader) string {
	t.Helper()
	br := bufio.NewReader(r)
	var buf strings.Builder
	for range 32 {
		line, err := br.ReadString('\n')
		buf.WriteString(line)
		if sid := sessionIDFromSSE(buf.String()); sid != "" {
			return sid
		}
		if err != nil {
			t.Fatalf("read sse: %v data=%q", err, buf.String())
		}
	}
	t.Fatalf("no sessionid in %q", buf.String())
	return ""
}

func sessionIDFromSSE(s string) string {
	const key = "sessionid="
	i := strings.Index(s, key)
	if i < 0 {
		return ""
	}
	rest := s[i+len(key):]
	cut := len(rest)
	for j, r := range rest {
		if r == '&' || r == '\n' || r == '\r' || r == ' ' {
			cut = j
			break
		}
	}
	return strings.TrimSpace(rest[:cut])
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type authUsers struct {
	byID map[string]*identitydomain.User
}

func (r *authUsers) UpsertByChannelExternal(context.Context, identitydomain.UpsertFrom) (*identitydomain.User, error) {
	return nil, errors.New("unused")
}
func (r *authUsers) SetAccess(context.Context, string, identitydomain.UserAccess) (*identitydomain.User, error) {
	return nil, errors.New("unused")
}
func (r *authUsers) List(context.Context, identitydomain.ListQuery) ([]*identitydomain.User, error) {
	return nil, nil
}
func (r *authUsers) Delete(context.Context, string) error {
	return errors.New("unused")
}
func (r *authUsers) GetByID(_ context.Context, id string) (*identitydomain.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, identitydomain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

type authChannels struct {
	byID map[string]channeldomain.Channel
}

func (r *authChannels) Get(_ context.Context, id string) (channeldomain.Channel, error) {
	ch, ok := r.byID[id]
	if !ok {
		return channeldomain.Channel{}, channeldomain.ErrNotFound
	}
	return ch, nil
}
