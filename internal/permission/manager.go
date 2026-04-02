// Package permission provides tool execution permission management.
package permission

import (
	"sync"
)

// Level represents permission level
type Level string

const (
	LevelAsk       Level = "ask"
	LevelAllow     Level = "allow"
	LevelDeny      Level = "deny"
	LevelAutoAllow Level = "auto_allow"
)

// Request represents a permission request
type Request struct {
	Tool      string
	Args      map[string]string
	SessionID string
}

// Response represents a permission response
type Response struct {
	Allowed bool
	Level   Level
}

// Manager manages tool permissions
type Manager struct {
	mu              sync.RWMutex
	toolPermissions map[string]Level
	sessionAuto     map[string]bool
}

// NewManager creates a new permission manager
func NewManager() *Manager {
	return &Manager{
		toolPermissions: make(map[string]Level),
		sessionAuto:     make(map[string]bool),
	}
}

// SetToolPermission sets permission for a tool
func (m *Manager) SetToolPermission(tool string, level Level) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolPermissions[tool] = level
}

// SetSessionAutoAllow enables auto-allow for a session
func (m *Manager) SetSessionAutoAllow(sessionID string, allow bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionAuto[sessionID] = allow
}

// Check checks if a tool execution is allowed
func (m *Manager) Check(req Request) Response {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check session auto-allow
	if m.sessionAuto[req.SessionID] {
		return Response{Allowed: true, Level: LevelAutoAllow}
	}

	// Check tool-specific permission
	level, ok := m.toolPermissions[req.Tool]
	if !ok {
		return Response{Allowed: false, Level: LevelAsk}
	}

	switch level {
	case LevelAllow, LevelAutoAllow:
		return Response{Allowed: true, Level: level}
	case LevelDeny:
		return Response{Allowed: false, Level: level}
	default:
		return Response{Allowed: false, Level: LevelAsk}
	}
}

// ListPermissions lists all tool permissions
func (m *Manager) ListPermissions() map[string]Level {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]Level)
	for k, v := range m.toolPermissions {
		result[k] = v
	}
	return result
}
