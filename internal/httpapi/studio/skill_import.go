package studio

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Mr9esx/Pixoma/internal/apierr"
	"github.com/Mr9esx/Pixoma/internal/response"
	"github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const (
	maxSkillZipBytes      = 4 << 20
	maxSkillZipEntries    = 256
	maxSkillMultipartOver = 64 << 10
)

func (h *Handler) importSkillZip(w http.ResponseWriter, r *http.Request) {
	accountID, ok := accountID(w, r)
	if !ok {
		return
	}
	if h.Capabilities == nil {
		response.Fail(w, apierr.ErrStudioListSkillsUnavailable, "能力配置服务不可用")
		return
	}
	files, metadata, err := readSkillZip(w, r)
	if err != nil {
		response.FailErr(w, apierr.ErrStudioInvalidBody, err)
		return
	}
	skill, err := h.Capabilities.CreateSkill(r.Context(), application.CreateSkillInput{
		AccountID:   accountID,
		Name:        metadata.Name,
		Description: metadata.Description,
		Enabled:     true,
		Files:       files,
	})
	if err != nil {
		failFromError(w, err)
		return
	}
	response.OKStatus(w, http.StatusCreated, skill)
}

func (h *Handler) inspectSkillZip(w http.ResponseWriter, r *http.Request) {
	if _, ok := accountID(w, r); !ok {
		return
	}
	files, metadata, err := readSkillZip(w, r)
	if err != nil {
		response.FailErr(w, apierr.ErrStudioInvalidBody, err)
		return
	}
	response.OKStatus(w, http.StatusOK, struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		Files       []domain.SkillFile `json:"files"`
	}{Name: metadata.Name, Description: metadata.Description, Files: files})
}

func readSkillZip(w http.ResponseWriter, r *http.Request) ([]domain.SkillFile, application.SkillMetadata, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSkillZipBytes+maxSkillMultipartOver)
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, application.SkillMetadata{}, fmt.Errorf("%w: Skill ZIP 请求格式无效", domain.ErrInvalid)
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || !strings.EqualFold(filepath.Ext(part.FileName()), ".zip") {
		return nil, application.SkillMetadata{}, fmt.Errorf("%w: 请上传 .zip 格式的 Skill 文件", domain.ErrInvalid)
	}
	archive, err := io.ReadAll(io.LimitReader(part, maxSkillZipBytes+1))
	if err != nil || len(archive) > maxSkillZipBytes {
		return nil, application.SkillMetadata{}, fmt.Errorf("%w: Skill ZIP 文件超过大小限制", domain.ErrInvalid)
	}
	if err := part.Close(); err != nil {
		return nil, application.SkillMetadata{}, err
	}
	if next, err := reader.NextPart(); err == nil {
		_ = next.Close()
		return nil, application.SkillMetadata{}, fmt.Errorf("%w: Skill ZIP 请求只能包含一个文件", domain.ErrInvalid)
	} else if err != io.EOF {
		return nil, application.SkillMetadata{}, err
	}
	return parseSkillArchive(archive)
}

func parseSkillArchive(archive []byte) ([]domain.SkillFile, application.SkillMetadata, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, application.SkillMetadata{}, fmt.Errorf("studio: 无法读取 Skill ZIP: %w", err)
	}
	if len(reader.File) == 0 {
		return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 文件数量无效")
	}
	if len(reader.File) > maxSkillZipEntries {
		return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 条目数量超过限制")
	}
	var manifestPath string
	entries := make([]*zip.File, 0, len(reader.File))
	directories := make([]string, 0)
	for _, entry := range reader.File {
		name := strings.TrimSuffix(entry.Name, "/")
		if !validSkillArchivePath(name) {
			return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 包含无效路径 %q", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if entry.UncompressedSize64 != 0 {
				return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 文件夹 %q 包含文件内容", entry.Name)
			}
			directories = append(directories, name)
			continue
		}
		if entry.Mode()&fs.ModeSymlink != 0 {
			return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 不支持符号链接")
		}
		if name == "SKILL.md" || strings.HasSuffix(name, "/SKILL.md") {
			if manifestPath != "" {
				return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 只能包含一个 SKILL.md")
			}
			manifestPath = name
		}
		entries = append(entries, entry)
	}
	if manifestPath == "" {
		return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 缺少 SKILL.md")
	}
	root := ""
	if manifestPath != "SKILL.md" {
		root = strings.TrimSuffix(manifestPath, "/SKILL.md")
	}
	files := make([]domain.SkillFile, 0, len(entries))
	for _, directory := range directories {
		if directory == root || strings.HasPrefix(root, directory+"/") {
			continue
		}
		relativePath := directory
		if root != "" {
			if !strings.HasPrefix(directory, root+"/") {
				return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 文件夹必须位于同一 Skill 目录中")
			}
			relativePath = strings.TrimPrefix(directory, root+"/")
		}
		files = append(files, domain.SkillFile{Path: relativePath, Directory: true})
	}
	totalBytes := 0
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name, "/")
		relativePath := name
		if root != "" {
			prefix := root + "/"
			if !strings.HasPrefix(name, prefix) {
				return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 文件必须位于同一 Skill 目录中")
			}
			relativePath = strings.TrimPrefix(name, prefix)
		}
		if entry.UncompressedSize64 > 256<<10 {
			return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill 文件 %q 超过大小限制", relativePath)
		}
		file, err := entry.Open()
		if err != nil {
			return nil, application.SkillMetadata{}, fmt.Errorf("studio: 无法读取 Skill 文件 %q: %w", relativePath, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(file, (256<<10)+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, application.SkillMetadata{}, readErr
		}
		if closeErr != nil {
			return nil, application.SkillMetadata{}, closeErr
		}
		if len(content) > 256<<10 {
			return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill 文件 %q 超过大小限制", relativePath)
		}
		totalBytes += len(content)
		if totalBytes > 1<<20 {
			return nil, application.SkillMetadata{}, fmt.Errorf("studio: Skill ZIP 解压后超过大小限制")
		}
		textContent := utf8.Valid(content) && !strings.ContainsRune(string(content), '\x00')
		value := string(content)
		if !textContent {
			value = base64.StdEncoding.EncodeToString(content)
		}
		files = append(files, domain.SkillFile{Path: relativePath, Content: value, Binary: !textContent})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	metadata, _, err := application.ValidateSkillFiles(files)
	if err != nil {
		return nil, application.SkillMetadata{}, err
	}
	if root != "" && path.Base(root) != metadata.Name {
		return nil, application.SkillMetadata{}, fmt.Errorf("studio: SKILL.md name 必须与 ZIP 中的 Skill 目录名一致")
	}
	return files, metadata, nil
}

func validSkillArchivePath(name string) bool {
	return name != "" && name != "." && fs.ValidPath(name) && !path.IsAbs(name) &&
		path.Clean(name) == name && !strings.ContainsAny(name, `:\`) && !strings.ContainsRune(name, '\x00') && !strings.HasPrefix(name, "../")
}
