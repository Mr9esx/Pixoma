package metrics

import (
	"testing"
)

func TestParseNvidiaSMIOutput(t *testing.T) {
	out := []byte("0, NVIDIA GeForce RTX 4090, 45, 12345, 24564\n" +
		"1, \"NVIDIA A100, 80GB\", 0, 100, 81920\n")
	gpus, err := parseNvidiaSMIOutput(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 2 {
		t.Fatalf("gpus=%d", len(gpus))
	}
	g0 := gpus[0]
	if g0.Name != "NVIDIA GeForce RTX 4090" || g0.UsagePercent == nil || *g0.UsagePercent != 45 {
		t.Fatalf("g0=%+v", g0)
	}
	if g0.VRAMUsedBytes != 12345<<20 || g0.VRAMTotalBytes != 24564<<20 {
		t.Fatalf("g0 vram=%d/%d", g0.VRAMUsedBytes, g0.VRAMTotalBytes)
	}
	if g0.VRAMUsagePercent == nil {
		t.Fatalf("g0 missing vram pct")
	}
	wantPct := float64(12345) / float64(24564) * 100
	if diff := *g0.VRAMUsagePercent - wantPct; diff > 1e-6 || diff < -1e-6 {
		t.Fatalf("g0 vram pct=%v want %v", *g0.VRAMUsagePercent, wantPct)
	}
	if g1 := gpus[1]; g1.Name != "NVIDIA A100, 80GB" {
		t.Fatalf("g1 name=%q", g1.Name)
	}
}

func TestParseNvidiaSMIOutput_BadRowSkipped(t *testing.T) {
	out := []byte("0, GPU A, not-a-number, 1, 2\n" +
		"1, GPU B, 10, 3, 0\n")
	gpus, err := parseNvidiaSMIOutput(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 0 {
		t.Fatalf("bad rows must be skipped, got %+v", gpus)
	}
}
