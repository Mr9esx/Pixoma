package persistence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type ProjectAssetRow struct {
	ID                        string `gorm:"primaryKey;size:64"`
	AccountID                 string `gorm:"size:64;not null;uniqueIndex:idx_studio_project_asset_identity;index:idx_studio_project_asset_page,priority:1"`
	ProjectID                 string `gorm:"size:64;not null;uniqueIndex:idx_studio_project_asset_identity;index:idx_studio_project_asset_page,priority:2"`
	AssetID                   string `gorm:"size:64;not null;uniqueIndex:idx_studio_project_asset_identity;index"`
	AssetVersionID            string `gorm:"size:64;not null;index"`
	DisplayName               string `gorm:"size:512;not null"`
	CategoryID                string `gorm:"size:64;index"`
	Rating                    int    `gorm:"not null;default:0;index"`
	SourceProjectAssetID      string `gorm:"size:64;index"`
	SourceProjectID           string `gorm:"size:64"`
	SourceProjectNameSnapshot string `gorm:"size:256"`
	SourceDisplayNameSnapshot string `gorm:"size:512"`
	SourceAssetVersionID      string `gorm:"size:64"`
	CopiedAt                  *time.Time
	AddedAt                   time.Time `gorm:"index:idx_studio_project_asset_page,priority:3"`
	UpdatedAt                 time.Time
	ArchivedAt                *time.Time
}

func (ProjectAssetRow) TableName() string { return "studio_project_assets" }

type SessionAssetUsageRow struct {
	ID                     string    `gorm:"primaryKey;size:64"`
	AccountID              string    `gorm:"size:64;not null;uniqueIndex:idx_studio_usage_identity;index:idx_studio_usage_session,priority:1"`
	SessionID              string    `gorm:"size:64;not null;uniqueIndex:idx_studio_usage_identity;index:idx_studio_usage_session,priority:2"`
	AssetID                string    `gorm:"size:64;not null;uniqueIndex:idx_studio_usage_identity;index"`
	AssetVersionID         string    `gorm:"size:64;not null;uniqueIndex:idx_studio_usage_identity"`
	UsageKind              string    `gorm:"size:32;not null;uniqueIndex:idx_studio_usage_identity"`
	OperationKey           string    `gorm:"size:128;not null;uniqueIndex:idx_studio_usage_identity"`
	RunID                  string    `gorm:"size:64"`
	MessageID              string    `gorm:"size:64"`
	SessionTitleSnapshot   string    `gorm:"size:256"`
	OperationLabelSnapshot string    `gorm:"size:256"`
	CreatedAt              time.Time `gorm:"index:idx_studio_usage_session,priority:3"`
}

func (SessionAssetUsageRow) TableName() string { return "studio_session_asset_usages" }

func projectAssetToRow(value *domain.ProjectAsset) *ProjectAssetRow {
	return &ProjectAssetRow{
		ID: value.ID, AccountID: value.AccountID, ProjectID: value.ProjectID, AssetID: value.AssetID,
		AssetVersionID: value.AssetVersionID, DisplayName: value.DisplayName, CategoryID: value.CategoryID,
		Rating: value.Rating, SourceProjectAssetID: value.SourceProjectAssetID, SourceProjectID: value.SourceProjectID,
		SourceProjectNameSnapshot: value.SourceProjectNameSnapshot, SourceDisplayNameSnapshot: value.SourceDisplayNameSnapshot,
		SourceAssetVersionID: value.SourceAssetVersionID, CopiedAt: value.CopiedAt,
		AddedAt: value.AddedAt, UpdatedAt: value.UpdatedAt, ArchivedAt: value.ArchivedAt,
	}
}

func projectAssetFromRow(row ProjectAssetRow) *domain.ProjectAsset {
	return &domain.ProjectAsset{
		ID: row.ID, AccountID: row.AccountID, ProjectID: row.ProjectID, AssetID: row.AssetID,
		AssetVersionID: row.AssetVersionID, DisplayName: row.DisplayName, CategoryID: row.CategoryID,
		Rating: row.Rating, SourceProjectAssetID: row.SourceProjectAssetID, SourceProjectID: row.SourceProjectID,
		SourceProjectNameSnapshot: row.SourceProjectNameSnapshot, SourceDisplayNameSnapshot: row.SourceDisplayNameSnapshot,
		SourceAssetVersionID: row.SourceAssetVersionID, CopiedAt: row.CopiedAt,
		AddedAt: row.AddedAt, UpdatedAt: row.UpdatedAt, ArchivedAt: row.ArchivedAt,
	}
}

func sessionAssetUsageToRow(value *domain.SessionAssetUsage) *SessionAssetUsageRow {
	return &SessionAssetUsageRow{
		ID: value.ID, AccountID: value.AccountID, SessionID: value.SessionID, AssetID: value.AssetID,
		AssetVersionID: value.AssetVersionID, UsageKind: value.UsageKind, OperationKey: value.OperationKey,
		RunID: value.RunID, MessageID: value.MessageID, SessionTitleSnapshot: value.SessionTitleSnapshot,
		OperationLabelSnapshot: value.OperationLabelSnapshot, CreatedAt: value.CreatedAt,
	}
}

func validProjectAsset(value *domain.ProjectAsset) error {
	if value == nil || value.ID == "" || value.AccountID == "" || value.AssetID == "" || value.AssetVersionID == "" || value.DisplayName == "" || value.Rating < 0 || value.Rating > 5 {
		return fmt.Errorf("%w: invalid project asset", domain.ErrInvalid)
	}
	return nil
}

func validateProjectAssetScope(tx *gorm.DB, value *domain.ProjectAsset) error {
	if value.ProjectID != "" {
		var project ProjectRow
		if err := tx.Where("account_id = ? AND id = ?", value.AccountID, value.ProjectID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
	}
	if value.CategoryID != "" {
		var category AssetCategoryRow
		if err := tx.Where("account_id = ? AND project_id = ? AND id = ?", value.AccountID, value.ProjectID, value.CategoryID).First(&category).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
	}
	var version AssetVersionRow
	if err := tx.Where("account_id = ? AND asset_id = ? AND id = ?", value.AccountID, value.AssetID, value.AssetVersionID).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *GormRepository) CreateAssetWithPlacement(ctx context.Context, asset *domain.Asset, placement *domain.ProjectAsset, usage *domain.SessionAssetUsage) error {
	if asset == nil || placement == nil || placement.AccountID != asset.AccountID || placement.AssetID != asset.ID {
		return fmt.Errorf("%w: invalid asset placement", domain.ErrInvalid)
	}
	if err := validProjectAsset(placement); err != nil {
		return err
	}
	return translateCreateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if placement.ProjectID != "" {
			var project ProjectRow
			if err := tx.Where("account_id = ? AND id = ?", asset.AccountID, placement.ProjectID).First(&project).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
		}
		if err := tx.Create(assetToRow(asset)).Error; err != nil {
			return err
		}
		versionFound := false
		for _, version := range asset.Versions {
			if version.ID == placement.AssetVersionID && version.AssetID == asset.ID && version.AccountID == asset.AccountID {
				versionFound = true
			}
			if err := tx.Create(assetVersionToRow(version)).Error; err != nil {
				return err
			}
			if supportsPalette(version.MIMEType) {
				if err := tx.Create(newPendingPaletteRow(version)).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("key_hash = ? AND account_id = ?", blobIntentKeyHash(asset.AccountID, version.BlobKey), asset.AccountID).Delete(&BlobWriteIntentRow{}).Error; err != nil {
				return err
			}
		}
		if !versionFound {
			return fmt.Errorf("%w: placement version does not belong to asset", domain.ErrInvalid)
		}
		if placement.CategoryID != "" {
			var category AssetCategoryRow
			if err := tx.Where("account_id = ? AND project_id = ? AND id = ?", placement.AccountID, placement.ProjectID, placement.CategoryID).First(&category).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return domain.ErrNotFound
				}
				return err
			}
		}
		if err := tx.Create(projectAssetToRow(placement)).Error; err != nil {
			return err
		}
		if usage != nil {
			if err := validateUsage(tx, usage, placement.AccountID, placement.AssetID, placement.ProjectID); err != nil {
				return err
			}
			return tx.Create(sessionAssetUsageToRow(usage)).Error
		}
		return nil
	}))
}

func validateUsage(tx *gorm.DB, value *domain.SessionAssetUsage, accountID, assetID, projectID string) error {
	if value.ID == "" || value.AccountID != accountID || value.AssetID != assetID || value.SessionID == "" || value.AssetVersionID == "" || value.OperationKey == "" || value.UsageKind == "" {
		return fmt.Errorf("%w: invalid session usage", domain.ErrInvalid)
	}
	var session SessionRow
	if err := tx.Where("account_id = ? AND id = ?", accountID, value.SessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	if session.ProjectID != projectID {
		return fmt.Errorf("%w: session and project differ", domain.ErrInvalid)
	}
	value.SessionTitleSnapshot = session.Title
	var version AssetVersionRow
	if err := tx.Where("account_id = ? AND asset_id = ? AND id = ?", accountID, assetID, value.AssetVersionID).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *GormRepository) ReferenceAssetInSession(ctx context.Context, placement *domain.ProjectAsset, usage *domain.SessionAssetUsage) error {
	if err := validProjectAsset(placement); err != nil {
		return err
	}
	if usage == nil {
		return fmt.Errorf("%w: missing session usage", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateProjectAssetScope(tx, placement); err != nil {
			return err
		}
		if err := validateUsage(tx, usage, placement.AccountID, placement.AssetID, placement.ProjectID); err != nil {
			return err
		}
		var existing ProjectAssetRow
		err := tx.Where("account_id = ? AND project_id = ? AND asset_id = ?", placement.AccountID, placement.ProjectID, placement.AssetID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(projectAssetToRow(placement)).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(sessionAssetUsageToRow(usage)).Error
	}))
}

func (r *GormRepository) CopyProjectAsset(ctx context.Context, accountID, sourceProjectAssetID string, target *domain.ProjectAsset) error {
	if target == nil || accountID == "" || target.ID == "" || target.AccountID != accountID {
		return fmt.Errorf("%w: invalid copy destination", domain.ErrInvalid)
	}
	return translateCreateError(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source ProjectAssetRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, sourceProjectAssetID).First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if target.ProjectID == source.ProjectID {
			return fmt.Errorf("%w: source and destination are the same project", domain.ErrInvalid)
		}
		target.AssetID = source.AssetID
		if target.AssetVersionID == "" {
			target.AssetVersionID = source.AssetVersionID
		}
		if target.DisplayName == "" {
			target.DisplayName = source.DisplayName
		}
		if err := validProjectAsset(target); err != nil {
			return err
		}
		if err := validateProjectAssetScope(tx, target); err != nil {
			return err
		}
		target.SourceProjectAssetID = source.ID
		target.SourceProjectID = source.ProjectID
		target.SourceDisplayNameSnapshot = source.DisplayName
		target.SourceAssetVersionID = source.AssetVersionID
		if source.ProjectID != "" {
			var project ProjectRow
			if err := tx.Where("account_id = ? AND id = ?", accountID, source.ProjectID).First(&project).Error; err != nil {
				return err
			}
			target.SourceProjectNameSnapshot = project.Name
		}
		copiedAt := target.AddedAt.UTC()
		target.CopiedAt = &copiedAt
		return tx.Create(projectAssetToRow(target)).Error
	}))
}

func (r *GormRepository) DeleteProjectAsset(ctx context.Context, accountID, projectAssetID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ProjectAssetRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, projectAssetID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if err := tx.Where("account_id = ? AND project_asset_id = ?", accountID, projectAssetID).Delete(&ProjectAssetTagRow{}).Error; err != nil {
			return err
		}
		return tx.Where("account_id = ? AND id = ?", accountID, projectAssetID).Delete(&ProjectAssetRow{}).Error
	})
}

func (r *GormRepository) GetProjectAsset(ctx context.Context, accountID, projectAssetID string) (*domain.ProjectAsset, error) {
	var row ProjectAssetRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, projectAssetID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	entry := projectAssetFromRow(row)
	var tags []AssetTagRow
	if err := r.db.WithContext(ctx).Table("studio_asset_tags AS t").
		Joins("JOIN studio_project_asset_tags AS j ON j.tag_id = t.id AND j.account_id = t.account_id").
		Where("j.account_id = ? AND j.project_asset_id = ?", accountID, projectAssetID).
		Order("t.name ASC, t.id ASC").Select("t.*").Find(&tags).Error; err != nil {
		return nil, err
	}
	entry.Tags = make([]domain.AssetTag, 0, len(tags))
	for _, tag := range tags {
		entry.Tags = append(entry.Tags, domain.AssetTag{ID: tag.ID, AccountID: tag.AccountID, Name: tag.Name, CreatedAt: tag.CreatedAt, UpdatedAt: tag.UpdatedAt})
	}
	if row.SourceProjectAssetID != "" {
		var sourceCount int64
		if err := r.db.WithContext(ctx).Model(&ProjectAssetRow{}).
			Where("account_id = ? AND id = ?", accountID, row.SourceProjectAssetID).Count(&sourceCount).Error; err != nil {
			return nil, err
		}
		entry.SourceDeleted = sourceCount == 0
	}
	asset, err := r.GetAsset(ctx, accountID, row.AssetID)
	if err != nil {
		return nil, err
	}
	entry.Asset = asset
	for _, version := range asset.Versions {
		if version.ID == row.AssetVersionID {
			entry.Version = version
			return entry, nil
		}
	}
	return nil, fmt.Errorf("%w: project asset references a missing version", domain.ErrInvalid)
}

func (r *GormRepository) GetProjectAssetByAsset(ctx context.Context, accountID, projectID, assetID string) (*domain.ProjectAsset, error) {
	if accountID == "" || assetID == "" {
		return nil, fmt.Errorf("%w: invalid project asset lookup", domain.ErrInvalid)
	}
	var row ProjectAssetRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND project_id = ? AND asset_id = ?", accountID, projectID, assetID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.GetProjectAsset(ctx, accountID, row.ID)
}

func (r *GormRepository) ListDuplicateProjectAssets(ctx context.Context, accountID, projectAssetID string) ([]*domain.ProjectAsset, error) {
	source, err := r.GetProjectAsset(ctx, accountID, projectAssetID)
	if err != nil {
		return nil, err
	}
	items := []*domain.ProjectAsset{}
	if source.Version.SHA256 == "" {
		return items, nil
	}
	baseQuery := func() *gorm.DB {
		return r.db.WithContext(ctx).Table("studio_project_assets AS p").
			Joins("JOIN studio_asset_versions AS v ON v.id = p.asset_version_id AND v.asset_id = p.asset_id AND v.account_id = p.account_id").
			Joins("LEFT JOIN studio_projects AS pr ON pr.id = p.project_id AND pr.account_id = p.account_id").
			Where("p.account_id = ? AND p.id <> ? AND p.asset_id <> ? AND v.sha256 = ?", accountID, projectAssetID, source.AssetID, source.Version.SHA256)
	}
	var sameProjectIDs []string
	if err := baseQuery().Where("p.project_id = ?", source.ProjectID).Order("p.id ASC").Pluck("p.id", &sameProjectIDs).Error; err != nil {
		return nil, err
	}
	var otherProjectIDs []string
	if err := baseQuery().Where("p.project_id <> ?", source.ProjectID).
		Order("COALESCE(pr.name, '') ASC, p.id ASC").Pluck("p.id", &otherProjectIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range append(sameProjectIDs, otherProjectIDs...) {
		item, err := r.GetProjectAsset(ctx, accountID, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *GormRepository) ListProjectAssets(ctx context.Context, accountID string, query domain.ProjectAssetListQuery) (*domain.ProjectAssetPage, error) {
	sortMode, err := projectAssetSortMode(query.Sort)
	if accountID == "" || query.Limit < 0 || err != nil || !validProjectAssetRanges(query) {
		return nil, fmt.Errorf("%w: invalid project asset query", domain.ErrInvalid)
	}
	if query.ProjectID != "" {
		var project ProjectRow
		if err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, query.ProjectID).First(&project).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
	}
	statement := r.db.WithContext(ctx).Table("studio_project_assets AS p").
		Joins("JOIN studio_assets AS a ON a.id = p.asset_id AND a.account_id = p.account_id").
		Joins("JOIN studio_asset_versions AS v ON v.id = p.asset_version_id AND v.asset_id = p.asset_id AND v.account_id = p.account_id").
		Where("p.account_id = ? AND p.project_id = ?", accountID, query.ProjectID)
	if query.ArchivedOnly {
		statement = statement.Where("p.archived_at IS NOT NULL")
	} else if !query.IncludeArchived {
		statement = statement.Where("p.archived_at IS NULL")
	}
	if query.Kind != "" {
		statement = statement.Where("a.kind = ?", query.Kind)
	}
	if query.Format != "" {
		statement = statement.Where("v.format = ?", query.Format)
	}
	if query.MinWidthPx != nil {
		statement = statement.Where("v.width_px >= ?", *query.MinWidthPx)
	}
	if query.MaxWidthPx != nil {
		statement = statement.Where("v.width_px <= ?", *query.MaxWidthPx)
	}
	if query.MinHeightPx != nil {
		statement = statement.Where("v.height_px >= ?", *query.MinHeightPx)
	}
	if query.MaxHeightPx != nil {
		statement = statement.Where("v.height_px <= ?", *query.MaxHeightPx)
	}
	if query.MinSizeBytes != nil {
		statement = statement.Where("v.size_bytes >= ?", *query.MinSizeBytes)
	}
	if query.MaxSizeBytes != nil {
		statement = statement.Where("v.size_bytes <= ?", *query.MaxSizeBytes)
	}
	if query.AddedFrom != nil {
		statement = statement.Where("p.added_at >= ?", query.AddedFrom.UTC())
	}
	if query.AddedTo != nil {
		statement = statement.Where("p.added_at < ?", query.AddedTo.UTC())
	}
	if query.DuplicatesOnly {
		statement = statement.Where("v.sha256 <> '' AND EXISTS (SELECT 1 FROM studio_project_assets AS p2 JOIN studio_asset_versions AS v2 ON v2.id = p2.asset_version_id AND v2.asset_id = p2.asset_id AND v2.account_id = p2.account_id WHERE p2.account_id = p.account_id AND p2.project_id = p.project_id AND p2.asset_id <> p.asset_id AND p2.archived_at IS NULL AND v2.sha256 = v.sha256)")
	}
	if query.CategoryID != "" {
		var categories []AssetCategoryRow
		if err := r.db.WithContext(ctx).Where("account_id = ? AND project_id = ?", accountID, query.ProjectID).Find(&categories).Error; err != nil {
			return nil, err
		}
		found := false
		children := make(map[string][]string, len(categories))
		for _, category := range categories {
			if category.ID == query.CategoryID {
				found = true
			}
			children[category.ParentID] = append(children[category.ParentID], category.ID)
		}
		if !found {
			return nil, domain.ErrNotFound
		}
		if query.CategoryDirect {
			statement = statement.Where("p.category_id = ?", query.CategoryID)
		} else {
			categoryIDs := []string{query.CategoryID}
			seen := map[string]bool{query.CategoryID: true}
			for index := 0; index < len(categoryIDs); index++ {
				for _, childID := range children[categoryIDs[index]] {
					if !seen[childID] {
						seen[childID] = true
						categoryIDs = append(categoryIDs, childID)
					}
				}
			}
			statement = statement.Where("p.category_id IN ?", categoryIDs)
		}
	}
	if query.Uncategorized {
		statement = statement.Where("p.category_id = ''")
	}
	if query.Rating != nil {
		statement = statement.Where("p.rating = ?", *query.Rating)
	}
	if query.SessionID != "" {
		statement = statement.Where("EXISTS (SELECT 1 FROM studio_session_asset_usages AS u WHERE u.account_id = p.account_id AND u.asset_id = p.asset_id AND u.session_id = ?)", query.SessionID)
	}
	if query.NoSession {
		statement = statement.Where("NOT EXISTS (SELECT 1 FROM studio_session_asset_usages AS u WHERE u.account_id = p.account_id AND u.asset_id = p.asset_id)")
	}
	for _, tagID := range query.TagIDs {
		statement = statement.Where("EXISTS (SELECT 1 FROM studio_project_asset_tags AS t WHERE t.project_asset_id = p.id AND t.account_id = p.account_id AND t.tag_id = ?)", tagID)
	}
	if query.Untagged {
		statement = statement.Where("NOT EXISTS (SELECT 1 FROM studio_project_asset_tags AS t WHERE t.project_asset_id = p.id AND t.account_id = p.account_id)")
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(strings.ToLower(search)) + "%"
		statement = statement.Where("(LOWER(p.display_name) LIKE ? ESCAPE '\\' OR LOWER(a.name) LIKE ? ESCAPE '\\' OR EXISTS (SELECT 1 FROM studio_project_asset_tags AS j JOIN studio_asset_tags AS t ON t.id = j.tag_id AND t.account_id = j.account_id WHERE j.project_asset_id = p.id AND j.account_id = p.account_id AND LOWER(t.name) LIKE ? ESCAPE '\\'))", pattern, pattern, pattern)
	}
	page := &domain.ProjectAssetPage{Items: []*domain.ProjectAsset{}}
	if err := statement.Count(&page.Total).Error; err != nil {
		return nil, err
	}
	if query.Cursor != "" {
		cursor, err := decodeProjectAssetCursor(query.Cursor, sortMode)
		if err != nil {
			return nil, err
		}
		statement, err = applyProjectAssetCursor(statement, cursor)
		if err != nil {
			return nil, err
		}
	}
	limit := normalizeLimit(query.Limit)
	var rows []ProjectAssetRow
	if err := statement.Select("p.*").Order(projectAssetOrder(sortMode)).Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	for _, row := range rows {
		item, err := r.GetProjectAsset(ctx, accountID, row.ID)
		if err != nil {
			return nil, err
		}
		page.Items = append(page.Items, item)
	}
	if more && len(rows) > 0 {
		cursor, err := projectAssetCursorFor(page.Items[len(page.Items)-1], sortMode)
		if err != nil {
			return nil, err
		}
		page.NextCursor = cursor
	}
	return page, nil
}

func validProjectAssetRanges(query domain.ProjectAssetListQuery) bool {
	if query.MinWidthPx != nil && (*query.MinWidthPx < 0 || query.MaxWidthPx != nil && *query.MinWidthPx > *query.MaxWidthPx) {
		return false
	}
	if query.MaxWidthPx != nil && *query.MaxWidthPx < 0 {
		return false
	}
	if query.MinHeightPx != nil && (*query.MinHeightPx < 0 || query.MaxHeightPx != nil && *query.MinHeightPx > *query.MaxHeightPx) {
		return false
	}
	if query.MaxHeightPx != nil && *query.MaxHeightPx < 0 {
		return false
	}
	if query.MinSizeBytes != nil && (*query.MinSizeBytes < 0 || query.MaxSizeBytes != nil && *query.MinSizeBytes > *query.MaxSizeBytes) {
		return false
	}
	if query.MaxSizeBytes != nil && *query.MaxSizeBytes < 0 {
		return false
	}
	return query.AddedFrom == nil || query.AddedTo == nil || query.AddedFrom.Before(*query.AddedTo)
}

type projectAssetCursor struct {
	Sort  string `json:"sort"`
	Value string `json:"value"`
	ID    string `json:"id"`
}

func projectAssetSortMode(value string) (string, error) {
	switch value {
	case "", "recent", "added_desc":
		return "recent", nil
	case "oldest", "added_asc":
		return "oldest", nil
	case "modified", "modified_desc":
		return "modified", nil
	case "name", "name_asc":
		return "name", nil
	case "size", "size_desc":
		return "size", nil
	case "rating", "rating_desc":
		return "rating", nil
	default:
		return "", domain.ErrInvalid
	}
}

func projectAssetOrder(sortMode string) string {
	switch sortMode {
	case "oldest":
		return "p.added_at ASC, p.id ASC"
	case "modified":
		return "CASE WHEN v.source_modified_at IS NULL THEN 1 ELSE 0 END ASC, v.source_modified_at DESC, p.id DESC"
	case "name":
		return "LOWER(p.display_name) ASC, p.id ASC"
	case "size":
		return "v.size_bytes DESC, p.id DESC"
	case "rating":
		return "p.rating DESC, p.id DESC"
	default:
		return "p.added_at DESC, p.id DESC"
	}
}

func decodeProjectAssetCursor(value, sortMode string) (projectAssetCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return projectAssetCursor{}, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
	}
	var cursor projectAssetCursor
	if err := json.Unmarshal(data, &cursor); err != nil || cursor.Sort != sortMode || cursor.ID == "" {
		return projectAssetCursor{}, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
	}
	return cursor, nil
}

func applyProjectAssetCursor(statement *gorm.DB, cursor projectAssetCursor) (*gorm.DB, error) {
	if cursor.Sort == "modified" {
		if cursor.Value == "" {
			return statement.Where("v.source_modified_at IS NULL AND p.id < ?", cursor.ID), nil
		}
		stamp, err := time.Parse(time.RFC3339Nano, cursor.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
		}
		return statement.Where("(v.source_modified_at < ? OR v.source_modified_at IS NULL OR (v.source_modified_at = ? AND p.id < ?))", stamp, stamp, cursor.ID), nil
	}
	column := "p.added_at"
	comparison := "<"
	var value any = cursor.Value
	switch cursor.Sort {
	case "recent", "oldest":
		stamp, err := time.Parse(time.RFC3339Nano, cursor.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
		}
		value = stamp
		if cursor.Sort == "oldest" {
			comparison = ">"
		}
	case "name":
		column, comparison = "LOWER(p.display_name)", ">"
	case "size", "rating":
		parsed, err := strconv.ParseInt(cursor.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid cursor", domain.ErrInvalid)
		}
		value = parsed
		if cursor.Sort == "size" {
			column = "v.size_bytes"
		} else {
			column = "p.rating"
		}
	}
	return statement.Where("("+column+" "+comparison+" ? OR ("+column+" = ? AND p.id "+comparison+" ?))", value, value, cursor.ID), nil
}

func projectAssetCursorFor(item *domain.ProjectAsset, sortMode string) (string, error) {
	cursor := projectAssetCursor{Sort: sortMode, ID: item.ID}
	switch sortMode {
	case "recent", "oldest":
		cursor.Value = item.AddedAt.UTC().Format(time.RFC3339Nano)
	case "modified":
		if item.Version.SourceModifiedAt != nil {
			cursor.Value = item.Version.SourceModifiedAt.UTC().Format(time.RFC3339Nano)
		}
	case "name":
		cursor.Value = strings.ToLower(item.DisplayName)
	case "size":
		cursor.Value = strconv.FormatInt(item.Version.SizeBytes, 10)
	case "rating":
		cursor.Value = strconv.Itoa(item.Rating)
	}
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func (r *GormRepository) ListSessionAssetUsages(ctx context.Context, accountID, sessionID string) ([]*domain.SessionAssetUsage, error) {
	var rows []SessionAssetUsageRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND session_id = ?", accountID, sessionID).
		Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.SessionAssetUsage, 0, len(rows))
	for _, row := range rows {
		out = append(out, usageFromRow(row))
	}
	return out, nil
}

func usageFromRow(row SessionAssetUsageRow) *domain.SessionAssetUsage {
	return &domain.SessionAssetUsage{
		ID: row.ID, AccountID: row.AccountID, SessionID: row.SessionID, AssetID: row.AssetID,
		AssetVersionID: row.AssetVersionID, UsageKind: row.UsageKind, OperationKey: row.OperationKey,
		RunID: row.RunID, MessageID: row.MessageID, SessionTitleSnapshot: row.SessionTitleSnapshot,
		OperationLabelSnapshot: row.OperationLabelSnapshot, CreatedAt: row.CreatedAt,
	}
}

func (r *GormRepository) ListAssetUsages(ctx context.Context, accountID, assetID string) ([]*domain.SessionAssetUsage, error) {
	var rows []SessionAssetUsageRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND asset_id = ?", accountID, assetID).
		Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.SessionAssetUsage, 0, len(rows))
	for _, row := range rows {
		out = append(out, usageFromRow(row))
	}
	return out, nil
}

func (r *GormRepository) GetAssetVersion(ctx context.Context, accountID, versionID string) (*domain.AssetVersion, error) {
	var row AssetVersionRow
	if err := r.db.WithContext(ctx).Where("account_id = ? AND id = ?", accountID, versionID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	version := assetVersionFromRow(row)
	return &version, nil
}

func (r *GormRepository) ListProjectAssetFormats(ctx context.Context, accountID, projectID string) ([]string, error) {
	if err := r.validateTreeProject(ctx, accountID, projectID); err != nil {
		return nil, err
	}
	var formats []string
	if err := r.db.WithContext(ctx).Table("studio_project_assets AS p").
		Joins("JOIN studio_asset_versions AS v ON v.id = p.asset_version_id AND v.asset_id = p.asset_id AND v.account_id = p.account_id").
		Where("p.account_id = ? AND p.project_id = ? AND p.archived_at IS NULL", accountID, projectID).
		Distinct("v.format").Order("v.format ASC").Pluck("v.format", &formats).Error; err != nil {
		return nil, err
	}
	return formats, nil
}

func (r *GormRepository) ListProjectAssetCounts(ctx context.Context, accountID string) (map[string]int64, error) {
	if accountID == "" {
		return nil, fmt.Errorf("%w: account is required", domain.ErrInvalid)
	}
	var rows []struct {
		ProjectID string
		Count     int64
	}
	if err := r.db.WithContext(ctx).Model(&ProjectAssetRow{}).
		Where("account_id = ? AND archived_at IS NULL", accountID).
		Select("project_id, COUNT(*) AS count").Group("project_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.ProjectID] = row.Count
	}
	return counts, nil
}

func (r *GormRepository) UpdateProjectAsset(ctx context.Context, accountID string, patch domain.ProjectAssetPatch, now time.Time) error {
	if accountID == "" || patch.ID == "" {
		return fmt.Errorf("%w: missing project asset", domain.ErrInvalid)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ProjectAssetRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, patch.ID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		updates := map[string]any{"updated_at": now.UTC()}
		if patch.DisplayName != nil {
			name := strings.TrimSpace(*patch.DisplayName)
			if name == "" || len([]rune(name)) > 256 {
				return fmt.Errorf("%w: invalid display name", domain.ErrInvalid)
			}
			updates["display_name"] = name
		}
		if patch.CategoryID != nil {
			if *patch.CategoryID != "" {
				var category AssetCategoryRow
				if err := tx.Where("account_id = ? AND project_id = ? AND id = ?", accountID, row.ProjectID, *patch.CategoryID).First(&category).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return domain.ErrNotFound
					}
					return err
				}
			}
			updates["category_id"] = *patch.CategoryID
		}
		if patch.Rating != nil {
			if *patch.Rating < 0 || *patch.Rating > 5 {
				return fmt.Errorf("%w: invalid rating", domain.ErrInvalid)
			}
			updates["rating"] = *patch.Rating
		}
		if patch.Archived != nil {
			if *patch.Archived {
				updates["archived_at"] = now.UTC()
			} else {
				updates["archived_at"] = nil
			}
		}
		return tx.Model(&ProjectAssetRow{}).Where("account_id = ? AND id = ?", accountID, patch.ID).Updates(updates).Error
	})
}

func (r *GormRepository) SetProjectAssetVersion(ctx context.Context, accountID, projectAssetID, versionID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ProjectAssetRow
		if err := tx.Where("account_id = ? AND id = ?", accountID, projectAssetID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		var version AssetVersionRow
		if err := tx.Where("account_id = ? AND asset_id = ? AND id = ?", accountID, row.AssetID, versionID).First(&version).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		return tx.Model(&ProjectAssetRow{}).Where("account_id = ? AND id = ?", accountID, projectAssetID).
			Updates(map[string]any{"asset_version_id": versionID, "updated_at": now.UTC()}).Error
	})
}
