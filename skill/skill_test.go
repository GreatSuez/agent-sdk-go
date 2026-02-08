package skill

import (
	"os"
	"path/filepath"
	"testing"
)

const testSkillMD = `---
name: test-skill
description: A test skill for unit testing
license: MIT
allowed-tools:
  - curl
  - jq
metadata:
  author: test@example.com
  version: "1.0"
---
# Test Skill

Follow these instructions during testing.

## Steps
1. Do step one
2. Do step two
`

func TestParse(t *testing.T) {
	s, err := Parse(testSkillMD)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if s.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "test-skill")
	}
	if s.Description != "A test skill for unit testing" {
		t.Errorf("Description = %q, want %q", s.Description, "A test skill for unit testing")
	}
	if s.License != "MIT" {
		t.Errorf("License = %q, want %q", s.License, "MIT")
	}
	if len(s.AllowedTools) != 2 || s.AllowedTools[0] != "curl" || s.AllowedTools[1] != "jq" {
		t.Errorf("AllowedTools = %v, want [curl jq]", s.AllowedTools)
	}
	if s.Metadata["author"] != "test@example.com" {
		t.Errorf("Metadata[author] = %q, want %q", s.Metadata["author"], "test@example.com")
	}
	if s.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %q, want %q", s.Metadata["version"], "1.0")
	}
	if s.Instructions == "" {
		t.Error("Instructions is empty")
	}
}

func TestParse_MissingName(t *testing.T) {
	_, err := Parse("---\ndescription: no name\n---\nbody")
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestParse_MissingDescription(t *testing.T) {
	_, err := Parse("---\nname: x\n---\nbody")
	if err == nil {
		t.Error("expected error for missing description")
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	_, err := Parse("just some text")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(testSkillMD), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := ParseFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if s.Name != "test-skill" {
		t.Errorf("Name = %q", s.Name)
	}
	if s.Source != "local" {
		t.Errorf("Source = %q, want local", s.Source)
	}
	if s.Path != skillDir {
		t.Errorf("Path = %q, want %q", s.Path, skillDir)
	}
}

func TestRegistry(t *testing.T) {
	Reset()
	defer Reset()

	s := &Skill{Name: "reg-test", Description: "test"}
	if err := Register(s); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Duplicate
	if err := Register(s); err == nil {
		t.Error("expected duplicate error")
	}

	// Get
	got, ok := Get("reg-test")
	if !ok || got.Name != "reg-test" {
		t.Error("Get failed")
	}
	_, ok = Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent")
	}

	// Names
	names := Names()
	if len(names) != 1 || names[0] != "reg-test" {
		t.Errorf("Names = %v", names)
	}

	// All
	all := All()
	if len(all) != 1 {
		t.Errorf("All len = %d", len(all))
	}

	// Count
	if Count() != 1 {
		t.Errorf("Count = %d", Count())
	}

	// Remove
	if !Remove("reg-test") {
		t.Error("Remove returned false")
	}
	if Remove("reg-test") {
		t.Error("second Remove should return false")
	}
	if Count() != 0 {
		t.Error("Count should be 0 after Remove")
	}
}

func TestRegistryNil(t *testing.T) {
	if err := Register(nil); err == nil {
		t.Error("expected error for nil skill")
	}
}

func TestRegistryEmptyName(t *testing.T) {
	if err := Register(&Skill{}); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestLoadFromDir(t *testing.T) {
	Reset()
	defer Reset()

	dir := t.TempDir()
	// Create two skill subdirectories
	for _, name := range []string{"skill-a", "skill-b"} {
		skillDir := filepath.Join(dir, name)
		os.MkdirAll(skillDir, 0755)
		content := "---\nname: " + name + "\ndescription: Test " + name + "\n---\nInstructions for " + name
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
	}

	n, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}
	if n != 2 {
		t.Errorf("loaded %d, want 2", n)
	}
	if Count() != 2 {
		t.Errorf("Count = %d, want 2", Count())
	}
}

func TestLoadFromDir_Nonexistent(t *testing.T) {
	n, err := LoadFromDir("/nonexistent/path/12345")
	if err != nil {
		t.Fatalf("should not error for nonexistent dir: %v", err)
	}
	if n != 0 {
		t.Errorf("loaded %d, want 0", n)
	}
}

func TestBuiltins(t *testing.T) {
	Reset()
	defer Reset()

	RegisterBuiltins()
	if Count() < 4 {
		t.Errorf("expected at least 4 built-in skills, got %d", Count())
	}

	// Verify a known built-in
	s, ok := Get("k8s-debug")
	if !ok {
		t.Fatal("k8s-debug built-in not found")
	}
	if s.Source != "builtin" {
		t.Errorf("Source = %q, want builtin", s.Source)
	}
	if len(s.AllowedTools) == 0 {
		t.Error("k8s-debug should have allowed tools")
	}
}

func TestLearnedPattern(t *testing.T) {
	Reset()
	defer Reset()

	dir := t.TempDir()
	patterns := []LearnedPattern{
		{Pattern: "Always check error return values", Source: "code-review"},
		{Pattern: "Use context.WithTimeout for HTTP calls", Source: "incident"},
	}

	s, err := CreateSkillFromPatterns("my-learned", "Learned behaviors", patterns, dir)
	if err != nil {
		t.Fatalf("CreateSkillFromPatterns failed: %v", err)
	}
	if s.Name != "my-learned" {
		t.Errorf("Name = %q", s.Name)
	}
	if s.Source != "learned" {
		t.Errorf("Source = %q, want learned", s.Source)
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(dir, "my-learned", "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not created: %v", err)
	}
}

func TestLearnedPattern_Errors(t *testing.T) {
	dir := t.TempDir()
	if _, err := CreateSkillFromPatterns("", "desc", []LearnedPattern{{Pattern: "x"}}, dir); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := CreateSkillFromPatterns("x", "desc", nil, dir); err == nil {
		t.Error("expected error for empty patterns")
	}
}

func TestMustRegister_Panics(t *testing.T) {
	Reset()
	defer Reset()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister should panic on duplicate")
		}
	}()

	s := &Skill{Name: "panic-test", Description: "test"}
	MustRegister(s)
	MustRegister(s) // should panic
}
