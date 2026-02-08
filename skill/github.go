package skill

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	parts := strings.SplitN(repoRef, "/", 3)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid repo reference %q — expected owner/repo or owner/repo/path", repoRef)
	}

	owner, repo := parts[0], parts[1]
	skillPath := ""
	if len(parts) == 3 {
		skillPath = parts[2]
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// If a specific skill path is given, install just that one
	if skillPath != "" {
		return installSingleSkill(owner, repo, skillPath, destDir)
	}

	// Otherwise, list the skills directory and install all
	return installAllSkills(owner, repo, destDir)
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
	// List contents of the "skills" directory in the repo
	searchPaths := []string{"skills", "skills/.curated", "skills/.experimental"}
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
		return 0, fmt.Errorf("no skills found in %s/%s", owner, repo)
	}
	return installed, nil
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
