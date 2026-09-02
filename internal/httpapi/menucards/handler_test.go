package menucards_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	menucardsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/menucards"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
)

type memTreeRepo struct {
	tree    mcdomain.MenuTree
	missing bool
}

func (m *memTreeRepo) GetTree(_ context.Context, _ string) (mcdomain.MenuTree, error) {
	if m.missing {
		return mcdomain.MenuTree{}, gorm.ErrRecordNotFound
	}
	return m.tree, nil
}
func (m *memTreeRepo) PutTree(_ context.Context, id string, tree mcdomain.MenuTree) error {
	tree.ID = id
	m.tree = tree
	m.missing = false
	return nil
}
func (m *memTreeRepo) WorkflowPlacements(_ context.Context, workflowID string) ([]persistence.WorkflowPlacement, error) {
	var out []persistence.WorkflowPlacement
	for _, p := range mcdomain.WalkWorkflowPlacements(m.tree) {
		if p.WorkflowID == workflowID {
			out = append(out, persistence.WorkflowPlacement{
				ChannelID: "ch1", ItemID: p.ButtonID, Label: p.Labels[len(p.Labels)-1],
				Kind: p.Kind, Labels: p.Labels,
			})
		}
	}
	return out, nil
}
func (m *memTreeRepo) RemoveWorkflowReferences(context.Context, string) ([]persistence.WorkflowPlacement, error) {
	return nil, nil
}

var _ persistence.CardRepository = (*memTreeRepo)(nil)

func mustReq(t *testing.T, method, url string, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestPutGetMenuTree(t *testing.T) {
	repo := &memTreeRepo{missing: true}
	h := menucardsapi.NewHandler(repo)
	r := chi.NewRouter()
	r.Route("/api/v1/channels/{id}", func(r chi.Router) { h.Mount(r) })
	r.Get("/api/v1/cases/{id}/menu-placements", h.ListWorkflowPlacements)
	srv := httptest.NewServer(r)
	defer srv.Close()

	get, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/api/v1/channels/ch1/menu", nil))
	if err != nil || get.StatusCode != http.StatusOK {
		t.Fatalf("get default status=%v err=%v", get.StatusCode, err)
	}
	var def mcdomain.MenuTree
	_ = json.NewDecoder(get.Body).Decode(&def)
	get.Body.Close()
	if def.ID != "ch1" || len(def.Items) != 2 {
		t.Fatalf("default=%+v", def)
	}

	tree := mcdomain.MenuTree{
		Columns: 2,
		Items: []mcdomain.TreeButton{
			{ID: "a", Label: "图", Action: mcdomain.TreeAction{Type: "open_workflow", WorkflowID: "10"}},
		},
	}
	body, _ := json.Marshal(tree)
	put, err := http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", body))
	if err != nil || put.StatusCode != http.StatusOK {
		t.Fatalf("put status=%v err=%v", put.StatusCode, err)
	}
	put.Body.Close()

	get2, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/api/v1/channels/ch1/menu", nil))
	if err != nil {
		t.Fatal(err)
	}
	var got mcdomain.MenuTree
	_ = json.NewDecoder(get2.Body).Decode(&got)
	get2.Body.Close()
	if len(got.Items) != 1 || got.Items[0].Action.WorkflowID != "10" {
		t.Fatalf("got=%+v", got)
	}

	wf, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/api/v1/cases/10/menu-placements", nil))
	if err != nil || wf.StatusCode != http.StatusOK {
		t.Fatalf("placements status=%v err=%v", wf.StatusCode, err)
	}
	var placements []map[string]any
	_ = json.NewDecoder(wf.Body).Decode(&placements)
	wf.Body.Close()
	if len(placements) != 1 || placements[0]["kind"] != "keyboard" {
		t.Fatalf("placements=%v", placements)
	}
}

func TestCardsEndpointsGone(t *testing.T) {
	h := menucardsapi.NewHandler(&memTreeRepo{})
	r := chi.NewRouter()
	r.Route("/api/v1/channels/{id}", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/api/v1/channels/ch1/cards", nil))
	if err != nil || resp.StatusCode != http.StatusGone {
		t.Fatalf("status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
}

func TestPutRejectsEmptyRoot(t *testing.T) {
	h := menucardsapi.NewHandler(&memTreeRepo{})
	r := chi.NewRouter()
	r.Route("/api/v1/channels/{id}", func(r chi.Router) { h.Mount(r) })
	srv := httptest.NewServer(r)
	defer srv.Close()
	body, _ := json.Marshal(mcdomain.MenuTree{Columns: 2, Items: nil})
	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", body))
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()
}