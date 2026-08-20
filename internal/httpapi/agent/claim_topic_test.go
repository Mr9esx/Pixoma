package agent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestAgent_ConcurrentClaimTopicSingleTask(t *testing.T) {
	dsn := "file:agent_claim_test_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &instpersist.EdgeRow{}); err != nil {
		t.Fatal(err)
	}
	edgeRepo := instpersist.NewEdgeRepository(gdb)
	now := time.Now().UTC()
	for _, id := range []string{"gpu-a", "gpu-b"} {
		_ = edgeRepo.Upsert(context.Background(), &edge.Record{
			ID:              sharedkernel.EdgeID(id),
			Name:            id,
			Enabled:         true,
			SubscribeTopics: []string{"fast-gpu"},
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	tasks := runtimedomain.NewMemoryTaskRepository()
	t0 := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t-one", "s1", sharedkernel.CaseID(1), "in", t0)
	_ = task.PrepareForTopic("fast-gpu", sharedkernel.BlobRef{Key: "jobs/t-one/job.json"}, t0)
	_ = tasks.Create(context.Background(), task)

	h := &agent.Handler{
		Token:    "tok",
		Tasks:    tasks,
		Presence: presence.NewStore(),
		Edges:    edgeRepo,
		Lease:    90 * time.Second,
		Now:      func() time.Time { return t0 },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)

	claim := func(edgeID string) int {
		req := httptest.NewRequest(http.MethodGet, "/agent/v1/jobs/claim?edge_id="+edgeID+"&wait=0s", nil)
		req.Header.Set("Authorization", "Bearer tok")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	codes := make(chan int, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for _, id := range []string{"gpu-a", "gpu-b"} {
		go func(eid string) {
			defer wg.Done()
			codes <- claim(eid)
		}(id)
	}
	wg.Wait()
	close(codes)
	got := map[int]int{}
	for c := range codes {
		got[c]++
	}
	if got[http.StatusOK] != 1 || got[http.StatusNoContent] != 1 {
		t.Fatalf("claim results = %v, want exactly one 200 and one 204", got)
	}
}
