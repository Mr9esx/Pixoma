package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/mr9esx/comfyui_tgbot/internal/menu/domain"
)

// ChannelMenuRow is the channel-scoped menu document (1:1 with a channel).
type ChannelMenuRow struct {
	ChannelID string    `gorm:"primaryKey;size:128"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (ChannelMenuRow) TableName() string { return "channel_menus" }

// ChannelMenuItemRow is a menu node under a channel.
type ChannelMenuItemRow struct {
	ChannelID       string `gorm:"primaryKey;size:128;not null;index;uniqueIndex:idx_ch_parent_label,priority:1;uniqueIndex:idx_ch_parent_order,priority:1"`
	ID              string `gorm:"primaryKey;size:128"`
	ParentID        string `gorm:"size:128;not null;default:'';uniqueIndex:idx_ch_parent_label,priority:2;uniqueIndex:idx_ch_parent_order,priority:2"`
	Label           string `gorm:"size:256;not null;uniqueIndex:idx_ch_parent_label,priority:3"`
	Order           int    `gorm:"column:sort_order;not null;uniqueIndex:idx_ch_parent_order,priority:3"`
	Enabled         bool   `gorm:"not null"`
	Kind            string `gorm:"size:64;not null"`
	PlaceholderText string `gorm:"size:512"`
	IntroText       string `gorm:"type:text"`
	ReplyJSON       string `gorm:"type:text"`
}

func (ChannelMenuItemRow) TableName() string { return "channel_menu_items" }

// ChannelMenuItemCaseRow links a menu item to a case.
type ChannelMenuItemCaseRow struct {
	ChannelID  string `gorm:"primaryKey;size:128;not null"`
	MenuItemID string `gorm:"primaryKey;size:128"`
	CaseID     string `gorm:"primaryKey;size:128"`
	Sort       int    `gorm:"not null"`
}

func (ChannelMenuItemCaseRow) TableName() string { return "channel_menu_item_cases" }

// ChannelMenuExtraRow stores platform-specific extras per channel+item+type.
type ChannelMenuExtraRow struct {
	ChannelID  string    `gorm:"primaryKey;size:128"`
	MenuItemID string    `gorm:"primaryKey;size:128"`
	ExtraType  string    `gorm:"primaryKey;size:64"`
	ExtraJSON  string    `gorm:"type:text;not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}

func (ChannelMenuExtraRow) TableName() string { return "channel_menu_item_extras" }

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetTree(ctx context.Context, channelID string) (domain.MenuTree, error) {
	var header ChannelMenuRow
	err := r.db.WithContext(ctx).First(&header, "channel_id = ?", channelID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.MenuTree{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MenuTree{}, err
	}

	var itemRows []ChannelMenuItemRow
	if err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("sort_order, id").
		Find(&itemRows).Error; err != nil {
		return domain.MenuTree{}, err
	}

	itemIDs := make([]string, 0, len(itemRows))
	for _, row := range itemRows {
		itemIDs = append(itemIDs, row.ID)
	}

	caseByItem := map[string][]string{}
	if len(itemIDs) > 0 {
		var caseRows []ChannelMenuItemCaseRow
		if err := r.db.WithContext(ctx).
			Where("menu_item_id IN ? AND channel_id = ?", itemIDs, channelID).
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
		ChannelID: header.ChannelID,
		Items:     nodes,
		UpdatedAt: header.UpdatedAt,
	}, nil
}

func (r *GormRepository) ReplaceTree(ctx context.Context, tree domain.MenuTree) error {
	if tree.ChannelID == "" {
		return fmt.Errorf("menu: ReplaceTree requires channel_id")
	}
	flat := domain.Flatten(tree.Items)
	now := time.Now().UTC()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var itemIDs []string
		if err := tx.Model(&ChannelMenuItemRow{}).
			Where("channel_id = ?", tree.ChannelID).
			Pluck("id", &itemIDs).Error; err != nil {
			return err
		}
		if len(itemIDs) > 0 {
			if err := tx.Where("menu_item_id IN ?", itemIDs).Delete(&ChannelMenuItemCaseRow{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("channel_id = ?", tree.ChannelID).Delete(&ChannelMenuItemRow{}).Error; err != nil {
			return err
		}

		header := ChannelMenuRow{ChannelID: tree.ChannelID, UpdatedAt: now}
		if err := tx.Save(&header).Error; err != nil {
			return err
		}

		for _, item := range flat {
			row, err := itemToRow(tree.ChannelID, item)
			if err != nil {
				return err
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			for sort, caseID := range item.CaseIDs {
				if err := tx.Create(&ChannelMenuItemCaseRow{
					ChannelID:  tree.ChannelID,
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
	var links []ChannelMenuItemCaseRow
	if err := r.db.WithContext(ctx).
		Where("case_id = ?", caseID).
		Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}

	seen := map[string]struct{}{}
	channelIDs := map[string]struct{}{}
	itemIDs := map[string]struct{}{}
	for _, link := range links {
		channelIDs[link.ChannelID] = struct{}{}
		itemIDs[link.MenuItemID] = struct{}{}
		key := channelItemKey(link.ChannelID, link.MenuItemID)
		seen[key] = struct{}{}
	}

	channelIDList := make([]string, 0, len(channelIDs))
	for id := range channelIDs {
		channelIDList = append(channelIDList, id)
	}
	itemIDList := make([]string, 0, len(itemIDs))
	for id := range itemIDs {
		itemIDList = append(itemIDList, id)
	}

	var itemRows []ChannelMenuItemRow
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND channel_id IN ?", itemIDList, channelIDList).
		Find(&itemRows).Error; err != nil {
		return nil, err
	}
	byKey := make(map[string]ChannelMenuItemRow, len(itemRows))
	for _, row := range itemRows {
		byKey[channelItemKey(row.ChannelID, row.ID)] = row
	}

	rowsByChannel := map[string][]ChannelMenuItemRow{}
	for channelID := range channelIDs {
		var rows []ChannelMenuItemRow
		if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).Find(&rows).Error; err != nil {
			return nil, err
		}
		rowsByChannel[channelID] = rows
	}

	placements := make([]domain.MenuPlacement, 0, len(links))
	for _, link := range links {
		item, ok := byKey[channelItemKey(link.ChannelID, link.MenuItemID)]
		if !ok {
			continue
		}
		path, err := buildPlacementPath(rowsByChannel[item.ChannelID], item.ID)
		if err != nil {
			return nil, err
		}
		placements = append(placements, domain.MenuPlacement{
			ChannelID: item.ChannelID,
			ItemID:    item.ID,
			Path:      path,
		})
	}

	sort.Slice(placements, func(i, j int) bool {
		if placements[i].ChannelID != placements[j].ChannelID {
			return placements[i].ChannelID < placements[j].ChannelID
		}
		return placements[i].ItemID < placements[j].ItemID
	})
	return placements, nil
}

func channelItemKey(channelID, itemID string) string {
	return channelID + "\x00" + itemID
}

func (r *GormRepository) EnsureDefault(
	ctx context.Context,
	channelID string,
	listImageCaseIDs func(context.Context) ([]string, error),
) (domain.MenuTree, error) {
	tree, err := r.GetTree(ctx, channelID)
	if err == nil {
		return tree, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.MenuTree{}, err
	}

	seed := domain.DefaultSeedTree(channelID)
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
	return r.GetTree(ctx, channelID)
}

func (r *GormRepository) ListExtras(ctx context.Context, channelID string) (map[string][]domain.Extra, error) {
	var rows []ChannelMenuExtraRow
	if err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("menu_item_id, extra_type").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string][]domain.Extra{}
	for _, row := range rows {
		out[row.MenuItemID] = append(out[row.MenuItemID], domain.Extra{
			ChannelID:  row.ChannelID,
			MenuItemID: row.MenuItemID,
			ExtraType:  row.ExtraType,
			ExtraJSON:  row.ExtraJSON,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepository) SaveExtras(ctx context.Context, channelID string, extras map[string][]domain.Extra) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", channelID).Delete(&ChannelMenuExtraRow{}).Error; err != nil {
			return err
		}
		for _, itemExtras := range extras {
			for _, extra := range itemExtras {
				if err := tx.Create(&ChannelMenuExtraRow{
					ChannelID:  channelID,
					MenuItemID: extra.MenuItemID,
					ExtraType:  extra.ExtraType,
					ExtraJSON:  extra.ExtraJSON,
					UpdatedAt:  now,
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func itemToRow(channelID string, item domain.MenuItem) (ChannelMenuItemRow, error) {
	row := ChannelMenuItemRow{
		ID:              item.ID,
		ChannelID:       channelID,
		Label:           item.Label,
		Order:           item.Order,
		Enabled:         item.Enabled,
		Kind:            string(item.Kind),
		PlaceholderText: item.PlaceholderText,
		IntroText:       item.IntroText,
	}
	row.ParentID = item.ParentID
	if item.Reply != nil {
		raw, err := json.Marshal(item.Reply)
		if err != nil {
			return ChannelMenuItemRow{}, fmt.Errorf("marshal reply for %q: %w", item.ID, err)
		}
		row.ReplyJSON = string(raw)
	}
	return row, nil
}

func rowToItem(row ChannelMenuItemRow, caseIDs []string) (domain.MenuItem, error) {
	item := domain.MenuItem{
		ID:              row.ID,
		Label:           row.Label,
		Order:           row.Order,
		Enabled:         row.Enabled,
		Kind:            domain.MenuKind(row.Kind),
		CaseIDs:         append([]string(nil), caseIDs...),
		PlaceholderText: row.PlaceholderText,
		IntroText:       row.IntroText,
	}
	item.ParentID = row.ParentID
	if row.ReplyJSON != "" {
		var reply domain.ReplyPayload
		if err := json.Unmarshal([]byte(row.ReplyJSON), &reply); err != nil {
			return domain.MenuItem{}, fmt.Errorf("unmarshal reply for %q: %w", row.ID, err)
		}
		item.Reply = &reply
	}
	return item, nil
}

func buildPlacementPath(rows []ChannelMenuItemRow, itemID string) ([]domain.PlacementStep, error) {
	byID := make(map[string]ChannelMenuItemRow, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	var path []domain.PlacementStep
	cur := itemID
	for cur != "" {
		row, ok := byID[cur]
		if !ok {
			return nil, fmt.Errorf("menu placement: unknown item %q", cur)
		}
		path = append([]domain.PlacementStep{{ID: row.ID, Label: row.Label}}, path...)
		if row.ParentID == "" {
			break
		}
		cur = row.ParentID
	}
	return path, nil
}
