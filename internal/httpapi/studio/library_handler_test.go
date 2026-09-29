package studio_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Mr9esx/Pixoma/internal/httpapi/apitest"
	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/blob/localfs"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

func newLibraryRouter(t *testing.T) (*chi.Mux, *studioapi.Handler) {
	t.Helper()
	gdb, err := db.Open(db.Options{DSN: "file:" + uuid.NewString() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, persistence.Models()...); err != nil {
		t.Fatal(err)
	}
	repo := persistence.NewGormRepository(gdb)
	blobs, err := localfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := &studioapi.Handler{Repo: repo, Service: &studioapp.Service{Repo: repo}, Blob: blobs}
	router := chi.NewRouter()
	handler.Mount(router)
	return router, handler
}

func TestStudioLibraryProjectsAndAssetsUseProjectScope(t *testing.T) {
	router, handler := newLibraryRouter(t)
	project, err := domain.NewProject(uuid.NewString(), "account-a", "漫画", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Repo.CreateProject(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	asset, err := handler.Service.UploadAsset(context.Background(), studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: "notes.txt",
		MIMEType: "text/plain", Content: bytes.NewBufferString("hello"),
	}, handler.Blob)
	if err != nil {
		t.Fatal(err)
	}
	projects := request(t, router, http.MethodGet, "/library/projects", nil, "account-a")
	if projects.Code != http.StatusOK {
		t.Fatalf("projects = %d %s", projects.Code, projects.Body.String())
	}
	var listedProjects []struct {
		ID         string `json:"id"`
		AssetCount int64  `json:"asset_count"`
	}
	if err := json.Unmarshal(apitest.DataBytes(projects), &listedProjects); err != nil {
		t.Fatal(err)
	}
	if len(listedProjects) != 2 || listedProjects[0].ID != project.ID || listedProjects[0].AssetCount != 1 || listedProjects[1].ID != "" {
		t.Fatalf("projects = %#v", listedProjects)
	}
	page := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID, nil, "account-a")
	if page.Code != http.StatusOK {
		t.Fatalf("assets = %d %s", page.Code, page.Body.String())
	}
	var listed struct {
		Items []struct {
			ID      string `json:"id"`
			AssetID string `json:"asset_id"`
			Asset   struct {
				ID string `json:"id"`
			} `json:"asset"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(apitest.DataBytes(page), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].AssetID != asset.ID || listed.Items[0].Asset.ID != asset.ID {
		t.Fatalf("assets = %#v", listed)
	}
	uncategorized := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID+"&category_id=none", nil, "account-a")
	var uncategorizedPage struct {
		Total int `json:"total"`
	}
	if uncategorized.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(uncategorized), &uncategorizedPage) != nil || uncategorizedPage.Total != 1 {
		t.Fatalf("uncategorized assets = %d %s", uncategorized.Code, uncategorized.Body.String())
	}
	foreign := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID, nil, "account-b")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign project = %d %s", foreign.Code, foreign.Body.String())
	}
	for _, path := range []string{"/assets/" + asset.ID + "/save-to-library", "/sessions/unused/assets/import"} {
		removed := request(t, router, http.MethodPost, path, nil, "account-a")
		if removed.Code != http.StatusNotFound {
			t.Fatalf("removed route %s = %d", path, removed.Code)
		}
	}
}

func TestStudioLibraryProjectsSortByNameWithUnassignedLast(t *testing.T) {
	router, handler := newLibraryRouter(t)
	for _, name := range []string{"角色", "背景"} {
		project, err := domain.NewProject(uuid.NewString(), "account-a", name, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if err := handler.Repo.CreateProject(context.Background(), project); err != nil {
			t.Fatal(err)
		}
	}
	response := request(t, router, http.MethodGet, "/library/projects", nil, "account-a")
	var projects []struct {
		Name string `json:"name"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(response), &projects) != nil ||
		len(projects) != 3 || projects[0].Name != "背景" || projects[1].Name != "角色" || projects[2].Name != "未归属项目" {
		t.Fatalf("project order = %d %s", response.Code, response.Body.String())
	}
}

func TestStudioUploadReturnsProjectAssetID(t *testing.T) {
	router, handler := newLibraryRouter(t)
	project, err := domain.NewProject(uuid.NewString(), "account-a", "上传测试", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Repo.CreateProject(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	file, err := writer.CreateFormFile("file", "说明.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("上传后的资产")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("project_id", project.ID); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("request_id", uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	uploadRequest := httptest.NewRequest(http.MethodPost, "/assets/upload", &form)
	uploadRequest.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRequest = uploadRequest.WithContext(setupapi.WithAccount(uploadRequest.Context(), setupapi.AccountSession{
		AccountID: "account-a", Username: "account-a", Role: "admin",
	}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, uploadRequest)
	var uploaded struct {
		ID             string `json:"id"`
		ProjectAssetID string `json:"project_asset_id"`
	}
	if recorder.Code != http.StatusCreated || json.Unmarshal(apitest.DataBytes(recorder), &uploaded) != nil ||
		uploaded.ID == "" || uploaded.ProjectAssetID == "" {
		t.Fatalf("upload response = %d %s", recorder.Code, recorder.Body.String())
	}
	entry, err := handler.Repo.GetProjectAsset(context.Background(), "account-a", uploaded.ProjectAssetID)
	if err != nil || entry.ProjectID != project.ID || entry.AssetID != uploaded.ID {
		t.Fatalf("uploaded project asset = %#v, %v", entry, err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="16" height="16" fill="red"/></svg>`
	svgAsset, err := handler.Service.UploadAsset(context.Background(), studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: "图标.svg",
		MIMEType: "image/svg+xml", Content: bytes.NewBufferString(svg),
	}, handler.Blob)
	if err != nil {
		t.Fatal(err)
	}
	content := request(t, router, http.MethodGet, "/assets/"+svgAsset.ID+"/content", nil, "account-a")
	if content.Code != http.StatusOK || content.Header().Get("Content-Type") != "image/svg+xml" ||
		content.Header().Get("Content-Security-Policy") != "sandbox" || content.Body.String() != svg {
		t.Fatalf("SVG content response = %d %s", content.Code, content.Body.String())
	}
}

func TestStudioLibraryListsSameContentAcrossProjects(t *testing.T) {
	router, handler := newLibraryRouter(t)
	ctx := context.Background()
	projects := make([]*domain.Project, 0, 2)
	for _, name := range []string{"本项目", "另一个项目"} {
		project, err := domain.NewProject(uuid.NewString(), "account-a", name, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if err := handler.Repo.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
		projects = append(projects, project)
	}
	for _, item := range []struct {
		accountID string
		projectID string
		name      string
		content   string
	}{
		{"account-a", projects[0].ID, "原文件.txt", "相同内容"},
		{"account-a", projects[0].ID, "同项目.txt", "相同内容"},
		{"account-a", projects[1].ID, "跨项目.txt", "相同内容"},
		{"account-a", projects[0].ID, "不同.txt", "不同内容"},
		{"account-b", "", "其他账号.txt", "相同内容"},
	} {
		if _, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
			AccountID: item.accountID, ProjectID: item.projectID, RequestID: uuid.NewString(),
			Name: item.name, MIMEType: "text/plain", Content: bytes.NewBufferString(item.content),
		}, handler.Blob); err != nil {
			t.Fatal(err)
		}
	}
	page, err := handler.Repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: projects[0].ID})
	if err != nil || len(page.Items) != 3 {
		t.Fatalf("source page = %#v, %v", page, err)
	}
	sourceID := ""
	for _, item := range page.Items {
		if item.DisplayName == "原文件.txt" {
			sourceID = item.ID
		}
	}
	if sourceID == "" {
		t.Fatal("source project asset missing")
	}
	response := request(t, router, http.MethodGet, "/library/assets/"+sourceID+"/duplicates", nil, "account-a")
	var duplicates struct {
		Items []struct {
			ID        string `json:"id"`
			ProjectID string `json:"project_id"`
			AssetID   string `json:"asset_id"`
		} `json:"items"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(response), &duplicates) != nil ||
		len(duplicates.Items) != 2 || duplicates.Items[0].ProjectID != projects[0].ID ||
		duplicates.Items[1].ProjectID != projects[1].ID || duplicates.Items[0].ID == sourceID || duplicates.Items[1].ID == sourceID {
		t.Fatalf("duplicates = %d %s", response.Code, response.Body.String())
	}
	foreign := request(t, router, http.MethodGet, "/library/assets/"+sourceID+"/duplicates", nil, "account-b")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign duplicates = %d %s", foreign.Code, foreign.Body.String())
	}
}

func TestStudioLibraryCategoryTreeAndBatchAddTags(t *testing.T) {
	router, handler := newLibraryRouter(t)
	ctx := context.Background()
	project, err := domain.NewProject(uuid.NewString(), "account-a", "插画", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"封面.txt", "内页.txt"} {
		if _, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
			AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: name,
			MIMEType: "text/plain", Content: bytes.NewBufferString(name),
		}, handler.Blob); err != nil {
			t.Fatal(err)
		}
	}
	list := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID, nil, "account-a")
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if list.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(list), &page) != nil || len(page.Items) != 2 {
		t.Fatalf("project assets = %d %s", list.Code, list.Body.String())
	}
	parentResponse := request(t, router, http.MethodPost, "/library/categories", map[string]any{"project_id": project.ID, "name": "角色"}, "account-a")
	var parent struct {
		ID string `json:"id"`
	}
	if parentResponse.Code != http.StatusCreated || json.Unmarshal(apitest.DataBytes(parentResponse), &parent) != nil {
		t.Fatalf("parent category = %d %s", parentResponse.Code, parentResponse.Body.String())
	}
	childResponse := request(t, router, http.MethodPost, "/library/categories", map[string]any{"project_id": project.ID, "parent_id": parent.ID, "name": "主角"}, "account-a")
	var child struct {
		ID string `json:"id"`
	}
	if childResponse.Code != http.StatusCreated || json.Unmarshal(apitest.DataBytes(childResponse), &child) != nil {
		t.Fatalf("child category = %d %s", childResponse.Code, childResponse.Body.String())
	}
	for index, categoryID := range []string{parent.ID, child.ID} {
		updated := request(t, router, http.MethodPatch, "/library/assets/"+page.Items[index].ID, map[string]any{"category_id": categoryID}, "account-a")
		if updated.Code != http.StatusOK {
			t.Fatalf("categorize asset = %d %s", updated.Code, updated.Body.String())
		}
	}
	group := request(t, router, http.MethodGet, "/library/tree?project_id="+project.ID+"&mode=category&parent_id=category:"+parent.ID, nil, "account-a")
	var tree struct {
		Nodes []struct {
			ID         string `json:"id"`
			GroupValue string `json:"group_value"`
		} `json:"nodes"`
	}
	if group.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(group), &tree) != nil {
		t.Fatalf("category tree = %d %s", group.Code, group.Body.String())
	}
	if len(tree.Nodes) != 2 || tree.Nodes[1].ID != "category-direct:"+parent.ID || tree.Nodes[1].GroupValue != "direct:"+parent.ID {
		t.Fatalf("category tree = %s", group.Body.String())
	}
	subtree := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID+"&category_id="+parent.ID, nil, "account-a")
	var subtreePage struct {
		Total int `json:"total"`
	}
	if subtree.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(subtree), &subtreePage) != nil || subtreePage.Total != 2 {
		t.Fatalf("category subtree = %d %s", subtree.Code, subtree.Body.String())
	}
	direct := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID+"&category_id=direct:"+parent.ID, nil, "account-a")
	var directPage struct {
		Total int `json:"total"`
	}
	if direct.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(direct), &directPage) != nil || directPage.Total != 1 {
		t.Fatalf("direct category = %d %s", direct.Code, direct.Body.String())
	}
	tagIDs := make([]string, 0, 2)
	for _, name := range []string{"常用", "待审"} {
		created := request(t, router, http.MethodPost, "/library/tags", map[string]any{"name": name}, "account-a")
		var tag struct {
			ID string `json:"id"`
		}
		if created.Code != http.StatusCreated || json.Unmarshal(apitest.DataBytes(created), &tag) != nil {
			t.Fatalf("create tag = %d %s", created.Code, created.Body.String())
		}
		tagIDs = append(tagIDs, tag.ID)
	}
	assigned := request(t, router, http.MethodPut, "/library/assets/"+page.Items[0].ID+"/tags", map[string]any{"tag_ids": tagIDs[:1]}, "account-a")
	if assigned.Code != http.StatusOK {
		t.Fatalf("assign tag = %d %s", assigned.Code, assigned.Body.String())
	}
	batch := request(t, router, http.MethodPost, "/library/assets/batch", map[string]any{
		"project_asset_ids": []string{page.Items[0].ID}, "action": "tags", "tag_ids": tagIDs[1:],
	}, "account-a")
	if batch.Code != http.StatusOK {
		t.Fatalf("batch add tag = %d %s", batch.Code, batch.Body.String())
	}
	detail := request(t, router, http.MethodGet, "/library/assets/"+page.Items[0].ID, nil, "account-a")
	var tagged struct {
		Tags []struct {
			ID string `json:"id"`
		} `json:"tags"`
	}
	if detail.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(detail), &tagged) != nil || len(tagged.Tags) != 2 {
		t.Fatalf("combined tags = %d %s", detail.Code, detail.Body.String())
	}
	removed := request(t, router, http.MethodPost, "/library/assets/batch", map[string]any{
		"project_asset_ids": []string{page.Items[0].ID}, "action": "tags_remove", "tag_ids": tagIDs[:1],
	}, "account-a")
	if removed.Code != http.StatusOK {
		t.Fatalf("batch remove tag = %d %s", removed.Code, removed.Body.String())
	}
	detail = request(t, router, http.MethodGet, "/library/assets/"+page.Items[0].ID, nil, "account-a")
	if detail.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(detail), &tagged) != nil ||
		len(tagged.Tags) != 1 || tagged.Tags[0].ID != tagIDs[1] {
		t.Fatalf("remaining tags = %d %s", detail.Code, detail.Body.String())
	}
}

func TestStudioLibraryCopyReferenceAndExport(t *testing.T) {
	router, handler := newLibraryRouter(t)
	ctx := context.Background()
	projects := make([]*domain.Project, 0, 2)
	for _, name := range []string{"原项目", "目标项目"} {
		project, err := domain.NewProject(uuid.NewString(), "account-a", name, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if err := handler.Repo.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
		projects = append(projects, project)
	}
	content := "角色资料：雨夜侦探"
	asset, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: projects[0].ID, RequestID: uuid.NewString(), Name: "角色.md",
		MIMEType: "text/markdown", Content: bytes.NewBufferString(content),
	}, handler.Blob)
	if err != nil {
		t.Fatal(err)
	}
	sourcePage, err := handler.Repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: projects[0].ID})
	if err != nil || len(sourcePage.Items) != 1 {
		t.Fatalf("source project assets = %#v, %v", sourcePage, err)
	}
	sourceID := sourcePage.Items[0].ID
	versionID := sourcePage.Items[0].AssetVersionID
	copied := request(t, router, http.MethodPost, "/library/assets/"+sourceID+"/projects", map[string]any{
		"project_id": projects[1].ID, "asset_version_id": versionID,
	}, "account-a")
	var copyView struct {
		ID         string `json:"id"`
		AssetID    string `json:"asset_id"`
		CopySource struct {
			ProjectAssetID      string `json:"project_asset_id"`
			ProjectNameSnapshot string `json:"project_name_snapshot"`
			Deleted             bool   `json:"deleted"`
		} `json:"copy_source"`
	}
	if copied.Code != http.StatusCreated || json.Unmarshal(apitest.DataBytes(copied), &copyView) != nil ||
		copyView.AssetID != asset.ID || copyView.CopySource.ProjectAssetID != sourceID || copyView.CopySource.ProjectNameSnapshot != projects[0].Name {
		t.Fatalf("copied asset = %d %s", copied.Code, copied.Body.String())
	}
	if denied := request(t, router, http.MethodGet, "/library/assets/"+copyView.ID, nil, "account-b"); denied.Code != http.StatusNotFound {
		t.Fatalf("foreign copy = %d %s", denied.Code, denied.Body.String())
	}
	session, err := domain.NewSession(uuid.NewString(), "account-a", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	session.ProjectID = projects[1].ID
	if err := handler.Repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	requestID := uuid.NewString()
	for attempt := 0; attempt < 2; attempt++ {
		referenced := request(t, router, http.MethodPost, "/sessions/"+session.ID+"/assets/references", map[string]any{
			"asset_id": asset.ID, "asset_version_id": versionID, "source_project_asset_id": sourceID, "request_id": requestID,
		}, "account-a")
		if referenced.Code != http.StatusCreated {
			t.Fatalf("reference asset = %d %s", referenced.Code, referenced.Body.String())
		}
	}
	usages, err := handler.Repo.ListSessionAssetUsages(ctx, "account-a", session.ID)
	if err != nil || len(usages) != 1 {
		t.Fatalf("session usages = %#v, %v", usages, err)
	}
	detail := request(t, router, http.MethodGet, "/library/assets/"+copyView.ID, nil, "account-a")
	var detailView struct {
		Usages []struct {
			SessionID string `json:"session_id"`
		} `json:"usages"`
		Versions []struct {
			ID string `json:"id"`
		} `json:"versions"`
	}
	if detail.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(detail), &detailView) != nil ||
		len(detailView.Usages) != 1 || detailView.Usages[0].SessionID != session.ID || len(detailView.Versions) != 1 {
		t.Fatalf("copy detail = %d %s", detail.Code, detail.Body.String())
	}
	exported := request(t, router, http.MethodPost, "/library/assets/export", map[string]any{"project_asset_ids": []string{copyView.ID}}, "account-a")
	if exported.Code != http.StatusOK || exported.Body.String() != content || exported.Header().Get("Content-Disposition") == "" {
		t.Fatalf("export = %d %s", exported.Code, exported.Body.String())
	}
	removed := request(t, router, http.MethodDelete, "/library/assets/"+sourceID, nil, "account-a")
	if removed.Code != http.StatusOK {
		t.Fatalf("delete source = %d %s", removed.Code, removed.Body.String())
	}
	copyAfterDelete := request(t, router, http.MethodGet, "/library/assets/"+copyView.ID, nil, "account-a")
	if copyAfterDelete.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(copyAfterDelete), &copyView) != nil || !copyView.CopySource.Deleted {
		t.Fatalf("copy source after delete = %d %s", copyAfterDelete.Code, copyAfterDelete.Body.String())
	}
	exported = request(t, router, http.MethodPost, "/library/assets/export", map[string]any{"project_asset_ids": []string{copyView.ID}}, "account-a")
	if exported.Code != http.StatusOK || exported.Body.String() != content {
		t.Fatalf("copy export after source delete = %d %s", exported.Code, exported.Body.String())
	}
}

func TestStudioLibraryFormatPalettePreferencesAndArchive(t *testing.T) {
	router, handler := newLibraryRouter(t)
	ctx := context.Background()
	project, err := domain.NewProject(uuid.NewString(), "account-a", "素材", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			frame.Set(x, y, color.RGBA{R: 230, G: 70, B: 50, A: 255})
		}
	}
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, frame); err != nil {
		t.Fatal(err)
	}
	imageContent := append([]byte(nil), imageBytes.Bytes()...)
	imageAsset, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: "缩略图.jpg",
		MIMEType: "image/jpeg", Content: bytes.NewReader(imageContent),
	}, handler.Blob)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: "说明.txt",
		MIMEType: "text/plain", Content: bytes.NewBufferString("配色说明"),
	}, handler.Blob); err != nil {
		t.Fatal(err)
	}
	formats := request(t, router, http.MethodGet, "/library/formats?project_id="+project.ID, nil, "account-a")
	var formatNames []string
	if formats.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(formats), &formatNames) != nil || len(formatNames) != 2 {
		t.Fatalf("formats = %d %s", formats.Code, formats.Body.String())
	}
	filtered := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID+"&format=png", nil, "account-a")
	var imagePage struct {
		Total int `json:"total"`
		Items []struct {
			ID      string `json:"id"`
			Version struct {
				ID       string `json:"id"`
				Format   string `json:"format"`
				MIMEType string `json:"mime_type"`
				WidthPx  int    `json:"width_px"`
				HeightPx int    `json:"height_px"`
				Palette  struct {
					Status string `json:"status"`
				} `json:"palette"`
			} `json:"version"`
		} `json:"items"`
	}
	if filtered.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(filtered), &imagePage) != nil || imagePage.Total != 1 ||
		imagePage.Items[0].Version.Format != "png" || imagePage.Items[0].Version.MIMEType != "image/png" ||
		imagePage.Items[0].Version.WidthPx != 3 || imagePage.Items[0].Version.HeightPx != 2 || imagePage.Items[0].Version.Palette.Status != "pending" {
		t.Fatalf("image metadata = %d %s", filtered.Code, filtered.Body.String())
	}
	versionID := imagePage.Items[0].Version.ID
	imageDetail := request(t, router, http.MethodGet, "/library/assets/"+imagePage.Items[0].ID, nil, "account-a")
	var imageDetailView struct {
		Versions []struct {
			Palette struct {
				Colors []domain.PaletteColor `json:"colors"`
			} `json:"palette"`
		} `json:"versions"`
	}
	if imageDetail.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(imageDetail), &imageDetailView) != nil ||
		len(imageDetailView.Versions) != 1 || imageDetailView.Versions[0].Palette.Colors == nil {
		t.Fatalf("pending palette colors = %d %s", imageDetail.Code, imageDetail.Body.String())
	}
	claimed, err := handler.Repo.ClaimPendingPaletteJobs(ctx, 1, time.Now().UTC())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim palette = %#v, %v", claimed, err)
	}
	if err := handler.Repo.FailAssetVersionPalette(ctx, "account-a", versionID, "decode_failed", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	retried := request(t, router, http.MethodPost, "/assets/"+imageAsset.ID+"/versions/"+versionID+"/palette/retry", nil, "account-a")
	if retried.Code != http.StatusOK {
		t.Fatalf("retry palette = %d %s", retried.Code, retried.Body.String())
	}
	palette, err := handler.Repo.GetAssetVersionPalette(ctx, "account-a", versionID)
	if err != nil || palette.Status != "pending" || palette.ErrorCode != "" {
		t.Fatalf("retried palette = %#v, %v", palette, err)
	}
	foreignRetry := request(t, router, http.MethodPost, "/assets/"+imageAsset.ID+"/versions/"+versionID+"/palette/retry", nil, "account-b")
	if foreignRetry.Code != http.StatusNotFound {
		t.Fatalf("foreign palette retry = %d %s", foreignRetry.Code, foreignRetry.Body.String())
	}
	preferences := request(t, router, http.MethodPut, "/library/preferences", map[string]any{"tree_mode": "format", "last_project_id": project.ID}, "account-a")
	if preferences.Code != http.StatusOK {
		t.Fatalf("save preferences = %d %s", preferences.Code, preferences.Body.String())
	}
	readPreferences := request(t, router, http.MethodGet, "/library/preferences", nil, "account-a")
	var saved struct {
		TreeMode      string `json:"tree_mode"`
		LastProjectID string `json:"last_project_id"`
	}
	if readPreferences.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(readPreferences), &saved) != nil ||
		saved.TreeMode != "format" || saved.LastProjectID != project.ID {
		t.Fatalf("preferences = %d %s", readPreferences.Code, readPreferences.Body.String())
	}
	page, err := handler.Repo.ListProjectAssets(ctx, "account-a", domain.ProjectAssetListQuery{ProjectID: project.ID})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("archive entries = %#v, %v", page, err)
	}
	archive := request(t, router, http.MethodPost, "/library/assets/export", map[string]any{
		"project_asset_ids": []string{page.Items[0].ID, page.Items[1].ID},
	}, "account-a")
	if archive.Code != http.StatusOK || archive.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("archive = %d %s", archive.Code, archive.Body.String())
	}
	reader, err := zip.NewReader(bytes.NewReader(archive.Body.Bytes()), int64(archive.Body.Len()))
	if err != nil || len(reader.File) != 2 {
		t.Fatalf("archive entries = %v, %v", reader, err)
	}
}

func TestStudioLibraryAdvancedFilters(t *testing.T) {
	router, handler := newLibraryRouter(t)
	ctx := context.Background()
	project, err := domain.NewProject(uuid.NewString(), "account-a", "图库", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Repo.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 3, 2))
	frame.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"甲.png", "乙.png"} {
		if _, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
			AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: name,
			MIMEType: "image/png", Content: bytes.NewReader(encoded.Bytes()),
		}, handler.Blob); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := handler.Service.UploadAsset(ctx, studioapp.UploadAssetInput{
		AccountID: "account-a", ProjectID: project.ID, RequestID: uuid.NewString(), Name: "说明.txt",
		MIMEType: "text/plain", Content: bytes.NewBufferString("说明"),
	}, handler.Blob); err != nil {
		t.Fatal(err)
	}
	start := url.QueryEscape(time.Now().UTC().Add(-time.Hour).Format(time.RFC3339))
	end := url.QueryEscape(time.Now().UTC().Add(time.Hour).Format(time.RFC3339))
	filtered := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID+
		"&width_min=3&width_max=3&height_min=2&height_max=2&size_min=1&size_max=1000&added_from="+start+
		"&added_to="+end+"&duplicates=true", nil, "account-a")
	var page struct {
		Total int `json:"total"`
	}
	if filtered.Code != http.StatusOK || json.Unmarshal(apitest.DataBytes(filtered), &page) != nil || page.Total != 2 {
		t.Fatalf("combined filter = %d %s", filtered.Code, filtered.Body.String())
	}
	for _, query := range []string{
		"width_min=-1", "width_max=abc", "width_min=4&width_max=3", "height_min=3&height_max=2",
		"size_min=-1", "size_max=abc", "size_min=10&size_max=9", "added_from=tomorrow",
		"added_from=2026-09-29T00:00:00Z&added_to=2026-09-29T00:00:00Z", "duplicates=maybe",
	} {
		invalid := request(t, router, http.MethodGet, "/library/assets?project_id="+project.ID+"&"+query, nil, "account-a")
		if invalid.Code != http.StatusBadRequest {
			t.Fatalf("invalid filter %q = %d %s", query, invalid.Code, invalid.Body.String())
		}
	}
}
