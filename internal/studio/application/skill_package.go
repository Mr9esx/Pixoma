package application

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

const (
	maxSkillFiles     = 128
	maxSkillFileBytes = 256 << 10
	maxSkillBytes     = 1 << 20
	maxSkillPathBytes = 512
	maxSkillPathParts = 16
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type SkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func ValidateSkillFiles(files []domain.SkillFile) (SkillMetadata, string, error) {
	if len(files) == 0 || len(files) > maxSkillFiles {
		return SkillMetadata{}, "", fmt.Errorf("studio: Skill 包文件数量必须在 1 到 %d 个之间", maxSkillFiles)
	}
	byPath := make(map[string]domain.SkillFile, len(files))
	totalBytes := 0
	for _, file := range files {
		if !utf8.ValidString(file.Path) || !fs.ValidPath(file.Path) || file.Path == "." || strings.ContainsAny(file.Path, `:\`) || strings.ContainsRune(file.Path, '\x00') || strings.HasPrefix(file.Path, "/") {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 包含无效文件路径 %q", file.Path)
		}
		if len(file.Path) > maxSkillPathBytes {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 文件路径 %q 超过 %d 字节", file.Path, maxSkillPathBytes)
		}
		if strings.Count(file.Path, "/") >= maxSkillPathParts {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 文件路径 %q 超过 %d 层", file.Path, maxSkillPathParts)
		}
		if _, exists := byPath[file.Path]; exists {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 包含重复文件 %q", file.Path)
		}
		if file.Directory {
			if file.Binary || file.Content != "" {
				return SkillMetadata{}, "", fmt.Errorf("studio: Skill 文件夹 %q 不能包含文件内容", file.Path)
			}
			byPath[file.Path] = file
			continue
		}
		contentBytes := []byte(file.Content)
		if file.Binary {
			decoded, err := base64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				return SkillMetadata{}, "", fmt.Errorf("studio: Skill 二进制文件 %q 编码无效", file.Path)
			}
			contentBytes = decoded
		} else if !utf8.ValidString(file.Content) || strings.ContainsRune(file.Content, '\x00') {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 文本文件 %q 编码无效", file.Path)
		}
		if len(contentBytes) > maxSkillFileBytes {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 文件 %q 超过大小限制", file.Path)
		}
		totalBytes += len(contentBytes)
		if totalBytes > maxSkillBytes {
			return SkillMetadata{}, "", fmt.Errorf("studio: Skill 包超过大小限制")
		}
		byPath[file.Path] = file
	}
	for _, file := range files {
		for parent := path.Dir(file.Path); parent != "."; parent = path.Dir(parent) {
			if entry, exists := byPath[parent]; exists && !entry.Directory {
				return SkillMetadata{}, "", fmt.Errorf("studio: Skill 路径 %q 同时作为文件和文件夹", parent)
			}
		}
	}
	manifest, ok := byPath["SKILL.md"]
	if !ok || manifest.Binary || manifest.Directory {
		return SkillMetadata{}, "", fmt.Errorf("studio: Skill 包必须包含文本文件 SKILL.md")
	}
	metadata, err := parseSkillMetadata(manifest.Content)
	if err != nil {
		return SkillMetadata{}, "", err
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if !file.Binary && !file.Directory {
			paths = append(paths, file.Path)
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		if paths[i] == "SKILL.md" {
			return true
		}
		if paths[j] == "SKILL.md" {
			return false
		}
		return paths[i] < paths[j]
	})
	var prompt strings.Builder
	for _, filePath := range paths {
		if prompt.Len() > 0 {
			prompt.WriteString("\n\n")
		}
		prompt.WriteString("[")
		prompt.WriteString(filePath)
		prompt.WriteString("]\n")
		prompt.WriteString(byPath[filePath].Content)
	}
	for _, file := range files {
		if file.Binary && !file.Directory {
			prompt.WriteString("\n\n[")
			prompt.WriteString(file.Path)
			prompt.WriteString("] binary resource")
		}
	}
	return metadata, prompt.String(), nil
}

func parseSkillMetadata(content string) (SkillMetadata, error) {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return SkillMetadata{}, fmt.Errorf("studio: SKILL.md 必须以 YAML 元数据开头")
	}
	var metadata SkillMetadata
	body, err := frontmatter.MustParse(strings.NewReader(content), &metadata, frontmatter.NewFormat("---", "---", yaml.Unmarshal))
	if errors.Is(err, frontmatter.ErrNotFound) {
		return SkillMetadata{}, fmt.Errorf("studio: SKILL.md YAML 元数据缺少结束标记")
	}
	if err != nil {
		return SkillMetadata{}, fmt.Errorf("studio: SKILL.md YAML 元数据无效: %w", err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return SkillMetadata{}, fmt.Errorf("studio: SKILL.md 缺少操作说明")
	}
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Description = strings.TrimSpace(metadata.Description)
	if len(metadata.Name) > 64 || !skillNamePattern.MatchString(metadata.Name) {
		return SkillMetadata{}, fmt.Errorf("studio: SKILL.md name 必须为 1 到 64 位小写字母、数字和连字符")
	}
	if metadata.Description == "" || utf8.RuneCountInString(metadata.Description) > 1024 {
		return SkillMetadata{}, fmt.Errorf("studio: SKILL.md description 必须在 1 到 1024 个字符之间")
	}
	return metadata, nil
}
