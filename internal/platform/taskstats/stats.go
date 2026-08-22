package taskstats

import (
	"context"
	"time"
)

// Status is a terminal task status recorded into daily stats.
type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// AddTerminalInput carries one terminal task transition to be recorded.
type AddTerminalInput struct {
	EdgeID      string
	ErrorCode   string
	Status      Status
	CompletedAt time.Time
	CreatedAt   time.Time
	CaseID      uint64
	// QueueDurationMS is created→started; ExecDurationMS is started→completed.
	QueueDurationMS int64
	ExecDurationMS  int64
}

// DailyRow is one day's aggregated terminal task counts.
type DailyRow struct {
	Date            string
	Processed       int
	Succeeded       int
	Failed          int
	Cancelled       int
	TotalDurationMS int64
	TotalQueueMS    int64
	TotalExecMS     int64
}

// ErrorRow is the aggregated count of one error code in a range.
type ErrorRow struct {
	ErrorCode string
	Count     int
}

// EdgeRow is the aggregated processed count of one edge in a range.
type EdgeRow struct {
	EdgeID    string
	Count     int
	Succeeded int
	Failed    int
}

// CaseRow is the aggregated processed count and duration of one case.
type CaseRow struct {
	CaseID          uint64
	Count           int
	TotalDurationMS int64
}

// Repository persists terminal task rollups and serves range queries.
type Repository interface {
	AddTerminal(ctx context.Context, in AddTerminalInput) error
	ListDaily(ctx context.Context, from, to string) ([]DailyRow, error)
	ListErrors(ctx context.Context, from, to string, limit int) ([]ErrorRow, error)
	ListEdges(ctx context.Context, from, to string) ([]EdgeRow, error)
	ListCases(ctx context.Context, from, to string, limit int) ([]CaseRow, error)
	Prune(ctx context.Context, before string) error
}

// DateOf buckets t into the configured timezone's calendar day.
func DateOf(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}
