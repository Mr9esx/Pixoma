package studio

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/response"
	"github.com/Mr9esx/Pixoma/internal/sharedkernel"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

type assetPaletteView struct {
	Status     string                `json:"status"`
	Colors     []domain.PaletteColor `json:"colors"`
	AnalyzedAt *time.Time            `json:"analyzed_at,omitempty"`
	ErrorCode  string                `json:"error_code,omitempty"`
}

type sessionUsageView struct {
	ID                   string    `json:"id"`
	SessionID            string    `json:"session_id"`
	SessionTitleSnapshot string    `json:"session_title_snapshot"`
	UsageKind            string    `json:"usage_kind"`
	RunID                string    `json:"run_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	SessionAvailable     bool      `json:"session_available"`
}

type assetTagView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type copySourceView struct {
	ProjectID           string `json:"project_id"`
	ProjectNameSnapshot string `json:"project_name_snapshot"`
	ProjectAssetID      string `json:"project_asset_id"`
	DisplayName         string `json:"display_name"`
	Deleted             bool   `json:"deleted"`
}

type projectAssetView struct {
	ID             string           `json:"id"`
	ProjectID      string           `json:"project_id"`
	AssetID        string           `json:"asset_id"`
	AssetVersionID string           `json:"asset_version_id"`
	DisplayName    string           `json:"display_name"`
	CategoryID     string           `json:"category_id"`
	Rating         int              `json:"rating"`
	Tags           []assetTagView   `json:"tags"`
	AddedAt        time.Time        `json:"added_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	ArchivedAt     *time.Time       `json:"archived_at,omitempty"`
	Asset          assetView        `json:"asset"`
	Version        assetVersionView `json:"version"`
	CopySource     *copySourceView  `json:"copy_source,omitempty"`
}

type libraryCategoryView struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	ParentID  string    `json:"parent_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func assetVersionToView(version domain.AssetVersion) assetVersionView {
	return assetVersionView{
		ID: version.ID, Version: version.Version, MIMEType: version.MIMEType,
		SizeBytes: version.SizeBytes, Metadata: version.Metadata,
		ContentURL: "/api/v1/studio/assets/" + url.PathEscape(version.AssetID) + "/content?version_id=" + url.QueryEscape(version.ID),
		CreatedAt:  version.CreatedAt, Format: version.Format, WidthPx: version.WidthPx, HeightPx: version.HeightPx,
		SourceCreatedAt: version.SourceCreatedAt, SourceModifiedAt: version.SourceModifiedAt,
		ContentOrigin: version.ContentOrigin,
	}
}

func toSessionUsageView(usage *domain.SessionAssetUsage) sessionUsageView {
	return sessionUsageView{
		ID: usage.ID, SessionID: usage.SessionID, SessionTitleSnapshot: usage.SessionTitleSnapshot,
		UsageKind: usage.UsageKind, RunID: usage.RunID, CreatedAt: usage.CreatedAt, SessionAvailable: true,
	}
}

func toLibraryCategoryView(category *domain.AssetCategory) libraryCategoryView {
	return libraryCategoryView{
		ID: category.ID, ProjectID: category.ProjectID, ParentID: category.ParentID,
		Name: category.Name, CreatedAt: category.CreatedAt, UpdatedAt: category.UpdatedAt,
	}
}

func (h *Handler) projectAssetToView(ctx context.Context, accountID string, item *domain.ProjectAsset) (projectAssetView, error) {
	if item == nil || item.Asset == nil {
		return projectAssetView{}, fmt.Errorf("%w: missing project asset data", domain.ErrNotFound)
	}
	view := projectAssetView{
		ID: item.ID, ProjectID: item.ProjectID, AssetID: item.AssetID, AssetVersionID: item.AssetVersionID,
		DisplayName: item.DisplayName, CategoryID: item.CategoryID, Rating: item.Rating,
		Tags: make([]assetTagView, 0, len(item.Tags)), AddedAt: item.AddedAt, UpdatedAt: item.UpdatedAt,
		ArchivedAt: item.ArchivedAt, Asset: assetsToViews([]*domain.Asset{item.Asset})[0],
		Version: assetVersionToView(item.Version),
	}
	for _, tag := range item.Tags {
		view.Tags = append(view.Tags, assetTagView{ID: tag.ID, Name: tag.Name})
	}
	if item.SourceProjectAssetID != "" {
		view.CopySource = &copySourceView{
			ProjectID: item.SourceProjectID, ProjectAssetID: item.SourceProjectAssetID,
			ProjectNameSnapshot: item.SourceProjectNameSnapshot,
			DisplayName:         item.SourceDisplayNameSnapshot, Deleted: item.SourceDeleted,
		}
	}
	palette, err := h.Repo.GetAssetVersionPalette(ctx, accountID, item.AssetVersionID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return projectAssetView{}, err
	}
	if palette != nil {
		view.Version.Palette = &assetPaletteView{
			Status: palette.Status, Colors: palette.Colors, AnalyzedAt: palette.AnalyzedAt,
			ErrorCode: palette.ErrorCode,
		}
		if view.Version.Palette.Colors == nil {
			view.Version.Palette.Colors = []domain.PaletteColor{}
		}
	}
	return view, nil
}

func libraryProjectID(w http.ResponseWriter, r *http.Request) (string, bool) {
	values, present := r.URL.Query()["project_id"]
	if !present || len(values) != 1 {
		response.Fail(w, apierr.ErrStudioInvalidBody, "请指定项目")
		return "", false
	}
	return strings.TrimSpace(values[0]), true
}

func libraryBadRequest(w http.ResponseWriter, message string) {
	response.Fail(w, apierr.ErrStudioInvalidBody, message)
}

func (h *Handler) listLibraryProjects(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	projects, err := h.Repo.ListProjects(r.Context(), accountID)
	if err != nil {
		failFromError(w, err)
		return
	}
	counts, err := h.Repo.ListProjectAssetCounts(r.Context(), accountID)
	if err != nil {
		failFromError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(projects)+1)
	slices.SortFunc(projects, func(left, right *domain.Project) int {
		if order := strings.Compare(left.Name, right.Name); order != 0 {
			return order
		}
		return strings.Compare(left.ID, right.ID)
	})
	for _, project := range projects {
		items = append(items, map[string]any{"id": project.ID, "name": project.Name, "asset_count": counts[project.ID]})
	}
	items = append(items, map[string]any{"id": "", "name": "未归属项目", "asset_count": counts[""]})
	response.OK(w, items)
}

func (h *Handler) listLibraryTree(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	projectID, ok := libraryProjectID(w, r)
	if !ok {
		return
	}
	page, err := h.Repo.ListProjectAssetTree(r.Context(), accountID, domain.ProjectAssetTreeQuery{
		ProjectID: projectID, Mode: r.URL.Query().Get("mode"), ParentID: r.URL.Query().Get("parent_id"),
		Limit: queryInt(r, "limit", 50), Cursor: r.URL.Query().Get("cursor"),
	})
	if err != nil {
		failFromError(w, err)
		return
	}
	nodes := make([]map[string]any, 0, len(page.Nodes))
	for _, node := range page.Nodes {
		value := map[string]any{"id": node.ID, "label": node.Name, "count": node.Count, "kind": node.Type}
		if node.ProjectAssetID != "" {
			value["asset_id"] = node.ProjectAssetID
			value["asset_kind"] = node.AssetKind
		} else if directID, direct := strings.CutPrefix(node.ID, "category-direct:"); direct {
			value["group_value"] = "direct:" + directID
		} else if _, group, found := strings.Cut(node.ID, ":"); found {
			value["group_value"] = group
		}
		nodes = append(nodes, value)
	}
	response.OK(w, map[string]any{"nodes": nodes, "next_cursor": page.NextCursor})
}

func (h *Handler) listProjectAssets(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	projectID, ok := libraryProjectID(w, r)
	if !ok {
		return
	}
	query := domain.ProjectAssetListQuery{
		ProjectID: projectID, Search: r.URL.Query().Get("q"), Kind: domain.AssetKind(r.URL.Query().Get("kind")),
		Format: r.URL.Query().Get("format"), SessionID: r.URL.Query().Get("session_id"),
		CategoryID: r.URL.Query().Get("category_id"), Sort: r.URL.Query().Get("sort"),
		Limit: queryInt(r, "limit", 50), Cursor: r.URL.Query().Get("cursor"),
	}
	if query.CategoryID == "uncategorized" || query.CategoryID == "none" {
		query.CategoryID = ""
		query.Uncategorized = true
	} else if directID, direct := strings.CutPrefix(query.CategoryID, "direct:"); direct {
		query.CategoryID = directID
		query.CategoryDirect = true
	}
	if query.SessionID == "none" {
		query.SessionID = ""
		query.NoSession = true
	}
	if raw := r.URL.Query().Get("tag_ids"); raw != "" {
		if raw == "untagged" {
			query.Untagged = true
		} else {
			query.TagIDs = strings.Split(raw, ",")
		}
	}
	if raw := r.URL.Query().Get("rating"); raw != "" {
		rating, err := strconv.Atoi(raw)
		if err != nil || rating < 0 || rating > 5 {
			libraryBadRequest(w, "评分筛选无效")
			return
		}
		query.Rating = &rating
	}
	if raw := r.URL.Query().Get("archived"); raw != "" {
		archived, err := strconv.ParseBool(raw)
		if err != nil {
			libraryBadRequest(w, "归档筛选无效")
			return
		}
		query.ArchivedOnly = archived
	}
	values := r.URL.Query()
	var filterErr error
	if query.MinWidthPx, filterErr = libraryNonnegativeInt(values, "width_min"); filterErr == nil {
		query.MaxWidthPx, filterErr = libraryNonnegativeInt(values, "width_max")
	}
	if filterErr == nil {
		query.MinHeightPx, filterErr = libraryNonnegativeInt(values, "height_min")
	}
	if filterErr == nil {
		query.MaxHeightPx, filterErr = libraryNonnegativeInt(values, "height_max")
	}
	if filterErr == nil {
		query.MinSizeBytes, filterErr = libraryNonnegativeInt64(values, "size_min")
	}
	if filterErr == nil {
		query.MaxSizeBytes, filterErr = libraryNonnegativeInt64(values, "size_max")
	}
	if filterErr == nil {
		query.AddedFrom, filterErr = libraryTime(values, "added_from")
	}
	if filterErr == nil {
		query.AddedTo, filterErr = libraryTime(values, "added_to")
	}
	if filterErr == nil {
		if raw, present := values["duplicates"]; present {
			if len(raw) != 1 {
				filterErr = fmt.Errorf("重复文件筛选无效")
			} else {
				query.DuplicatesOnly, filterErr = strconv.ParseBool(raw[0])
			}
		}
	}
	if filterErr != nil || invalidLibraryRange(query.MinWidthPx, query.MaxWidthPx) ||
		invalidLibraryRange(query.MinHeightPx, query.MaxHeightPx) ||
		invalidLibraryRange(query.MinSizeBytes, query.MaxSizeBytes) ||
		query.AddedFrom != nil && query.AddedTo != nil && !query.AddedFrom.Before(*query.AddedTo) {
		libraryBadRequest(w, "资产筛选条件无效")
		return
	}
	page, err := h.Repo.ListProjectAssets(r.Context(), accountID, query)
	if err != nil {
		failFromError(w, err)
		return
	}
	items := make([]projectAssetView, 0, len(page.Items))
	for _, item := range page.Items {
		view, err := h.projectAssetToView(r.Context(), accountID, item)
		if err != nil {
			failFromError(w, err)
			return
		}
		items = append(items, view)
	}
	response.OK(w, map[string]any{"items": items, "total": page.Total, "next_cursor": page.NextCursor})
}

func libraryNonnegativeInt(values url.Values, key string) (*int, error) {
	raw, present := values[key]
	if !present {
		return nil, nil
	}
	if len(raw) != 1 {
		return nil, fmt.Errorf("%s 筛选无效", key)
	}
	value, err := strconv.Atoi(raw[0])
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s 筛选无效", key)
	}
	return &value, nil
}

func libraryNonnegativeInt64(values url.Values, key string) (*int64, error) {
	raw, present := values[key]
	if !present {
		return nil, nil
	}
	if len(raw) != 1 {
		return nil, fmt.Errorf("%s 筛选无效", key)
	}
	value, err := strconv.ParseInt(raw[0], 10, 64)
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s 筛选无效", key)
	}
	return &value, nil
}

func libraryTime(values url.Values, key string) (*time.Time, error) {
	raw, present := values[key]
	if !present {
		return nil, nil
	}
	if len(raw) != 1 {
		return nil, fmt.Errorf("%s 筛选无效", key)
	}
	value, err := time.Parse(time.RFC3339, raw[0])
	if err != nil {
		return nil, fmt.Errorf("%s 筛选无效", key)
	}
	return &value, nil
}

func invalidLibraryRange[T int | int64](minimum, maximum *T) bool {
	return minimum != nil && maximum != nil && *minimum > *maximum
}

func (h *Handler) getProjectAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	item, err := h.Repo.GetProjectAsset(r.Context(), accountID, chi.URLParam(r, "projectAssetID"))
	if err != nil {
		failFromError(w, err)
		return
	}
	view, err := h.projectAssetToView(r.Context(), accountID, item)
	if err != nil {
		failFromError(w, err)
		return
	}
	usages, err := h.Repo.ListAssetUsages(r.Context(), accountID, item.AssetID)
	if err != nil {
		failFromError(w, err)
		return
	}
	usageViews := make([]sessionUsageView, 0, len(usages))
	for _, usage := range usages {
		usageView := toSessionUsageView(usage)
		if _, err := h.Repo.GetSession(r.Context(), accountID, usage.SessionID); errors.Is(err, domain.ErrNotFound) {
			usageView.SessionAvailable = false
		} else if err != nil {
			failFromError(w, err)
			return
		}
		usageViews = append(usageViews, usageView)
	}
	versions := make([]assetVersionView, 0, len(item.Asset.Versions))
	for _, version := range item.Asset.Versions {
		versionView := assetVersionToView(version)
		palette, err := h.Repo.GetAssetVersionPalette(r.Context(), accountID, version.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			failFromError(w, err)
			return
		}
		if palette != nil {
			versionView.Palette = &assetPaletteView{Status: palette.Status, Colors: palette.Colors, AnalyzedAt: palette.AnalyzedAt, ErrorCode: palette.ErrorCode}
			if versionView.Palette.Colors == nil {
				versionView.Palette.Colors = []domain.PaletteColor{}
			}
		}
		versions = append(versions, versionView)
	}
	response.OK(w, struct {
		projectAssetView
		Usages   []sessionUsageView `json:"usages"`
		Versions []assetVersionView `json:"versions"`
	}{view, usageViews, versions})
}

func (h *Handler) listDuplicateProjectAssets(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	items, err := h.Repo.ListDuplicateProjectAssets(r.Context(), accountID, chi.URLParam(r, "projectAssetID"))
	if err != nil {
		failFromError(w, err)
		return
	}
	views := make([]projectAssetView, 0, len(items))
	for _, item := range items {
		view, err := h.projectAssetToView(r.Context(), accountID, item)
		if err != nil {
			failFromError(w, err)
			return
		}
		views = append(views, view)
	}
	response.OK(w, map[string]any{"items": views})
}

func (h *Handler) patchProjectAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		DisplayName *string `json:"display_name"`
		CategoryID  *string `json:"category_id"`
		Rating      *int    `json:"rating"`
		Archived    *bool   `json:"archived"`
	}
	if err := decodeJSON(r, &body); err != nil {
		libraryBadRequest(w, "请求内容格式不正确")
		return
	}
	id := chi.URLParam(r, "projectAssetID")
	if err := h.Repo.UpdateProjectAsset(r.Context(), accountID, domain.ProjectAssetPatch{
		ID: id, DisplayName: body.DisplayName, CategoryID: body.CategoryID,
		Rating: body.Rating, Archived: body.Archived,
	}, time.Now().UTC()); err != nil {
		failFromError(w, err)
		return
	}
	h.respondProjectAsset(w, r, accountID, id)
}

func (h *Handler) deleteProjectAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if err := h.Repo.DeleteProjectAsset(r.Context(), accountID, chi.URLParam(r, "projectAssetID")); err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, nil)
}

func (h *Handler) patchProjectAssetVersion(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		AssetVersionID string `json:"asset_version_id"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.AssetVersionID) == "" {
		libraryBadRequest(w, "请选择资产版本")
		return
	}
	id := chi.URLParam(r, "projectAssetID")
	if err := h.Repo.SetProjectAssetVersion(r.Context(), accountID, id, body.AssetVersionID, time.Now().UTC()); err != nil {
		failFromError(w, err)
		return
	}
	h.respondProjectAsset(w, r, accountID, id)
}

func (h *Handler) copyProjectAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		ProjectID      string `json:"project_id"`
		AssetVersionID string `json:"asset_version_id"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.AssetVersionID) == "" {
		libraryBadRequest(w, "请选择资产版本")
		return
	}
	now := time.Now().UTC()
	target := &domain.ProjectAsset{
		ID: uuid.NewString(), AccountID: accountID, ProjectID: body.ProjectID,
		AssetVersionID: body.AssetVersionID, AddedAt: now, UpdatedAt: now,
	}
	if err := h.Repo.CopyProjectAsset(r.Context(), accountID, chi.URLParam(r, "projectAssetID"), target); err != nil {
		failFromError(w, err)
		return
	}
	h.respondProjectAssetStatus(w, r, accountID, target.ID, http.StatusCreated)
}

func (h *Handler) putProjectAssetTags(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		TagIDs []string `json:"tag_ids"`
	}
	if err := decodeJSON(r, &body); err != nil || body.TagIDs == nil {
		libraryBadRequest(w, "请提交标签列表")
		return
	}
	id := chi.URLParam(r, "projectAssetID")
	if err := h.Repo.SetProjectAssetTags(r.Context(), accountID, id, body.TagIDs); err != nil {
		failFromError(w, err)
		return
	}
	h.respondProjectAsset(w, r, accountID, id)
}

func (h *Handler) respondProjectAsset(w http.ResponseWriter, r *http.Request, accountID, id string) {
	h.respondProjectAssetStatus(w, r, accountID, id, http.StatusOK)
}

func (h *Handler) respondProjectAssetStatus(w http.ResponseWriter, r *http.Request, accountID, id string, status int) {
	item, err := h.Repo.GetProjectAsset(r.Context(), accountID, id)
	if err != nil {
		failFromError(w, err)
		return
	}
	view, err := h.projectAssetToView(r.Context(), accountID, item)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OKStatus(w, status, view)
}

func (h *Handler) batchProjectAssets(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		ProjectAssetIDs []string `json:"project_asset_ids"`
		Action          string   `json:"action"`
		CategoryID      string   `json:"category_id"`
		TagIDs          []string `json:"tag_ids"`
		Rating          *int     `json:"rating"`
		Archived        *bool    `json:"archived"`
		ProjectID       string   `json:"project_id"`
	}
	if err := decodeJSON(r, &body); err != nil || len(body.ProjectAssetIDs) == 0 || len(body.ProjectAssetIDs) > 100 {
		libraryBadRequest(w, "请选择 1 至 100 项资产")
		return
	}
	switch body.Action {
	case "category", "tags", "tags_remove", "rating", "archive", "project":
	default:
		libraryBadRequest(w, "批量操作无效")
		return
	}
	if (body.Action == "rating" && body.Rating == nil) || (body.Action == "archive" && body.Archived == nil) ||
		((body.Action == "tags" || body.Action == "tags_remove") && body.TagIDs == nil) {
		libraryBadRequest(w, "缺少批量操作内容")
		return
	}
	results := make([]map[string]any, 0, len(body.ProjectAssetIDs))
	for _, id := range body.ProjectAssetIDs {
		var err error
		switch body.Action {
		case "category":
			err = h.Repo.UpdateProjectAsset(r.Context(), accountID, domain.ProjectAssetPatch{ID: id, CategoryID: &body.CategoryID}, time.Now().UTC())
		case "tags", "tags_remove":
			item, getErr := h.Repo.GetProjectAsset(r.Context(), accountID, id)
			if getErr != nil {
				err = getErr
				break
			}
			merged := make([]string, 0, len(item.Tags)+len(body.TagIDs))
			selected := make(map[string]bool, len(body.TagIDs))
			for _, tagID := range body.TagIDs {
				selected[tagID] = true
			}
			if body.Action == "tags_remove" {
				for _, tag := range item.Tags {
					if !selected[tag.ID] {
						merged = append(merged, tag.ID)
					}
				}
			} else {
				for _, tag := range item.Tags {
					merged = append(merged, tag.ID)
					delete(selected, tag.ID)
				}
				for _, tagID := range body.TagIDs {
					if selected[tagID] {
						merged = append(merged, tagID)
						delete(selected, tagID)
					}
				}
			}
			err = h.Repo.SetProjectAssetTags(r.Context(), accountID, id, merged)
		case "rating":
			err = h.Repo.UpdateProjectAsset(r.Context(), accountID, domain.ProjectAssetPatch{ID: id, Rating: body.Rating}, time.Now().UTC())
		case "archive":
			err = h.Repo.UpdateProjectAsset(r.Context(), accountID, domain.ProjectAssetPatch{ID: id, Archived: body.Archived}, time.Now().UTC())
		case "project":
			now := time.Now().UTC()
			err = h.Repo.CopyProjectAsset(r.Context(), accountID, id, &domain.ProjectAsset{
				ID: uuid.NewString(), AccountID: accountID, ProjectID: body.ProjectID, AddedAt: now, UpdatedAt: now,
			})
		}
		result := map[string]any{"project_asset_id": id, "success": err == nil}
		if err != nil {
			result["error"] = err.Error()
		}
		results = append(results, result)
	}
	response.OK(w, map[string]any{"results": results})
}

func (h *Handler) listProjectCategories(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	projectID, ok := libraryProjectID(w, r)
	if !ok {
		return
	}
	if projectID != "" {
		if _, err := h.Repo.GetProject(r.Context(), accountID, projectID); err != nil {
			failFromError(w, err)
			return
		}
	}
	categories, err := h.Repo.ListAssetCategories(r.Context(), accountID, projectID)
	if err != nil {
		failFromError(w, err)
		return
	}
	views := make([]libraryCategoryView, 0, len(categories))
	for _, category := range categories {
		views = append(views, toLibraryCategoryView(category))
	}
	response.OK(w, views)
}

func (h *Handler) createProjectCategory(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		ProjectID string `json:"project_id"`
		ParentID  string `json:"parent_id"`
		Name      string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		libraryBadRequest(w, "分类名称不能为空")
		return
	}
	now := time.Now().UTC()
	category := &domain.AssetCategory{ID: uuid.NewString(), AccountID: accountID, ProjectID: body.ProjectID, ParentID: body.ParentID, Name: body.Name, CreatedAt: now, UpdatedAt: now}
	if err := h.Repo.CreateAssetCategory(r.Context(), category); err != nil {
		failFromError(w, err)
		return
	}
	response.OKStatus(w, http.StatusCreated, toLibraryCategoryView(category))
}

func (h *Handler) patchProjectCategory(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		ParentID *string `json:"parent_id"`
		Name     *string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || (body.Name == nil && body.ParentID == nil) {
		libraryBadRequest(w, "请提交分类修改内容")
		return
	}
	id := chi.URLParam(r, "categoryID")
	category, err := h.Repo.GetAssetCategory(r.Context(), accountID, id)
	if err != nil {
		failFromError(w, err)
		return
	}
	name, parentID := category.Name, category.ParentID
	if body.Name != nil {
		name = *body.Name
	}
	if body.ParentID != nil {
		parentID = *body.ParentID
	}
	if err := h.Repo.UpdateAssetCategory(r.Context(), accountID, id, parentID, name, time.Now().UTC()); err != nil {
		failFromError(w, err)
		return
	}
	category, err = h.Repo.GetAssetCategory(r.Context(), accountID, id)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, toLibraryCategoryView(category))
}

func (h *Handler) deleteProjectCategory(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if err := h.Repo.DeleteAssetCategory(r.Context(), accountID, chi.URLParam(r, "categoryID")); err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, nil)
}

func (h *Handler) listLibraryFormats(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	projectID, ok := libraryProjectID(w, r)
	if !ok {
		return
	}
	formats, err := h.Repo.ListProjectAssetFormats(r.Context(), accountID, projectID)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, formats)
}

func (h *Handler) listLibraryTags(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	tags, err := h.Repo.ListAssetTags(r.Context(), accountID)
	if err != nil {
		failFromError(w, err)
		return
	}
	views := make([]assetTagView, 0, len(tags))
	for _, tag := range tags {
		views = append(views, assetTagView{ID: tag.ID, Name: tag.Name})
	}
	response.OK(w, views)
}

func (h *Handler) createLibraryTag(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		libraryBadRequest(w, "标签名称不能为空")
		return
	}
	now := time.Now().UTC()
	tag := &domain.AssetTag{ID: uuid.NewString(), AccountID: accountID, Name: body.Name, CreatedAt: now, UpdatedAt: now}
	if err := h.Repo.CreateAssetTag(r.Context(), tag); err != nil {
		failFromError(w, err)
		return
	}
	response.OKStatus(w, http.StatusCreated, assetTagView{ID: tag.ID, Name: strings.TrimSpace(tag.Name)})
}

func (h *Handler) patchLibraryTag(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		libraryBadRequest(w, "标签名称不能为空")
		return
	}
	id := chi.URLParam(r, "tagID")
	if err := h.Repo.RenameAssetTag(r.Context(), accountID, id, body.Name, time.Now().UTC()); err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, assetTagView{ID: id, Name: strings.TrimSpace(body.Name)})
}

func (h *Handler) getLibraryPreferences(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	preferences, err := h.Repo.GetAssetLibraryPreferences(r.Context(), accountID)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, map[string]any{"tree_mode": preferences.TreeMode, "last_project_id": preferences.LastProjectID})
}

func (h *Handler) putLibraryPreferences(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		TreeMode      string `json:"tree_mode"`
		LastProjectID string `json:"last_project_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		libraryBadRequest(w, "请求内容格式不正确")
		return
	}
	preferences := &domain.AssetLibraryPreferences{AccountID: accountID, TreeMode: body.TreeMode, LastProjectID: body.LastProjectID, UpdatedAt: time.Now().UTC()}
	if err := h.Repo.SaveAssetLibraryPreferences(r.Context(), preferences); err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, map[string]any{"tree_mode": preferences.TreeMode, "last_project_id": preferences.LastProjectID})
}

func (h *Handler) retryAssetPalette(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	assetID, versionID := chi.URLParam(r, "assetID"), chi.URLParam(r, "versionID")
	version, err := h.Repo.GetAssetVersion(r.Context(), accountID, versionID)
	if err != nil {
		failFromError(w, err)
		return
	}
	if version.AssetID != assetID {
		failFromError(w, domain.ErrNotFound)
		return
	}
	if err := h.Repo.RetryAssetVersionPalette(r.Context(), accountID, versionID, time.Now().UTC()); err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, nil)
}

func (h *Handler) sessionAssetViews(ctx context.Context, accountID, sessionID string) ([]assetView, error) {
	usages, err := h.Repo.ListSessionAssetUsages(ctx, accountID, sessionID)
	if err != nil {
		return nil, err
	}
	views := make([]assetView, 0)
	indices := make(map[string]int)
	for _, usage := range usages {
		index, found := indices[usage.AssetID]
		if !found {
			asset, err := h.Repo.GetAsset(ctx, accountID, usage.AssetID)
			if err != nil {
				return nil, err
			}
			index = len(views)
			indices[usage.AssetID] = index
			views = append(views, assetsToViews([]*domain.Asset{asset})[0])
		}
		views[index].Usages = append(views[index].Usages, toSessionUsageView(usage))
	}
	return views, nil
}

func (h *Handler) listSessionAssetViews(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	sessionID := chi.URLParam(r, "sessionID")
	if _, err := h.Repo.GetSession(r.Context(), accountID, sessionID); err != nil {
		failFromError(w, err)
		return
	}
	views, err := h.sessionAssetViews(r.Context(), accountID, sessionID)
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OK(w, views)
}

func (h *Handler) referenceSessionAsset(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	var body struct {
		AssetID              string `json:"asset_id"`
		AssetVersionID       string `json:"asset_version_id"`
		RequestID            string `json:"request_id"`
		SourceProjectAssetID string `json:"source_project_asset_id"`
	}
	if err := decodeJSON(r, &body); err != nil || body.AssetID == "" || body.AssetVersionID == "" {
		libraryBadRequest(w, "请选择资产和版本")
		return
	}
	requestID, err := uuid.Parse(body.RequestID)
	if err != nil {
		libraryBadRequest(w, "request_id 必须为 UUID")
		return
	}
	sessionID := chi.URLParam(r, "sessionID")
	session, err := h.Repo.GetSession(r.Context(), accountID, sessionID)
	if err != nil {
		failFromError(w, err)
		return
	}
	asset, err := h.Repo.GetAsset(r.Context(), accountID, body.AssetID)
	if err != nil {
		failFromError(w, err)
		return
	}
	version, err := h.Repo.GetAssetVersion(r.Context(), accountID, body.AssetVersionID)
	if err != nil {
		failFromError(w, err)
		return
	}
	if version.AssetID != asset.ID {
		failFromError(w, domain.ErrNotFound)
		return
	}
	now := time.Now().UTC()
	placement := &domain.ProjectAsset{
		ID: uuid.NewString(), AccountID: accountID, ProjectID: session.ProjectID,
		AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name,
		AddedAt: now, UpdatedAt: now,
	}
	if body.SourceProjectAssetID != "" {
		source, err := h.Repo.GetProjectAsset(r.Context(), accountID, body.SourceProjectAssetID)
		if err != nil {
			failFromError(w, err)
			return
		}
		if source.AssetID != asset.ID {
			failFromError(w, domain.ErrNotFound)
			return
		}
		if source.ProjectID != session.ProjectID {
			placement.SourceProjectAssetID = source.ID
			placement.SourceProjectID = source.ProjectID
			placement.SourceDisplayNameSnapshot = source.DisplayName
			placement.SourceAssetVersionID = version.ID
			placement.CopiedAt = &now
			if source.ProjectID != "" {
				project, err := h.Repo.GetProject(r.Context(), accountID, source.ProjectID)
				if err != nil {
					failFromError(w, err)
					return
				}
				placement.SourceProjectNameSnapshot = project.Name
			}
		}
	}
	usage := &domain.SessionAssetUsage{
		ID:        uuid.NewSHA1(uuid.NameSpaceOID, []byte(accountID+"\x00"+sessionID+"\x00"+requestID.String())).String(),
		AccountID: accountID, SessionID: sessionID, AssetID: asset.ID, AssetVersionID: version.ID,
		UsageKind: "referenced", OperationKey: requestID.String(), SessionTitleSnapshot: session.Title,
		CreatedAt: now,
	}
	if err := h.Repo.ReferenceAssetInSession(r.Context(), placement, usage); err != nil {
		failFromError(w, err)
		return
	}
	view := assetsToViews([]*domain.Asset{asset})[0]
	view.Usages = []sessionUsageView{toSessionUsageView(usage)}
	response.OKStatus(w, http.StatusCreated, view)
}

func (h *Handler) exportProjectAssets(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Blob == nil {
		response.Fail(w, apierr.ErrStudioCreateTextAssetUnavailable, "资产存储服务不可用")
		return
	}
	var body struct {
		ProjectAssetIDs []string `json:"project_asset_ids"`
	}
	if err := decodeJSON(r, &body); err != nil || len(body.ProjectAssetIDs) == 0 || len(body.ProjectAssetIDs) > 100 {
		libraryBadRequest(w, "请选择 1 至 100 项资产")
		return
	}
	items := make([]*domain.ProjectAsset, 0, len(body.ProjectAssetIDs))
	for _, id := range body.ProjectAssetIDs {
		item, err := h.Repo.GetProjectAsset(r.Context(), accountID, id)
		if err != nil {
			failFromError(w, err)
			return
		}
		items = append(items, item)
	}
	if len(items) == 1 {
		item := items[0]
		reader, err := h.Blob.Get(r.Context(), sharedkernel.BlobRef{Key: item.Version.BlobKey, MIME: item.Version.MIMEType, Size: item.Version.SizeBytes})
		if err != nil {
			failFromError(w, err)
			return
		}
		defer reader.Close()
		w.Header().Set("Content-Type", item.Version.MIMEType)
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(exportName(item.DisplayName, item.ID)))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Length", strconv.FormatInt(item.Version.SizeBytes, 10))
		if _, err := io.Copy(w, reader); err != nil {
			slog.Error("studio asset export failed", "error", err, "project_asset_id", item.ID)
		}
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''studio-assets.zip")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	archive := zip.NewWriter(w)
	used := make(map[string]bool, len(items))
	for _, item := range items {
		reader, err := h.Blob.Get(r.Context(), sharedkernel.BlobRef{Key: item.Version.BlobKey, MIME: item.Version.MIMEType, Size: item.Version.SizeBytes})
		if err != nil {
			slog.Error("studio asset export failed", "error", err, "project_asset_id", item.ID)
			return
		}
		name := exportName(item.DisplayName, item.ID)
		base, ext := strings.TrimSuffix(name, path.Ext(name)), path.Ext(name)
		for sequence := 2; used[name]; sequence++ {
			name = fmt.Sprintf("%s (%d)%s", base, sequence, ext)
		}
		used[name] = true
		entry, err := archive.Create(name)
		if err != nil {
			reader.Close()
			slog.Error("studio asset archive failed", "error", err)
			return
		}
		_, copyErr := io.Copy(entry, reader)
		closeErr := reader.Close()
		if copyErr != nil || closeErr != nil {
			slog.Error("studio asset archive failed", "copy_error", copyErr, "close_error", closeErr)
			return
		}
	}
	if err := archive.Close(); err != nil {
		slog.Error("studio asset archive failed", "error", err)
	}
}

func exportName(name, assetID string) string {
	name = path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	name = strings.Map(func(value rune) rune {
		if value < 0x20 || value == 0x7f {
			return -1
		}
		return value
	}, name)
	if name == "" || name == "." || name == ".." {
		return assetID
	}
	return name
}
