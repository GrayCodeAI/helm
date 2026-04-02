// Package heartbeat provides session heartbeat monitoring.
package heartbeat

import (
	"context"
	"sync"
	"time"
)

// Session represents a monitored session
type Session struct {
	ID            string
	LastHeartbeat time.Time
	Status        string // alive, dead, unknown
}

// Monitor monitors session heartbeats
type Monitor struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	interval time.Duration
	onDead   func(sessionID string)
}

// NewMonitor creates a new heartbeat monitor
func NewMonitor(interval time.Duration, onDead func(string)) *Monitor {
	return &Monitor{
		sessions: make(map[string]*Session),
		interval: interval,
		onDead:   onDead,
	}
}

// Register registers a session for monitoring
func (m *Monitor) Register(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions[sessionID] = &Session{
		ID:            sessionID,
		LastHeartbeat: time.Now(),
		Status:        "alive",
	}
}

// Unregister unregisters a session
func (m *Monitor) Unregister(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// Beat records a heartbeat
func (m *Monitor) Beat(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[sessionID]; ok {
		sess.LastHeartbeat = time.Now()
		sess.Status = "alive"
	}
}

// Start starts the monitoring loop
func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkSessions()
		}
	}
}

func (m *Monitor) checkSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	deadThreshold := m.interval * 3

	for id, sess := range m.sessions {
		if time.Since(sess.LastHeartbeat) > deadThreshold && sess.Status == "alive" {
			sess.Status = "dead"
			if m.onDead != nil {
				m.onDead(id)
			}
		}
	}
}

// GetStatus gets a session's status
func (m *Monitor) GetStatus(sessionID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return "unknown", false
	}
	return sess.Status, true
}

// ListSessions lists all monitored sessions
func (m *Monitor) ListSessions() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for id, sess := range m.sessions {
		result[id] = sess.Status
	}
	return result
}
