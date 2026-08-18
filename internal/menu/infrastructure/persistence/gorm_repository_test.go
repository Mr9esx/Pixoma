package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/db"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
	"github.com/mr9esx/comfyui_tgbot/internal/menu/infrastructure/persistence"
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
		&persistence.ChannelMenuRow{},
		&persistence.ChannelMenuItemRow{},
		&persistence.ChannelMenuItemCaseRow{},
		&persistence.ChannelMenuExtraRow{},
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
	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].CaseIDs = []string{"case-a"}
	tree.Items[0].Children = []domain.MenuNode{{
		ID: "folder-x", ParentID: "btn-image", Label: "子夹", Enabled: true, Kind: domain.KindFolder,
	}}
	if err := repo.ReplaceTree(context.Background(), tree); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTree(context.Background(), "tg-default")
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

func TestReplaceTree_PreservesIntroText(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewGormRepository(gdb)
	migrateMenuTables(t, gdb)

	tree := domain.DefaultSeedTree("tg-default")
	tree.Items[0].IntroText = "点模板先看预览图，再上传图片生成。"
	if err := repo.ReplaceTree(context.Background(), tree); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTree(context.Background(), "tg-default")
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0].IntroText != tree.Items[0].IntroText {
		t.Fatalf("intro_text=%q", got.Items[0].IntroText)
	}
}

func TestGetTree_PreservesRootAndChildOrder(t *testing.T) {
	gdb := openTestDB(t)
	repo := persistence.NewGormRepository(gdb)
	migrateMenuTables(t, gdb)

	tree := domain.MenuTree{
		ChannelID: "tg-default",
		Items: []domain.MenuNode{
			{
				ID: "root-z", Label: "Z", Order: 1, Enabled: true, Kind: domain.KindPlaceholder,
			},
			{
				ID: "root-a", Label: "A", Order: 0, Enabled: true, Kind: domain.KindFolder,
				CaseIDs: []string{"case-b", "case-a"},
				Children: []domain.MenuNode{
					{ID: "child-b", Label: "B", Order: 1, Enabled: true, Kind: domain.KindFolder},
					{ID: "child-a", Label: "A", Order: 0, Enabled: true, Kind: domain.KindFolder},
				},
			},
		},
	}
	if err := repo.ReplaceTree(context.Background(), tree); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTree(context.Background(), "tg-default")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("roots=%d", len(got.Items))
	}
	if got.Items[0].ID != "root-a" || got.Items[1].ID != "root-z" {
		t.Fatalf("root order: %q, %q", got.Items[0].ID, got.Items[1].ID)
	}
	if len(got.Items[0].Children) != 2 {
		t.Fatalf("children=%d", len(got.Items[0].Children))
	}
	if got.Items[0].Children[0].ID != "child-a" || got.Items[0].Children[1].ID != "child-b" {
		t.Fatalf("child order: %q, %q", got.Items[0].Children[0].ID, got.Items[0].Children[1].ID)
	}
	wantCases := []string{"case-b", "case-a"}
	for i, id := range wantCases {
		if got.Items[0].CaseIDs[i] != id {
			t.Fatalf("case order=%v", got.Items[0].CaseIDs)
		}
	}
}

func TestEnsureDefault_SeedsWhenEmpty(t *testing.T) {
	gdb := openTestDB(t)
	migrateMenuTables(t, gdb)
	repo := persistence.NewGormRepository(gdb)

	tree, err := repo.EnsureDefault(context.Background(), "tg-default", nil)
	if err != nil {
		t.Fatal(err)
	}
	if tree.ChannelID != "tg-default" {
		t.Fatalf("id=%q", tree.ChannelID)
	}
	if len(tree.Items) != 6 {
		t.Fatalf("items=%d", len(tree.Items))
	}
	if tree.Items[0].Kind != domain.KindFolder {
		t.Fatalf("kind=%q", tree.Items[0].Kind)
	}

	again, err := repo.EnsureDefault(context.Background(), "tg-default", nil)
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
	tree, err := repo.EnsureDefault(context.Background(), "tg-default", list)
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

func TestChannelMenuExtras_IsolatedAndPerItem(t *testing.T) {
	gdb := openTestDB(t)
	migrateMenuTables(t, gdb)
	repo := persistence.NewGormRepository(gdb)
	ctx := context.Background()

	tree := domain.DefaultSeedTree("tg-default")
	if err := repo.ReplaceTree(ctx, tree); err != nil {
		t.Fatal(err)
	}

	extras := map[string][]domain.Extra{
		"btn-image": {{ChannelID: "tg-default", MenuItemID: "btn-image", ExtraType: "tg_root_layout", ExtraJSON: `{"columns":2}`}},
	}
	if err := repo.SaveExtras(ctx, "tg-default", extras); err != nil {
		t.Fatal(err)
	}

	got, err := repo.ListExtras(ctx, "tg-default")
	if err != nil {
		t.Fatal(err)
	}
	if len(got["btn-image"]) != 1 || got["btn-image"][0].ExtraType != "tg_root_layout" {
		t.Fatalf("extras=%+v", got)
	}

	// 跨渠道隔离
	other, err := repo.ListExtras(ctx, "feishu-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatalf("cross-channel leak: %+v", other)
	}

	// 覆盖保存（同 item 同 type 替换）
	extras["btn-image"] = []domain.Extra{{ChannelID: "tg-default", MenuItemID: "btn-image", ExtraType: "tg_root_layout", ExtraJSON: `{"columns":3}`}}
	if err := repo.SaveExtras(ctx, "tg-default", extras); err != nil {
		t.Fatal(err)
	}
	got2, _ := repo.ListExtras(ctx, "tg-default")
	if len(got2["btn-image"]) != 1 || got2["btn-image"][0].ExtraJSON != `{"columns":3}` {
		t.Fatalf("replace extras=%+v", got2)
	}
}

func TestChannelMenuItem_UniqueOrderWithinParent(t *testing.T) {
	gdb := openTestDB(t)
	migrateMenuTables(t, gdb)

	now := time.Now().UTC()
	if err := gdb.Create(&persistence.ChannelMenuRow{ChannelID: "tg-default", UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	one := &persistence.ChannelMenuItemRow{ID: "a", ChannelID: "tg-default", Label: "A", Order: 0, Enabled: true, Kind: string(domain.KindPlaceholder)}
	two := &persistence.ChannelMenuItemRow{ID: "b", ChannelID: "tg-default", Label: "B", Order: 0, Enabled: true, Kind: string(domain.KindPlaceholder)}
	if err := gdb.Create(one).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(two).Error; err == nil {
		t.Fatal("duplicate order within same parent must be rejected by DB")
	}
}
