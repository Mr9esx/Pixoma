package topics_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/topics"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
)

type fakeRepo struct {
	topics map[string]topic.Topic
}

func (f *fakeRepo) List(_ context.Context, enabled *bool) ([]topic.Topic, error) {
	var out []topic.Topic
	for _, t := range f.topics {
		if enabled != nil && t.Enabled != *enabled {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, key string) (*topic.Topic, error) {
	t, ok := f.topics[key]
	if !ok {
		return nil, topic.ErrTopicNotFound
	}
	return &t, nil
}

func (f *fakeRepo) Create(_ context.Context, t topic.Topic) error {
	if _, ok := f.topics[t.Key]; ok {
		return topic.ErrTopicConflict
	}
	f.topics[t.Key] = t
	return nil
}

func (f *fakeRepo) Update(_ context.Context, t topic.Topic) error {
	if _, ok := f.topics[t.Key]; !ok {
		return topic.ErrTopicNotFound
	}
	f.topics[t.Key] = t
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, key string) error {
	if _, ok := f.topics[key]; !ok {
		return topic.ErrTopicNotFound
	}
	delete(f.topics, key)
	return nil
}

func newTestRouter(repo *fakeRepo, caseRefs, edgeRefs int) http.Handler {
	h := &topics.Handler{
		Repo:          repo,
		CountCaseRefs: func(context.Context, string) (int, error) { return caseRefs, nil },
		CountEdgeRefs: func(context.Context, string) (int, error) { return edgeRefs, nil },
	}
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func do(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func seedRepo(repo *fakeRepo) {
	now := time.Now().UTC()
	repo.topics["default"] = topic.Topic{Key: "default", Name: "Default", Enabled: true, CreatedAt: now, UpdatedAt: now}
}

func TestTopics_CreateAndList(t *testing.T) {
	repo := &fakeRepo{topics: map[string]topic.Topic{}}
	r := newTestRouter(repo, 0, 0)

	rec := do(t, r, http.MethodPost, "/", `{"key":"fast-gpu","name":"Fast GPU"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}
	rec = do(t, r, http.MethodGet, "/", "")
	var list []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("list decode: %v", err)
	}
	if len(list) != 1 || list[0]["key"] != "fast-gpu" {
		t.Fatalf("list = %+v", list)
	}
}

func TestTopics_InvalidKey(t *testing.T) {
	repo := &fakeRepo{topics: map[string]topic.Topic{}}
	r := newTestRouter(repo, 0, 0)
	for _, key := range []string{"Bad Key", "UPPER", "-lead", "trail-", "a_b", ""} {
		rec := do(t, r, http.MethodPost, "/", `{"key":"`+key+`","name":"x"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("key %q status = %d", key, rec.Code)
		}
	}
}

func TestTopics_DuplicateConflict(t *testing.T) {
	repo := &fakeRepo{topics: map[string]topic.Topic{}}
	r := newTestRouter(repo, 0, 0)
	_ = do(t, r, http.MethodPost, "/", `{"key":"dup","name":"D"}`)
	rec := do(t, r, http.MethodPost, "/", `{"key":"dup","name":"D2"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d", rec.Code)
	}
}

func TestTopics_Update(t *testing.T) {
	repo := &fakeRepo{topics: map[string]topic.Topic{}}
	r := newTestRouter(repo, 0, 0)
	_ = do(t, r, http.MethodPost, "/", `{"key":"k","name":"K"}`)
	rec := do(t, r, http.MethodPut, "/k", `{"name":"K2","enabled":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d", rec.Code)
	}
	got, _ := repo.Get(context.Background(), "k")
	if got.Name != "K2" || got.Enabled {
		t.Fatalf("update not applied: %+v", got)
	}
}

func TestTopics_DeleteProtections(t *testing.T) {
	repo := &fakeRepo{topics: map[string]topic.Topic{}}
	seedRepo(repo)
	now := time.Now().UTC()
	repo.topics["fast-gpu"] = topic.Topic{Key: "fast-gpu", Name: "F", Enabled: true, CreatedAt: now, UpdatedAt: now}

	r := newTestRouter(repo, 0, 0)
	rec := do(t, r, http.MethodDelete, "/default", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete default status = %d", rec.Code)
	}

	ref := newTestRouter(repo, 1, 0)
	rec = do(t, ref, http.MethodDelete, "/fast-gpu", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete referenced status = %d", rec.Code)
	}

	rec = do(t, r, http.MethodDelete, "/fast-gpu", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete ok status = %d", rec.Code)
	}
	if _, err := repo.Get(context.Background(), "fast-gpu"); err != topic.ErrTopicNotFound {
		t.Fatalf("topic still present: %v", err)
	}
}
