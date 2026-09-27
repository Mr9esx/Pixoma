package persistence_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/platform/db"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type legacyLibraryFolderRow struct {
	ID        string `gorm:"primaryKey"`
	AccountID string
	ParentID  string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (legacyLibraryFolderRow) TableName() string { return "studio_library_folders" }

type legacyLibraryAssetRow struct {
	ID             uint64 `gorm:"primaryKey;autoIncrement"`
	AccountID      string
	AssetID        string
	AssetVersionID string
	FolderID       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (legacyLibraryAssetRow) TableName() string { return "studio_library_assets" }

func TestMigrateLibraryCategoriesPreservesAssignments(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, err := gdb.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := gdb.AutoMigrate(&legacyLibraryFolderRow{}, &legacyLibraryAssetRow{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	for _, category := range []legacyLibraryFolderRow{
		{ID: "parent", AccountID: "account-a", Name: "故事", CreatedAt: now, UpdatedAt: now},
		{ID: "child", AccountID: "account-a", ParentID: "parent", Name: "人物", CreatedAt: now, UpdatedAt: now},
	} {
		if err := gdb.Create(&category).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := gdb.Create(&legacyLibraryAssetRow{
		AccountID: "account-a", AssetID: "asset-a", AssetVersionID: "version-a", FolderID: "child", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := persistence.MigrateLibraryCategories(ctx, gdb); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	categories, err := repo.ListLibraryCategories(ctx, "account-a")
	if err != nil || len(categories) != 2 {
		t.Fatalf("categories = (%#v, %v)", categories, err)
	}
	parents := make(map[string]string, len(categories))
	for _, category := range categories {
		parents[category.ID] = category.ParentID
	}
	_, hasParent := parents["parent"]
	if !hasParent || parents["child"] != "parent" {
		t.Fatalf("category hierarchy = %#v", categories)
	}
	var assignment persistence.LibraryAssetRow
	if err := gdb.Where("account_id = ? AND asset_id = ?", "account-a", "asset-a").First(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if assignment.CategoryID != "child" {
		t.Fatalf("category = %q", assignment.CategoryID)
	}
	if err := repo.MoveLibraryAsset(ctx, "account-a", "asset-a", "", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := persistence.MigrateLibraryCategories(ctx, gdb); err != nil {
		t.Fatal(err)
	}
	if err := gdb.Where("account_id = ? AND asset_id = ?", "account-a", "asset-a").First(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if assignment.CategoryID != "" {
		t.Fatalf("category after repeat migration = %q", assignment.CategoryID)
	}
}
