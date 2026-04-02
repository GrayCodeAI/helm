// Package conductor provides meta-agent orchestration.
package conductor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourname/helm/internal/agent"
	"github.com/yourname/helm/internal/pubsub"
)

// Agent represents a specialized agent
type Agent struct {
	ID         string
	Name       string
	Role       string
	Model      string
	MaxTurns   int
	Tools      []string
	Status     string
	LastActive time.Time
}

// Conductor manages multiple agents
type Conductor struct {
	mu     sync.RWMutex
	agents map[string]*Agent
	broker *pubsub.Broker
}

// NewConductor creates a new conductor
func NewConductor(broker *pubsub.Broker) *Conductor {
	return &Conductor{
		agents: make(map[string]*Agent),
		broker: broker,
	}
}

// RegisterAgent registers a specialized agent
func (c *Conductor) RegisterAgent(a *Agent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.agents[a.ID] = a
}

// GetAgent gets an agent by ID
func (c *Conductor) GetAgent(id string) (*Agent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.agents[id]
	return a, ok
}

// ListAgents lists all registered agents
func (c *Conductor) ListAgents() []*Agent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	agents := make([]*Agent, 0, len(c.agents))
	for _, a := range c.agents {
		agents = append(agents, a)
	}
	return agents
}

// SelectAgent selects the best agent for a task
func (c *Conductor) SelectAgent(taskType string) *Agent {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, a := range c.agents {
		if a.Role == taskType && a.Status == "available" {
			return a
		}
	}

	// Fallback to first available agent
	for _, a := range c.agents {
		if a.Status == "available" {
			return a
		}
	}

	return nil
}

// ExecuteTask executes a task using the selected agent
func (c *Conductor) ExecuteTask(ctx context.Context, taskType, prompt string) (*agent.Coordinator, error) {
	agent := c.SelectAgent(taskType)
	if agent == nil {
		return nil, fmt.Errorf("no available agent for task type: %s", taskType)
	}

	agent.Status = "busy"
	agent.LastActive = time.Now()

	if c.broker != nil {
		c.broker.Publish("conductor.task_started", map[string]interface{}{
			"agent_id": agent.ID,
			"task":     taskType,
		})
	}

	// In production, would create and run coordinator
	return nil, nil
}

// GetStats returns conductor statistics
func (c *Conductor) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := len(c.agents)
	busy := 0
	available := 0

	for _, a := range c.agents {
		if a.Status == "busy" {
			busy++
		} else {
			available++
		}
	}

	return map[string]interface{}{
		"total":     total,
		"busy":      busy,
		"available": available,
	}
}
