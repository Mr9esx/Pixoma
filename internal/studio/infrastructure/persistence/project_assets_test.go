package persistence_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestProjectAssetCopyKeepsContentAndSourceAfterSourceDeletion(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	for _, id := range []string{"source-project", "target-project"} {
		project, err := domain.NewProject(id, "account-a", id, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
	}
	asset, err := domain.NewAsset("asset-a", "account-a", "原始图片.png", domain.AssetImage, domain.AssetOriginUser, now)
	if err != nil {
		t.Fatal(err)
	}
	version, err := asset.AppendVersion("version-a", "image/png", "studio/account-a/asset-a/v1.png", 64, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := asset.AppendVersion("version-b", "image/png", "studio/account-a/asset-a/v2.png", 96, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	source := &domain.ProjectAsset{ID: "entry-source", AccountID: "account-a", ProjectID: "source-project", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: "封面", AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, source, nil); err != nil {
		t.Fatal(err)
	}
	target := &domain.ProjectAsset{ID: "entry-target", AccountID: "account-a", ProjectID: "target-project", AssetVersionID: second.ID, AddedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)}
	if err := repo.CopyProjectAsset(ctx, "account-a", source.ID, target); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteProjectAsset(ctx, "account-a", source.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetProjectAsset(ctx, "account-a", source.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("source remains: %v", err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AssetID != asset.ID || got.AssetVersionID != second.ID || got.SourceProjectAssetID != source.ID || got.SourceProjectID != "source-project" || got.SourceProjectNameSnapshot != "source-project" || got.SourceDisplayNameSnapshot != "封面" || got.SourceAssetVersionID != version.ID || !got.SourceDeleted {
		t.Fatalf("copied entry = %#v", got)
	}
	if got.Asset == nil || got.Asset.ID != asset.ID || got.Version.ID != second.ID {
		t.Fatalf("copied content = %#v", got)
	}
	if _, err := repo.GetProjectAsset(ctx, "account-b", target.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other account read = %v", err)
	}
}

func TestProjectAssetListUsesProjectAndPinnedVersion(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 11, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ id, name, format string }{{"one", "蓝色封面", "png"}, {"two", "红色封面", "jpeg"}, {"three", "蓝色草图", "png"}} {
		asset, err := domain.NewAsset(item.id, "account-a", item.name, domain.AssetImage, domain.AssetOriginUser, now)
		if err != nil {
			t.Fatal(err)
		}
		version, err := asset.AppendVersion(item.id+"-v1", "image/"+item.format, "studio/account-a/"+item.id, 10, now)
		if err != nil {
			t.Fatal(err)
		}
		asset.Versions[0].Format = item.format
		entry := &domain.ProjectAsset{ID: "entry-" + item.id, AccountID: "account-a", ProjectID: "project-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: item.name, Rating: 4, AddedAt: now, UpdatedAt: now}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: "project-a", Format: "png", Search: "蓝色", Limit: 1})
	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("first page = (%#v, %v)", page, err)
	}
	second, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: "project-a", Format: "png", Search: "蓝色", Limit: 1, Cursor: page.NextCursor})
	if err != nil || second.Total != 2 || len(second.Items) != 1 || second.Items[0].ID == page.Items[0].ID || second.NextCursor != "" {
		t.Fatalf("second page = (%#v, %v)", second, err)
	}
	if second.Items[0].Version.Format != "png" {
		t.Fatalf("pinned version = %#v", second.Items[0].Version)
	}
	other, err := repo.ListProjectAssets(ctx, "account-b", domain.ProjectAssetListQuery{ProjectID: "project-a", Limit: 10})
	if !errors.Is(err, domain.ErrNotFound) || other != nil {
		t.Fatalf("cross-account list = (%#v, %v)", other, err)
	}
}

func TestCreateAssetWithPlacementRollsBackInvalidUsage(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	asset, _ := domain.NewAsset("asset-invalid", "account-a", "草图.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-invalid-v1", "image/png", "studio/account-a/invalid", 4, now)
	placement := &domain.ProjectAsset{ID: "entry-invalid", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	usage := &domain.SessionAssetUsage{ID: "usage-invalid", AccountID: "account-a", SessionID: "missing", AssetID: asset.ID, AssetVersionID: version.ID, UsageKind: "created", OperationKey: "request-invalid", CreatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, placement, usage); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing session = %v", err)
	}
	if _, err := repo.GetAsset(ctx, "account-a", asset.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("asset survived failed transaction: %v", err)
	}
	if _, err := repo.GetProjectAsset(ctx, "account-a", placement.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("entry survived failed transaction: %v", err)
	}
}

func TestSessionAssetsRemainInProjectAfterClear(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 13, 0, 0, 0, time.UTC)
	session, _ := domain.NewSession("session-a", "account-a", now)
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "文稿.md", domain.AssetDocument, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "text/markdown", "studio/account-a/a.md", 5, now)
	placement := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	usage := &domain.SessionAssetUsage{ID: "usage-a", AccountID: "account-a", SessionID: session.ID, AssetID: asset.ID, AssetVersionID: version.ID, UsageKind: "created", OperationKey: "request-a", SessionTitleSnapshot: "新对话", CreatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, placement, usage); err != nil {
		t.Fatal(err)
	}
	if err := repo.ClearSessions(ctx, "account-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetAsset(ctx, "account-a", asset.ID); err != nil {
		t.Fatalf("asset removed: %v", err)
	}
	if _, err := repo.GetProjectAsset(ctx, "account-a", placement.ID); err != nil {
		t.Fatalf("entry removed: %v", err)
	}
	usages, err := repo.ListSessionAssetUsages(ctx, "account-a", session.ID)
	if err != nil || len(usages) != 1 || usages[0].SessionTitleSnapshot != "新对话" {
		t.Fatalf("source snapshot = (%#v, %v)", usages, err)
	}
	byAsset, err := repo.ListAssetUsages(ctx, "account-a", asset.ID)
	if err != nil || len(byAsset) != 1 || byAsset[0].SessionID != session.ID {
		t.Fatalf("asset sources = (%#v, %v)", byAsset, err)
	}
}

func TestReferenceAssetInSessionKeepsExistingProjectVersion(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 14, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	session, _ := domain.NewSession("session-a", "account-a", now)
	session.ProjectID = project.ID
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "文稿.md", domain.AssetDocument, domain.AssetOriginUser, now)
	first, _ := asset.AppendVersion("asset-a-v1", "text/markdown", "studio/account-a/a-v1.md", 5, now)
	placement := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: first.ID, DisplayName: "项目文稿", AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, placement, nil); err != nil {
		t.Fatal(err)
	}
	second, _ := asset.AppendVersion("asset-a-v2", "text/markdown", "studio/account-a/a-v2.md", 6, now.Add(time.Minute))
	if err := repo.AppendAssetVersion(ctx, asset.ID, "account-a", second); err != nil {
		t.Fatal(err)
	}
	usage := &domain.SessionAssetUsage{ID: "usage-a", AccountID: "account-a", SessionID: session.ID, AssetID: asset.ID, AssetVersionID: second.ID, UsageKind: "referenced", OperationKey: "reference-a", CreatedAt: now.Add(time.Minute)}
	proposed := &domain.ProjectAsset{ID: "entry-b", AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: second.ID, DisplayName: "覆盖名称", AddedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)}
	if err := repo.ReferenceAssetInSession(ctx, proposed, usage); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", placement.ID)
	if err != nil || got.AssetVersionID != first.ID || got.DisplayName != "项目文稿" {
		t.Fatalf("existing project entry = (%#v, %v)", got, err)
	}
	if _, err := repo.GetProjectAsset(ctx, "account-a", proposed.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("duplicate entry = %v", err)
	}
	usages, err := repo.ListSessionAssetUsages(ctx, "account-a", session.ID)
	if err != nil || len(usages) != 1 || usages[0].AssetVersionID != second.ID || usages[0].SessionTitleSnapshot != domain.DefaultSessionTitle {
		t.Fatalf("session usage = (%#v, %v)", usages, err)
	}
}

func TestProjectCategoryAndTagsStayWithinProjectAndAccount(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)
	for _, id := range []string{"project-a", "project-b"} {
		project, _ := domain.NewProject(id, "account-a", id, now)
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
	}
	category := &domain.AssetCategory{ID: "category-a", AccountID: "account-a", ProjectID: "project-a", Name: "封面", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetCategory(ctx, category); err != nil {
		t.Fatal(err)
	}
	tag := &domain.AssetTag{ID: "tag-a", AccountID: "account-a", Name: "蓝色", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetTag(ctx, tag); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "封面.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "image/png", "studio/account-a/a.png", 4, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", ProjectID: "project-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, CategoryID: category.ID, Rating: 5, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetProjectAssetTags(ctx, "account-a", entry.ID, []string{tag.ID}); err != nil {
		t.Fatal(err)
	}
	page, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: "project-a", CategoryID: category.ID, TagIDs: []string{tag.ID}, Limit: 10})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || len(page.Items[0].Tags) != 1 || page.Items[0].Tags[0].Name != "蓝色" {
		t.Fatalf("filtered entry = (%#v, %v)", page, err)
	}
	otherProject, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: "project-b", TagIDs: []string{tag.ID}, Limit: 10})
	if err != nil || otherProject.Total != 0 {
		t.Fatalf("other project = (%#v, %v)", otherProject, err)
	}
	if err := repo.SetProjectAssetTags(ctx, "account-b", entry.ID, []string{tag.ID}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account tag edit = %v", err)
	}
	if err := repo.SetProjectAssetTags(ctx, "account-a", entry.ID, []string{"missing"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing tag = %v", err)
	}
	if err := repo.DeleteProjectAsset(ctx, "account-a", entry.ID); err != nil {
		t.Fatal(err)
	}
	replacement, _ := domain.NewAsset("asset-b", "account-a", "另一张.png", domain.AssetImage, domain.AssetOriginUser, now)
	replacementVersion, _ := replacement.AppendVersion("asset-b-v1", "image/png", "studio/account-a/b.png", 4, now)
	reusedID := &domain.ProjectAsset{ID: entry.ID, AccountID: "account-a", ProjectID: "project-a", AssetID: replacement.ID, AssetVersionID: replacementVersion.ID, DisplayName: replacement.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, replacement, reusedID, nil); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", reusedID.ID)
	if err != nil || len(got.Tags) != 0 {
		t.Fatalf("reused entry tags = (%#v, %v)", got, err)
	}
}

func TestProjectAssetPatchAndVersionStayWithinEntry(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 16, 0, 0, 0, time.UTC)
	asset, _ := domain.NewAsset("asset-a", "account-a", "草图.png", domain.AssetImage, domain.AssetOriginUser, now)
	first, _ := asset.AppendVersion("asset-v1", "image/png", "studio/account-a/v1.png", 4, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: first.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	second, _ := asset.AppendVersion("asset-v2", "image/png", "studio/account-a/v2.png", 5, now.Add(time.Minute))
	if err := repo.AppendAssetVersion(ctx, asset.ID, "account-a", second); err != nil {
		t.Fatal(err)
	}
	name := "正式封面"
	rating := 5
	if err := repo.UpdateProjectAsset(ctx, "account-a", domain.ProjectAssetPatch{ID: entry.ID, DisplayName: &name, Rating: &rating}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetProjectAssetVersion(ctx, "account-a", entry.ID, second.ID, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", entry.ID)
	if err != nil || got.DisplayName != name || got.Rating != 5 || got.Version.ID != second.ID {
		t.Fatalf("updated entry = (%#v, %v)", got, err)
	}
	bad := 6
	if err := repo.UpdateProjectAsset(ctx, "account-a", domain.ProjectAssetPatch{ID: entry.ID, Rating: &bad}, now); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid rating = %v", err)
	}
	if err := repo.SetProjectAssetVersion(ctx, "account-b", entry.ID, first.ID, now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-account version change = %v", err)
	}
}

func TestMovingSessionAddsAssetsToDestinationProject(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 17, 0, 0, 0, time.UTC)
	for _, id := range []string{"project-a", "project-b"} {
		project, _ := domain.NewProject(id, "account-a", id, now)
		if err := repo.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
	}
	session, _ := domain.NewSession("session-a", "account-a", now)
	session.ProjectID = "project-a"
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "图.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "image/png", "studio/account-a/a.png", 5, now)
	source := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", ProjectID: "project-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: "项目图", AddedAt: now, UpdatedAt: now}
	usage := &domain.SessionAssetUsage{ID: "usage-a", AccountID: "account-a", SessionID: session.ID, AssetID: asset.ID, AssetVersionID: version.ID, UsageKind: "created", OperationKey: "request-a", CreatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, source, usage); err != nil {
		t.Fatal(err)
	}
	if err := repo.MoveSessionToProject(ctx, "account-a", session.ID, "project-b"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"project-a", "project-b"} {
		page, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: id, Limit: 10})
		if err != nil || page.Total != 1 || page.Items[0].AssetID != asset.ID {
			t.Fatalf("project %s = (%#v, %v)", id, page, err)
		}
		if id == "project-b" && (page.Items[0].SourceProjectAssetID != source.ID || page.Items[0].SourceProjectNameSnapshot != "project-a") {
			t.Fatalf("moved source = %#v", page.Items[0])
		}
	}
}

func TestDeletingProjectMovesAssetsToUnassignedProject(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 18, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目 A", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "图.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "image/png", "studio/account-a/a.png", 5, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteProject(ctx, "account-a", project.ID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", entry.ID)
	if err != nil || got.ProjectID != "" || got.AssetVersionID != version.ID {
		t.Fatalf("unassigned entry = (%#v, %v)", got, err)
	}
	if _, err := repo.GetProject(ctx, "account-a", project.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted project = %v", err)
	}
}

func TestDeletingProjectKeepsCategoryHierarchyAndAssignments(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 18, 30, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目 A", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	category := &domain.AssetCategory{ID: "category-a", AccountID: "account-a", ProjectID: project.ID, Name: "封面", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetCategory(ctx, category); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "图.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "image/png", "studio/account-a/a.png", 5, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, CategoryID: category.ID, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteProject(ctx, "account-a", project.ID); err != nil {
		t.Fatal(err)
	}
	categories, err := repo.ListAssetCategories(ctx, "account-a", "")
	if err != nil || len(categories) != 2 {
		t.Fatalf("unassigned categories = (%#v, %v)", categories, err)
	}
	var rootID string
	for _, item := range categories {
		if item.Name == "项目 A" {
			rootID = item.ID
		}
	}
	if rootID == "" {
		t.Fatalf("missing project category: %#v", categories)
	}
	gotCategory, err := repo.GetAssetCategory(ctx, "account-a", category.ID)
	if err != nil || gotCategory.ProjectID != "" || gotCategory.ParentID != rootID {
		t.Fatalf("moved category = (%#v, %v)", gotCategory, err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", entry.ID)
	if err != nil || got.ProjectID != "" || got.CategoryID != category.ID {
		t.Fatalf("moved entry = (%#v, %v)", got, err)
	}
}

func TestImageAndNewVersionCreatePaletteJobs(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 19, 0, 0, 0, time.UTC)
	asset, _ := domain.NewAsset("asset-a", "account-a", "图.png", domain.AssetImage, domain.AssetOriginUser, now)
	first, _ := asset.AppendVersion("asset-a-v1", "image/png", "studio/account-a/v1.png", 4, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: first.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	second, _ := asset.AppendVersion("asset-a-v2", "image/png", "studio/account-a/v2.png", 5, now.Add(time.Minute))
	if err := repo.AppendAssetVersion(ctx, asset.ID, "account-a", second); err != nil {
		t.Fatal(err)
	}
	jobs, err := repo.ClaimPendingPaletteJobs(ctx, 10, now.Add(2*time.Minute))
	if err != nil || len(jobs) != 2 {
		t.Fatalf("claimed palettes = (%#v, %v)", jobs, err)
	}
	if err := repo.CompleteAssetVersionPalette(ctx, "account-a", first.ID, []domain.PaletteColor{{Hex: "#112233", Ratio: 1}}, nil, 256, 128, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetAssetVersionPalette(ctx, "account-a", first.ID)
	if err != nil || got.Status != "ready" || len(got.Colors) != 1 || got.Colors[0].Hex != "#112233" {
		t.Fatalf("completed palette = (%#v, %v)", got, err)
	}
	version, err := repo.GetAssetVersion(ctx, "account-a", first.ID)
	if err != nil || version.WidthPx == nil || *version.WidthPx != 256 || version.HeightPx == nil || *version.HeightPx != 128 {
		t.Fatalf("analyzed dimensions = (%#v, %v)", version, err)
	}
}

func TestUnsupportedImageDoesNotCreatePaletteJob(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 19, 0, 0, 0, time.UTC)
	asset, err := domain.NewAsset("asset-tiff", "account-a", "图.tiff", domain.AssetImage, domain.AssetOriginUser, now)
	if err != nil {
		t.Fatal(err)
	}
	version, err := asset.AppendVersion("asset-tiff-v1", "image/tiff", "studio/account-a/image.tiff", 4, now)
	if err != nil {
		t.Fatal(err)
	}
	entry := &domain.ProjectAsset{ID: "entry-tiff", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetAssetVersionPalette(ctx, "account-a", version.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unsupported palette = %v", err)
	}
}

func TestPaletteFailuresWaitForManualRetryAfterFiveAttempts(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 19, 0, 0, 0, time.UTC)
	asset, err := domain.NewAsset("asset-retry", "account-a", "图.png", domain.AssetImage, domain.AssetOriginUser, now)
	if err != nil {
		t.Fatal(err)
	}
	version, err := asset.AppendVersion("asset-retry-v1", "image/png", "studio/account-a/image.png", 4, now)
	if err != nil {
		t.Fatal(err)
	}
	entry := &domain.ProjectAsset{ID: "entry-retry", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 5; attempt++ {
		current := now.Add(time.Duration(attempt) * time.Hour)
		jobs, err := repo.ClaimPendingPaletteJobs(ctx, 10, current)
		if err != nil || len(jobs) != 1 || jobs[0].Attempts != attempt {
			t.Fatalf("attempt %d jobs = (%#v, %v)", attempt, jobs, err)
		}
		if err := repo.FailAssetVersionPalette(ctx, "account-a", version.ID, "analysis_failed", current.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := repo.ClaimPendingPaletteJobs(ctx, 10, now.Add(24*time.Hour))
	if err != nil || len(jobs) != 0 {
		t.Fatalf("automatic retries = (%#v, %v)", jobs, err)
	}
	if err := repo.RetryAssetVersionPalette(ctx, "account-a", version.ID, now.Add(25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	jobs, err = repo.ClaimPendingPaletteJobs(ctx, 10, now.Add(25*time.Hour))
	if err != nil || len(jobs) != 1 || jobs[0].Attempts != 1 {
		t.Fatalf("manual retry = (%#v, %v)", jobs, err)
	}
}

func TestAssetCreationKeyRejectsDuplicateRequest(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 20, 0, 0, 0, time.UTC)
	for _, id := range []string{"asset-one", "asset-two"} {
		asset, _ := domain.NewAsset(id, "account-a", id+".png", domain.AssetImage, domain.AssetOriginUser, now)
		asset.CreationKey = "upload-request-a"
		version, _ := asset.AppendVersion(id+"-v1", "image/png", "studio/account-a/"+id, 4, now)
		entry := &domain.ProjectAsset{ID: "entry-" + id, AccountID: "account-a", AssetID: id, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
		err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil)
		if id == "asset-one" && err != nil {
			t.Fatal(err)
		}
		if id == "asset-two" && !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("duplicate creation key = %v", err)
		}
	}
	got, err := repo.GetAssetByCreationKey(ctx, "account-a", "upload-request-a")
	if err != nil || got.ID != "asset-one" {
		t.Fatalf("original asset = (%#v, %v)", got, err)
	}
}

func TestAssetVersionOperationKeyRejectsDuplicateEdit(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 20, 30, 0, 0, time.UTC)
	asset, _ := domain.NewAsset("asset-a", "account-a", "文稿.md", domain.AssetDocument, domain.AssetOriginUser, now)
	first, _ := asset.AppendVersion("asset-a-v1", "text/markdown", "studio/account-a/v1.md", 4, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: first.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	second, _ := asset.AppendVersion("asset-a-v2", "text/markdown", "studio/account-a/v2.md", 5, now.Add(time.Minute))
	second.OperationKey = "edit-request-a"
	if err := repo.AppendAssetVersion(ctx, asset.ID, "account-a", second); err != nil {
		t.Fatal(err)
	}
	third, _ := asset.AppendVersion("asset-a-v3", "text/markdown", "studio/account-a/v3.md", 6, now.Add(2*time.Minute))
	third.OperationKey = "edit-request-a"
	if err := repo.AppendAssetVersion(ctx, asset.ID, "account-a", third); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("duplicate edit key = %v", err)
	}
}

func TestAssetLibraryPreferencesPersistTreeMode(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 21, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveAssetLibraryPreferences(ctx, &domain.AssetLibraryPreferences{AccountID: "account-a", TreeMode: "format", LastProjectID: project.ID, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetAssetLibraryPreferences(ctx, "account-a")
	if err != nil || got.TreeMode != "format" || got.LastProjectID != project.ID {
		t.Fatalf("preferences = (%#v, %v)", got, err)
	}
	other, err := repo.GetAssetLibraryPreferences(ctx, "account-b")
	if err != nil || other.TreeMode != "asset" || other.LastProjectID != "" {
		t.Fatalf("other account default = (%#v, %v)", other, err)
	}
	if err := repo.SaveAssetLibraryPreferences(ctx, &domain.AssetLibraryPreferences{AccountID: "account-a", TreeMode: "unknown", UpdatedAt: now}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid tree mode = %v", err)
	}
}

func TestProjectAssetTreeGroupsFormatsAndExpandsAssets(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ id, format string }{{"one", "png"}, {"two", "png"}, {"three", "jpeg"}} {
		asset, _ := domain.NewAsset(item.id, "account-a", item.id+".png", domain.AssetImage, domain.AssetOriginUser, now)
		version, _ := asset.AppendVersion(item.id+"-v1", "image/"+item.format, "studio/account-a/"+item.id, 5, now)
		asset.Versions[0].Format = item.format
		entry := &domain.ProjectAsset{ID: "entry-" + item.id, AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	root, err := repo.ListProjectAssetTree(ctx, "account-a", domain.ProjectAssetTreeQuery{ProjectID: project.ID, Mode: "format", Limit: 10})
	if err != nil || len(root.Nodes) != 2 || root.Nodes[0].ID != "format:jpeg" || root.Nodes[0].Count != 1 || root.Nodes[1].ID != "format:png" || root.Nodes[1].Count != 2 {
		t.Fatalf("format groups = (%#v, %v)", root, err)
	}
	children, err := repo.ListProjectAssetTree(ctx, "account-a", domain.ProjectAssetTreeQuery{ProjectID: project.ID, Mode: "format", ParentID: "format:png", Limit: 10})
	if err != nil || len(children.Nodes) != 2 || children.Nodes[0].Type != "asset" {
		t.Fatalf("format children = (%#v, %v)", children, err)
	}
	formats, err := repo.ListProjectAssetFormats(ctx, "account-a", project.ID)
	if err != nil || len(formats) != 2 || formats[0] != "jpeg" || formats[1] != "png" {
		t.Fatalf("project formats = (%#v, %v)", formats, err)
	}
}

func TestAssetCategoryMoveRejectsCycleAndDeleteMovesContents(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	parent := &domain.AssetCategory{ID: "category-parent", AccountID: "account-a", ProjectID: project.ID, Name: "插画", CreatedAt: now, UpdatedAt: now}
	child := &domain.AssetCategory{ID: "category-child", AccountID: "account-a", ProjectID: project.ID, ParentID: parent.ID, Name: "人物", CreatedAt: now, UpdatedAt: now}
	for _, category := range []*domain.AssetCategory{parent, child} {
		if err := repo.CreateAssetCategory(ctx, category); err != nil {
			t.Fatal(err)
		}
	}
	gotCategory, err := repo.GetAssetCategory(ctx, "account-a", child.ID)
	if err != nil || gotCategory.ProjectID != project.ID || gotCategory.ParentID != parent.ID {
		t.Fatalf("category details = (%#v, %v)", gotCategory, err)
	}
	if err := repo.UpdateAssetCategory(ctx, "account-a", parent.ID, child.ID, "插画", now); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("category cycle = %v", err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "人像.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "image/png", "studio/account-a/a.png", 5, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, CategoryID: child.ID, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteAssetCategory(ctx, "account-a", child.ID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetProjectAsset(ctx, "account-a", entry.ID)
	if err != nil || got.CategoryID != parent.ID {
		t.Fatalf("entry category = (%#v, %v)", got, err)
	}
}

func TestProjectAssetCategoryFilterIncludesDescendants(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 15, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-category", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for _, category := range []*domain.AssetCategory{
		{ID: "parent", AccountID: "account-a", ProjectID: project.ID, Name: "主分类", CreatedAt: now, UpdatedAt: now},
		{ID: "child", AccountID: "account-a", ProjectID: project.ID, ParentID: "parent", Name: "子分类", CreatedAt: now, UpdatedAt: now},
	} {
		if err := repo.CreateAssetCategory(ctx, category); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct{ id, category string }{{"one", "parent"}, {"two", "child"}} {
		asset, _ := domain.NewAsset(item.id, "account-a", item.id+".png", domain.AssetImage, domain.AssetOriginUser, now)
		version, _ := asset.AppendVersion(item.id+"-v1", "image/png", "studio/account-a/"+item.id, 5, now)
		entry := &domain.ProjectAsset{ID: "entry-" + item.id, AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, CategoryID: item.category, AddedAt: now, UpdatedAt: now}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	all, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: project.ID, CategoryID: "parent"})
	if err != nil || all.Total != 2 {
		t.Fatalf("category subtree = (%#v, %v)", all, err)
	}
	direct, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: project.ID, CategoryID: "parent", CategoryDirect: true})
	if err != nil || direct.Total != 1 || direct.Items[0].ID != "entry-one" {
		t.Fatalf("direct category = (%#v, %v)", direct, err)
	}
}

func TestProjectAssetSortAndCursor(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 45, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-sort", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for index, item := range []struct {
		id     string
		name   string
		rating int
		modify bool
	}{
		{"a", "Charlie", 1, true},
		{"b", "Alpha", 5, false},
		{"c", "Bravo", 3, true},
	} {
		createdAt := now.Add(time.Duration(index) * time.Minute)
		asset, _ := domain.NewAsset(item.id, "account-a", item.name, domain.AssetImage, domain.AssetOriginUser, createdAt)
		version, _ := asset.AppendVersion(item.id+"-v1", "image/png", "studio/account-a/"+item.id, int64(index+1)*100, createdAt)
		if item.modify {
			asset.Versions[0].SourceModifiedAt = &createdAt
		}
		entry := &domain.ProjectAsset{ID: "entry-" + item.id, AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: item.name, Rating: item.rating, AddedAt: createdAt, UpdatedAt: createdAt}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		sort string
		want []string
	}{
		{"recent", []string{"entry-c", "entry-b", "entry-a"}},
		{"oldest", []string{"entry-a", "entry-b", "entry-c"}},
		{"modified", []string{"entry-c", "entry-a", "entry-b"}},
		{"name", []string{"entry-b", "entry-c", "entry-a"}},
		{"size", []string{"entry-c", "entry-b", "entry-a"}},
		{"rating", []string{"entry-b", "entry-c", "entry-a"}},
	} {
		t.Run(test.sort, func(t *testing.T) {
			cursor := ""
			var got []string
			for range test.want {
				page, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: project.ID, Sort: test.sort, Limit: 1, Cursor: cursor})
				if err != nil || page.Total != 3 || len(page.Items) != 1 {
					t.Fatalf("page = (%#v, %v)", page, err)
				}
				got = append(got, page.Items[0].ID)
				cursor = page.NextCursor
			}
			if strings.Join(got, ",") != strings.Join(test.want, ",") || cursor != "" {
				t.Fatalf("sort %s = %v, cursor %q", test.sort, got, cursor)
			}
		})
	}
}

func TestProjectAssetSearchMatchesOriginalNameAndTags(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 50, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-search", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-search", "account-a", "蓝色封面.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-search-v1", "image/png", "studio/account-a/search", 5, now)
	entry := &domain.ProjectAsset{ID: "entry-search", AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: "主视觉", AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	tag := &domain.AssetTag{ID: "tag-search", AccountID: "account-a", Name: "海报", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetTag(ctx, tag); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetProjectAssetTags(ctx, "account-a", entry.ID, []string{tag.ID}); err != nil {
		t.Fatal(err)
	}
	for _, search := range []string{"主视觉", "蓝色封面", "海报"} {
		page, err := repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: project.ID, Search: search})
		if err != nil || page.Total != 1 || page.Items[0].ID != entry.ID {
			t.Fatalf("search %q = (%#v, %v)", search, page, err)
		}
	}
}

func TestProjectAssetCombinedRangeAndDuplicateFilters(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-ranges", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for index, item := range []struct {
		id     string
		sha256 string
		width  int
		height int
		size   int64
	}{
		{"a", strings.Repeat("a", 64), 100, 200, 500},
		{"b", strings.Repeat("a", 64), 120, 220, 600},
		{"c", strings.Repeat("b", 64), 120, 220, 600},
	} {
		addedAt := now.Add(time.Duration(index) * time.Minute)
		asset, _ := domain.NewAsset(item.id, "account-a", item.id+".png", domain.AssetImage, domain.AssetOriginUser, addedAt)
		version, _ := asset.AppendVersion(item.id+"-v1", "image/png", "studio/account-a/"+item.id, item.size, addedAt)
		asset.Versions[0].SHA256 = item.sha256
		asset.Versions[0].WidthPx = &item.width
		asset.Versions[0].HeightPx = &item.height
		entry := &domain.ProjectAsset{ID: "entry-" + item.id, AccountID: "account-a", ProjectID: project.ID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: addedAt, UpdatedAt: addedAt}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	minWidth, maxWidth := 110, 130
	minHeight, maxHeight := 210, 230
	minSize, maxSize := int64(550), int64(650)
	addedFrom, addedTo := now.Add(time.Minute), now.Add(2*time.Minute)
	query := domain.ProjectAssetListQuery{
		ProjectID: project.ID, MinWidthPx: &minWidth, MaxWidthPx: &maxWidth,
		MinHeightPx: &minHeight, MaxHeightPx: &maxHeight,
		MinSizeBytes: &minSize, MaxSizeBytes: &maxSize,
		AddedFrom: &addedFrom, AddedTo: &addedTo, DuplicatesOnly: true,
	}
	page, err := repo.ListProjectAssets(ctx, "account-a", query)
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "entry-b" {
		t.Fatalf("combined filters = (%#v, %v)", page, err)
	}
	query.DuplicatesOnly = false
	addedFrom = addedTo
	if _, err := repo.ListProjectAssets(ctx, "account-a", query); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("empty added range = %v", err)
	}
	addedFrom = now.Add(time.Minute)
	page, err = repo.ListProjectAssets(ctx, "account-a", query)
	if err != nil || page.Total != 1 || page.Items[0].ID != "entry-b" {
		t.Fatalf("exclusive added_to = (%#v, %v)", page, err)
	}
	minWidth = 140
	if _, err := repo.ListProjectAssets(ctx, "account-a", query); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid width range = %v", err)
	}
}

func TestListDuplicateProjectAssetsScopesAccountAndPrioritizesCurrentProject(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 30, 0, 30, 0, 0, time.UTC)
	for _, project := range []struct{ id, accountID string }{
		{"project-a", "account-a"}, {"project-b", "account-a"}, {"project-c", "account-b"},
	} {
		value, _ := domain.NewProject(project.id, project.accountID, project.id, now)
		if err := repo.CreateProject(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	sha := strings.Repeat("a", 64)
	for _, item := range []struct {
		id, accountID, projectID, hash string
	}{
		{"source", "account-a", "project-a", sha},
		{"same-project", "account-a", "project-a", sha},
		{"other-project", "account-a", "project-b", sha},
		{"other-account", "account-b", "project-c", sha},
		{"without-hash", "account-a", "project-a", ""},
	} {
		asset, _ := domain.NewAsset(item.id, item.accountID, item.id+".png", domain.AssetImage, domain.AssetOriginUser, now)
		version, _ := asset.AppendVersion(item.id+"-v1", "image/png", "studio/"+item.accountID+"/"+item.id, 10, now)
		asset.Versions[0].SHA256 = item.hash
		entry := &domain.ProjectAsset{ID: "entry-" + item.id, AccountID: item.accountID, ProjectID: item.projectID, AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.CopyProjectAsset(ctx, "account-a", "entry-source", &domain.ProjectAsset{
		ID: "entry-shared-asset", AccountID: "account-a", ProjectID: "project-b", AddedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	items, err := repo.ListDuplicateProjectAssets(ctx, "account-a", "entry-source")
	if err != nil || len(items) != 2 || items[0].ID != "entry-same-project" || items[1].ID != "entry-other-project" || items[0].Asset == nil || items[1].Version.ID == "" {
		t.Fatalf("duplicates = (%#v, %v)", items, err)
	}
	if items, err := repo.ListDuplicateProjectAssets(ctx, "account-a", "entry-without-hash"); err != nil || len(items) != 0 {
		t.Fatalf("missing hash = (%#v, %v)", items, err)
	}
	if _, err := repo.ListDuplicateProjectAssets(ctx, "account-b", "entry-source"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other account duplicates = %v", err)
	}
	entry, err := repo.GetProjectAssetByAsset(ctx, "account-a", "project-b", "other-project")
	if err != nil || entry.ID != "entry-other-project" {
		t.Fatalf("entry by asset = (%#v, %v)", entry, err)
	}
	if _, err := repo.GetProjectAssetByAsset(ctx, "account-b", "project-b", "other-project"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other account entry by asset = %v", err)
	}
}

func TestTagRenameUpdatesProjectAssetDetails(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 30, 0, 0, time.UTC)
	tag := &domain.AssetTag{ID: "tag-a", AccountID: "account-a", Name: "原标签", CreatedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetTag(ctx, tag); err != nil {
		t.Fatal(err)
	}
	if err := repo.RenameAssetTag(ctx, "account-a", tag.ID, "新标签", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	tags, err := repo.ListAssetTags(ctx, "account-a")
	if err != nil || len(tags) != 1 || tags[0].Name != "新标签" {
		t.Fatalf("renamed tags = (%#v, %v)", tags, err)
	}
}

func TestProjectAssetCountsExcludeArchivedEntries(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 45, 0, 0, time.UTC)
	project, _ := domain.NewProject("project-a", "account-a", "项目", now)
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"one", "two"} {
		asset, _ := domain.NewAsset(id, "account-a", id+".png", domain.AssetImage, domain.AssetOriginUser, now)
		version, _ := asset.AppendVersion(id+"-v1", "image/png", "studio/account-a/"+id, 5, now)
		entry := &domain.ProjectAsset{ID: "entry-" + id, AccountID: "account-a", ProjectID: project.ID, AssetID: id, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
		if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
			t.Fatal(err)
		}
	}
	archived := true
	if err := repo.UpdateProjectAsset(ctx, "account-a", domain.ProjectAssetPatch{ID: "entry-two", Archived: &archived}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	counts, err := repo.ListProjectAssetCounts(ctx, "account-a")
	if err != nil || counts[project.ID] != 1 {
		t.Fatalf("project counts = (%#v, %v)", counts, err)
	}
}

func TestBlobWriteIntentClearsWhenAssetTransactionCommits(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 50, 0, 0, time.UTC)
	key := "studio/account-a/blob-a"
	intent := &domain.BlobWriteIntent{BlobKey: key, AccountID: "account-a", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := repo.RegisterBlobWriteIntent(ctx, intent); err != nil {
		t.Fatal(err)
	}
	asset, _ := domain.NewAsset("asset-a", "account-a", "图.png", domain.AssetImage, domain.AssetOriginUser, now)
	version, _ := asset.AppendVersion("asset-a-v1", "image/png", key, 5, now)
	entry := &domain.ProjectAsset{ID: "entry-a", AccountID: "account-a", AssetID: asset.ID, AssetVersionID: version.ID, DisplayName: asset.Name, AddedAt: now, UpdatedAt: now}
	if err := repo.CreateAssetWithPlacement(ctx, asset, entry, nil); err != nil {
		t.Fatal(err)
	}
	jobs, err := repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(2*time.Minute))
	if err != nil || len(jobs) != 0 {
		t.Fatalf("committed blob intent = (%#v, %v)", jobs, err)
	}
	referenced, err := repo.IsBlobReferenced(ctx, "account-a", key)
	if err != nil || !referenced {
		t.Fatalf("blob reference = (%v, %v)", referenced, err)
	}
}

func TestExpiredBlobWriteIntentCanBeClaimedAndRemoved(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 55, 0, 0, time.UTC)
	key := "studio/account-a/orphan"
	if err := repo.RegisterBlobWriteIntent(ctx, &domain.BlobWriteIntent{BlobKey: key, AccountID: "account-a", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RegisterBlobWriteIntent(ctx, &domain.BlobWriteIntent{BlobKey: key, AccountID: "account-b", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	jobs, err := repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(2*time.Minute))
	if err != nil || len(jobs) != 2 || jobs[0].BlobKey != key || jobs[1].BlobKey != key {
		t.Fatalf("expired intent = (%#v, %v)", jobs, err)
	}
	referenced, err := repo.IsBlobReferenced(ctx, "account-a", key)
	if err != nil || referenced {
		t.Fatalf("orphan reference = (%v, %v)", referenced, err)
	}
	if err := repo.DeleteBlobWriteIntent(ctx, "account-a", key); err != nil {
		t.Fatal(err)
	}
	jobs, err = repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(3*time.Minute))
	if err != nil || len(jobs) != 1 || jobs[0].AccountID != "account-b" {
		t.Fatalf("other account intent = (%#v, %v)", jobs, err)
	}
	if err := repo.DeleteBlobWriteIntent(ctx, "account-b", key); err != nil {
		t.Fatal(err)
	}
	jobs, err = repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(4*time.Minute))
	if err != nil || len(jobs) != 0 {
		t.Fatalf("removed intent = (%#v, %v)", jobs, err)
	}
}

func TestBlobWriteIntentSupportsLongBlobKeys(t *testing.T) {
	repo := openRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 23, 58, 0, 0, time.UTC)
	key := "studio/account-a/" + strings.Repeat("x", 850)
	if err := repo.RegisterBlobWriteIntent(ctx, &domain.BlobWriteIntent{BlobKey: key, AccountID: "account-a", CreatedAt: now, ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	jobs, err := repo.ClaimExpiredBlobWriteIntents(ctx, 10, now.Add(2*time.Minute))
	if err != nil || len(jobs) != 1 || jobs[0].BlobKey != key {
		t.Fatalf("long key intent = (%#v, %v)", jobs, err)
	}
}
