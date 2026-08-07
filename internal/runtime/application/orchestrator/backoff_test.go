package orchestrator_test

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/runtime/application/orchestrator"
)

func TestNextBackoffGrowsThenCaps(t *testing.T) {
	d0, ok := orchestrator.NextBackoff(0, time.Second, time.Minute)
	if !ok || d0 < time.Second {
		t.Fatalf("d0=%v ok=%v", d0, ok)
	}
	d3, ok := orchestrator.NextBackoff(3, time.Second, time.Minute)
	if !ok || d3 <= d0 {
		t.Fatalf("d3=%v d0=%v", d3, d0)
	}
	d20, ok := orchestrator.NextBackoff(20, time.Second, time.Minute)
	if ok || d20 != 0 {
		t.Fatalf("want give up, got %v %v", d20, ok)
	}
}
