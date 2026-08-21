package stats

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/taskstats"
)

// Handler serves /api/v1/stats/tasks/* aggregation endpoints.
type Handler struct {
	Repo taskstats.Repository
	Loc  *time.Location
}

func (h *Handler) Mount(r chi.Router) {
	r.Get("/tasks/daily", h.daily)
	r.Get("/tasks/errors", h.errors)
	r.Get("/tasks/edges", h.edges)
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
}

type dailyResp struct {
	Range struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"range"`
	Days []dayResp `json:"days"`
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
		EdgeID string `json:"edge_id"`
		Count  int    `json:"count"`
	}
	items := make([]item, 0, len(rows))
	total := 0
	for _, row := range rows {
		items = append(items, item{EdgeID: row.EdgeID, Count: row.Count})
		total += row.Count
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
