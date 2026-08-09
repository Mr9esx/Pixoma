package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/tgmenu/domain"
)

type MenuHeaderRow struct {
	ID        string    `gorm:"primaryKey;size:64"`
	BotID     string    `gorm:"size:64;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (MenuHeaderRow) TableName() string { return "tg_menus" }

type MenuItemRow struct {
	ID              string  `gorm:"primaryKey;size:128"`
	MenuID          string  `gorm:"size:64;not null;index"`
	ParentID        *string `gorm:"size:128;index"`
	Label           string  `gorm:"size:256;not null"`
	Row             int     `gorm:"not null"`
	Col             int     `gorm:"not null"`
	Enabled         bool    `gorm:"not null"`
	Kind            string  `gorm:"size:64;not null"`
	PlaceholderText string  `gorm:"size:512"`
	Tag             string  `gorm:"size:128"`
	ReplyJSON       string  `gorm:"type:text"`
}

func (MenuItemRow) TableName() string { return "tg_menu_items" }

type MenuItemCaseRow struct {
	MenuItemID string `gorm:"primaryKey;size:128"`
	CaseID     string `gorm:"primaryKey;size:128"`
	Sort       int    `gorm:"not null"`
}

func (MenuItemCaseRow) TableName() string { return "tg_menu_item_cases" }

// LegacyMenuRow is the old JSON-backed menu config table used for one-time migration.
type LegacyMenuRow struct {
	ID        string    `gorm:"primaryKey;size:64"`
	ItemsJSON string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (LegacyMenuRow) TableName() string { return "tg_menu_configs" }

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetTree(ctx context.Context, id string) (domain.MenuTree, error) {
	var header MenuHeaderRow
	err := r.db.WithContext(ctx).First(&header, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.MenuTree{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MenuTree{}, err
	}

	var itemRows []MenuItemRow
	if err := r.db.WithContext(ctx).Where("menu_id = ?", id).Find(&itemRows).Error; err != nil {
		return domain.MenuTree{}, err
	}

	itemIDs := make([]string, 0, len(itemRows))
	for _, row := range itemRows {
		itemIDs = append(itemIDs, row.ID)
	}

	caseByItem := map[string][]string{}
	if len(itemIDs) > 0 {
		var caseRows []MenuItemCaseRow
		if err := r.db.WithContext(ctx).
			Where("menu_item_id IN ?", itemIDs).
			Order("menu_item_id, sort").
			Find(&caseRows).Error; err != nil {
			return domain.MenuTree{}, err
		}
		for _, row := range caseRows {
			caseByItem[row.MenuItemID] = append(caseByItem[row.MenuItemID], row.CaseID)
		}
	}

	items := make([]domain.MenuItem, 0, len(itemRows))
	for _, row := range itemRows {
		item, err := rowToItem(row, caseByItem[row.ID])
		if err != nil {
			return domain.MenuTree{}, err
		}
		items = append(items, item)
	}

	nodes, err := domain.BuildTree(items)
	if err != nil {
		return domain.MenuTree{}, err
	}

	return domain.MenuTree{
		ID:        header.ID,
		BotID:     header.BotID,
		Items:     nodes,
		UpdatedAt: header.UpdatedAt,
	}, nil
}

func (r *GormRepository) ReplaceTree(ctx context.Context, tree domain.MenuTree) error {
	if tree.ID == "" {
		tree.ID = domain.DocumentIDDefault
	}
	if tree.BotID == "" {
		tree.BotID = domain.BotIDDefault
	}

	flat := domain.Flatten(tree.Items)
	now := time.Now().UTC()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var itemIDs []string
		if err := tx.Model(&MenuItemRow{}).
			Where("menu_id = ?", tree.ID).
			Pluck("id", &itemIDs).Error; err != nil {
			return err
		}
		if len(itemIDs) > 0 {
			if err := tx.Where("menu_item_id IN ?", itemIDs).Delete(&MenuItemCaseRow{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("menu_id = ?", tree.ID).Delete(&MenuItemRow{}).Error; err != nil {
			return err
		}

		header := MenuHeaderRow{
			ID:        tree.ID,
			BotID:     tree.BotID,
			UpdatedAt: now,
		}
		if err := tx.Save(&header).Error; err != nil {
			return err
		}

		for _, item := range flat {
			row, err := itemToRow(tree.ID, item)
			if err != nil {
				return err
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			for sort, caseID := range item.CaseIDs {
				if err := tx.Create(&MenuItemCaseRow{
					MenuItemID: item.ID,
					CaseID:     caseID,
					Sort:       sort,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *GormRepository) ListPlacementsByCase(ctx context.Context, caseID string) ([]domain.MenuPlacement, error) {
	var links []MenuItemCaseRow
	if err := r.db.WithContext(ctx).
		Where("case_id = ?", caseID).
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}

	itemIDs := make([]string, 0, len(links))
	seen := map[string]struct{}{}
	for _, link := range links {
		if _, ok := seen[link.MenuItemID]; ok {
			continue
		}
		seen[link.MenuItemID] = struct{}{}
		itemIDs = append(itemIDs, link.MenuItemID)
	}

	var itemRows []MenuItemRow
	if err := r.db.WithContext(ctx).Where("id IN ?", itemIDs).Find(&itemRows).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]MenuItemRow, len(itemRows))
	menuIDs := map[string]struct{}{}
	for _, row := range itemRows {
		byID[row.ID] = row
		menuIDs[row.MenuID] = struct{}{}
	}

	menuItemRows := map[string][]MenuItemRow{}
	for menuID := range menuIDs {
		var rows []MenuItemRow
		if err := r.db.WithContext(ctx).Where("menu_id = ?", menuID).Find(&rows).Error; err != nil {
			return nil, err
		}
		menuItemRows[menuID] = rows
	}

	placements := make([]domain.MenuPlacement, 0, len(links))
	for _, link := range links {
		item, ok := byID[link.MenuItemID]
		if !ok {
			continue
		}
		path, err := buildPlacementPath(menuItemRows[item.MenuID], item.ID)
		if err != nil {
			return nil, err
		}
		placements = append(placements, domain.MenuPlacement{
			MenuID: item.MenuID,
			ItemID: item.ID,
			Path:   path,
		})
	}

	sort.Slice(placements, func(i, j int) bool {
		if placements[i].MenuID != placements[j].MenuID {
			return placements[i].MenuID < placements[j].MenuID
		}
		return placements[i].ItemID < placements[j].ItemID
	})
	return placements, nil
}

func (r *GormRepository) EnsureDefault(
	ctx context.Context,
	listImageCaseIDs func(context.Context) ([]string, error),
) (domain.MenuTree, error) {
	tree, err := r.GetTree(ctx, domain.DocumentIDDefault)
	if err == nil {
		return tree, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MenuTree{}, err
	}

	migrated, err := r.migrateLegacyJSON(ctx)
	if err != nil {
		return domain.MenuTree{}, err
	}
	if migrated {
		return r.GetTree(ctx, domain.DocumentIDDefault)
	}

	seed := domain.DefaultSeedTree()
	if listImageCaseIDs != nil {
		ids, err := listImageCaseIDs(ctx)
		if err != nil {
			return domain.MenuTree{}, err
		}
		for i := range seed.Items {
			if seed.Items[i].ID == "btn-image" {
				seed.Items[i].CaseIDs = append([]string(nil), ids...)
				break
			}
		}
	}
	if err := r.ReplaceTree(ctx, seed); err != nil {
		return domain.MenuTree{}, err
	}
	return r.GetTree(ctx, domain.DocumentIDDefault)
}

// MigrateFromLegacyIfNeeded imports legacy JSON when relational tables are empty.
func (r *GormRepository) MigrateFromLegacyIfNeeded(ctx context.Context) error {
	_, err := r.migrateLegacyJSON(ctx)
	return err
}

func (r *GormRepository) migrateLegacyJSON(ctx context.Context) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&MenuHeaderRow{}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	if !r.db.WithContext(ctx).Migrator().HasTable(&LegacyMenuRow{}) {
		return false, nil
	}

	var legacy LegacyMenuRow
	err := r.db.WithContext(ctx).First(&legacy, "id = ?", domain.DocumentIDDefault).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	tree, err := legacyJSONToTree(legacy)
	if err != nil {
		return false, err
	}
	if err := r.ReplaceTree(ctx, tree); err != nil {
		return false, err
	}
	return true, nil
}

type legacyMenuItem struct {
	ID              string               `json:"id"`
	Label           string               `json:"label"`
	Row             int                  `json:"row"`
	Col             int                  `json:"col"`
	Enabled         bool                 `json:"enabled"`
	Action          string               `json:"action"`
	CaseID          string               `json:"case_id"`
	Tag             string               `json:"tag"`
	PlaceholderText string               `json:"placeholder_text"`
	Reply           *domain.ReplyPayload `json:"reply"`
}

func legacyJSONToTree(legacy LegacyMenuRow) (domain.MenuTree, error) {
	var legacyItems []legacyMenuItem
	if err := json.Unmarshal([]byte(legacy.ItemsJSON), &legacyItems); err != nil {
		return domain.MenuTree{}, fmt.Errorf("unmarshal legacy menu items: %w", err)
	}

	items := make([]domain.MenuNode, 0, len(legacyItems))
	for _, it := range legacyItems {
		node := domain.MenuNode{
			ID:              it.ID,
			Label:           it.Label,
			Row:             it.Row,
			Col:             it.Col,
			Enabled:         it.Enabled,
			Kind:            domain.MenuKind(it.Action),
			Tag:             it.Tag,
			PlaceholderText: it.PlaceholderText,
			Reply:           it.Reply,
		}
		if it.CaseID != "" {
			node.CaseIDs = []string{it.CaseID}
		}
		items = append(items, node)
	}

	return domain.MenuTree{
		ID:        domain.DocumentIDDefault,
		BotID:     domain.BotIDDefault,
		Items:     items,
		UpdatedAt: legacy.UpdatedAt,
	}, nil
}

func itemToRow(menuID string, item domain.MenuItem) (MenuItemRow, error) {
	row := MenuItemRow{
		ID:              item.ID,
		MenuID:          menuID,
		Label:           item.Label,
		Row:             item.Row,
		Col:             item.Col,
		Enabled:         item.Enabled,
		Kind:            string(item.Kind),
		PlaceholderText: item.PlaceholderText,
		Tag:             item.Tag,
	}
	if item.ParentID != "" {
		parentID := item.ParentID
		row.ParentID = &parentID
	}
	if item.Reply != nil {
		raw, err := json.Marshal(item.Reply)
		if err != nil {
			return MenuItemRow{}, fmt.Errorf("marshal reply for %q: %w", item.ID, err)
		}
		row.ReplyJSON = string(raw)
	}
	return row, nil
}

func rowToItem(row MenuItemRow, caseIDs []string) (domain.MenuItem, error) {
	item := domain.MenuItem{
		ID:              row.ID,
		Label:           row.Label,
		Row:             row.Row,
		Col:             row.Col,
		Enabled:         row.Enabled,
		Kind:            domain.MenuKind(row.Kind),
		CaseIDs:         append([]string(nil), caseIDs...),
		PlaceholderText: row.PlaceholderText,
		Tag:             row.Tag,
	}
	if row.ParentID != nil {
		item.ParentID = *row.ParentID
	}
	if row.ReplyJSON != "" {
		var reply domain.ReplyPayload
		if err := json.Unmarshal([]byte(row.ReplyJSON), &reply); err != nil {
			return domain.MenuItem{}, fmt.Errorf("unmarshal reply for %q: %w", row.ID, err)
		}
		item.Reply = &reply
	}
	return item, nil
}

func buildPlacementPath(rows []MenuItemRow, itemID string) ([]domain.PlacementStep, error) {
	byID := make(map[string]MenuItemRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}

	var steps []domain.PlacementStep
	current := itemID
	for current != "" {
		row, ok := byID[current]
		if !ok {
			return nil, fmt.Errorf("unknown menu item %q while building placement path", current)
		}
		steps = append(steps, domain.PlacementStep{ID: row.ID, Label: row.Label})
		if row.ParentID == nil || *row.ParentID == "" {
			break
		}
		current = *row.ParentID
	}

	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
	return steps, nil
}
