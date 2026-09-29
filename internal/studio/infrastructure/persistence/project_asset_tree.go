package persistence

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func (r *GormRepository) ListProjectAssetTree(ctx context.Context, accountID string, query domain.ProjectAssetTreeQuery) (*domain.ProjectAssetTreePage, error) {
	if accountID == "" || query.Limit < 0 {
		return nil, fmt.Errorf("%w: invalid tree query", domain.ErrInvalid)
	}
	if query.Mode == "" {
		query.Mode = "asset"
	}
	if !validTreeMode(query.Mode) {
		return nil, fmt.Errorf("%w: invalid tree mode", domain.ErrInvalid)
	}
	if err := r.validateTreeProject(ctx, accountID, query.ProjectID); err != nil {
		return nil, err
	}
	if query.Mode == "asset" {
		if query.ParentID != "" {
			return nil, fmt.Errorf("%w: asset tree has no groups", domain.ErrInvalid)
		}
		return r.assetTreeLeaves(ctx, accountID, domain.ProjectAssetListQuery{ProjectID: query.ProjectID, Limit: query.Limit, Cursor: query.Cursor})
	}
	if query.ParentID != "" {
		return r.treeChildren(ctx, accountID, query)
	}
	var nodes []domain.ProjectAssetTreeNode
	var err error
	switch query.Mode {
	case "format":
		nodes, err = r.formatTreeGroups(ctx, accountID, query.ProjectID)
	case "rating":
		nodes, err = r.ratingTreeGroups(ctx, accountID, query.ProjectID)
	case "tag":
		nodes, err = r.tagTreeGroups(ctx, accountID, query.ProjectID)
	case "session":
		nodes, err = r.sessionTreeGroups(ctx, accountID, query.ProjectID)
	case "category":
		nodes, err = r.categoryTreeGroups(ctx, accountID, query.ProjectID, "")
	}
	if err != nil {
		return nil, err
	}
	return paginateTreeNodes(nodes, query.Limit, query.Cursor)
}

func (r *GormRepository) validateTreeProject(ctx context.Context, accountID, projectID string) error {
	if projectID == "" {
		return nil
	}
	var project ProjectRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, projectID).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *GormRepository) assetTreeLeaves(ctx context.Context, accountID string, query domain.ProjectAssetListQuery) (*domain.ProjectAssetTreePage, error) {
	page, err := r.ListProjectAssets(ctx, accountID, query)
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.ProjectAssetTreeNode, 0, len(page.Items))
	for _, item := range page.Items {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "asset:" + item.ID, Name: item.DisplayName, Type: "asset", ProjectAssetID: item.ID})
	}
	return &domain.ProjectAssetTreePage{Nodes: nodes, NextCursor: page.NextCursor}, nil
}

func (r *GormRepository) treeChildren(ctx context.Context, accountID string, query domain.ProjectAssetTreeQuery) (*domain.ProjectAssetTreePage, error) {
	mode, value, ok := strings.Cut(query.ParentID, ":")
	if !ok || (mode != query.Mode && !(query.Mode == "category" && mode == "category-direct")) || value == "" {
		return nil, fmt.Errorf("%w: invalid tree parent", domain.ErrInvalid)
	}
	filters := domain.ProjectAssetListQuery{ProjectID: query.ProjectID, Limit: query.Limit, Cursor: query.Cursor}
	switch mode {
	case "category-direct":
		filters.CategoryID = value
		filters.CategoryDirect = true
	case "format":
		filters.Format = value
	case "rating":
		rating, err := strconv.Atoi(value)
		if err != nil || rating < 0 || rating > 5 {
			return nil, fmt.Errorf("%w: invalid rating group", domain.ErrInvalid)
		}
		filters.Rating = &rating
	case "tag":
		if value == "untagged" {
			filters.Untagged = true
		} else {
			filters.TagIDs = []string{value}
		}
	case "session":
		if value == "none" {
			filters.NoSession = true
		} else {
			filters.SessionID = value
		}
	case "category":
		if value == "uncategorized" {
			filters.Uncategorized = true
		} else {
			var category AssetCategoryRow
			if err := r.db.WithContext(ctx).Where("account_id = ? AND project_id = ? AND id = ?", accountID, query.ProjectID, value).First(&category).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, domain.ErrNotFound
				}
				return nil, err
			}
			groups, err := r.categoryTreeGroups(ctx, accountID, query.ProjectID, value)
			if err != nil {
				return nil, err
			}
			if len(groups) > 0 {
				return paginateTreeNodes(groups, query.Limit, query.Cursor)
			}
			filters.CategoryID = value
			filters.CategoryDirect = true
		}
	}
	return r.assetTreeLeaves(ctx, accountID, filters)
}

func (r *GormRepository) formatTreeGroups(ctx context.Context, accountID, projectID string) ([]domain.ProjectAssetTreeNode, error) {
	var rows []struct {
		Format string
		Count  int64
	}
	err := r.db.WithContext(ctx).Table("studio_project_assets AS p").
		Joins("JOIN studio_asset_versions AS v ON v.id = p.asset_version_id AND v.account_id = p.account_id AND v.asset_id = p.asset_id").
		Where("p.account_id = ? AND p.project_id = ? AND p.archived_at IS NULL", accountID, projectID).
		Select("v.format AS format, COUNT(*) AS count").Group("v.format").Order("v.format ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.ProjectAssetTreeNode, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "format:" + row.Format, Name: row.Format, Type: "group", Count: row.Count, HasChildren: true})
	}
	return nodes, nil
}

func (r *GormRepository) ratingTreeGroups(ctx context.Context, accountID, projectID string) ([]domain.ProjectAssetTreeNode, error) {
	var rows []struct {
		Rating int
		Count  int64
	}
	err := r.db.WithContext(ctx).Model(&ProjectAssetRow{}).
		Where("account_id = ? AND project_id = ? AND archived_at IS NULL", accountID, projectID).
		Select("rating, COUNT(*) AS count").Group("rating").Order("rating DESC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.ProjectAssetTreeNode, 0, len(rows))
	for _, row := range rows {
		name := strconv.Itoa(row.Rating)
		if row.Rating == 0 {
			name = "未评分"
		}
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "rating:" + strconv.Itoa(row.Rating), Name: name, Type: "group", Count: row.Count, HasChildren: true})
	}
	return nodes, nil
}

func (r *GormRepository) tagTreeGroups(ctx context.Context, accountID, projectID string) ([]domain.ProjectAssetTreeNode, error) {
	var rows []struct {
		ID    string
		Name  string
		Count int64
	}
	err := r.db.WithContext(ctx).Table("studio_project_assets AS p").
		Joins("JOIN studio_project_asset_tags AS j ON j.project_asset_id = p.id AND j.account_id = p.account_id").
		Joins("JOIN studio_asset_tags AS t ON t.id = j.tag_id AND t.account_id = j.account_id").
		Where("p.account_id = ? AND p.project_id = ? AND p.archived_at IS NULL", accountID, projectID).
		Select("t.id AS id, t.name AS name, COUNT(DISTINCT p.id) AS count").Group("t.id, t.name").Order("t.name ASC, t.id ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.ProjectAssetTreeNode, 0, len(rows)+1)
	for _, row := range rows {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "tag:" + row.ID, Name: row.Name, Type: "group", Count: row.Count, HasChildren: true})
	}
	var untagged int64
	err = r.db.WithContext(ctx).Table("studio_project_assets AS p").Where("p.account_id = ? AND p.project_id = ? AND p.archived_at IS NULL", accountID, projectID).
		Where("NOT EXISTS (SELECT 1 FROM studio_project_asset_tags AS j WHERE j.project_asset_id = p.id AND j.account_id = p.account_id)").Count(&untagged).Error
	if err != nil {
		return nil, err
	}
	if untagged > 0 {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "tag:untagged", Name: "无标签", Type: "group", Count: untagged, HasChildren: true})
	}
	return nodes, nil
}

func (r *GormRepository) sessionTreeGroups(ctx context.Context, accountID, projectID string) ([]domain.ProjectAssetTreeNode, error) {
	var rows []struct {
		ID    string
		Name  string
		Count int64
	}
	err := r.db.WithContext(ctx).Table("studio_project_assets AS p").
		Joins("JOIN studio_session_asset_usages AS u ON u.asset_id = p.asset_id AND u.account_id = p.account_id").
		Joins("LEFT JOIN studio_sessions AS s ON s.id = u.session_id AND s.account_id = u.account_id").
		Where("p.account_id = ? AND p.project_id = ? AND p.archived_at IS NULL", accountID, projectID).
		Select("u.session_id AS id, MAX(COALESCE(s.title, u.session_title_snapshot)) AS name, COUNT(DISTINCT p.id) AS count").
		Group("u.session_id").Order("name ASC, id ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.ProjectAssetTreeNode, 0, len(rows)+1)
	for _, row := range rows {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "session:" + row.ID, Name: row.Name, Type: "group", Count: row.Count, HasChildren: true})
	}
	var noSource int64
	err = r.db.WithContext(ctx).Table("studio_project_assets AS p").Where("p.account_id = ? AND p.project_id = ? AND p.archived_at IS NULL", accountID, projectID).
		Where("NOT EXISTS (SELECT 1 FROM studio_session_asset_usages AS u WHERE u.asset_id = p.asset_id AND u.account_id = p.account_id)").Count(&noSource).Error
	if err != nil {
		return nil, err
	}
	if noSource > 0 {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "session:none", Name: "无来源对话", Type: "group", Count: noSource, HasChildren: true})
	}
	return nodes, nil
}

func (r *GormRepository) categoryTreeGroups(ctx context.Context, accountID, projectID, parentID string) ([]domain.ProjectAssetTreeNode, error) {
	var categories []AssetCategoryRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND project_id = ?", accountID, projectID).Find(&categories).Error; err != nil {
		return nil, err
	}
	var counts []struct {
		CategoryID string
		Count      int64
	}
	if err := r.db.WithContext(ctx).Model(&ProjectAssetRow{}).
		Where("account_id = ? AND project_id = ? AND archived_at IS NULL", accountID, projectID).
		Select("category_id, COUNT(*) AS count").Group("category_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	direct := make(map[string]int64, len(counts))
	for _, count := range counts {
		direct[count.CategoryID] = count.Count
	}
	children := make(map[string][]AssetCategoryRow, len(categories))
	for _, category := range categories {
		children[category.ParentID] = append(children[category.ParentID], category)
	}
	var countSubtree func(string, map[string]bool) int64
	countSubtree = func(id string, seen map[string]bool) int64 {
		if seen[id] {
			return 0
		}
		seen[id] = true
		total := direct[id]
		for _, child := range children[id] {
			total += countSubtree(child.ID, seen)
		}
		return total
	}
	nodes := make([]domain.ProjectAssetTreeNode, 0, len(children[parentID])+1)
	for _, category := range children[parentID] {
		count := countSubtree(category.ID, map[string]bool{})
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "category:" + category.ID, Name: category.Name, Type: "group", Count: count, HasChildren: count > 0 || len(children[category.ID]) > 0})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Name == nodes[j].Name {
			return nodes[i].ID < nodes[j].ID
		}
		return nodes[i].Name < nodes[j].Name
	})
	if parentID == "" && direct[""] > 0 {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "category:uncategorized", Name: "未分类", Type: "group", Count: direct[""], HasChildren: true})
	}
	if parentID != "" && direct[parentID] > 0 {
		nodes = append(nodes, domain.ProjectAssetTreeNode{ID: "category-direct:" + parentID, Name: "本级资产", Type: "group", Count: direct[parentID], HasChildren: true})
	}
	return nodes, nil
}

func paginateTreeNodes(nodes []domain.ProjectAssetTreeNode, limit int, cursor string) (*domain.ProjectAssetTreePage, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	start := 0
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
		}
		start, err = strconv.Atoi(string(data))
		if err != nil || start < 0 || start > len(nodes) {
			return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
		}
	}
	if start >= len(nodes) {
		return &domain.ProjectAssetTreePage{Nodes: []domain.ProjectAssetTreeNode{}}, nil
	}
	end := start + limit
	if end > len(nodes) {
		end = len(nodes)
	}
	page := &domain.ProjectAssetTreePage{Nodes: nodes[start:end]}
	if end < len(nodes) {
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	return page, nil
}
