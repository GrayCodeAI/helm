// Package git provides git operations and worktree management.
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Worktree manages git worktrees for parallel sessions
type Worktree struct {
	baseDir string
}

// NewWorktree creates a new worktree manager
func NewWorktree(baseDir string) *Worktree {
	return &Worktree{baseDir: baseDir}
}

// Create creates a new worktree for a session
func (w *Worktree) Create(name, commit string) (string, error) {
	dir := fmt.Sprintf("%s/.helm/worktrees/%s", w.baseDir, name)
	cmd := exec.Command("git", "worktree", "add", "-b", name, dir, commit)
	cmd.Dir = w.baseDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without -b flag (branch might already exist)
		cmd = exec.Command("git", "worktree", "add", dir, commit)
		cmd.Dir = w.baseDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("create worktree: %w (%s)", err, string(output))
		}
	}

	return dir, nil
}

// Remove removes a worktree
func (w *Worktree) Remove(name string) error {
	dir := fmt.Sprintf("%s/.helm/worktrees/%s", w.baseDir, name)
	cmd := exec.Command("git", "worktree", "remove", "-f", dir)
	cmd.Dir = w.baseDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove worktree: %w (%s)", err, string(output))
	}

	return nil
}

// List returns all worktrees
func (w *Worktree) List() ([]string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = w.baseDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	var worktrees []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			worktrees = append(worktrees, path)
		}
	}

	return worktrees, nil
}

// IsWorktree checks if a directory is a worktree
func (w *Worktree) IsWorktree(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-path", ".")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "worktrees")
}
