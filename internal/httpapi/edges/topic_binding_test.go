package edges_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/edges"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type fakeTopicRepo struct {
	topics map[string]topic.Topic
}

func (f *fakeTopicRepo) List(_ context.Context, _ *bool) ([]topic.Topic, error) {
	var out []topic.Topic
	for _, t := range f.topics {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeTopicRepo) Get(_ context.Context, key string) (*topic.Topic, error) {
	t, ok := f.topics[key]
	if !ok {
		return nil, topic.ErrTopicNotFound
	}
	return &t, nil
}

func (f *fakeTopicRepo) Create(_ context.Context, t topic.Topic) error { f.topics[t.Key] = t; return nil }
func (f *fakeTopicRepo) Update(_ context.Context, t topic.Topic) error { f.topics[t.Key] = t; return nil }
func (f *fakeTopicRepo) Delete(_ context.Context, key string) error    { delete(f.topics, key); return nil }

func TestEdges_PatchSubscribeTopics(t *testing.T) {
	dsn := "file:edge_binding_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatal(err)
	}
	repo := instpersist.NewEdgeRepository(gdb)
	now := time.Now().UTC()
	_ = repo.Upsert(context.Background(), &edge.Record{ID: "gpu-1", Name: "gpu-1", Enabled: true, CreatedAt: now, UpdatedAt: now})

	topicsRepo := &fakeTopicRepo{topics: map[string]topic.Topic{
		"default":   {Key: "default", Name: "Default", Enabled: true},
		"fast-gpu":  {Key: "fast-gpu", Name: "Fast", Enabled: true},
		"disabled":  {Key: "disabled", Name: "D", Enabled: false},
	}}
	h := &edges.Handler{Repo: repo, Topics: topicsRepo}
	r := chi.NewRouter()
	r.Route("/api/v1/edges", h.Mount)

	do := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	rec := do("/api/v1/edges/gpu-1", `{"subscribe_topics":["default","fast-gpu"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d body=%s", rec.Code, rec.Body.String())
	}
	var dto map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &dto)
	if dto["subscribe_topics"] == nil || len(dto["subscribe_topics"].([]any)) != 2 {
		t.Fatalf("dto subscribe_topics = %v", dto["subscribe_topics"])
	}

	rec = do("/api/v1/edges/gpu-1", `{"subscribe_topics":["nope"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown topic status = %d body=%s", rec.Code, rec.Body.String())
	}
	rec = do("/api/v1/edges/gpu-1", `{"subscribe_topics":["disabled"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("disabled topic status = %d body=%s", rec.Code, rec.Body.String())
	}
	rec = do("/api/v1/edges/gpu-1", `{"subscribe_topics":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("empty topics status = %d body=%s", rec.Code, rec.Body.String())
	}
	got, _ := repo.Get(context.Background(), sharedkernel.EdgeID("gpu-1"))
	eff := got.EffectiveTopics()
	if len(eff) != 1 || eff[0] != "default" {
		t.Fatalf("empty subscribe should normalize to default, got %v", eff)
	}
}
