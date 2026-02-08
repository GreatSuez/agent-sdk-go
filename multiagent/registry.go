package multiagent

import (
	"sync"
)

// Registry provides agent discovery and capability matching.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]AgentInfo
}

// AgentInfo describes an agent's capabilities.
type AgentInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Role         AgentRole `json:"role"`
	Capabilities []string  `json:"capabilities"`
	Status       string    `json:"status"`
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]AgentInfo),
	}
}

// Register adds an agent to the registry.
func (r *Registry) Register(info AgentInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info.Status == "" {
		info.Status = "available"
	}
	r.agents[info.ID] = info
}

// Unregister removes an agent from the registry.
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, id)
}

// Get returns an agent by ID.
func (r *Registry) Get(id string) (AgentInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.agents[id]
	return info, ok
}

// List returns all registered agents.
func (r *Registry) List() []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]AgentInfo, 0, len(r.agents))
	for _, info := range r.agents {
		agents = append(agents, info)
	}
	return agents
}

// FindByRole returns agents with a specific role.
func (r *Registry) FindByRole(role AgentRole) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var results []AgentInfo
	for _, info := range r.agents {
		if info.Role == role {
			results = append(results, info)
		}
	}
	return results
}

// FindByCapability returns agents with a specific capability.
func (r *Registry) FindByCapability(capability string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var results []AgentInfo
	for _, info := range r.agents {
		for _, cap := range info.Capabilities {
			if cap == capability {
				results = append(results, info)
				break
			}
		}
	}
	return results
}

// UpdateStatus updates an agent's status.
func (r *Registry) UpdateStatus(id, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.agents[id]; ok {
		info.Status = status
		r.agents[id] = info
	}
}

// FindAvailable returns all available agents.
func (r *Registry) FindAvailable() []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var results []AgentInfo
	for _, info := range r.agents {
		if info.Status == "available" {
			results = append(results, info)
		}
	}
	return results
}
