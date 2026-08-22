package stats

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

// Handler serves /api/v1/stats/tasks/* aggregation endpoints.
type Handler struct {
	Repo    taskstats.Repository
	Loc     *time.Location
	Metrics edge.MetricsRepository
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/tasks/daily", h.daily)
	r.Get("/tasks/errors", h.errors)
	r.Get("/tasks/edges", h.edges)
	r.Get("/cases/top", h.casesTop)
	r.Get("/fleet", h.fleet)
}

const (
	dateLayout      = "2006-01-02"
	maxRangeDays    = 365
	defaultRangeDay = 29
)

type dayResp struct {
	Date          string `json:"date"`
	Processed     int    `json:"processed"`
	Succeeded     int    `json:"succeeded"`
	Failed        int    `json:"failed"`
	Cancelled     int    `json:"cancelled"`
	AvgDurationMS *int64 `json:"avg_duration_ms"`
	AvgQueueMS    *int64 `json:"avg_queue_ms"`
	AvgExecMS     *int64 `json:"avg_exec_ms"`
}

type dailyResp struct {
	Range struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"range"`
	Days    []dayResp `json:"days"`
	Summary struct {
		Processed   int      `json:"processed"`
		Succeeded   int      `json:"succeeded"`
		Failed      int      `json:"failed"`
		Cancelled   int      `json:"cancelled"`
		SuccessRate *float64 `json:"success_rate"`
	} `json:"summary"`
}

func (h *Handler) parseRange(r *http.Request) (from, to string, code int, msg string) {
	today := time.Now().In(h.Loc).Format(dateLayout)
	to = r.URL.Query().Get("to")
	if to == "" {
		to = today
	}
	from = r.URL.Query().Get("from")
	if from == "" {
		from = time.Now().In(h.Loc).AddDate(0, 0, -defaultRangeDay).Format(dateLayout)
	}
	tf, err1 := time.ParseInLocation(dateLayout, from, h.Loc)
	tt, err2 := time.ParseInLocation(dateLayout, to, h.Loc)
	if err1 != nil || err2 != nil {
		return "", "", http.StatusBadRequest, "invalid from/to: expected YYYY-MM-DD"
	}
	if tf.After(tt) {
		return "", "", http.StatusBadRequest, "from must not be after to"
	}
	if int(tt.Sub(tf).Hours()/24)+1 > maxRangeDays {
		return "", "", http.StatusBadRequest, "range exceeds 365 days"
	}
	return from, to, 0, ""
}

func (h *Handler) daily(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	rows, err := h.Repo.ListDaily(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	byDate := make(map[string]taskstats.DailyRow, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}
	var resp dailyResp
	resp.Range.From, resp.Range.To = from, to
	cur, _ := time.ParseInLocation(dateLayout, from, h.Loc)
	end, _ := time.ParseInLocation(dateLayout, to, h.Loc)
	for !cur.After(end) {
		d := cur.Format(dateLayout)
		row, ok := byDate[d]
		if !ok {
			row = taskstats.DailyRow{Date: d}
		}
		day := dayResp{
			Date: d, Processed: row.Processed, Succeeded: row.Succeeded,
			Failed: row.Failed, Cancelled: row.Cancelled,
		}
		if row.Processed > 0 {
			avg := row.TotalDurationMS / int64(row.Processed)
			day.AvgDurationMS = &avg
			avgQ := row.TotalQueueMS / int64(row.Processed)
			avgE := row.TotalExecMS / int64(row.Processed)
			day.AvgQueueMS = &avgQ
			day.AvgExecMS = &avgE
		}
		resp.Days = append(resp.Days, day)
		resp.Summary.Processed += row.Processed
		resp.Summary.Succeeded += row.Succeeded
		resp.Summary.Failed += row.Failed
		resp.Summary.Cancelled += row.Cancelled
		cur = cur.AddDate(0, 0, 1)
	}
	if den := resp.Summary.Succeeded + resp.Summary.Failed; den > 0 {
		rate := float64(resp.Summary.Succeeded) / float64(den)
		resp.Summary.SuccessRate = &rate
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) errors(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 100 {
			writeErr(w, http.StatusBadRequest, "invalid limit: must be 1..100")
			return
		}
		limit = n
	}
	rows, err := h.Repo.ListErrors(r.Context(), from, to, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type item struct {
		ErrorCode string `json:"error_code"`
		Count     int    `json:"count"`
	}
	items := make([]item, 0, len(rows))
	for _, row := range rows {
		items = append(items, item{ErrorCode: row.ErrorCode, Count: row.Count})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) edges(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	rows, err := h.Repo.ListEdges(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type item struct {
		EdgeID      string   `json:"edge_id"`
		Count       int      `json:"count"`
		Succeeded   int      `json:"succeeded"`
		Failed      int      `json:"failed"`
		SuccessRate *float64 `json:"success_rate"`
	}
	items := make([]item, 0, len(rows))
	total := 0
	for _, row := range rows {
		it := item{EdgeID: row.EdgeID, Count: row.Count, Succeeded: row.Succeeded, Failed: row.Failed}
		if den := row.Succeeded + row.Failed; den > 0 {
			rate := float64(row.Succeeded) / float64(den)
			it.SuccessRate = &rate
		}
		items = append(items, it)
		total += row.Count
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (h *Handler) casesTop(w http.ResponseWriter, r *http.Request) {
	from, to, code, msg := h.parseRange(r)
	if code != 0 {
		writeErr(w, code, msg)
		return
	}
	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 || n > 20 {
			writeErr(w, http.StatusBadRequest, "invalid limit: must be 1..20")
			return
		}
		limit = n
	}
	rows, err := h.Repo.ListCases(r.Context(), from, to, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type item struct {
		CaseID        uint64 `json:"case_id"`
		Count         int    `json:"count"`
		AvgDurationMS *int64 `json:"avg_duration_ms"`
	}
	items := make([]item, 0, len(rows))
	for _, row := range rows {
		it := item{CaseID: row.CaseID, Count: row.Count}
		if row.Count > 0 {
			avg := row.TotalDurationMS / int64(row.Count)
			it.AvgDurationMS = &avg
		}
		items = append(items, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) fleet(w http.ResponseWriter, r *http.Request) {
	if h.Metrics == nil {
		writeErr(w, http.StatusInternalServerError, "metrics not configured")
		return
	}
	since := time.Now().UTC().Add(-24 * time.Hour)
	latest, err := h.Metrics.LatestAll(r.Context(), since)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type node struct {
		EdgeID          string   `json:"edge_id"`
		CPUUsagePercent float64  `json:"cpu_usage_percent"`
		MemUsagePercent float64  `json:"mem_usage_percent"`
		GPUUsagePercent *float64 `json:"gpu_usage_percent,omitempty"`
		VRAMUsedBytes   uint64   `json:"vram_used_bytes,omitempty"`
		VRAMTotalBytes  uint64   `json:"vram_total_bytes,omitempty"`
	}
	var nodes []node
	var cpuSum, memSum float64
	var gpuSum float64
	var gpuN int
	var vramUsed, vramTotal uint64
	var hottest *node
	for id, m := range latest {
		n := node{
			EdgeID:          string(id),
			CPUUsagePercent: m.CPUUsagePercent,
			MemUsagePercent: m.MemUsagePercent,
		}
		var gpuAcc float64
		var gpuCnt int
		for _, g := range m.GPUs {
			if g.UsagePercent != nil {
				gpuAcc += *g.UsagePercent
				gpuCnt++
			}
			vramUsed += g.VRAMUsedBytes
			vramTotal += g.VRAMTotalBytes
		}
		if gpuCnt > 0 {
			avg := gpuAcc / float64(gpuCnt)
			n.GPUUsagePercent = &avg
			gpuSum += avg
			gpuN++
		}
		n.VRAMUsedBytes = vramUsed
		n.VRAMTotalBytes = vramTotal
		nodes = append(nodes, n)
		cpuSum += m.CPUUsagePercent
		memSum += m.MemUsagePercent
		if hottest == nil || m.CPUUsagePercent > hottest.CPUUsagePercent {
			hottest = &nodes[len(nodes)-1]
		}
	}
	out := map[string]any{
		"online":                len(nodes),
		"avg_cpu_usage_percent": avg(cpuSum, len(nodes)),
		"avg_mem_usage_percent": avg(memSum, len(nodes)),
		"vram_used_bytes":       vramUsed,
		"vram_total_bytes":      vramTotal,
		"nodes":                 nodes,
	}
	if gpuN > 0 {
		g := gpuSum / float64(gpuN)
		out["avg_gpu_usage_percent"] = &g
	}
	if hottest != nil {
		out["hottest"] = map[string]any{"edge_id": hottest.EdgeID, "cpu_usage_percent": hottest.CPUUsagePercent}
	}
	writeJSON(w, http.StatusOK, out)
}

func avg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
