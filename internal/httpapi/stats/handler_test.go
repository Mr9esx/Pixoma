package stats_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/httpapi/stats"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

type fakeRepo struct {
	days  []taskstats.DailyRow
	errs  []taskstats.ErrorRow
	edges []taskstats.EdgeRow
	cases []taskstats.CaseRow
}

func (f *fakeRepo) AddTerminal(context.Context, taskstats.AddTerminalInput) error { return nil }
func (f *fakeRepo) ListDaily(_ context.Context, _, _ string) ([]taskstats.DailyRow, error) {
	return f.days, nil
}
func (f *fakeRepo) ListErrors(_ context.Context, _, _ string, _ int) ([]taskstats.ErrorRow, error) {
	return f.errs, nil
}
func (f *fakeRepo) ListEdges(_ context.Context, _, _ string) ([]taskstats.EdgeRow, error) {
	return f.edges, nil
}
func (f *fakeRepo) ListCases(_ context.Context, _, _ string, _ int) ([]taskstats.CaseRow, error) {
	return f.cases, nil
}
func (f *fakeRepo) Prune(context.Context, string) error { return nil }

type metricsFake struct {
	latest map[sharedkernel.EdgeID]edge.Metrics
}

func (m *metricsFake) Append(context.Context, sharedkernel.EdgeID, edge.Metrics) error { return nil }
func (m *metricsFake) ListSince(context.Context, sharedkernel.EdgeID, time.Time, int) ([]edge.Metrics, error) {
	return nil, nil
}
func (m *metricsFake) LatestAll(context.Context, time.Time) (map[sharedkernel.EdgeID]edge.Metrics, error) {
	return m.latest, nil
}

func newRouter(repo taskstats.Repository) http.Handler {
	h := &stats.Handler{
		Repo: repo, Loc: time.FixedZone("CST", 8*3600),
		Metrics: &metricsFake{},
	}
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

func TestDailyZeroFillAndSummary(t *testing.T) {
	repo := &fakeRepo{days: []taskstats.DailyRow{
		{Date: "2026-08-20", Processed: 3, Succeeded: 2, Failed: 1, TotalDurationMS: 3000, TotalQueueMS: 1000, TotalExecMS: 2000},
	}}
	rec := httptest.NewRecorder()
	newRouter(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/daily?from=2026-08-19&to=2026-08-21", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Days []struct {
			Date          string `json:"date"`
			Processed     int    `json:"processed"`
			AvgDurationMS *int64 `json:"avg_duration_ms"`
			AvgQueueMS    *int64 `json:"avg_queue_ms"`
			AvgExecMS     *int64 `json:"avg_exec_ms"`
		} `json:"days"`
		Summary struct {
			Processed   int      `json:"processed"`
			SuccessRate *float64 `json:"success_rate"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Days) != 3 || resp.Days[0].Processed != 0 || resp.Days[1].Processed != 3 ||
		resp.Days[2].Processed != 0 {
		t.Fatalf("days=%+v", resp.Days)
	}
	if resp.Days[1].AvgDurationMS == nil || *resp.Days[1].AvgDurationMS != 1000 {
		t.Fatalf("avg=%v", resp.Days[1].AvgDurationMS)
	}
	if resp.Days[0].AvgDurationMS != nil {
		t.Fatalf("empty day avg should be null: %v", resp.Days[0].AvgDurationMS)
	}
	if resp.Days[1].AvgQueueMS == nil || *resp.Days[1].AvgQueueMS != 333 ||
		resp.Days[1].AvgExecMS == nil || *resp.Days[1].AvgExecMS != 666 {
		t.Fatalf("queue/exec avg: %+v", resp.Days[1])
	}
	if resp.Summary.Processed != 3 || resp.Summary.SuccessRate == nil ||
		*resp.Summary.SuccessRate < 0.66 || *resp.Summary.SuccessRate > 0.67 {
		t.Fatalf("summary=%+v", resp.Summary)
	}
}

func TestDailyEmptyRange(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter(&fakeRepo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/daily?from=2026-08-01&to=2026-08-02", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var resp struct {
		Summary struct {
			SuccessRate *float64 `json:"success_rate"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Summary.SuccessRate != nil {
		t.Fatalf("empty success_rate should be null: %v", resp.Summary.SuccessRate)
	}
}

func TestDailyInvalidRanges(t *testing.T) {
	for _, q := range []string{
		"/tasks/daily?from=2026-08-03&to=2026-08-01",
		"/tasks/daily?from=2026/08/01&to=2026-08-01",
		"/tasks/daily?from=2025-01-01&to=2026-08-01",
	} {
		rec := httptest.NewRecorder()
		newRouter(&fakeRepo{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, q, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("query=%s status=%d", q, rec.Code)
		}
	}
}

func TestErrorsTopN(t *testing.T) {
	repo := &fakeRepo{errs: []taskstats.ErrorRow{{ErrorCode: "timeout", Count: 5}, {ErrorCode: "oom", Count: 2}}}
	rec := httptest.NewRecorder()
	newRouter(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/errors?from=2026-08-01&to=2026-08-02&limit=5", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []struct {
			ErrorCode string `json:"error_code"`
			Count     int    `json:"count"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 2 || resp.Items[0].ErrorCode != "timeout" {
		t.Fatalf("items=%+v", resp.Items)
	}
	for _, bad := range []string{"limit=0", "limit=101", "limit=abc"} {
		rec := httptest.NewRecorder()
		newRouter(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/errors?from=2026-08-01&to=2026-08-02&"+bad, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("limit=%s status=%d", bad, rec.Code)
		}
	}
}

func TestEdgesTotal(t *testing.T) {
	repo := &fakeRepo{edges: []taskstats.EdgeRow{
		{EdgeID: "gpu-1", Count: 3, Succeeded: 2, Failed: 1},
		{EdgeID: "gpu-2", Count: 1, Succeeded: 1, Failed: 0},
	}}
	rec := httptest.NewRecorder()
	newRouter(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks/edges?from=2026-08-01&to=2026-08-02", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []struct {
			EdgeID      string   `json:"edge_id"`
			Count       int      `json:"count"`
			Succeeded   int      `json:"succeeded"`
			Failed      int      `json:"failed"`
			SuccessRate *float64 `json:"success_rate"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 2 || resp.Items[0].EdgeID != "gpu-1" || resp.Total != 4 ||
		resp.Items[0].Succeeded != 2 || resp.Items[0].Failed != 1 ||
		resp.Items[0].SuccessRate == nil || *resp.Items[0].SuccessRate < 0.66 || *resp.Items[0].SuccessRate > 0.67 {
		t.Fatalf("resp=%+v", resp)
	}
	if resp.Items[1].SuccessRate == nil || *resp.Items[1].SuccessRate != 1 {
		t.Fatalf("edge2 rate=%v", resp.Items[1].SuccessRate)
	}
}

func TestCasesTop(t *testing.T) {
	repo := &fakeRepo{cases: []taskstats.CaseRow{
		{CaseID: 1, CaseName: "Portrait", Count: 3, TotalDurationMS: 9000},
		{CaseID: 2, CaseName: "Video", Count: 1, TotalDurationMS: 1000},
	}}
	rec := httptest.NewRecorder()
	newRouter(repo).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cases/top?from=2026-08-01&to=2026-08-02&limit=5", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []struct {
			CaseID        uint64 `json:"case_id"`
			CaseName      string `json:"case_name"`
			Count         int    `json:"count"`
			AvgDurationMS *int64 `json:"avg_duration_ms"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 2 || resp.Items[0].CaseID != 1 || resp.Items[0].CaseName != "Portrait" || resp.Items[0].Count != 3 ||
		resp.Items[0].AvgDurationMS == nil || *resp.Items[0].AvgDurationMS != 3000 {
		t.Fatalf("items=%+v", resp.Items)
	}
	rec2 := httptest.NewRecorder()
	newRouter(repo).ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/cases/top?from=2026-08-01&to=2026-08-02&limit=0", nil))
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("limit=0 status=%d", rec2.Code)
	}
}

func TestFleetAggregates(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	gpu := 72.0
	latest := map[sharedkernel.EdgeID]edge.Metrics{
		"gpu-1": {CPUUsagePercent: 78, MemUsagePercent: 64, GPUs: []edge.GPUMetric{{Name: "g0", UsagePercent: &gpu, VRAMUsedBytes: 12 << 30, VRAMTotalBytes: 24 << 30}}},
		"gpu-2": {CPUUsagePercent: 50, MemUsagePercent: 60},
	}
	h := &stats.Handler{Repo: &fakeRepo{}, Loc: loc, Metrics: &metricsFake{latest: latest}}
	r := chi.NewRouter()
	h.Mount(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fleet", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Online         int      `json:"online"`
		AvgCPU         float64  `json:"avg_cpu_usage_percent"`
		VRAMUsedBytes  uint64   `json:"vram_used_bytes"`
		VRAMTotalBytes uint64   `json:"vram_total_bytes"`
		AvgGPU         *float64 `json:"avg_gpu_usage_percent"`
		Hottest        struct {
			EdgeID string  `json:"edge_id"`
			CPU    float64 `json:"cpu_usage_percent"`
		} `json:"hottest"`
		Nodes []struct {
			EdgeID string  `json:"edge_id"`
			CPU    float64 `json:"cpu_usage_percent"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Online != 2 || resp.AvgCPU != 64 || resp.VRAMUsedBytes != 12<<30 || resp.VRAMTotalBytes != 24<<30 ||
		resp.Hottest.EdgeID != "gpu-1" || resp.AvgGPU == nil || *resp.AvgGPU != 72 || len(resp.Nodes) != 2 {
		t.Fatalf("fleet=%+v", resp)
	}
}
