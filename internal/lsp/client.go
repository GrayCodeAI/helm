// Package lsp provides Language Server Protocol integration.
package lsp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Client represents an LSP client connection
type Client struct {
	Name    string
	Command string
	Args    []string
	Running bool
}

// Manager manages LSP clients
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewManager creates a new LSP manager
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// RegisterClient registers an LSP client
func (m *Manager) RegisterClient(name, command string, args []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[name] = &Client{
		Name:    name,
		Command: command,
		Args:    args,
	}
}

// StartClient starts an LSP client
func (m *Manager) StartClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("client not found: %s", name)
	}

	if client.Running {
		return nil
	}

	// Check if command exists
	cmd := exec.Command("which", client.Command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("LSP server not found: %s", client.Command)
	}

	client.Running = true
	return nil
}

// StopClient stops an LSP client
func (m *Manager) StopClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("client not found: %s", name)
	}

	client.Running = false
	return nil
}

// GetDiagnostics gets diagnostics from an LSP client
func (m *Manager) GetDiagnostics(ctx context.Context, name, filePath string) ([]string, error) {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()

	if !ok || !client.Running {
		return nil, fmt.Errorf("client not running: %s", name)
	}

	// In production, would communicate via JSON-RPC
	// For now, run linter directly
	cmd := exec.CommandContext(ctx, client.Command, filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.Split(string(output), "\n"), nil
	}

	return strings.Split(string(output), "\n"), nil
}

// ListClients lists all registered clients
func (m *Manager) ListClients() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	return clients
}

// AutoDiscover discovers available LSP servers
func (m *Manager) AutoDiscover() []string {
	servers := map[string]string{
		"gopls":         "gopls",
		"pyright":       "pyright-langserver",
		"tsserver":      "typescript-language-server",
		"rust-analyzer": "rust-analyzer",
	}

	var discovered []string
	for name, cmd := range servers {
		if exec.Command("which", cmd).Run() == nil {
			discovered = append(discovered, name)
			m.RegisterClient(name, cmd, nil)
		}
	}

	return discovered
}
