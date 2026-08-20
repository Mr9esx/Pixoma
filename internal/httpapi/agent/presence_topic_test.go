package agent_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/agent"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	instpersist "github.com/mr9esx/comfyui_tgbot/internal/platform/edge/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestAgent_PresenceWritesSubscribeTopicsOnce(t *testing.T) {
	dsn := "file:agent_topic_test_" + t.Name() + "?mode=memory&cache=shared"
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

	h := &agent.Handler{
		Token:    "tok",
		Tasks:    runtimedomain.NewMemoryTaskRepository(),
		Presence: presence.NewStore(),
		Edges:    repo,
		Now:      func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)

	post := func(body string) int {
		req := httptest.NewRequest(http.MethodPost, "/agent/v1/presence", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer tok")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := post(`{"edge_id":"gpu-1","subscribe_topics":["fast-gpu"]}`); code != http.StatusOK {
		t.Fatalf("first presence status = %d", code)
	}
	rec, _ := repo.Get(context.Background(), sharedkernel.EdgeID("gpu-1"))
	if len(rec.SubscribeTopics) != 1 || rec.SubscribeTopics[0] != "fast-gpu" {
		t.Fatalf("first presence subscription = %v", rec.SubscribeTopics)
	}

	// Second presence with different topics must NOT overwrite (first write wins).
	if code := post(`{"edge_id":"gpu-1","subscribe_topics":["other"]}`); code != http.StatusOK {
		t.Fatalf("second presence status = %d", code)
	}
	rec, _ = repo.Get(context.Background(), sharedkernel.EdgeID("gpu-1"))
	if len(rec.SubscribeTopics) != 1 || rec.SubscribeTopics[0] != "fast-gpu" {
		t.Fatalf("second presence overwrote subscription: %v", rec.SubscribeTopics)
	}

	// Admin override is authoritative and presence won't clobber it.
	if err := repo.UpdateSubscribeTopics(context.Background(), "gpu-1", []string{"default"}); err != nil {
		t.Fatal(err)
	}
	if code := post(`{"edge_id":"gpu-1","subscribe_topics":["fast-gpu"]}`); code != http.StatusOK {
		t.Fatalf("post-admin presence status = %d", code)
	}
	rec, _ = repo.Get(context.Background(), sharedkernel.EdgeID("gpu-1"))
	if len(rec.SubscribeTopics) != 1 || rec.SubscribeTopics[0] != "default" {
		t.Fatalf("admin override clobbered: %v", rec.SubscribeTopics)
	}
}
