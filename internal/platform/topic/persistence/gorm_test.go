package persistence_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/topic/persistence"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := "topic-test-" + strings.ReplaceAll(t.Name(), "/", "-")
	gdb, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := gdb.AutoMigrate(&persistence.TopicRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gdb
}

func TestTopicRepository_RoundTrip(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewTopicRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Create(ctx, topic.Topic{Key: "fast-gpu", Name: "Fast GPU", Enabled: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, "fast-gpu")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Key != "fast-gpu" || got.Name != "Fast GPU" || !got.Enabled {
		t.Fatalf("unexpected topic: %+v", got)
	}
}

func TestTopicRepository_ListFiltersEnabled(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewTopicRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = repo.Create(ctx, topic.Topic{Key: "a", Name: "A", Enabled: true, CreatedAt: now, UpdatedAt: now})
	_ = repo.Create(ctx, topic.Topic{Key: "b", Name: "B", Enabled: false, CreatedAt: now, UpdatedAt: now})

	all, err := repo.List(ctx, nil)
	if err != nil || len(all) != 2 {
		t.Fatalf("list all: %v len=%d", err, len(all))
	}
	enabled := true
	on, err := repo.List(ctx, &enabled)
	if err != nil || len(on) != 1 || on[0].Key != "a" {
		t.Fatalf("list enabled: %v %+v", err, on)
	}
}

func TestTopicRepository_CreateDuplicateConflict(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewTopicRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = repo.Create(ctx, topic.Topic{Key: "dup", Name: "D", Enabled: true, CreatedAt: now, UpdatedAt: now})
	err := repo.Create(ctx, topic.Topic{Key: "dup", Name: "D2", Enabled: true, CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, topic.ErrTopicConflict) {
		t.Fatalf("expected ErrTopicConflict, got %v", err)
	}
}

func TestTopicRepository_UpdateAndDelete(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewTopicRepository(gdb)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = repo.Create(ctx, topic.Topic{Key: "k", Name: "K", Enabled: true, CreatedAt: now, UpdatedAt: now})
	if err := repo.Update(ctx, topic.Topic{Key: "k", Name: "K2", Enabled: false, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := repo.Get(ctx, "k")
	if got.Name != "K2" || got.Enabled {
		t.Fatalf("update not applied: %+v", got)
	}
	if err := repo.Delete(ctx, "k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, "k"); !errors.Is(err, topic.ErrTopicNotFound) {
		t.Fatalf("expected ErrTopicNotFound after delete, got %v", err)
	}
	if err := repo.Delete(ctx, "k"); !errors.Is(err, topic.ErrTopicNotFound) {
		t.Fatalf("expected ErrTopicNotFound on missing delete, got %v", err)
	}
}
