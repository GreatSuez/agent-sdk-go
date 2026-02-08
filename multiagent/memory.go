package multiagent

import (
	"sync"
	"time"
)

// SharedMemory provides a shared key-value store for agent collaboration.
type SharedMemory struct {
	mu      sync.RWMutex
	entries map[string]*MemoryEntry
}

// MemoryEntry represents a value in shared memory.
type MemoryEntry struct {
	Key       string         `json:"key"`
	Value     any            `json:"value"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedBy string         `json:"createdBy"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedBy string         `json:"updatedBy,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt"`
	TTL       time.Duration  `json:"ttl,omitempty"`
}

// NewSharedMemory creates a new shared memory instance.
func NewSharedMemory() *SharedMemory {
	return &SharedMemory{
		entries: make(map[string]*MemoryEntry),
	}
}

// Set stores a value in shared memory.
func (m *SharedMemory) Set(key string, value any, agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if existing, ok := m.entries[key]; ok {
		existing.Value = value
		existing.UpdatedBy = agentID
		existing.UpdatedAt = now
	} else {
		m.entries[key] = &MemoryEntry{
			Key:       key,
			Value:     value,
			CreatedBy: agentID,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
}

// SetWithTTL stores a value with a time-to-live.
func (m *SharedMemory) SetWithTTL(key string, value any, agentID string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	m.entries[key] = &MemoryEntry{
		Key:       key,
		Value:     value,
		CreatedBy: agentID,
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       ttl,
	}
}

// Get retrieves a value from shared memory.
func (m *SharedMemory) Get(key string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[key]
	if !ok {
		return nil, false
	}

	// Check TTL
	if entry.TTL > 0 {
		if time.Since(entry.CreatedAt) > entry.TTL {
			return nil, false
		}
	}

	return entry.Value, true
}

// GetEntry retrieves the full entry from shared memory.
func (m *SharedMemory) GetEntry(key string) (*MemoryEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[key]
	if !ok {
		return nil, false
	}

	// Check TTL
	if entry.TTL > 0 {
		if time.Since(entry.CreatedAt) > entry.TTL {
			return nil, false
		}
	}

	// Return a copy
	copy := *entry
	return &copy, true
}

// Delete removes a value from shared memory.
func (m *SharedMemory) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
}

// Keys returns all keys in shared memory.
func (m *SharedMemory) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.entries))
	for k, entry := range m.entries {
		// Skip expired entries
		if entry.TTL > 0 && time.Since(entry.CreatedAt) > entry.TTL {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}

// All returns all entries in shared memory.
func (m *SharedMemory) All() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]any)
	for k, entry := range m.entries {
		// Skip expired entries
		if entry.TTL > 0 && time.Since(entry.CreatedAt) > entry.TTL {
			continue
		}
		result[k] = entry.Value
	}
	return result
}

// Clear removes all entries from shared memory.
func (m *SharedMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]*MemoryEntry)
}

// CleanupExpired removes all expired entries.
func (m *SharedMemory) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for k, entry := range m.entries {
		if entry.TTL > 0 && time.Since(entry.CreatedAt) > entry.TTL {
			delete(m.entries, k)
			count++
		}
	}
	return count
}

// SetMetadata adds metadata to an entry.
func (m *SharedMemory) SetMetadata(key string, metadata map[string]any) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[key]
	if !ok {
		return false
	}

	if entry.Metadata == nil {
		entry.Metadata = make(map[string]any)
	}
	for k, v := range metadata {
		entry.Metadata[k] = v
	}
	entry.UpdatedAt = time.Now().UTC()
	return true
}

// GetByCreator returns all entries created by a specific agent.
func (m *SharedMemory) GetByCreator(agentID string) map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]any)
	for k, entry := range m.entries {
		if entry.CreatedBy == agentID {
			// Skip expired entries
			if entry.TTL > 0 && time.Since(entry.CreatedAt) > entry.TTL {
				continue
			}
			result[k] = entry.Value
		}
	}
	return result
}

// Size returns the number of entries in shared memory.
func (m *SharedMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}
