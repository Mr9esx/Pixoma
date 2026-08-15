package agent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/agent"
	runtimedomain "github.com/mr9esx/comfyui_tgbot/internal/runtime/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type statusSpy struct {
	events []sharedkernel.TaskStatusEvent
}

func (s *statusSpy) OnStatus(_ context.Context, ev sharedkernel.TaskStatusEvent) error {
	s.events = append(s.events, ev)
	return nil
}

func mountAgent(t *testing.T, token string, tasks runtimedomain.TaskRepository, status agent.StatusApplier) *httptest.Server {
	t.Helper()
	h := &agent.Handler{
		Token:  token,
		Tasks:  tasks,
		Status: status,
		Lease:  90 * time.Second,
		Now:    func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	r := chi.NewRouter()
	r.Route("/agent/v1", h.Mount)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestAgent_Unauthorized(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	srv := mountAgent(t, "secret-token", tasks, &statusSpy{})
	res, err := http.Get(srv.URL + "/agent/v1/jobs/claim?instance_id=gpu-1&wait=0s")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestAgent_ClaimReturnsJob(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", "c1", "in", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "jobs/t1/job.json"}, now)
	if err := tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	srv := mountAgent(t, "secret-token", tasks, &statusSpy{})

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?instance_id=gpu-1&wait=0s", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["task_id"] != "t1" {
		t.Fatalf("body=%v", body)
	}
	jobRef, _ := body["job_ref"].(map[string]any)
	if jobRef["key"] != "jobs/t1/job.json" {
		t.Fatalf("job_ref=%v", jobRef)
	}
	got, _ := tasks.Get(ctx, "t1")
	if got.Status != sharedkernel.TaskRunning {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestAgent_ClaimEmptyWhenNone(t *testing.T) {
	tasks := runtimedomain.NewMemoryTaskRepository()
	srv := mountAgent(t, "secret-token", tasks, &statusSpy{})
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/agent/v1/jobs/claim?instance_id=gpu-1&wait=0s", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	if res.StatusCode == http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		if len(bytes.TrimSpace(raw)) != 0 && string(raw) != "null" && string(raw) != "{}" {
			// allow empty JSON object / null for "no job"
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err == nil {
				if body["task_id"] != nil && body["task_id"] != "" {
					t.Fatalf("unexpected job %v", body)
				}
			}
		}
	}
}

func TestAgent_StatusReports(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", "c1", "in", now)
	_ = task.PrepareForClaim("gpu-1", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-1", time.Minute, now)
	_ = tasks.Create(ctx, task)
	spy := &statusSpy{}
	srv := mountAgent(t, "tok", tasks, spy)

	payload, _ := json.Marshal(map[string]any{
		"instance_id": "gpu-1",
		"status":      "succeeded",
		"outputs":     []map[string]any{{"key": "out.png"}},
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/jobs/t1/status", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	if len(spy.events) != 1 || spy.events[0].Status != sharedkernel.TaskSucceeded {
		t.Fatalf("events=%+v", spy.events)
	}
}

func TestAgent_StatusRejectsWrongInstance(t *testing.T) {
	ctx := context.Background()
	tasks := runtimedomain.NewMemoryTaskRepository()
	now := time.Unix(1000, 0).UTC()
	task := runtimedomain.NewPending("t1", "s1", "c1", "in", now)
	_ = task.PrepareForClaim("gpu-2", sharedkernel.BlobRef{Key: "j"}, now)
	_ = task.ClaimWithLease("gpu-2", time.Minute, now)
	_ = tasks.Create(ctx, task)
	spy := &statusSpy{}
	srv := mountAgent(t, "tok", tasks, spy)

	payload, _ := json.Marshal(map[string]any{
		"instance_id": "gpu-1",
		"status":      "succeeded",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/agent/v1/jobs/t1/status", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	if len(spy.events) != 0 {
		t.Fatalf("stale status must not apply: %+v", spy.events)
	}
}
