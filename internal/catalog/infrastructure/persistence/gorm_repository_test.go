package persistence_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/catalog/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/catalog/infrastructure/persistence"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

func openTestDB(t *testing.T) *persistence.GormRepository {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file::memory:?cache=shared"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(gdb, &persistence.CaseRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return persistence.NewGormRepository(gdb)
}

func sampleCase(id sharedkernel.CaseID, tags ...string) *domain.Case {
	return &domain.Case{
		Enabled: true,
		Document: domain.CaseDocument{
			ID:   id,
			Name: "Demo " + strconv.FormatUint(uint64(id), 10),
			Tags: tags,
			Inputs: []domain.InputField{
				{Key: "prompt", Type: "string", Required: true},
			},
			Outputs: []domain.OutputField{{Key: "image", Type: "image"}},
			Bindings: domain.ComfyBindings{
				WorkflowJSON: map[string]any{"1": map[string]any{}},
				Inputs:       []domain.InputBinding{{Key: "prompt", NodeID: "1", FieldPath: "text"}},
			},
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string"},
				},
			},
		},
	}
}

func TestCreateGetListDisable(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	c := sampleCase(1, "text2img")
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Create(ctx, c); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}

	got, err := repo.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Document.Name != c.Document.Name || !got.Enabled {
		t.Fatalf("unexpected get: %+v", got)
	}

	c2 := sampleCase(2, "img2img")
	_ = repo.Create(ctx, c2)

	list, err := repo.List(ctx, domain.ListQuery{Tag: "img2img"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Document.ID != 2 {
		t.Fatalf("tag filter: %+v", list)
	}

	if err := repo.Disable(ctx, 1); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got, _ = repo.Get(ctx, 1)
	if got.Enabled {
		t.Fatal("expected disabled")
	}
}

func TestEnableAndListFilters(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()
	c1 := sampleCase(3, "t1")
	c1.Document.Name = "Alpha Workflow"
	c1.Document.Categories = []string{"gen"}
	_ = repo.Create(ctx, c1)
	c2 := sampleCase(4, "t2")
	c2.Document.Name = "Beta Other"
	_ = repo.Create(ctx, c2)

	if err := repo.Disable(ctx, 3); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enable(ctx, 3); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.Get(ctx, 3)
	if !got.Enabled {
		t.Fatal("expected enabled after Enable")
	}
	if err := repo.Enable(ctx, 999999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	list, err := repo.List(ctx, domain.ListQuery{Q: "Alpha", Limit: 10})
	if err != nil || len(list) != 1 || list[0].Document.ID != 3 {
		t.Fatalf("q filter: err=%v list=%+v", err, list)
	}
	en := true
	list, err = repo.List(ctx, domain.ListQuery{Enabled: &en, Category: "gen", Limit: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("enabled+category: err=%v n=%d", err, len(list))
	}
}

func TestRepository_Delete(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	if err := repo.Create(ctx, sampleCase(7)); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.Delete(ctx, 7); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, 7); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
	if err := repo.Delete(ctx, 7); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound for missing delete, got %v", err)
	}
}
