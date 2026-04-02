// Package groups provides session grouping and organization.
package groups

import (
	"path/filepath"
	"strings"
	"sync"
)

// Group represents a session group
type Group struct {
	ID       string
	Name     string
	Path     string
	Sessions []string
	Expanded bool
	Order    int
}

// Manager manages session groups
type Manager struct {
	mu     sync.RWMutex
	groups []*Group
}

// NewManager creates a new group manager
func NewManager() *Manager {
	return &Manager{
		groups: make([]*Group, 0),
	}
}

// AutoGroupByProject auto-groups sessions by project directory
func (m *Manager) AutoGroupByProject(sessions map[string]string) []*Group {
	// sessions: sessionID -> projectPath
	projectSessions := make(map[string][]string)

	for sessionID, projectPath := range sessions {
		// Use directory name as group name
		groupName := filepath.Base(projectPath)
		if groupName == "" || groupName == "/" {
			groupName = "root"
		}
		projectSessions[groupName] = append(projectSessions[groupName], sessionID)
	}

	m.mu.Lock()
	m.groups = make([]*Group, 0, len(projectSessions))

	i := 0
	for name, sessionIDs := range projectSessions {
		m.groups = append(m.groups, &Group{
			ID:       "group-" + name,
			Name:     name,
			Path:     name,
			Sessions: sessionIDs,
			Expanded: true,
			Order:    i,
		})
		i++
	}
	m.mu.Unlock()

	return m.groups
}

// CreateGroup creates a new group
func (m *Manager) CreateGroup(name string) *Group {
	m.mu.Lock()
	defer m.mu.Unlock()

	group := &Group{
		ID:       "group-" + strings.ToLower(name),
		Name:     name,
		Expanded: true,
		Order:    len(m.groups),
	}
	m.groups = append(m.groups, group)
	return group
}

// AddSession adds a session to a group
func (m *Manager) AddSession(groupID, sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, g := range m.groups {
		if g.ID == groupID {
			g.Sessions = append(g.Sessions, sessionID)
			return true
		}
	}
	return false
}

// RemoveSession removes a session from a group
func (m *Manager) RemoveSession(groupID, sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, g := range m.groups {
		if g.ID == groupID {
			for i, s := range g.Sessions {
				if s == sessionID {
					g.Sessions = append(g.Sessions[:i], g.Sessions[i+1:]...)
					return true
				}
			}
		}
	}
	return false
}

// ToggleExpand toggles group expansion
func (m *Manager) ToggleExpand(groupID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, g := range m.groups {
		if g.ID == groupID {
			g.Expanded = !g.Expanded
			return
		}
	}
}

// ListGroups lists all groups
func (m *Manager) ListGroups() []*Group {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.groups
}

// GetGroup gets a group by ID
func (m *Manager) GetGroup(id string) (*Group, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, g := range m.groups {
		if g.ID == id {
			return g, true
		}
	}
	return nil, false
}
