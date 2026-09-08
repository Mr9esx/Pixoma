package persistence_test

import (
	"context"
	"testing"
	"time"

	edge "github.com/Mr9esx/Pixoma/internal/edge/domain"
	"github.com/Mr9esx/Pixoma/internal/edge/infrastructure/persistence"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
)

func TestEdgeRepository_SubscribeTopicsRoundTrip(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	rec := &edge.Record{
		ID:              sharedkernel.EdgeID("gpu-1"),
		Name:            "gpu-1",
		Enabled:         true,
		SubscribeTopics: []string{"default", "fast-gpu"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.Upsert(ctx, rec); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repo.Get(ctx, "gpu-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.SubscribeTopics) != 2 || got.SubscribeTopics[0] != "default" || got.SubscribeTopics[1] != "fast-gpu" {
		t.Fatalf("subscribe topics = %v", got.SubscribeTopics)
	}
}

func TestEdgeRepository_UpdateSubscribeTopics(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewEdgeRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	rec := &edge.Record{
		ID:              sharedkernel.EdgeID("gpu-2"),
		Name:            "gpu-2",
		Enabled:         true,
		SubscribeTopics: []string{"default"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_ = repo.Upsert(ctx, rec)
	if err := repo.UpdateSubscribeTopics(ctx, "gpu-2", []string{"fast-gpu"}); err != nil {
		t.Fatalf("update subscribe topics: %v", err)
	}
	got, _ := repo.Get(ctx, "gpu-2")
	if len(got.SubscribeTopics) != 1 || got.SubscribeTopics[0] != "fast-gpu" {
		t.Fatalf("after update = %v", got.SubscribeTopics)
	}
	if err := repo.UpdateSubscribeTopics(ctx, "missing", []string{"x"}); err == nil {
		t.Fatal("expected ErrNotFound for missing edge")
	}
}
