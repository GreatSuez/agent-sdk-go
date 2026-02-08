// Package skill implements the Agent Skills open standard (SKILL.md).
// Skills are composable prompt+tool bundles that can be loaded from local
// directories, GitHub repositories, or embedded as built-ins.
//
// Format: each skill is a folder containing a SKILL.md file with YAML
// frontmatter (name, description, allowed-tools, etc.) followed by
// markdown instructions injected into the agent's system prompt.
//
// Compatible with: OpenAI Codex skills, GitHub Copilot agent skills,
// Claude Code skills.
package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill represents a parsed Agent Skills open standard skill.
type Skill struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	License      string            `json:"license,omitempty"`
	AllowedTools []string          `json:"allowedTools,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Instructions string            `json:"instructions"`
	Path         string            `json:"path,omitempty"`
	Source       string            `json:"source,omitempty"` // "builtin", "local", "github:<owner>/<repo>"
}

// ParseFile parses a SKILL.md file into a Skill.
func ParseFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file %q: %w", path, err)
	}
	s, err := Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill file %q: %w", path, err)
	}
	s.Path = filepath.Dir(path)
	s.Source = "local"
	return s, nil
}

// Parse parses SKILL.md content (YAML frontmatter + markdown body).
func Parse(content string) (*Skill, error) {
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	s := &Skill{
		Metadata:     make(map[string]string),
		Instructions: strings.TrimSpace(body),
	}

	if err := parseFrontmatter(frontmatter, s); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}

	if s.Name == "" {
		return nil, fmt.Errorf("skill name is required in frontmatter")
	}
	if s.Description == "" {
		return nil, fmt.Errorf("skill description is required in frontmatter")
	}

	return s, nil
}

// splitFrontmatter splits YAML frontmatter (between --- delimiters) from the markdown body.
func splitFrontmatter(content string) (frontmatter string, body string, err error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("SKILL.md must start with --- (YAML frontmatter)")
	}

	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", fmt.Errorf("SKILL.md missing closing --- for frontmatter")
	}

	frontmatter = strings.TrimSpace(rest[:idx])
	body = strings.TrimSpace(rest[idx+4:]) // skip \n---
	return frontmatter, body, nil
}

// parseFrontmatter parses simple YAML key-value pairs and lists from frontmatter.
// Supports: scalar values, simple lists (- item), and nested maps (one level for metadata).
func parseFrontmatter(fm string, s *Skill) error {
	scanner := bufio.NewScanner(strings.NewReader(fm))

	var currentKey string
	var inList bool
	var listItems []string
	var inMetadata bool
	metadataMap := make(map[string]string)

	flushList := func() {
		if currentKey == "allowed-tools" {
			s.AllowedTools = listItems
		}
		listItems = nil
		inList = false
	}

	flushMetadata := func() {
		if inMetadata {
			s.Metadata = metadataMap
			metadataMap = make(map[string]string)
			inMetadata = false
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		// List item
		if inList && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
			item = strings.Trim(item, `"'`)
			listItems = append(listItems, item)
			continue
		} else if inList {
			flushList()
		}

		// Metadata nested map item (2-space or tab indented key: value)
		if inMetadata && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
			trimmed := strings.TrimSpace(line)
			if colonIdx := strings.Index(trimmed, ":"); colonIdx > 0 {
				k := strings.TrimSpace(trimmed[:colonIdx])
				v := strings.TrimSpace(trimmed[colonIdx+1:])
				v = strings.Trim(v, `"'`)
				metadataMap[k] = v
			}
			continue
		} else if inMetadata {
			flushMetadata()
		}

		// Top-level key: value
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		value = strings.Trim(value, `"'`)
		currentKey = key

		switch key {
		case "name":
			s.Name = value
		case "description":
			s.Description = value
		case "license":
			s.License = value
		case "allowed-tools":
			if value == "" {
				inList = true
				listItems = nil
			}
		case "metadata":
			if value == "" {
				inMetadata = true
			}
		default:
			// Store unknown top-level keys in metadata
			if value != "" {
				s.Metadata[key] = value
			}
		}
	}

	// Flush any pending state
	if inList {
		flushList()
	}
	if inMetadata {
		flushMetadata()
	}

	return scanner.Err()
}
