package studio_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/Mr9esx/Pixoma/internal/httpapi/apitest"
	setupapi "github.com/Mr9esx/Pixoma/internal/httpapi/setup"
	studioapi "github.com/Mr9esx/Pixoma/internal/httpapi/studio"
	"github.com/Mr9esx/Pixoma/internal/platform/db"
	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/Mr9esx/Pixoma/internal/studio/infrastructure/persistence"
)

type skillZipEntry struct {
	path    string
	content []byte
}

func TestCreateSkillWithIDEFiles(t *testing.T) {
	router := skillImportRouter(t)
	manifest := "---\nname: comic-storyboard\ndescription: \"Use this skill: plan a storyboard.\"\n---\n# Steps\nRead references/guide.md."
	created := request(t, router, http.MethodPost, "/skills", map[string]any{
		"name":        "comic-storyboard",
		"description": "Use this skill: plan a storyboard.",
		"prompt":      manifest,
		"enabled":     true,
		"files": []map[string]any{
			{"path": "SKILL.md", "content": manifest},
			{"path": "references/guide.md", "content": "Choose a camera angle."},
		},
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.Code, created.Body.String())
	}
	var skill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(created.Result())).Decode(&skill); err != nil {
		t.Fatal(err)
	}
	if skill.Name != "comic-storyboard" || len(skill.Files) != 2 || !strings.Contains(skill.Prompt, "[references/guide.md]\nChoose a camera angle.") {
		t.Fatalf("created Skill = %#v", skill)
	}
}

func TestImportSkillZipKeepsFilesAndSupportsEditing(t *testing.T) {
	router := skillImportRouter(t)
	manifest := "---\nname: sample-skill\ndescription: Use this skill to prepare a sample.\n---\n# Sample\nRead references/guide.md and scripts/check.py."
	archive := skillZip(t, []skillZipEntry{
		{path: "sample-skill/SKILL.md", content: []byte(manifest)},
		{path: "sample-skill/references/guide.md", content: []byte("Original guide")},
		{path: "sample-skill/scripts/check.py", content: []byte("print('ok')\n")},
		{path: "sample-skill/assets/icon.png", content: []byte{0x89, 0x50, 0x4e, 0x47, 0x00}},
	})
	created := uploadSkillZip(t, router, archive, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("import status = %d, body = %s", created.Code, created.Body.String())
	}
	var skill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(created.Result())).Decode(&skill); err != nil {
		t.Fatal(err)
	}
	if skill.Name != "sample-skill" || len(skill.Files) != 4 || !strings.Contains(skill.Prompt, "[references/guide.md]\nOriginal guide") || !strings.Contains(skill.Prompt, "[scripts/check.py]") {
		t.Fatalf("imported skill = %#v", skill)
	}
	if skill.Files[1].Path != "assets/icon.png" || !skill.Files[1].Binary {
		t.Fatalf("binary resource = %#v", skill.Files)
	}
	for i := range skill.Files {
		if skill.Files[i].Path == "references/guide.md" {
			skill.Files[i].Content = "Updated guide"
		}
	}
	updated := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
		"name": skill.Name, "description": skill.Description, "prompt": skill.Prompt,
		"version": "1.0.1", "updated_at": skill.UpdatedAt, "files": skill.Files,
	}, "account-a")
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), "Updated guide") {
		t.Fatalf("update status = %d, body = %s", updated.Code, updated.Body.String())
	}
	old := request(t, router, http.MethodGet, "/skills/"+skill.ID+"/versions/1.0.0", nil, "account-a")
	if old.Code != http.StatusOK {
		t.Fatalf("old version status = %d, body = %s", old.Code, old.Body.String())
	}
	var original studioapp.SkillVersionView
	if err := json.NewDecoder(apitest.DataReader(old.Result())).Decode(&original); err != nil {
		t.Fatal(err)
	}
	foundOriginalGuide := false
	for _, file := range original.Files {
		if file.Path == "references/guide.md" {
			foundOriginalGuide = file.Content == "Original guide"
		}
	}
	if !foundOriginalGuide {
		t.Fatalf("original files = %#v", original.Files)
	}
	list := request(t, router, http.MethodGet, "/skills", nil, "account-a")
	var listed []studioapp.SkillSummary
	if err := json.NewDecoder(apitest.DataReader(list.Result())).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != skill.ID {
		t.Fatalf("saved skill = %#v", listed)
	}
	detail := request(t, router, http.MethodGet, "/skills/"+skill.ID, nil, "account-a")
	var saved studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(detail.Result())).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Files) != 4 || !strings.Contains(saved.Prompt, "Updated guide") {
		t.Fatalf("saved detail = %#v", saved)
	}
	foreign := request(t, router, http.MethodGet, "/skills", nil, "account-b")
	var other []studioapp.SkillSummary
	if err := json.NewDecoder(apitest.DataReader(foreign.Result())).Decode(&other); err != nil {
		t.Fatal(err)
	}
	if len(other) != 0 {
		t.Fatalf("other account skills = %#v", other)
	}
}

func TestImportSkillZipWithRootManifest(t *testing.T) {
	router := skillImportRouter(t)
	manifest := "---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions"
	archive := skillZip(t, []skillZipEntry{
		{path: "SKILL.md", content: []byte(manifest)},
		{path: "references/guide.md", content: []byte("Read this guide")},
	})
	for _, endpoint := range []struct {
		path   string
		status int
	}{
		{path: "/skills/import/inspect", status: http.StatusOK},
		{path: "/skills/import", status: http.StatusCreated},
	} {
		result := uploadSkillZipTo(t, router, archive, "account-a", endpoint.path)
		if result.Code != endpoint.status {
			t.Fatalf("%s 状态 = %d，响应 = %s", endpoint.path, result.Code, result.Body.String())
		}
		var packageView struct {
			Files []domain.SkillFile `json:"files"`
		}
		if err := json.NewDecoder(apitest.DataReader(result.Result())).Decode(&packageView); err != nil {
			t.Fatal(err)
		}
		if len(packageView.Files) != 2 || packageView.Files[0].Path != "SKILL.md" || packageView.Files[1].Path != "references/guide.md" {
			t.Fatalf("%s 文件 = %#v", endpoint.path, packageView.Files)
		}
	}
}

func TestImportSkillZipWithMaximumFilesAndRootDirectory(t *testing.T) {
	router := skillImportRouter(t)
	manifest := "---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions"
	entries := []skillZipEntry{
		{path: "sample-skill/"},
		{path: "sample-skill/SKILL.md", content: []byte(manifest)},
	}
	for index := 0; index < 127; index++ {
		entries = append(entries, skillZipEntry{
			path:    fmt.Sprintf("sample-skill/references/guide-%03d.md", index),
			content: []byte("Guide"),
		})
	}
	archive := skillZip(t, entries)
	inspected := uploadSkillZipTo(t, router, archive, "account-a", "/skills/import/inspect")
	if inspected.Code != http.StatusOK {
		t.Fatalf("检查状态 = %d，响应 = %s", inspected.Code, inspected.Body.String())
	}
	var packageView struct {
		Files []domain.SkillFile `json:"files"`
	}
	if err := json.NewDecoder(apitest.DataReader(inspected.Result())).Decode(&packageView); err != nil {
		t.Fatal(err)
	}
	if len(packageView.Files) != 128 {
		t.Fatalf("检查文件数量 = %d", len(packageView.Files))
	}
	created := uploadSkillZip(t, router, archive, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("导入状态 = %d，响应 = %s", created.Code, created.Body.String())
	}
	var skill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(created.Result())).Decode(&skill); err != nil {
		t.Fatal(err)
	}
	if len(skill.Files) != 128 {
		t.Fatalf("导入文件数量 = %d", len(skill.Files))
	}
}

func TestImportSkillZipRejectsExcessiveEntries(t *testing.T) {
	router := skillImportRouter(t)
	manifest := []byte("---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions")
	tooManyFiles := []skillZipEntry{{path: "sample-skill/"}, {path: "sample-skill/SKILL.md", content: manifest}}
	for index := 0; index < 128; index++ {
		tooManyFiles = append(tooManyFiles, skillZipEntry{path: fmt.Sprintf("sample-skill/file-%03d.md", index), content: []byte("Guide")})
	}
	tooManyRawEntries := []skillZipEntry{{path: "sample-skill/SKILL.md", content: manifest}}
	for index := 0; index < 256; index++ {
		tooManyRawEntries = append(tooManyRawEntries, skillZipEntry{path: "sample-skill/"})
	}
	for name, entries := range map[string][]skillZipEntry{
		"converted files": tooManyFiles,
		"raw ZIP entries": tooManyRawEntries,
	} {
		t.Run(name, func(t *testing.T) {
			archive := skillZip(t, entries)
			for _, endpoint := range []string{"/skills/import/inspect", "/skills/import"} {
				result := uploadSkillZipTo(t, router, archive, "account-a", endpoint)
				if result.Code != http.StatusBadRequest {
					t.Fatalf("%s 状态 = %d，响应 = %s", endpoint, result.Code, result.Body.String())
				}
			}
		})
	}
}

func TestImportSkillZipRejectsInvalidPackages(t *testing.T) {
	router := skillImportRouter(t)
	valid := []byte("---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions")
	cases := map[string][]skillZipEntry{
		"missing manifest":      {{path: "sample-skill/README.md", content: []byte("readme")}},
		"missing description":   {{path: "sample-skill/SKILL.md", content: []byte("---\nname: sample-skill\n---\n# Instructions")}},
		"name mismatch":         {{path: "other-skill/SKILL.md", content: valid}},
		"path traversal":        {{path: "sample-skill/SKILL.md", content: valid}, {path: "sample-skill/../outside.txt", content: []byte("outside")}},
		"duplicate file":        {{path: "sample-skill/SKILL.md", content: valid}, {path: "sample-skill/SKILL.md", content: valid}},
		"file folder collision": {{path: "sample-skill/SKILL.md", content: valid}, {path: "sample-skill/references", content: []byte("text")}, {path: "sample-skill/references/guide.md", content: []byte("guide")}},
		"NUL path":              {{path: "sample-skill/SKILL.md", content: valid}, {path: "sample-skill/ref\x00.md", content: []byte("guide")}},
		"long path":             {{path: "sample-skill/SKILL.md", content: valid}, {path: "sample-skill/" + strings.Repeat("a", 513), content: []byte("guide")}},
		"deep path":             {{path: "sample-skill/SKILL.md", content: valid}, {path: "sample-skill/" + strings.Repeat("d/", 16) + "guide.md", content: []byte("guide")}},
		"sibling file":          {{path: "sample-skill/SKILL.md", content: valid}, {path: "other-skill/guide.md", content: []byte("guide")}},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			archive := skillZip(t, files)
			for _, endpoint := range []string{"/skills/import/inspect", "/skills/import"} {
				result := uploadSkillZipTo(t, router, archive, "account-a", endpoint)
				if result.Code != http.StatusBadRequest {
					t.Fatalf("%s 状态 = %d，响应 = %s", endpoint, result.Code, result.Body.String())
				}
			}
		})
	}
	list := request(t, router, http.MethodGet, "/skills", nil, "account-a")
	var skills []studioapp.SkillSummary
	if err := json.NewDecoder(apitest.DataReader(list.Result())).Decode(&skills); err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Fatalf("invalid imports created skills: %#v", skills)
	}
}

func TestImportedSkillWithLargeBinaryResourcesCanBeUpdated(t *testing.T) {
	router := skillImportRouter(t)
	entries := []skillZipEntry{{
		path:    "sample-skill/SKILL.md",
		content: []byte("---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions"),
	}}
	for index := 0; index < 4; index++ {
		entries = append(entries, skillZipEntry{
			path:    fmt.Sprintf("sample-skill/assets/resource-%d.bin", index),
			content: bytes.Repeat([]byte{0x00, 0xff}, 120<<10),
		})
	}
	created := uploadSkillZip(t, router, skillZip(t, entries), "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("import status = %d, body = %s", created.Code, created.Body.String())
	}
	var skill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(created.Result())).Decode(&skill); err != nil {
		t.Fatal(err)
	}
	requestBody := map[string]any{
		"name": skill.Name, "description": skill.Description,
		"version": "1.0.1", "updated_at": skill.UpdatedAt, "files": skill.Files,
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) <= 1<<20 {
		t.Fatalf("request body = %d bytes, expected over 1 MiB", len(encoded))
	}
	updated := request(t, router, http.MethodPatch, "/skills/"+skill.ID, requestBody, "account-a")
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updated.Code, updated.Body.String())
	}
}

func TestInspectSkillZipAndKeepEmptyDirectory(t *testing.T) {
	router := skillImportRouter(t)
	manifest := "---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions"
	archive := skillZip(t, []skillZipEntry{
		{path: "sample-skill/"},
		{path: "sample-skill/SKILL.md", content: []byte(manifest)},
		{path: "sample-skill/references/"},
	})
	inspected := uploadSkillZipTo(t, router, archive, "account-a", "/skills/import/inspect")
	if inspected.Code != http.StatusOK {
		t.Fatalf("检查状态 = %d，响应 = %s", inspected.Code, inspected.Body.String())
	}
	var packageView struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		Files       []domain.SkillFile `json:"files"`
	}
	if err := json.NewDecoder(apitest.DataReader(inspected.Result())).Decode(&packageView); err != nil {
		t.Fatal(err)
	}
	if packageView.Name != "sample-skill" || len(packageView.Files) != 2 || !packageView.Files[0].Directory && !packageView.Files[1].Directory {
		t.Fatalf("检查结果 = %#v", packageView)
	}
	list := request(t, router, http.MethodGet, "/skills", nil, "account-a")
	var listed []studioapp.SkillSummary
	if err := json.NewDecoder(apitest.DataReader(list.Result())).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("检查操作创建了 Skill：%#v", listed)
	}
	created := request(t, router, http.MethodPost, "/skills", map[string]any{
		"files": packageView.Files, "enabled": true,
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("创建状态 = %d，响应 = %s", created.Code, created.Body.String())
	}
	var skill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(created.Result())).Decode(&skill); err != nil {
		t.Fatal(err)
	}
	detail := request(t, router, http.MethodGet, "/skills/"+skill.ID, nil, "account-a")
	var saved studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(detail.Result())).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Files) != 2 || !saved.Files[1].Directory {
		t.Fatalf("保存后文件夹 = %#v", saved.Files)
	}
}

func TestSkillDetailEnabledAndVersionConflict(t *testing.T) {
	router := skillImportRouter(t)
	manifest := "---\nname: sample-skill\ndescription: Use this skill for samples.\n---\n# Instructions"
	created := request(t, router, http.MethodPost, "/skills", map[string]any{
		"files": []map[string]any{{"path": "SKILL.md", "content": manifest}}, "enabled": true,
	}, "account-a")
	if created.Code != http.StatusCreated {
		t.Fatalf("创建状态 = %d，响应 = %s", created.Code, created.Body.String())
	}
	var skill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(created.Result())).Decode(&skill); err != nil {
		t.Fatal(err)
	}
	list := request(t, router, http.MethodGet, "/skills", nil, "account-a")
	if strings.Contains(list.Body.String(), "SKILL.md") || strings.Contains(list.Body.String(), "prompt") {
		t.Fatalf("列表包含完整文件内容：%s", list.Body.String())
	}
	detail := request(t, router, http.MethodGet, "/skills/"+skill.ID, nil, "account-a")
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "SKILL.md") {
		t.Fatalf("详情状态 = %d，响应 = %s", detail.Code, detail.Body.String())
	}
	foreign := request(t, router, http.MethodGet, "/skills/"+skill.ID, nil, "account-b")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("其他账户详情状态 = %d", foreign.Code)
	}
	disabled := request(t, router, http.MethodPatch, "/skills/"+skill.ID+"/enabled", map[string]any{"enabled": false}, "account-a")
	if disabled.Code != http.StatusOK {
		t.Fatalf("停用状态 = %d，响应 = %s", disabled.Code, disabled.Body.String())
	}
	var disabledSkill studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(disabled.Result())).Decode(&disabledSkill); err != nil {
		t.Fatal(err)
	}
	if disabledSkill.Enabled || len(disabledSkill.Files) != 1 || !disabledSkill.UpdatedAt.Equal(skill.UpdatedAt) {
		t.Fatalf("停用后内容 = %#v", disabledSkill)
	}
	missingVersion := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
		"files": skill.Files,
	}, "account-a")
	if missingVersion.Code != http.StatusBadRequest {
		t.Fatalf("缺少版本状态 = %d", missingVersion.Code)
	}
	skill.Files[0].Content = strings.Replace(skill.Files[0].Content, "Instructions", "Updated", 1)
	updated := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
		"version": "1.0.1", "updated_at": skill.UpdatedAt, "files": skill.Files,
	}, "account-a")
	if updated.Code != http.StatusOK {
		t.Fatalf("保存状态 = %d，响应 = %s", updated.Code, updated.Body.String())
	}
	var current studioapp.SkillView
	if err := json.NewDecoder(apitest.DataReader(updated.Result())).Decode(&current); err != nil {
		t.Fatal(err)
	}
	if current.Enabled || !current.UpdatedAt.After(skill.UpdatedAt) {
		t.Fatalf("保存后状态 = %#v", current)
	}
	stale := request(t, router, http.MethodPatch, "/skills/"+skill.ID, map[string]any{
		"version": "1.0.2", "updated_at": skill.UpdatedAt, "files": skill.Files,
	}, "account-a")
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "4091204") {
		t.Fatalf("过期版本状态 = %d，响应 = %s", stale.Code, stale.Body.String())
	}
}

func skillImportRouter(t *testing.T) http.Handler {
	t.Helper()
	dsn := fmt.Sprintf("file:skill_import_%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := db.Open(db.Options{DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(gdb, &persistence.SkillRow{}, &persistence.SkillVersionRow{}); err != nil {
		t.Fatal(err)
	}
	service := &studioapp.CapabilityConfigService{Repo: persistence.NewGormRepository(gdb)}
	handler := &studioapi.Handler{Capabilities: service}
	router := chi.NewRouter()
	handler.Mount(router)
	return router
}

func skillZip(t *testing.T, entries []skillZipEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, entry := range entries {
		file, err := writer.Create(entry.path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(entry.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func uploadSkillZip(t *testing.T, handler http.Handler, archive []byte, accountID string) *httptest.ResponseRecorder {
	return uploadSkillZipTo(t, handler, archive, accountID, "/skills/import")
}

func uploadSkillZipTo(t *testing.T, handler http.Handler, archive []byte, accountID, path string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "skill.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(setupapi.WithAccount(req.Context(), setupapi.AccountSession{AccountID: accountID, Username: accountID, Role: "admin"}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
