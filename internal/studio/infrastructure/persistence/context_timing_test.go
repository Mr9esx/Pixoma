package persistence

import (
	"testing"
	"time"
)

func TestContextTimingBreakdown(t *testing.T) {
	start := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	interval := func(from, to int) contextInterval {
		return contextInterval{start.Add(time.Duration(from) * time.Second), start.Add(time.Duration(to) * time.Second)}
	}
	timing := contextTimingBreakdown(
		[]contextInterval{interval(0, 10)},
		[]contextInterval{interval(7, 8)},
		[]contextInterval{interval(1, 4)},
		nil,
		nil,
		[]contextInterval{interval(3, 6)},
	)
	if timing.ActiveMS != 9000 || timing.ModelWaitMS != 2000 || timing.ToolsMS != 2000 || timing.OverlapMS != 1000 || timing.OtherMS != 4000 {
		t.Fatalf("timing = %#v", timing)
	}
}
