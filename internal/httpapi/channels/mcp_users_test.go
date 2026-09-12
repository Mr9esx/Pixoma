package channels_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	channelapp "github.com/Mr9esx/Pixoma/internal/channels/application"
	channelpersist "github.com/Mr9esx/Pixoma/internal/channels/infrastructure/persistence"
	channelsapi "github.com/Mr9esx/Pixoma/internal/httpapi/channels"
	"github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	usersapi "github.com/Mr9esx/Pixoma/internal/httpapi/users"
	pixmcp "github.com/Mr9esx/Pixoma/internal/mcp"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	identitydomain "github.com/Mr9esx/Pixoma/internal/users/domain"
	userpersist "github.com/Mr9esx/Pixoma/internal/users/infrastructure/persistence"
)

func TestMCPUser_MintAndRotate(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:mcp_users_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&channelpersist.ChannelRow{},
		&userpersist.UserRow{},
		&userpersist.UserExternalIdentityRow{},
		&pixmcp.TokenRow{},
	); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	chStore := channelpersist.NewGormRepository(gdb)
	chSvc := &channelapp.Service{Store: chStore, Key: key}
	users := userpersist.NewUserRepository(gdb)
	tokens := pixmcp.NewGormTokenStore(gdb)
	chH := &channelsapi.Handler{Svc: chSvc, Users: users, Tokens: tokens, Key: key}
	usersH := &usersapi.Handler{Repo: users, Channels: chStore, Tokens: tokens, Key: key, ChannelGet: chSvc}

	r := chi.NewRouter()
	r.Route("/api/v1/channels", func(r chi.Router) { chH.Mount(r) })
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Get("/{id}/mcp-token", usersH.GetMCPToken)
		r.Post("/{id}/mcp-token/rotate", usersH.RotateMCPToken)
	})

	_, chBody := serveJSON(t, r, http.MethodPost, "/api/v1/channels", map[string]any{
		"platform": "mcp", "name": "Cursor",
	}, "")
	if chBody["platform"] != "mcp" {
		t.Fatalf("create mcp=%v", chBody)
	}
	chID, _ := chBody["id"].(string)

	code, mint := serveJSON(t, r, http.MethodPost, "/api/v1/channels/"+chID+"/mcp-users", map[string]any{
		"name": "我的 Cursor",
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("mint status=%d body=%v", code, mint)
	}
	token, _ := mint["token"].(string)
	user, _ := mint["user"].(map[string]any)
	userID, _ := user["id"].(string)
	if token == "" || userID == "" {
		t.Fatalf("mint=%v", mint)
	}
	if user["access"] != string(identitydomain.UserAccessAlwaysAllowed) {
		t.Fatalf("access=%v", user["access"])
	}

	_, adminGot := serveJSON(t, r, http.MethodGet, "/api/v1/users/"+userID+"/mcp-token", nil, "admin")
	if adminGot["token"] != token {
		t.Fatalf("admin token=%v", adminGot)
	}
	_, viewerGot := serveJSON(t, r, http.MethodGet, "/api/v1/users/"+userID+"/mcp-token", nil, "viewer")
	if tok, ok := viewerGot["token"].(string); ok && tok != "" {
		t.Fatalf("viewer saw token")
	}

	code, rot := serveJSON(t, r, http.MethodPost, "/api/v1/users/"+userID+"/mcp-token/rotate", map[string]any{}, "admin")
	if code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%v", code, rot)
	}
	newTok, _ := rot["token"].(string)
	if newTok == "" || newTok == token {
		t.Fatalf("rotate=%v old=%s", rot, token)
	}

	mcpH := pixmcp.NewHandler(pixmcp.Deps{Resolver: pixmcp.NewResolver(tokens, users, chSvc, key)})
	oldReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	oldReq.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mcpH.ServeHTTP(rec, oldReq)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("old token status=%d", rec.Code)
	}

	code, _ = serveJSON(t, r, http.MethodDelete, "/api/v1/channels/"+chID+"/mcp-users/"+userID, nil, "")
	if code != http.StatusOK {
		t.Fatalf("delete status=%d", code)
	}
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+chID+"/mcp-users", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list after delete status=%d body=%s", listRec.Code, listRec.Body.Bytes())
	}
	var listed []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("list json: %v body=%s", err, listRec.Body.Bytes())
	}
	for _, item := range listed {
		if item["id"] == userID {
			t.Fatalf("deleted user still listed: %v", listed)
		}
	}

	delReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	delReq.Header.Set("Authorization", "Bearer "+newTok)
	delRec := httptest.NewRecorder()
	mcpH.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusUnauthorized {
		t.Fatalf("deleted token status=%d", delRec.Code)
	}

	_, tgBody := serveJSON(t, r, http.MethodPost, "/api/v1/channels", map[string]any{
		"platform": "telegram", "name": "主机器人", "token": "1234567890",
	}, "")
	tgID, _ := tgBody["id"].(string)
	code, rej := serveJSON(t, r, http.MethodDelete, "/api/v1/channels/"+tgID+"/mcp-users/"+userID, nil, "")
	if code != http.StatusBadRequest {
		t.Fatalf("non-mcp delete status=%d body=%v", code, rej)
	}
}

func TestMCPUser_MintFailureRollsBackUser(t *testing.T) {
	gdb, err := db.Open(db.Options{DSN: "file:mcp_users_rollback_" + t.Name() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb,
		&channelpersist.ChannelRow{},
		&userpersist.UserRow{},
		&userpersist.UserExternalIdentityRow{},
		&pixmcp.TokenRow{},
	); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	chStore := channelpersist.NewGormRepository(gdb)
	chSvc := &channelapp.Service{Store: chStore, Key: key}
	users := userpersist.NewUserRepository(gdb)
	chH := &channelsapi.Handler{
		Svc: chSvc, Users: users, Tokens: failPutTokens{}, Key: key,
	}

	r := chi.NewRouter()
	r.Route("/api/v1/channels", func(r chi.Router) { chH.Mount(r) })

	_, chBody := serveJSON(t, r, http.MethodPost, "/api/v1/channels", map[string]any{
		"platform": "mcp", "name": "Cursor",
	}, "")
	chID, _ := chBody["id"].(string)

	code, mint := serveJSON(t, r, http.MethodPost, "/api/v1/channels/"+chID+"/mcp-users", map[string]any{
		"name": "半成品",
	}, "")
	if code != http.StatusInternalServerError {
		t.Fatalf("mint fail status=%d body=%v", code, mint)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+chID+"/mcp-users", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.Bytes())
	}
	var listed []map[string]any
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("list json: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("mint failure left users=%v", listed)
	}
}

type failPutTokens struct{}

func (failPutTokens) Put(context.Context, pixmcp.TokenRecord) error {
	return errors.New("store down")
}
func (failPutTokens) GetByHash(context.Context, string) (pixmcp.TokenRecord, error) {
	return pixmcp.TokenRecord{}, pixmcp.ErrTokenNotFound
}
func (failPutTokens) GetByUserID(context.Context, string) (pixmcp.TokenRecord, error) {
	return pixmcp.TokenRecord{}, pixmcp.ErrTokenNotFound
}
func (failPutTokens) DeleteByUserID(context.Context, string) error { return nil }

func serveJSON(t *testing.T, h http.Handler, method, path string, body any, role string) (int, map[string]any) {
	t.Helper()
	var buf *bytes.Reader
	if body == nil {
		buf = bytes.NewReader(nil)
	} else {
		raw, _ := json.Marshal(body)
		buf = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		req = req.WithContext(setup.WithAccount(req.Context(), setup.AccountSession{
			AccountID: "acct-" + role, Username: role, Role: role,
		}))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var m map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	return rec.Code, m
}
