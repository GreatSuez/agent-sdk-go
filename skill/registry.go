package skill

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu     sync.RWMutex
	skills = map[string]*Skill{}
)

// Register adds a skill to the global registry.
func Register(s *Skill) error {
	if s == nil {
		return fmt.Errorf("skill is nil")
	}
	if s.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := skills[s.Name]; exists {
		return fmt.Errorf("skill %q already registered", s.Name)
	}
	skills[s.Name] = s
	return nil
}

// MustRegister registers a skill or panics.
func MustRegister(s *Skill) {
	if err := Register(s); err != nil {
		panic(err)
	}
}

// Get returns a skill by name.
func Get(name string) (*Skill, bool) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := skills[name]
	return s, ok
}

// Names returns sorted skill names.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(skills))
	for name := range skills {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// All returns all registered skills sorted by name.
func All() []*Skill {
	names := Names()
	mu.RLock()
	defer mu.RUnlock()
	out := make([]*Skill, 0, len(names))
	for _, name := range names {
		out = append(out, skills[name])
	}
	return out
}

// Remove removes a skill by name. Returns true if it existed.
func Remove(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := skills[name]; ok {
		delete(skills, name)
		return true
	}
	return false
}

// Count returns the number of registered skills.
func Count() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(skills)
}

// Reset clears all registered skills (for testing).
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	skills = map[string]*Skill{}
}
