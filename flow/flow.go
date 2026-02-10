// Package flow provides a Genkit-style flow registry. A flow is a named,
// reusable agent configuration (tools, system prompt, workflow) that can be
// discovered and tested from the DevUI.
package flow

import (
	"fmt"
	"sort"
	"sync"
)

// Definition describes a named agent flow that can be executed from the DevUI.
type Definition struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Workflow     string         `json:"workflow,omitempty"`
	Tools        []string       `json:"tools,omitempty"`
	Skills       []string       `json:"skills,omitempty"`
	SystemPrompt string         `json:"systemPrompt,omitempty"`
	InputExample string         `json:"inputExample,omitempty"`
	InputSchema  map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

var (
	mu    sync.RWMutex
	flows = map[string]*Definition{}
)

// Register adds a flow definition to the global registry.
func Register(f *Definition) error {
	if f == nil {
		return fmt.Errorf("flow definition is nil")
	}
	if f.Name == "" {
		return fmt.Errorf("flow name is required")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := flows[f.Name]; exists {
		return fmt.Errorf("flow %q already registered", f.Name)
	}
	flows[f.Name] = f
	return nil
}

// MustRegister registers a flow and panics on error.
func MustRegister(f *Definition) {
	if err := Register(f); err != nil {
		panic(err)
	}
}

// Upsert registers a flow definition or replaces an existing one with the same name.
func Upsert(f *Definition) error {
	if f == nil {
		return fmt.Errorf("flow definition is nil")
	}
	if f.Name == "" {
		return fmt.Errorf("flow name is required")
	}
	mu.Lock()
	defer mu.Unlock()
	flows[f.Name] = f
	return nil
}

// MustUpsert upserts a flow and panics on error.
func MustUpsert(f *Definition) {
	if err := Upsert(f); err != nil {
		panic(err)
	}
}

// Get returns a flow definition by name.
func Get(name string) (*Definition, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := flows[name]
	return f, ok
}

// Names returns all registered flow names sorted alphabetically.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(flows))
	for name := range flows {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// All returns all registered flow definitions sorted by name.
func All() []*Definition {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]*Definition, 0, len(flows))
	for _, f := range flows {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Delete removes a flow by name.
func Delete(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := flows[name]; !ok {
		return false
	}
	delete(flows, name)
	return true
}

// Reset clears the registry. Intended for tests only.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	flows = map[string]*Definition{}
}
