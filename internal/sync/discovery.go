// Package sync provides session file discovery and auto-sync.
package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/yourname/helm/internal/pubsub"
)

// AgentDef defines an agent's session file location
type AgentDef struct {
	Name        string
	SessionDir  string
	FilePattern string
	Parser      string
}

// DiscoveryEngine discovers and syncs agent sessions
type DiscoveryEngine struct {
	agents  []AgentDef
	watcher *fsnotify.Watcher
	broker  *pubsub.Broker
	mu      sync.RWMutex
}

// NewDiscoveryEngine creates a new discovery engine
func NewDiscoveryEngine(broker *pubsub.Broker) (*DiscoveryEngine, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	return &DiscoveryEngine{
		watcher: watcher,
		broker:  broker,
		agents:  DefaultAgents(),
	}, nil
}

// DefaultAgents returns the default agent definitions
func DefaultAgents() []AgentDef {
	home, _ := os.UserHomeDir()
	return []AgentDef{
		{
			Name:        "claude",
			SessionDir:  filepath.Join(home, ".claude", "projects"),
			FilePattern: "*.jsonl",
			Parser:      "claude",
		},
		{
			Name:        "codex",
			SessionDir:  filepath.Join(home, ".codex", "sessions"),
			FilePattern: "*.jsonl",
			Parser:      "codex",
		},
		{
			Name:        "gemini",
			SessionDir:  filepath.Join(home, ".gemini", "sessions"),
			FilePattern: "*.jsonl",
			Parser:      "gemini",
		},
		{
			Name:        "opencode",
			SessionDir:  filepath.Join(home, ".local", "state", "opencode"),
			FilePattern: "*.jsonl",
			Parser:      "opencode",
		},
	}
}

// Start starts the discovery engine
func (d *DiscoveryEngine) Start(ctx context.Context) error {
	for _, agent := range d.agents {
		if _, err := os.Stat(agent.SessionDir); err == nil {
			if err := d.watcher.Add(agent.SessionDir); err != nil {
				return fmt.Errorf("watch %s: %w", agent.SessionDir, err)
			}
		}
	}

	go d.watchLoop(ctx)
	return nil
}

// Stop stops the discovery engine
func (d *DiscoveryEngine) Stop() error {
	return d.watcher.Close()
}

func (d *DiscoveryEngine) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if strings.HasSuffix(event.Name, ".jsonl") {
					d.broker.Publish(pubsub.EventSessionCreated, map[string]interface{}{
						"file": event.Name,
						"time": time.Now(),
					})
				}
			}
		case err, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "watcher error: %v\n", err)
		}
	}
}

// DiscoverSessions discovers all existing sessions
func (d *DiscoveryEngine) DiscoverSessions() ([]string, error) {
	var sessions []string

	d.mu.RLock()
	agents := d.agents
	d.mu.RUnlock()

	for _, agent := range agents {
		entries, err := os.ReadDir(agent.SessionDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), agent.FilePattern) || agent.FilePattern == "*.jsonl" {
				sessions = append(sessions, filepath.Join(agent.SessionDir, entry.Name()))
			}
		}
	}

	return sessions, nil
}
