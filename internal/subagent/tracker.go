// Package subagent provides subagent session tracking and management.
package subagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/pubsub"
)

// Subagent represents a child agent session
type Subagent struct {
	ID          string
	ParentID    string
	Name        string
	Status      string
	StartedAt   time.Time
	CompletedAt *time.Time
	Output      string
	Error       string
	Tokens      int
	Cost        float64
}

// Tracker tracks subagent sessions
type Tracker struct {
	mu        sync.RWMutex
	subagents map[string][]*Subagent // parentID -> subagents
	broker    *pubsub.Broker
	db        db.Querier
}

// NewTracker creates a new subagent tracker
func NewTracker(broker *pubsub.Broker, db db.Querier) *Tracker {
	return &Tracker{
		subagents: make(map[string][]*Subagent),
		broker:    broker,
		db:        db,
	}
}

// Spawn registers a new subagent
func (t *Tracker) Spawn(parentID, name string) *Subagent {
	sub := &Subagent{
		ID:        fmt.Sprintf("sub-%s-%d", parentID, time.Now().UnixNano()),
		ParentID:  parentID,
		Name:      name,
		Status:    "running",
		StartedAt: time.Now(),
	}

	t.mu.Lock()
	t.subagents[parentID] = append(t.subagents[parentID], sub)
	t.mu.Unlock()

	if t.broker != nil {
		t.broker.Publish("subagent.spawned", sub)
	}

	return sub
}

// Complete marks a subagent as complete
func (t *Tracker) Complete(subID string, output string, tokens int, cost float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for parentID, subs := range t.subagents {
		for _, sub := range subs {
			if sub.ID == subID {
				sub.Status = "completed"
				sub.Output = output
				sub.Tokens = tokens
				sub.Cost = cost
				now := time.Now()
				sub.CompletedAt = &now

				if t.broker != nil {
					t.broker.Publish("subagent.completed", sub)
				}
				return
			}
		}
		_ = parentID
	}
}

// Fail marks a subagent as failed
func (t *Tracker) Fail(subID string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, subs := range t.subagents {
		for _, sub := range subs {
			if sub.ID == subID {
				sub.Status = "failed"
				sub.Error = err.Error()
				now := time.Now()
				sub.CompletedAt = &now
				return
			}
		}
	}
}

// GetSubagents gets all subagents for a parent
func (t *Tracker) GetSubagents(parentID string) []*Subagent {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.subagents[parentID]
}

// GetActiveSubagents gets all running subagents
func (t *Tracker) GetActiveSubagents() []*Subagent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var active []*Subagent
	for _, subs := range t.subagents {
		for _, sub := range subs {
			if sub.Status == "running" {
				active = append(active, sub)
			}
		}
	}
	return active
}

// GetOrphanedSubagents gets subagents whose parent is no longer running
func (t *Tracker) GetOrphanedSubagents(runningParents map[string]bool) []*Subagent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var orphans []*Subagent
	for parentID, subs := range t.subagents {
		if !runningParents[parentID] {
			for _, sub := range subs {
				if sub.Status == "running" {
					orphans = append(orphans, sub)
				}
			}
		}
	}
	return orphans
}

// GetStats returns subagent statistics
func (t *Tracker) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0
	running := 0
	completed := 0
	failed := 0
	totalCost := 0.0
	totalTokens := 0

	for _, subs := range t.subagents {
		for _, sub := range subs {
			total++
			switch sub.Status {
			case "running":
				running++
			case "completed":
				completed++
				totalCost += sub.Cost
				totalTokens += sub.Tokens
			case "failed":
				failed++
			}
		}
	}

	return map[string]interface{}{
		"total":        total,
		"running":      running,
		"completed":    completed,
		"failed":       failed,
		"total_cost":   totalCost,
		"total_tokens": totalTokens,
	}
}

// DetectSubagentsFromMessages detects subagent events from session messages
func DetectSubagentsFromMessages(messages []db.Message) []*Subagent {
	var subagents []*Subagent

	for _, msg := range messages {
		content := msg.Content
		if strings.Contains(content, "subagent") || strings.Contains(content, "spawn") {
			sub := &Subagent{
				ID:        fmt.Sprintf("sub-%s-%d", msg.SessionID, msg.ID),
				ParentID:  msg.SessionID,
				Name:      extractSubagentName(content),
				Status:    "completed",
				StartedAt: parseMessageTime(msg.Timestamp),
			}
			subagents = append(subagents, sub)
		}
	}

	return subagents
}

func extractSubagentName(content string) string {
	// Extract name from content like "Spawning subagent: name"
	if idx := strings.Index(content, "subagent:"); idx != -1 {
		name := strings.TrimSpace(content[idx+9:])
		if end := strings.Index(name, "\n"); end != -1 {
			name = name[:end]
		}
		return name
	}
	return "unknown"
}

func parseMessageTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
}

// LinkSubagentToSession links a subagent to a child session in the database
func (t *Tracker) LinkSubagentToSession(ctx context.Context, subagentID, sessionID string) error {
	// Update session with parent reference
	// This would require a parent_session_id column in the sessions table
	return nil
}
