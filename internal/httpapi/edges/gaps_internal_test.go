package edges

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
)

func TestFillMetricsGaps(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	since := now.Add(-30 * time.Minute)
	until := now
	series := []edge.Metrics{
		{CPUUsagePercent: 11, CollectedAt: since.Add(2 * time.Minute)},
		{CPUUsagePercent: 22, CollectedAt: since.Add(12 * time.Minute)},
	}
	filled := fillMetricsGaps(series, since, until, time.Minute)
	if len(filled) != 30 {
		t.Fatalf("want 30 buckets, got %d", len(filled))
	}
	if filled[2].CPUUsagePercent != 11 || filled[12].CPUUsagePercent != 22 {
		t.Fatalf("real samples misplaced: %+v", filled)
	}
	if filled[0].CPUUsagePercent != 0 || filled[0].DiskReadBytesPerSec == nil || *filled[0].DiskReadBytesPerSec != 0 {
		t.Fatalf("zero bucket must carry zero disk rates: %+v", filled[0])
	}
	if filled[0].CollectedAt.Sub(since) != 30*time.Second {
		t.Fatalf("zero bucket timestamp should be bucket midpoint: %v", filled[0].CollectedAt)
	}
	if got := fillMetricsGaps(nil, since, until, time.Minute); got != nil {
		t.Fatalf("empty series must stay empty, got %v", got)
	}
}

func TestMetricsBucketSize(t *testing.T) {
	cases := []struct {
		window time.Duration
		want   time.Duration
	}{
		{time.Hour, 30 * time.Second},
		{6 * time.Hour, time.Minute},
		{24 * time.Hour, 5 * time.Minute},
		{7 * 24 * time.Hour, 30 * time.Minute},
		{30 * 24 * time.Hour, 2 * time.Hour},
		{90 * 24 * time.Hour, 6 * time.Hour},
	}
	for _, tc := range cases {
		if got := metricsBucketSize(tc.window); got != tc.want {
			t.Fatalf("bucket(%v)=%v want %v", tc.window, got, tc.want)
		}
	}
}
