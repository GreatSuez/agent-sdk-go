package skill

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const skillFileName = "SKILL.md"

// DefaultSearchPaths returns the default directories to scan for skills.
func DefaultSearchPaths() []string {
	paths := []string{
		"./skills",
		"./.github/skills",
		"./.agents/skills",
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".agent", "skills"))
	}
	return paths
}

// LoadFromDir scans a directory for skill folders (each containing SKILL.md)
// and registers them. Returns the number of skills loaded.
func LoadFromDir(dir string) (int, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // directory doesn't exist, skip silently
		}
		return 0, fmt.Errorf("failed to stat skills directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%q is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read skills directory %q: %w", dir, err)
	}

	loaded := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			// Check if it's a SKILL.md file directly in the dir
			if entry.Name() == skillFileName {
				if err := loadSkillFile(filepath.Join(dir, skillFileName)); err != nil {
					log.Printf("⚠️  Failed to load skill from %s: %v", dir, err)
				} else {
					loaded++
				}
			}
			continue
		}

		skillPath := filepath.Join(dir, entry.Name(), skillFileName)
		if _, err := os.Stat(skillPath); err != nil {
			// Also check subdirectories (e.g., .curated/skill-name/, .experimental/)
			if entry.Name() == ".curated" || entry.Name() == ".experimental" || entry.Name() == ".system" {
				subLoaded, subErr := LoadFromDir(filepath.Join(dir, entry.Name()))
				if subErr != nil {
					log.Printf("⚠️  Failed to scan %s: %v", filepath.Join(dir, entry.Name()), subErr)
				}
				loaded += subLoaded
			}
			continue
		}

		if err := loadSkillFile(skillPath); err != nil {
			log.Printf("⚠️  Failed to load skill %q: %v", entry.Name(), err)
			continue
		}
		loaded++
	}

	return loaded, nil
}

// LoadFromPaths scans multiple directories for skills.
func LoadFromPaths(paths []string) int {
	total := 0
	for _, p := range paths {
		n, err := LoadFromDir(p)
		if err != nil {
			log.Printf("⚠️  Error scanning skills directory %q: %v", p, err)
			continue
		}
		total += n
	}
	return total
}

// ScanDefaults scans all default search paths for skills.
func ScanDefaults() int {
	return LoadFromPaths(DefaultSearchPaths())
}

func loadSkillFile(path string) error {
	s, err := ParseFile(path)
	if err != nil {
		return err
	}
	// Skip if already registered (first loaded wins)
	if _, exists := Get(s.Name); exists {
		return nil
	}
	return Register(s)
}
