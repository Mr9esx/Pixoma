package menucards_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	menucardsapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/menucards"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menucard/infrastructure/persistence"
)

type memCardRepo struct {
	menu  mcdomain.Menu
	cards map[string]mcdomain.Card
}

func (m *memCardRepo) GetMenu(_ context.Context, _ string) (mcdomain.Menu, error) { return m.menu, nil }
func (m *memCardRepo) PutMenu(_ context.Context, _ string, menu mcdomain.Menu) error {
	m.menu = menu
	return nil
}
func (m *memCardRepo) ListCards(_ context.Context, _ string) ([]mcdomain.Card, error) {
	out := make([]mcdomain.Card, 0, len(m.cards))
	for _, c := range m.cards {
		out = append(out, c)
	}
	return out, nil
}
func (m *memCardRepo) GetCard(_ context.Context, _, id string) (mcdomain.Card, error) {
	c, ok := m.cards[id]
	if !ok {
		return mcdomain.Card{}, http.ErrNotSupported
	}
	return c, nil
}
func (m *memCardRepo) CreateCard(_ context.Context, _ string, card mcdomain.Card) error {
	if m.cards == nil {
		m.cards = map[string]mcdomain.Card{}
	}
	m.cards[card.ID] = card
	return nil
}
func (m *memCardRepo) UpdateCard(_ context.Context, _ string, card mcdomain.Card) error {
	m.cards[card.ID] = card
	return nil
}
func (m *memCardRepo) DeleteCard(_ context.Context, _, id string) error {
	delete(m.cards, id)
	return nil
}
func (m *memCardRepo) CardReferences(ctx context.Context, channelID, id string) ([]string, error) {
	return referencesOf(m.menu, m.cards, id), nil
}

func (m *memCardRepo) WorkflowPlacements(_ context.Context, workflowID string) ([]persistence.WorkflowPlacement, error) {
	var out []persistence.WorkflowPlacement
	for _, it := range m.menu.Items {
		if it.Action.Type == "open_workflow" && it.Action.WorkflowID == workflowID {
			out = append(out, persistence.WorkflowPlacement{ChannelID: "ch1", ItemID: it.ID, Label: it.Label, Kind: "menu_item"})
		}
	}
	return out, nil
}

func (m *memCardRepo) RemoveWorkflowReferences(_ context.Context, workflowID string) ([]persistence.WorkflowPlacement, error) {
	var removed []persistence.WorkflowPlacement
	items := m.menu.Items[:0]
	for _, it := range m.menu.Items {
		if it.Action.Type != "open_workflow" || it.Action.WorkflowID != workflowID {
			items = append(items, it)
			continue
		}
		removed = append(removed, persistence.WorkflowPlacement{ChannelID: "ch1", ItemID: it.ID, Label: it.Label, Kind: "menu_item"})
	}
	m.menu.Items = items
	for id, card := range m.cards {
		buttons := card.Buttons[:0]
		for _, b := range card.Buttons {
			if b.Action.Type != "open_workflow" || b.Action.WorkflowID != workflowID {
				buttons = append(buttons, b)
				continue
			}
			removed = append(removed, persistence.WorkflowPlacement{ChannelID: "ch1", ItemID: b.ID, Label: b.Label, Kind: "card_button"})
		}
		card.Buttons = buttons
		m.cards[id] = card
	}
	return removed, nil
}

func referencesOf(menu mcdomain.Menu, cards map[string]mcdomain.Card, id string) []string {
	refs := []string{}
	for _, it := range menu.Items {
		if it.Action.Type == "open_card" && it.Action.CardID == id {
			refs = append(refs, "menu:"+it.ID)
		}
	}
	for _, card := range cards {
		for _, b := range card.Buttons {
			if b.Action.Type == "open_card" && b.Action.CardID == id {
				refs = append(refs, "card:"+card.ID+":"+b.ID)
			}
		}
	}
	return refs
}

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

func TestMenuCardsHandlerCRUD(t *testing.T) {
	repo := &memCardRepo{cards: map[string]mcdomain.Card{}}
	h := menucardsapi.NewHandler(repo)
	r := chi.NewRouter()
	r.Route("/api/v1/channels/{id}", func(r chi.Router) { h.Mount(r) })
	r.Get("/api/v1/cases/{id}/menu-placements", h.ListWorkflowPlacements)
	srv := httptest.NewServer(r)
	defer srv.Close()

	body, _ := json.Marshal(mcdomain.Menu{ID: "m", Name: "主", Columns: 2})
	resp, err := http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", body))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("put menu status=%v err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()

	cardBody, _ := json.Marshal(mcdomain.Card{ID: "c1", Name: "x", Text: "hi", Buttons: []mcdomain.CardButton{
		{ID: "b", Label: "B", Action: mcdomain.Action{Type: "send_text", Text: "ok"}},
	}})
	cardResp, err := http.DefaultClient.Do(mustReq(t, http.MethodPost, srv.URL+"/api/v1/channels/ch1/cards", cardBody))
	if err != nil || cardResp.StatusCode != http.StatusCreated {
		t.Fatalf("post card status=%v err=%v", cardResp.StatusCode, err)
	}
	cardResp.Body.Close()

	menuWithRef, _ := json.Marshal(mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_card", CardID: "c1"}},
	}})
	_, _ = http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", menuWithRef))

	delResp, err := http.DefaultClient.Do(mustReq(t, http.MethodDelete, srv.URL+"/api/v1/channels/ch1/cards/c1", nil))
	if err != nil || delResp.StatusCode != http.StatusConflict {
		t.Fatalf("delete referenced status=%v err=%v", delResp.StatusCode, err)
	}
	delResp.Body.Close()

	refResp, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/api/v1/channels/ch1/cards/c1/references", nil))
	if err != nil || refResp.StatusCode != http.StatusOK {
		t.Fatalf("references status=%v err=%v", refResp.StatusCode, err)
	}
	var refs []string
	_ = json.NewDecoder(refResp.Body).Decode(&refs)
	refResp.Body.Close()
	if len(refs) != 1 || refs[0] != "menu:mi" {
		t.Fatalf("refs=%v", refs)
	}

	// workflow placements reverse lookup
	wfMenu, _ := json.Marshal(mcdomain.Menu{ID: "m", Name: "主", Columns: 2, Items: []mcdomain.MenuItem{
		{ID: "mi", Label: "L", Action: mcdomain.Action{Type: "open_workflow", WorkflowID: "w1"}},
	}})
	_, _ = http.DefaultClient.Do(mustReq(t, http.MethodPut, srv.URL+"/api/v1/channels/ch1/menu", wfMenu))
	wfResp, err := http.DefaultClient.Do(mustReq(t, http.MethodGet, srv.URL+"/api/v1/cases/w1/menu-placements", nil))
	if err != nil || wfResp.StatusCode != http.StatusOK {
		t.Fatalf("workflow placements status=%v err=%v", wfResp.StatusCode, err)
	}
	var placements []map[string]any
	_ = json.NewDecoder(wfResp.Body).Decode(&placements)
	wfResp.Body.Close()
	if len(placements) != 1 || placements[0]["kind"] != "menu_item" {
		t.Fatalf("placements=%v", placements)
	}
}

var _ persistence.CardRepository = (*memCardRepo)(nil)
