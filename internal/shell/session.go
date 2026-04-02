// Package shell provides PTY-based session management for running agent CLIs.
package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// Session represents a running agent session in a PTY
type Session struct {
	ID        string
	Command   string
	Args      []string
	Dir       string
	PTY       *os.File
	Cmd       *exec.Cmd
	Status    string // running, stopped, exited
	ExitCode  int
	StartedAt time.Time
	ExitedAt  *time.Time
	mu        sync.RWMutex
}

// Config configures a shell session
type Config struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
	Rows    uint16
	Cols    uint16
}

// NewSession creates a new PTY session
func NewSession(cfg Config) *Session {
	return &Session{
		ID:      fmt.Sprintf("shell-%d", time.Now().UnixNano()),
		Command: cfg.Command,
		Args:    cfg.Args,
		Dir:     cfg.Dir,
		Status:  "stopped",
	}
}

// Start starts the session in a PTY
func (s *Session) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.CommandContext(ctx, s.Command, s.Args...)
	cmd.Dir = s.Dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start with PTY
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start pty: %w", err)
	}

	s.PTY = ptyFile
	s.Cmd = cmd
	s.Status = "running"
	s.StartedAt = time.Now()

	return nil
}

// Write writes input to the PTY
func (s *Session) Write(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.PTY == nil {
		return 0, fmt.Errorf("session not started")
	}

	return s.PTY.Write(data)
}

// Read reads output from the PTY
func (s *Session) Read(buf []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.PTY == nil {
		return 0, fmt.Errorf("session not started")
	}

	return s.PTY.Read(buf)
}

// Resize resizes the PTY
func (s *Session) Resize(rows, cols uint16) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.PTY == nil {
		return fmt.Errorf("session not started")
	}

	return pty.Setsize(s.PTY, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Close closes the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.PTY != nil {
		s.PTY.Close()
		s.PTY = nil
	}

	if s.Cmd != nil && s.Cmd.Process != nil {
		s.Cmd.Process.Signal(syscall.SIGTERM)
		s.Cmd.Wait()
	}

	s.Status = "exited"
	now := time.Now()
	s.ExitedAt = &now

	return nil
}

// Wait waits for the session to exit
func (s *Session) Wait() error {
	s.mu.RLock()
	cmd := s.Cmd
	s.mu.RUnlock()

	if cmd == nil {
		return fmt.Errorf("session not started")
	}

	err := cmd.Wait()
	now := time.Now()
	s.mu.Lock()
	s.ExitedAt = &now
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			s.ExitCode = exitErr.ExitCode()
			s.Status = "exited"
		}
	} else {
		s.ExitCode = 0
		s.Status = "exited"
	}
	s.mu.Unlock()

	return err
}

// IsRunning checks if session is running
func (s *Session) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == "running"
}

// OutputReader returns a reader for session output
func (s *Session) OutputReader() io.Reader {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PTY
}

// Manager manages multiple shell sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates and starts a new session
func (m *Manager) CreateSession(ctx context.Context, cfg Config) (*Session, error) {
	sess := NewSession(cfg)
	if err := sess.Start(ctx); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[sess.ID] = sess
	m.mu.Unlock()

	return sess, nil
}

// GetSession gets a session by ID
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// ListSessions lists all sessions
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// CloseSession closes and removes a session
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	if err := sess.Close(); err != nil {
		return err
	}

	delete(m.sessions, id)
	return nil
}

// CloseAll closes all sessions
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for id, sess := range m.sessions {
		if err := sess.Close(); err != nil {
			lastErr = err
		}
		delete(m.sessions, id)
	}

	return lastErr
}
