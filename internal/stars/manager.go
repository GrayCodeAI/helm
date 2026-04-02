// Package stars provides session starring and pinning.
package stars

import (
	"sync"
	"time"
)

// Star represents a starred session
type Star struct {
	SessionID string
	StarredAt time.Time
	Note      string
}

// Pin represents a pinned session
type Pin struct {
	SessionID string
	PinnedAt  time.Time
	Note      string
	Position  int
}

// Manager manages stars and pins
type Manager struct {
	mu    sync.RWMutex
	stars map[string]*Star
	pins  map[string]*Pin
}

// NewManager creates a new stars manager
func NewManager() *Manager {
	return &Manager{
		stars: make(map[string]*Star),
		pins:  make(map[string]*Pin),
	}
}

// Star stars a session
func (m *Manager) Star(sessionID, note string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stars[sessionID] = &Star{
		SessionID: sessionID,
		StarredAt: time.Now(),
		Note:      note,
	}
}

// Unstar unstars a session
func (m *Manager) Unstar(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.stars, sessionID)
}

// IsStarred checks if a session is starred
func (m *Manager) IsStarred(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.stars[sessionID]
	return ok
}

// ListStars lists all starred sessions
func (m *Manager) ListStars() []*Star {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stars := make([]*Star, 0, len(m.stars))
	for _, s := range m.stars {
		stars = append(stars, s)
	}
	return stars
}

// Pin pins a session
func (m *Manager) Pin(sessionID, note string, position int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pins[sessionID] = &Pin{
		SessionID: sessionID,
		PinnedAt:  time.Now(),
		Note:      note,
		Position:  position,
	}
}

// Unpin unpins a session
func (m *Manager) Unpin(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pins, sessionID)
}

// IsPinned checks if a session is pinned
func (m *Manager) IsPinned(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.pins[sessionID]
	return ok
}

// ListPins lists all pinned sessions
func (m *Manager) ListPins() []*Pin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pins := make([]*Pin, 0, len(m.pins))
	for _, p := range m.pins {
		pins = append(pins, p)
	}
	return pins
}
