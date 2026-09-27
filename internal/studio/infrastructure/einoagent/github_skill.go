package einoagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

const (
	maxGitHubSkillFiles     = 128
	maxGitHubSkillFileBytes = 256 << 10
	maxGitHubSkillBytes     = 1 << 20
	maxGitHubSkillDescBytes = 4 << 10
)

type githubSkillSource struct {
	Owner     string
	Repo      string
	Ref       string
	Directory string
}

type githubSkillPackage struct {
	Name        string
	Description string
	Prompt      string
}

func parseGitHubSkillURL(rawURL string) (githubSkillSource, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return githubSkillSource{}, fmt.Errorf("studio: Skill URL must be a public GitHub file or directory link")
	}
	segments, err := githubPathSegments(parsed.EscapedPath())
	if err != nil || len(segments) < 4 {
		return githubSkillSource{}, fmt.Errorf("studio: invalid GitHub Skill URL path")
	}
	source := githubSkillSource{Owner: segments[0], Repo: segments[1]}
	mode := segments[2]
	source.Ref = segments[3]
	if source.Owner == "" || source.Repo == "" || source.Ref == "" {
		return githubSkillSource{}, fmt.Errorf("studio: invalid GitHub Skill URL path")
	}
	switch mode {
	case "blob":
		if len(segments) < 5 || segments[len(segments)-1] != "SKILL.md" {
			return githubSkillSource{}, fmt.Errorf("studio: GitHub file link must point to SKILL.md")
		}
		source.Directory = strings.Join(segments[4:len(segments)-1], "/")
	case "tree":
		source.Directory = strings.Join(segments[4:], "/")
	default:
		return githubSkillSource{}, fmt.Errorf("studio: GitHub Skill URL must use blob or tree")
	}
	return source, nil
}

func githubPathSegments(escapedPath string) ([]string, error) {
	parts := strings.Split(strings.Trim(escapedPath, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		segment, err := url.PathUnescape(part)
		if err != nil || segment == "" || segment == "." || segment == ".." || strings.ContainsAny(segment, `/\\`) {
			return nil, fmt.Errorf("studio: invalid GitHub Skill URL path")
		}
		segments = append(segments, segment)
	}
	return segments, nil
}

func fetchGitHubSkill(ctx context.Context, source githubSkillSource) (githubSkillPackage, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 || request.URL.Scheme != "https" || request.URL.Host != "raw.githubusercontent.com" {
				return fmt.Errorf("studio: GitHub Skill download redirected outside GitHub content hosts")
			}
			return nil
		},
	}
	skillPath := path.Join(source.Directory, "SKILL.md")
	skillContent, err := fetchGitHubSkillFile(ctx, client, githubSkillRawURL(source, skillPath))
	if err != nil {
		return githubSkillPackage{}, err
	}
	files := map[string]string{"SKILL.md": string(skillContent)}
	pending := githubSkillReferences(string(skillContent))
	totalBytes := len(skillContent)
	for len(pending) > 0 {
		referencePath := pending[0]
		pending = pending[1:]
		if _, loaded := files[referencePath]; loaded {
			continue
		}
		if len(files) >= maxGitHubSkillFiles || totalBytes >= maxGitHubSkillBytes {
			return githubSkillPackage{}, fmt.Errorf("studio: GitHub Skill text content exceeds the supported size")
		}
		content, err := fetchGitHubSkillFile(ctx, client, githubSkillRawURL(source, path.Join(source.Directory, referencePath)))
		if err != nil {
			return githubSkillPackage{}, err
		}
		if totalBytes+len(content) > maxGitHubSkillBytes {
			return githubSkillPackage{}, fmt.Errorf("studio: GitHub Skill text content exceeds the supported size")
		}
		files[referencePath] = string(content)
		totalBytes += len(content)
		pending = append(pending, githubSkillReferences(string(content))...)
	}
	return buildGitHubSkillPackage(files)
}

var githubSkillReferencePattern = regexp.MustCompile(`(?i)references/[a-z0-9_.-]+(?:/[a-z0-9_.-]+)*\.(?:md|txt)`)

func githubSkillReferences(content string) []string {
	seen := make(map[string]struct{})
	var references []string
	appendReferences := func(value []byte) {
		for _, match := range githubSkillReferencePattern.FindAll(value, -1) {
			referencePath := string(match)
			if _, ok := githubSkillRelativePath("", referencePath); !ok {
				continue
			}
			if _, exists := seen[referencePath]; exists {
				continue
			}
			seen[referencePath] = struct{}{}
			references = append(references, referencePath)
		}
	}
	source := []byte(content)
	document := goldmark.New().Parser().Parse(text.NewReader(source))
	if err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch current := node.(type) {
		case *ast.Text:
			appendReferences(current.Text(source))
		case *ast.Link:
			appendReferences(current.Destination)
		case *ast.Image:
			appendReferences(current.Destination)
		}
		return ast.WalkContinue, nil
	}); err != nil {
		panic(err)
	}
	sort.Strings(references)
	return references
}

func githubSkillRawURL(source githubSkillSource, relativePath string) string {
	return (&url.URL{
		Scheme: "https",
		Host:   "raw.githubusercontent.com",
		Path:   path.Join(source.Owner, source.Repo, source.Ref, relativePath),
	}).String()
}

func fetchGitHubSkillFile(ctx context.Context, client *http.Client, downloadURL string) ([]byte, error) {
	parsed, err := url.Parse(downloadURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "raw.githubusercontent.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("studio: GitHub returned an unsupported Skill file address")
	}
	return githubSkillRequest(ctx, client, parsed.String(), "text/plain", maxGitHubSkillFileBytes)
}

func githubSkillRequest(ctx context.Context, client *http.Client, endpoint, accept string, maxBytes int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "Pixoma-Studio-Skill-Installer")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("studio: read GitHub Skill content: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("studio: GitHub Skill request returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("studio: read GitHub Skill response: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("studio: GitHub Skill response exceeds the supported size")
	}
	return body, nil
}

func githubSkillRelativePath(directory, itemPath string) (string, bool) {
	cleanPath := path.Clean(itemPath)
	if cleanPath != itemPath || strings.HasPrefix(cleanPath, "/") || strings.Contains(cleanPath, `\`) {
		return "", false
	}
	if directory != "" {
		prefix := directory + "/"
		if !strings.HasPrefix(cleanPath, prefix) {
			return "", false
		}
		cleanPath = strings.TrimPrefix(cleanPath, prefix)
	}
	if cleanPath == "" || cleanPath == "." || strings.HasPrefix(cleanPath, "../") {
		return "", false
	}
	return cleanPath, true
}

func isGitHubSkillTextFile(relativePath string) bool {
	return relativePath == "SKILL.md" || strings.EqualFold(path.Ext(relativePath), ".md") || strings.EqualFold(path.Ext(relativePath), ".txt")
}

func buildGitHubSkillPackage(files map[string]string) (githubSkillPackage, error) {
	skillContent, ok := files["SKILL.md"]
	if !ok {
		return githubSkillPackage{}, fmt.Errorf("studio: GitHub Skill directory does not contain SKILL.md")
	}
	metadata, err := parseGitHubSkillMetadata(skillContent)
	if err != nil {
		return githubSkillPackage{}, err
	}
	paths := make([]string, 0, len(files))
	for relativePath := range files {
		if isGitHubSkillTextFile(relativePath) {
			if _, ok := githubSkillRelativePath("", relativePath); !ok {
				return githubSkillPackage{}, fmt.Errorf("studio: invalid relative path in GitHub Skill package")
			}
			paths = append(paths, relativePath)
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
	for _, relativePath := range paths {
		if prompt.Len() > 0 {
			prompt.WriteString("\n\n")
		}
		prompt.WriteString("[")
		prompt.WriteString(relativePath)
		prompt.WriteString("]\n")
		prompt.WriteString(files[relativePath])
	}
	return githubSkillPackage{Name: metadata.Name, Description: metadata.Description, Prompt: prompt.String()}, nil
}

type githubSkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func parseGitHubSkillMetadata(skillContent string) (githubSkillMetadata, error) {
	if !strings.HasPrefix(skillContent, "---\n") && !strings.HasPrefix(skillContent, "---\r\n") {
		return githubSkillMetadata{}, fmt.Errorf("studio: SKILL.md must begin with YAML metadata")
	}
	var metadata githubSkillMetadata
	_, err := frontmatter.MustParse(strings.NewReader(skillContent), &metadata, frontmatter.NewFormat("---", "---", yaml.Unmarshal))
	if errors.Is(err, frontmatter.ErrNotFound) {
		return githubSkillMetadata{}, fmt.Errorf("studio: SKILL.md YAML metadata is not closed")
	}
	if err != nil {
		return githubSkillMetadata{}, fmt.Errorf("studio: invalid SKILL.md YAML metadata: %w", err)
	}
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Description = strings.TrimSpace(metadata.Description)
	if metadata.Name == "" || len(metadata.Name) > 256 || metadata.Description == "" || len(metadata.Description) > maxGitHubSkillDescBytes {
		return githubSkillMetadata{}, fmt.Errorf("studio: SKILL.md metadata requires a valid name and description")
	}
	return metadata, nil
}
