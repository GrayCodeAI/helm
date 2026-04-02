// Package trash provides soft delete and trash management.
package trash

import (
	"sync"
	"time"
)

// Item represents a trashed item
type Item struct {
	ID        string
	Type      string // session, memory, prompt
	DeletedAt time.Time
	Data      interface{}
}

// Manager manages the trash bin
type Manager struct {
	mu    sync.RWMutex
	items map[string]*Item
}

// NewManager creates a new trash manager
func NewManager() *Manager {
	return &Manager{
		items: make(map[string]*Item),
	}
}

// MoveToTrash moves an item to trash
func (m *Manager) MoveToTrash(id, itemType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[id] = &Item{
		ID:        id,
		Type:      itemType,
		DeletedAt: time.Now(),
		Data:      data,
	}
}

// Restore restores an item from trash
func (m *Manager) Restore(id string) (string, interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[id]
	if !ok {
		return "", nil, false
	}

	itemType := item.Type
	data := item.Data
	delete(m.items, id)
	return itemType, data, true
}

// DeletePermanently permanently deletes an item
func (m *Manager) DeletePermanently(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.items[id]
	if !ok {
		return false
	}

	delete(m.items, id)
	return true
}

// EmptyTrash permanently deletes all items
func (m *Manager) EmptyTrash() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := len(m.items)
	m.items = make(map[string]*Item)
	return count
}

// List lists all trashed items
func (m *Manager) List() []*Item {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]*Item, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, item)
	}
	return items
}

// Count returns the number of trashed items
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

// CleanupOld removes items older than maxAge
func (m *Manager) CleanupOld(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	count := 0

	for id, item := range m.items {
		if item.DeletedAt.Before(cutoff) {
			delete(m.items, id)
			count++
		}
	}

	return count
}
