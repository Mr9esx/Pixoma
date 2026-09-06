package linkhealth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	linkhealthapi "github.com/mr9esx/comfyui_tgbot/internal/httpapi/linkhealth"
	mcdomain "github.com/mr9esx/comfyui_tgbot/internal/menucard/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/packaging/linkhealth"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
)

func TestHandlerGetAssemblesWithoutProbing(t *testing.T) {
	probed := 0
	h := &linkhealthapi.Handler{
		Snapshot: func(*http.Request) (linkhealth.Snapshot, error) {
			return linkhealth.Snapshot{
				AdapterKnown:  true,
				PresenceKnown: true,
				Channels: []linkhealth.ChannelSnap{{
					ID:            "tg",
					Name:          "电报",
					Enabled:       true,
					AdapterFound:  true,
					AdapterState:  "running",
					LastCheckKind: "ok",
				}},
				Menus: map[string]mcdomain.MenuTree{
					"tg": {
						ID:      "tg",
						Columns: 1,
						Items: []mcdomain.TreeButton{{
							ID:    "btn",
							Label: "run",
							Action: mcdomain.TreeAction{
								Type:       "open_workflow",
								WorkflowID: "10",
							},
						}},
					},
				},
				Cases: []linkhealth.CaseSnap{{
					ID: 10, Name: "海报", Topics: []string{"jobs"}, Enabled: true,
				}},
				Topics: []linkhealth.TopicSnap{{Key: "jobs", Name: "出图", Enabled: true}},
				Edges: []linkhealth.EdgeSnap{{
					ID: "e1", Name: "机房-1", Enabled: true, Topics: []string{"jobs"},
				}},
				Presence: map[string]presence.Snapshot{
					"e1": {ID: "e1", EdgeOnline: true, ComfyRunning: true},
				},
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/link-health", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var g linkhealth.Graph
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatal(err)
	}
	health := map[string]linkhealth.Health{}
	for _, n := range g.Nodes {
		health[n.ID] = n.Health
	}
	if health["case:10"] != linkhealth.HealthOK {
		t.Fatalf("case health=%s nodes=%+v", health["case:10"], g.Nodes)
	}
	if probed != 0 {
		t.Fatalf("handler must not probe, probed=%d", probed)
	}
}
