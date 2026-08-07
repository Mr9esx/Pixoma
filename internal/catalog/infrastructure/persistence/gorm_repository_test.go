package persistence_test

import (
	"context"
	"errors"
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

func sampleCase(id string, tags ...string) *domain.Case {
	return &domain.Case{
		Enabled: true,
		Document: domain.CaseDocument{
			ID:   sharedkernel.CaseID(id),
			Name: "Demo " + id,
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
			MenuKey: "create",
		},
	}
}

func TestCreateGetListDisable(t *testing.T) {
	repo := openTestDB(t)
	ctx := context.Background()

	c := sampleCase("text2img-demo", "text2img")
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Create(ctx, c); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}

	got, err := repo.Get(ctx, "text2img-demo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Document.Name != c.Document.Name || !got.Enabled {
		t.Fatalf("unexpected get: %+v", got)
	}

	c2 := sampleCase("img2img-demo", "img2img")
	_ = repo.Create(ctx, c2)

	list, err := repo.List(ctx, domain.ListQuery{Tag: "img2img"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Document.ID != "img2img-demo" {
		t.Fatalf("tag filter: %+v", list)
	}

	if err := repo.Disable(ctx, "text2img-demo"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	got, _ = repo.Get(ctx, "text2img-demo")
	if got.Enabled {
		t.Fatal("expected disabled")
	}
}
