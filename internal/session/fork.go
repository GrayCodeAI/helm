// Package session provides session forking capabilities
package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/helm/internal/git"
	"github.com/yourname/helm/internal/memory"
)

// Fork creates a branch of an existing session
type Fork struct {
	ID               string
	ParentID         string
	Session          *Session
	BranchName       string
	WorktreeDir      string
	InheritedContext ForkContext
	CreatedAt        time.Time
}

// ForkContext defines what context is inherited from the parent
type ForkContext struct {
	ProjectMemory  bool
	FileState      bool
	PromptContext  bool
	SessionHistory bool
	SelectedFiles  []string
}

// FullContext returns a ForkContext with all options enabled
func FullContext() ForkContext {
	return ForkContext{
		ProjectMemory:  true,
		FileState:      true,
		PromptContext:  true,
		SessionHistory: true,
	}
}

// RelevantOnly returns a ForkContext with only relevant context
func RelevantOnly(files []string) ForkContext {
	return ForkContext{
		ProjectMemory:  true,
		FileState:      true,
		PromptContext:  true,
		SessionHistory: false,
		SelectedFiles:  files,
	}
}

// MemoryOnly returns a ForkContext with only project memory
func MemoryOnly() ForkContext {
	return ForkContext{
		ProjectMemory:  true,
		FileState:      false,
		PromptContext:  false,
		SessionHistory: false,
	}
}

// ForkManager manages session forking
type ForkManager struct {
	sessionManager *Manager
	memoryEngine   *memory.Engine
	worktree       *git.Worktree
	forks          map[string]*Fork // forkID -> Fork
}

// NewForkManager creates a new fork manager
func NewForkManager(sm *Manager, me *memory.Engine, wt *git.Worktree) *ForkManager {
	return &ForkManager{
		sessionManager: sm,
		memoryEngine:   me,
		worktree:       wt,
		forks:          make(map[string]*Fork),
	}
}

// Create creates a new fork from an existing session
func (fm *ForkManager) Create(ctx context.Context, parentID, newPrompt string, context ForkContext) (*Fork, error) {
	// Get parent session
	parent, err := fm.sessionManager.Get(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("get parent session: %w", err)
	}

	// Create branch name
	branchName := fmt.Sprintf("helm-fork-%s-%d", parentID[:8], time.Now().Unix())

	// Create git worktree
	var worktreeDir string
	if fm.worktree != nil {
		dir, err := fm.worktree.Create(branchName, "HEAD")
		if err != nil {
			// Fallback to simple branch without worktree
			worktreeDir = ""
		} else {
			worktreeDir = dir
		}
	}

	// Create new session
	newSession := &Session{
		ID:        uuid.New().String(),
		Provider:  parent.Provider,
		Model:     parent.Model,
		Project:   parent.Project,
		Prompt:    newPrompt,
		Status:    "running",
		StartedAt: time.Now(),
	}

	if context.PromptContext {
		// Include parent prompt context
		if parent.Summary != "" {
			newSession.Prompt = fmt.Sprintf("Based on previous attempt (%s):\n%s\n\nNew approach:\n%s",
				parent.ID[:8], parent.Summary, newPrompt)
		} else {
			newSession.Prompt = fmt.Sprintf("Previous attempt context:\n%s\n\nNew approach:\n%s",
				parent.Prompt, newPrompt)
		}
	}

	// Store the new session
	if err := fm.sessionManager.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("create forked session: %w", err)
	}

	fork := &Fork{
		ID:               uuid.New().String(),
		ParentID:         parentID,
		Session:          newSession,
		BranchName:       branchName,
		WorktreeDir:      worktreeDir,
		InheritedContext: context,
		CreatedAt:        time.Now(),
	}

	fm.forks[fork.ID] = fork

	return fork, nil
}

// Get retrieves a fork by ID
func (fm *ForkManager) Get(forkID string) (*Fork, bool) {
	fork, ok := fm.forks[forkID]
	return fork, ok
}

// List returns all forks for a parent session
func (fm *ForkManager) List(parentID string) []*Fork {
	var result []*Fork
	for _, fork := range fm.forks {
		if fork.ParentID == parentID {
			result = append(result, fork)
		}
	}
	return result
}

// Delete removes a fork
func (fm *ForkManager) Delete(ctx context.Context, forkID string) error {
	fork, ok := fm.forks[forkID]
	if !ok {
		return fmt.Errorf("fork not found: %s", forkID)
	}

	// Remove worktree if exists
	if fork.WorktreeDir != "" && fm.worktree != nil {
		if err := fm.worktree.Remove(fork.BranchName); err != nil {
			// Log but don't fail
		}
	}

	// Delete session
	if err := fm.sessionManager.Delete(ctx, fork.Session.ID); err != nil {
		return fmt.Errorf("delete forked session: %w", err)
	}

	delete(fm.forks, forkID)
	return nil
}

// Compare compares two forks
func (fm *ForkManager) Compare(forkID1, forkID2 string) (*Comparison, error) {
	fork1, ok := fm.forks[forkID1]
	if !ok {
		return nil, fmt.Errorf("fork not found: %s", forkID1)
	}

	fork2, ok := fm.forks[forkID2]
	if !ok {
		return nil, fmt.Errorf("fork not found: %s", forkID2)
	}

	return &Comparison{
		Fork1:     fork1,
		Fork2:     fork2,
		Cost1:     fork1.Session.Cost,
		Cost2:     fork2.Session.Cost,
		Duration1: fork1.Session.EndedAt.Sub(fork1.Session.StartedAt),
		Duration2: fork2.Session.EndedAt.Sub(fork2.Session.StartedAt),
	}, nil
}

// Comparison represents a comparison between two forks
type Comparison struct {
	Fork1     *Fork
	Fork2     *Fork
	Cost1     float64
	Cost2     float64
	Duration1 time.Duration
	Duration2 time.Duration
}

// Winner returns which fork performed better
func (c *Comparison) Winner() *Fork {
	// Compare by cost first
	if c.Cost1 < c.Cost2 {
		return c.Fork1
	}
	if c.Cost2 < c.Cost1 {
		return c.Fork2
	}

	// If costs are equal, compare by duration
	if c.Duration1 < c.Duration2 {
		return c.Fork1
	}
	return c.Fork2
}

// Diff returns a diff of the changes between forks
func (c *Comparison) Diff() string {
	return fmt.Sprintf(`## Fork Comparison

### Fork 1 (%s)
- Cost: $%.4f
- Duration: %s
- Status: %s
- Model: %s

### Fork 2 (%s)
- Cost: $%.4f
- Duration: %s
- Status: %s
- Model: %s

### Winner: %s
`,
		c.Fork1.ID[:8], c.Cost1, c.Duration1.Round(time.Second), c.Fork1.Session.Status, c.Fork1.Session.Model,
		c.Fork2.ID[:8], c.Cost2, c.Duration2.Round(time.Second), c.Fork2.Session.Status, c.Fork2.Session.Model,
		c.Winner().ID[:8])
}

// GetContextForFork retrieves the inherited context for a fork
func (fm *ForkManager) GetContextForFork(ctx context.Context, forkID string) (string, error) {
	fork, ok := fm.forks[forkID]
	if !ok {
		return "", fmt.Errorf("fork not found: %s", forkID)
	}

	var contextParts []string

	// Get project memory if requested
	if fork.InheritedContext.ProjectMemory && fm.memoryEngine != nil {
		memories, err := fm.memoryEngine.List(ctx, fork.Session.Project)
		if err == nil && len(memories) > 0 {
			contextParts = append(contextParts, "## Project Memory")
			for _, m := range memories {
				contextParts = append(contextParts, fmt.Sprintf("- [%s] %s: %s", m.Type, m.Key, m.Value))
			}
		}
	}

	// Get parent session info if requested
	if fork.InheritedContext.SessionHistory {
		parent, err := fm.sessionManager.Get(ctx, fork.ParentID)
		if err == nil {
			contextParts = append(contextParts, "## Parent Session")
			contextParts = append(contextParts, fmt.Sprintf("- ID: %s", parent.ID[:8]))
			contextParts = append(contextParts, fmt.Sprintf("- Status: %s", parent.Status))
			if parent.Summary != "" {
				contextParts = append(contextParts, fmt.Sprintf("- Summary: %s", parent.Summary))
			}
		}
	}

	// Join context parts
	return strings.Join(contextParts, "\n") + "\n", nil
}

// CleanupWorktrees removes all fork worktrees
func (fm *ForkManager) CleanupWorktrees() error {
	if fm.worktree == nil {
		return nil
	}

	for _, fork := range fm.forks {
		if fork.WorktreeDir != "" {
			_ = fm.worktree.Remove(fork.BranchName)
		}
	}

	return nil
}
