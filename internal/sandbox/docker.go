// Package sandbox provides Docker sandbox support for agent sessions.
package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Config represents sandbox configuration
type Config struct {
	Image       string
	CPULimit    float64
	MemoryLimit string
	Volumes     map[string]string
	Network     string
	WorkDir     string
	Env         map[string]string
}

// DefaultConfig returns default sandbox config
func DefaultConfig() Config {
	return Config{
		Image:       "ubuntu:22.04",
		CPULimit:    2.0,
		MemoryLimit: "4g",
		Network:     "none",
		WorkDir:     "/workspace",
		Env:         make(map[string]string),
		Volumes:     make(map[string]string),
	}
}

// Container represents a running sandbox container
type Container struct {
	ID        string
	Config    Config
	StartedAt time.Time
}

// Manager manages Docker sandbox containers
type Manager struct {
	containers map[string]*Container
}

// NewManager creates a new sandbox manager
func NewManager() *Manager {
	return &Manager{
		containers: make(map[string]*Container),
	}
}

// Create creates a new sandbox container
func (m *Manager) Create(ctx context.Context, cfg Config) (*Container, error) {
	args := []string{"run", "-d"}

	// Resource limits
	if cfg.CPULimit > 0 {
		args = append(args, fmt.Sprintf("--cpus=%.1f", cfg.CPULimit))
	}
	if cfg.MemoryLimit != "" {
		args = append(args, "--memory", cfg.MemoryLimit)
	}

	// Network
	if cfg.Network != "" {
		args = append(args, "--network", cfg.Network)
	}

	// Working directory
	if cfg.WorkDir != "" {
		args = append(args, "-w", cfg.WorkDir)
	}

	// Volumes
	for hostPath, containerPath := range cfg.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// Environment variables
	for key, value := range cfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, cfg.Image, "sleep", "infinity")

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run: %s: %w", string(output), err)
	}

	containerID := strings.TrimSpace(string(output))
	container := &Container{
		ID:        containerID,
		Config:    cfg,
		StartedAt: time.Now(),
	}

	m.containers[containerID] = container
	return container, nil
}

// Execute runs a command inside a sandbox container
func (m *Manager) Execute(ctx context.Context, containerID, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "exec", containerID, "bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker exec: %w", err)
	}
	return string(output), nil
}

// Stop stops a sandbox container
func (m *Manager) Stop(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker stop: %w", err)
	}

	cmd = exec.CommandContext(ctx, "docker", "rm", containerID)
	cmd.Run()

	delete(m.containers, containerID)
	return nil
}

// StopAll stops all sandbox containers
func (m *Manager) StopAll(ctx context.Context) error {
	for id := range m.containers {
		m.Stop(ctx, id)
	}
	return nil
}

// ListContainers lists all running containers
func (m *Manager) ListContainers() []*Container {
	containers := make([]*Container, 0, len(m.containers))
	for _, c := range m.containers {
		containers = append(containers, c)
	}
	return containers
}

// IsAvailable checks if Docker is available
func IsAvailable() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}
