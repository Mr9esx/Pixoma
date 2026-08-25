package presence_test

import (
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/presence"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func TestStore_ReportOnlineThenTimeoutClearsComfy(t *testing.T) {
	now := time.Unix(1_000, 0).UTC()
	s := presence.NewStore()
	s.Now = func() time.Time { return now }

	s.Report(sharedkernel.EdgeID("gpu-1"), true)
	got := s.Snapshot("gpu-1")
	if !got.EdgeOnline || !got.ComfyRunning || got.ID != "gpu-1" {
		t.Fatalf("fresh report: %+v", got)
	}

	now = now.Add(15 * time.Second)
	got = s.Snapshot("gpu-1")
	if got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("at 15s must be offline and comfy down: %+v", got)
	}
}

func TestStore_TouchDoesNotSetComfyRunning(t *testing.T) {
	now := time.Unix(1_000, 0).UTC()
	s := presence.NewStore()
	s.Now = func() time.Time { return now }

	s.Touch("gpu-1")
	got := s.Snapshot("gpu-1")
	if !got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("touch-only: %+v", got)
	}

	s.Report("gpu-1", true)
	now = now.Add(10 * time.Second)
	s.Touch("gpu-1")
	now = now.Add(10 * time.Second)
	got = s.Snapshot("gpu-1")
	if !got.EdgeOnline || !got.ComfyRunning {
		t.Fatalf("touch must keep last comfy_running while still online: %+v", got)
	}
}

func TestStore_UnknownInstanceOffline(t *testing.T) {
	s := presence.NewStore()
	got := s.Snapshot("missing")
	if got.ID != "missing" || got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("%+v", got)
	}
}

func TestStore_RemoveClearsRecord(t *testing.T) {
	s := presence.NewStore()
	s.Report("gpu-1", true)
	if got := s.Snapshot("gpu-1"); !got.EdgeOnline {
		t.Fatalf("snapshot before remove: %+v", got)
	}
	s.Remove("gpu-1")
	if got := s.Snapshot("gpu-1"); got.EdgeOnline || got.ComfyRunning {
		t.Fatalf("snapshot after remove: %+v", got)
	}
}
