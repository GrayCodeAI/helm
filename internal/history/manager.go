// Package history provides prompt history management.
package history

import (
	"sync"
	"time"
)

// Entry represents a prompt history entry
type Entry struct {
	ID        string
	Prompt    string
	Timestamp time.Time
	SessionID string
	Provider  string
	Model     string
}

// Manager manages prompt history
type Manager struct {
	mu      sync.RWMutex
	entries []Entry
	maxSize int
}

// NewManager creates a new history manager
func NewManager(maxSize int) *Manager {
	if maxSize == 0 {
		maxSize = 1000
	}
	return &Manager{
		entries: make([]Entry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a prompt to history
func (m *Manager) Add(prompt, sessionID, provider, model string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := time.Now().Format("20060102150405.000")
	entry := Entry{
		ID:        id,
		Prompt:    prompt,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Provider:  provider,
		Model:     model,
	}

	m.entries = append(m.entries, entry)

	// Trim if exceeds max
	if len(m.entries) > m.maxSize {
		m.entries = m.entries[len(m.entries)-m.maxSize:]
	}

	return id
}

// Get gets a history entry by ID
func (m *Manager) Get(id string) (*Entry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, e := range m.entries {
		if e.ID == id {
			return &e, true
		}
	}
	return nil, false
}

// List lists all history entries
func (m *Manager) List(limit int) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.entries) {
		limit = len(m.entries)
	}

	// Return most recent first
	result := make([]Entry, limit)
	for i := 0; i < limit; i++ {
		result[i] = m.entries[len(m.entries)-1-i]
	}
	return result
}

// Search searches history by prompt content
func (m *Manager) Search(query string) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []Entry
	for _, e := range m.entries {
		if contains(e.Prompt, query) {
			results = append(results, e)
		}
	}
	return results
}

// Clear clears all history
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = nil
}

// Count returns the number of history entries
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s[1:], substr)) || (len(s) >= len(substr) && s[:len(substr)] == substr)
}
