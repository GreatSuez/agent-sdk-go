package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// InstallFromGitHub downloads a skill from a GitHub repository and saves it locally.
// repoRef can be: "owner/repo/path/to/skill" or "owner/repo" (installs all skills).
// destDir is the local directory to save into (e.g., "./skills").
// Returns the number of skills installed.
func InstallFromGitHub(repoRef string, destDir string) (int, error) {
	owner, repo, skillPath, err := parseGitHubRef(repoRef)
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// If a specific skill path is given, install just that one
	if skillPath != "" {
		if isSkillCollectionPath(skillPath) {
			return installAllSkillsFrom(owner, repo, skillPath, destDir)
		}
		return installSingleSkill(owner, repo, skillPath, destDir)
	}

	// Otherwise, list the skills directory and install all
	return installAllSkills(owner, repo, destDir)
}

func parseGitHubRef(repoRef string) (owner, repo, skillPath string, err error) {
	ref := strings.TrimSpace(repoRef)
	if ref == "" {
		return "", "", "", fmt.Errorf("invalid repo reference %q — expected owner/repo or GitHub URL", repoRef)
	}

	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		u, parseErr := url.Parse(ref)
		if parseErr != nil {
			return "", "", "", fmt.Errorf("invalid repo URL %q: %w", repoRef, parseErr)
		}
		host := strings.ToLower(strings.TrimSpace(u.Host))
		if host != "github.com" && host != "www.github.com" {
			return "", "", "", fmt.Errorf("unsupported repo host %q (expected github.com)", u.Host)
		}
		parts := splitPath(u.Path)
		if len(parts) < 2 {
			return "", "", "", fmt.Errorf("invalid GitHub URL %q — expected /owner/repo", repoRef)
		}
		owner = parts[0]
		repo = strings.TrimSuffix(parts[1], ".git")
		if len(parts) > 4 && parts[2] == "tree" {
			// /owner/repo/tree/<branch>/<path...>
			skillPath = strings.Join(parts[4:], "/")
		} else if len(parts) > 4 && parts[2] == "blob" {
			// /owner/repo/blob/<branch>/<path...>
			skillPath = strings.Join(parts[4:], "/")
		} else if len(parts) > 2 {
			skillPath = strings.Join(parts[2:], "/")
		}
		return owner, repo, skillPath, nil
	}

	parts := strings.SplitN(ref, "/", 3)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid repo reference %q — expected owner/repo or owner/repo/path", repoRef)
	}
	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	if len(parts) == 3 {
		skillPath = parts[2]
	}
	return owner, repo, skillPath, nil
}

func installSingleSkill(owner, repo, skillPath, destDir string) (int, error) {
	// Try to fetch SKILL.md from the path
	skillMDPath := skillPath + "/SKILL.md"
	content, err := fetchGitHubFile(owner, repo, skillMDPath)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch %s from %s/%s: %w", skillMDPath, owner, repo, err)
	}

	// Parse to get the skill name
	s, err := Parse(content)
	if err != nil {
		return 0, fmt.Errorf("failed to parse skill from %s/%s/%s: %w", owner, repo, skillPath, err)
	}

	// Save locally
	localDir := filepath.Join(destDir, s.Name)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create skill directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(localDir, skillFileName), []byte(content), 0644); err != nil {
		return 0, fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	s.Path = localDir
	s.Source = fmt.Sprintf("github:%s/%s", owner, repo)

	// Register if not already present
	if _, exists := Get(s.Name); !exists {
		if err := Register(s); err != nil {
			return 0, err
		}
	}

	return 1, nil
}

func installAllSkills(owner, repo, destDir string) (int, error) {
	return installAllSkillsFrom(owner, repo, "", destDir)
}

func installAllSkillsFrom(owner, repo, basePath, destDir string) (int, error) {
	// List contents of the "skills" directory in the repo
	searchPaths := []string{"skills", "skills/.curated", "skills/.experimental"}
	if bp := strings.Trim(strings.TrimSpace(basePath), "/"); bp != "" {
		searchPaths = []string{bp, bp + "/.curated", bp + "/.experimental"}
	}
	installed := 0

	for _, searchPath := range searchPaths {
		entries, err := listGitHubDir(owner, repo, searchPath)
		if err != nil {
			continue // directory might not exist
		}

		for _, entry := range entries {
			if entry.Type != "dir" {
				continue
			}
			n, err := installSingleSkill(owner, repo, searchPath+"/"+entry.Name, destDir)
			if err != nil {
				continue // skip skills that fail to install
			}
			installed += n
		}
	}

	if installed == 0 {
		fallback, err := installFromRootDirs(owner, repo, basePath, destDir)
		if err == nil {
			installed += fallback
		}
	}

	if installed == 0 {
		if strings.TrimSpace(basePath) == "" {
			return 0, fmt.Errorf("no skills found in %s/%s", owner, repo)
		}
		return 0, fmt.Errorf("no skills found in %s/%s under %q", owner, repo, basePath)
	}
	return installed, nil
}

func installFromRootDirs(owner, repo, basePath, destDir string) (int, error) {
	root := strings.Trim(strings.TrimSpace(basePath), "/")
	skillDirs, err := discoverSkillCollectionPaths(owner, repo, root, 5)
	if err != nil {
		return 0, err
	}
	installed := 0
	seen := map[string]bool{}
	for _, dir := range skillDirs {
		dir = strings.Trim(strings.TrimSpace(dir), "/")
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		n, installErr := installAllSkillsFrom(owner, repo, dir, destDir)
		if installErr != nil {
			continue
		}
		installed += n
	}
	if installed == 0 {
		return 0, fmt.Errorf("no installable skills found in root directories")
	}
	return installed, nil
}

func discoverSkillCollectionPaths(owner, repo, base string, maxDepth int) ([]string, error) {
	if maxDepth < 0 {
		return nil, nil
	}
	entries, err := listGitHubDir(owner, repo, strings.Trim(strings.TrimSpace(base), "/"))
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}
		path := entry.Name
		if strings.TrimSpace(base) != "" {
			path = strings.Trim(strings.TrimSpace(base), "/") + "/" + entry.Name
		}
		if entry.Name == "skills" {
			paths = append(paths, path)
		}
		if maxDepth == 0 {
			continue
		}
		nested, nestedErr := discoverSkillCollectionPaths(owner, repo, path, maxDepth-1)
		if nestedErr != nil {
			continue
		}
		paths = append(paths, nested...)
	}
	return paths, nil
}

func isSkillCollectionPath(path string) bool {
	p := strings.Trim(strings.TrimSpace(path), "/")
	if p == "" {
		return false
	}
	if p == "skills" || strings.HasSuffix(p, "/skills") {
		return true
	}
	if strings.HasSuffix(p, "/.curated") || strings.HasSuffix(p, "/.experimental") {
		return true
	}
	return false
}

func splitPath(path string) []string {
	raw := strings.Split(path, "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

type githubEntry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file" or "dir"
	Path string `json:"path"`
}

func fetchGitHubFile(owner, repo, path string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, path)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Try HEAD branch
		url = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/%s", owner, repo, path)
		resp2, err := httpClient.Get(url)
		if err != nil {
			return "", err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != 200 {
			return "", fmt.Errorf("file not found: %s (HTTP %d)", path, resp2.StatusCode)
		}
		body, err := io.ReadAll(resp2.Body)
		return string(body), err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

func listGitHubDir(owner, repo, path string) ([]githubEntry, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d listing %s", resp.StatusCode, url)
	}

	var entries []githubEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}
