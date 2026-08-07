package orchestrator

import "time"

// NextBackoff returns exponential backoff with simple jitter based on attempt.
// false means give up (attempt too high).
func NextBackoff(attempt int, base, cap time.Duration) (time.Duration, bool) {
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 16 {
		return 0, false
	}
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
		if d >= cap {
			d = cap
			break
		}
	}
	j := d / 8
	if attempt%2 == 0 {
		d += j
	} else if d > j {
		d -= j
	}
	if d > cap {
		d = cap
	}
	return d, true
}
