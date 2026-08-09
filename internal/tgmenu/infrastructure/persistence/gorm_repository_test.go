package persistence_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/infrastructure/persistence"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:tgmenu_" + t.Name() + "?mode=memory&cache=shared"
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	return gdb
}

func migrateMenuTables(t *testing.T, gdb *gorm.DB, extra ...any) {
	t.Helper()
	models := []any{
		&persistence.MenuHeaderRow{},
		&persistence.MenuItemRow{},
		&persistence.MenuItemCaseRow{},
	}
	models = append(models, extra...)
	if err := db.AutoMigrate(gdb, models...); err != nil {
		t.Fatal(err)
	}
}

func TestReplaceTree_RoundTripAndPlacements(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewGormRepository(gdb)
	migrateMenuTables(t, gdb)
	tree := domain.DefaultSeedTree()
	tree.Items[0].CaseIDs = []string{"case-a"}
	tree.Items[0].Children = []domain.MenuNode{{
		ID: "folder-x", ParentID: "btn-image", Label: "子夹", Enabled: true, Kind: domain.KindFolder,
	}}
	if err := repo.ReplaceTree(context.Background(), tree); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTree(context.Background(), domain.DocumentIDDefault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items[0].CaseIDs) != 1 || got.Items[0].CaseIDs[0] != "case-a" {
		t.Fatalf("case ids: %+v", got.Items[0].CaseIDs)
	}
	if len(got.Items[0].Children) != 1 {
		t.Fatalf("children=%d", len(got.Items[0].Children))
	}

	ps, err := repo.ListPlacementsByCase(context.Background(), "case-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 {
		t.Fatalf("placements=%d", len(ps))
	}
	if ps[0].ItemID != "btn-image" {
		t.Fatalf("item_id=%q", ps[0].ItemID)
	}
	if ps[0].Path[len(ps[0].Path)-1].Label != "🖼 图片" {
		t.Fatalf("path label=%q", ps[0].Path[len(ps[0].Path)-1].Label)
	}
}

func TestEnsureDefault_SeedsWhenEmpty(t *testing.T) {
	gdb := openTestDB(t)
	migrateMenuTables(t, gdb)
	repo := persistence.NewGormRepository(gdb)

	tree, err := repo.EnsureDefault(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if tree.ID != domain.DocumentIDDefault {
		t.Fatalf("id=%q", tree.ID)
	}
	if len(tree.Items) != 6 {
		t.Fatalf("items=%d", len(tree.Items))
	}
	if tree.Items[0].Kind != domain.KindFolder {
		t.Fatalf("kind=%q", tree.Items[0].Kind)
	}

	again, err := repo.EnsureDefault(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if again.Items[0].Label != tree.Items[0].Label {
		t.Fatalf("label changed: %q -> %q", tree.Items[0].Label, again.Items[0].Label)
	}
}

func TestEnsureDefault_AppliesImageCaseIDs(t *testing.T) {
	gdb := openTestDB(t)
	migrateMenuTables(t, gdb)
	repo := persistence.NewGormRepository(gdb)

	list := func(context.Context) ([]string, error) {
		return []string{"img-1", "img-2"}, nil
	}
	tree, err := repo.EnsureDefault(context.Background(), list)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"img-1", "img-2"}
	if len(tree.Items[0].CaseIDs) != len(want) {
		t.Fatalf("case ids=%v", tree.Items[0].CaseIDs)
	}
	for i, id := range want {
		if tree.Items[0].CaseIDs[i] != id {
			t.Fatalf("case ids=%v", tree.Items[0].CaseIDs)
		}
	}
}

func TestEnsureDefault_MigratesLegacyJSON(t *testing.T) {
	gdb := openTestDB(t)
	migrateMenuTables(t, gdb, &persistence.LegacyMenuRow{})
	legacyItems := []map[string]any{
		{
			"id": "btn-go", "label": "Go", "row": 0, "col": 0, "enabled": true,
			"action": "open_case", "case_id": "legacy-case",
		},
	}
	raw, err := json.Marshal(legacyItems)
	if err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&persistence.LegacyMenuRow{
		ID:        domain.DocumentIDDefault,
		ItemsJSON: string(raw),
	}).Error; err != nil {
		t.Fatal(err)
	}

	repo := persistence.NewGormRepository(gdb)
	tree, err := repo.EnsureDefault(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Items) != 1 {
		t.Fatalf("items=%d", len(tree.Items))
	}
	if tree.Items[0].ID != "btn-go" {
		t.Fatalf("id=%q", tree.Items[0].ID)
	}
	if tree.Items[0].Kind != domain.KindOpenCase {
		t.Fatalf("kind=%q", tree.Items[0].Kind)
	}
	if len(tree.Items[0].CaseIDs) != 1 || tree.Items[0].CaseIDs[0] != "legacy-case" {
		t.Fatalf("case ids=%v", tree.Items[0].CaseIDs)
	}
}
