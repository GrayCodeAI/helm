// Package tmux provides tmux session management for agent sessions.
package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager manages tmux sessions
type Manager struct {
	prefix string
}

// NewManager creates a new tmux manager
func NewManager(prefix string) *Manager {
	if prefix == "" {
		prefix = "helm"
	}
	return &Manager{prefix: prefix}
}

// SessionName returns the tmux session name for a given session ID
func (m *Manager) SessionName(sessionID string) string {
	return fmt.Sprintf("%s-%s", m.prefix, sessionID)
}

// CreateSession creates a new tmux session
func (m *Manager) CreateSession(sessionID, command string, args []string) error {
	name := m.SessionName(sessionID)

	// Check if session already exists
	exists, _ := m.SessionExists(name)
	if exists {
		return fmt.Errorf("session %s already exists", name)
	}

	// Build command
	cmdArgs := []string{"new-session", "-d", "-s", name}
	cmdArgs = append(cmdArgs, command)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("tmux", cmdArgs...)
	return cmd.Run()
}

// KillSession kills a tmux session
func (m *Manager) KillSession(sessionID string) error {
	name := m.SessionName(sessionID)
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// SessionExists checks if a session exists
func (m *Manager) SessionExists(name string) (bool, error) {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

// ListSessions lists all helm tmux sessions
func (m *Manager) ListSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(line, m.prefix+"-") {
			sessions = append(sessions, line)
		}
	}

	return sessions, nil
}

// CapturePane captures the contents of a pane
func (m *Manager) CapturePane(sessionID string) (string, error) {
	name := m.SessionName(sessionID)
	cmd := exec.Command("tmux", "capture-pane", "-p", "-t", name)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// SendKeys sends keys to a session
func (m *Manager) SendKeys(sessionID, keys string) error {
	name := m.SessionName(sessionID)
	cmd := exec.Command("tmux", "send-keys", "-t", name, keys)
	return cmd.Run()
}

// ResizeSession resizes a session
func (m *Manager) ResizeSession(sessionID string, rows, cols int) error {
	name := m.SessionName(sessionID)
	cmd := exec.Command("tmux", "resize-window", "-t", name, "-x", fmt.Sprintf("%d", cols), "-y", fmt.Sprintf("%d", rows))
	return cmd.Run()
}
