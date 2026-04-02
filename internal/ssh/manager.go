// Package ssh provides SSH remote session support.
package ssh

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Connection represents an SSH connection
type Connection struct {
	Host        string
	User        string
	Port        int
	KeyPath     string
	Connected   bool
	LastConnect time.Time
}

// Manager manages SSH connections and remote sessions
type Manager struct {
	mu          sync.RWMutex
	connections map[string]*Connection
}

// NewManager creates a new SSH manager
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
	}
}

// Connect establishes an SSH connection using ControlMaster
func (m *Manager) Connect(host, user string, port int, keyPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s@%s:%d", user, host, port)

	// Check if already connected
	if conn, ok := m.connections[key]; ok && conn.Connected {
		return nil
	}

	// Create control socket directory
	socketDir := filepath.Join(os.TempDir(), "helm-ssh")
	os.MkdirAll(socketDir, 0700)

	controlSocket := filepath.Join(socketDir, fmt.Sprintf("%s-%s-%d", user, host, port))

	// Start ControlMaster connection
	args := []string{
		"-o", "ControlMaster=yes",
		"-o", fmt.Sprintf("ControlPath=%s", controlSocket),
		"-o", "ControlPersist=600",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
	}

	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}

	if port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", port))
	}

	args = append(args, fmt.Sprintf("%s@%s", user, host), "true")

	cmd := exec.Command("ssh", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh connect: %w", err)
	}

	m.connections[key] = &Connection{
		Host:        host,
		User:        user,
		Port:        port,
		KeyPath:     keyPath,
		Connected:   true,
		LastConnect: time.Now(),
	}

	return nil
}

// Execute runs a command on a remote host
func (m *Manager) Execute(host, user string, port int, command string) (string, error) {
	key := fmt.Sprintf("%s@%s:%d", user, host, port)

	m.mu.RLock()
	conn, exists := m.connections[key]
	m.mu.RUnlock()

	if !exists || !conn.Connected {
		if err := m.Connect(host, user, port, conn.KeyPath); err != nil {
			return "", err
		}
	}

	socketDir := filepath.Join(os.TempDir(), "helm-ssh")
	controlSocket := filepath.Join(socketDir, fmt.Sprintf("%s-%s-%d", user, host, port))

	args := []string{
		"-o", fmt.Sprintf("ControlPath=%s", controlSocket),
		"-o", "ControlMaster=no",
		fmt.Sprintf("%s@%s", user, host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("ssh execute: %w", err)
	}

	return string(output), nil
}

// DeployBinary deploys the helm binary to a remote host
func (m *Manager) DeployBinary(host, user string, port int, localPath, remotePath string) error {
	args := []string{
		"-o", "ControlMaster=auto",
		"-o", "ControlPersist=60",
		localPath,
		fmt.Sprintf("%s@%s:%s", user, host, remotePath),
	}

	if port != 22 {
		args = append(args, "-P", fmt.Sprintf("%d", port))
	}

	cmd := exec.Command("scp", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp deploy: %w", err)
	}

	// Make executable
	_, err := m.Execute(host, user, port, fmt.Sprintf("chmod +x %s", remotePath))
	return err
}

// DetectPlatform detects the remote platform
func (m *Manager) DetectPlatform(host, user string, port int) (string, error) {
	output, err := m.Execute(host, user, port, "uname -s -m")
	if err != nil {
		return "", err
	}

	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected uname output: %s", output)
	}

	return fmt.Sprintf("%s-%s", strings.ToLower(parts[0]), strings.ToLower(parts[1])), nil
}

// Disconnect closes an SSH connection
func (m *Manager) Disconnect(host, user string, port int) error {
	key := fmt.Sprintf("%s@%s:%d", user, host, port)

	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[key]
	if !ok {
		return nil
	}

	socketDir := filepath.Join(os.TempDir(), "helm-ssh")
	controlSocket := filepath.Join(socketDir, fmt.Sprintf("%s-%s-%d", user, host, port))

	cmd := exec.Command("ssh", "-o", fmt.Sprintf("ControlPath=%s", controlSocket), "-O", "exit", fmt.Sprintf("%s@%s", user, host))
	cmd.Run()

	conn.Connected = false
	delete(m.connections, key)
	return nil
}

// ListConnections lists all active connections
func (m *Manager) ListConnections() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]*Connection, 0, len(m.connections))
	for _, c := range m.connections {
		conns = append(conns, c)
	}
	return conns
}

// IsConnected checks if a connection is active
func (m *Manager) IsConnected(host, user string, port int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s@%s:%d", user, host, port)
	conn, ok := m.connections[key]
	return ok && conn.Connected
}

// CheckConnection checks if an SSH connection is alive
func CheckConnection(host string, port int) bool {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
