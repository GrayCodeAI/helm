// Package git provides snapshot/state management for natural language git commands
package git

import (
	"fmt"
	"os/exec"
	"time"
)

// Snapshot represents a saved state
type Snapshot struct {
	ID          string
	Name        string
	Description string
	Commit      string
	Branch      string
	CreatedAt   time.Time
}

// SnapshotManager manages git snapshots
type SnapshotManager struct {
	repoPath string
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(repoPath string) *SnapshotManager {
	return &SnapshotManager{repoPath: repoPath}
}

// Save creates a snapshot
func (sm *SnapshotManager) Save(name, description string) (*Snapshot, error) {
	// Create a commit with the snapshot name
	cmd := exec.Command("git", "-C", sm.repoPath, "add", "-A")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("stage changes: %w", err)
	}

	commitMsg := fmt.Sprintf("[helm-snapshot] %s\n\n%s", name, description)
	cmd = exec.Command("git", "-C", sm.repoPath, "commit", "-m", commitMsg, "--allow-empty")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("commit: %w - %s", err, output)
	}

	// Get commit hash
	cmd = exec.Command("git", "-C", sm.repoPath, "rev-parse", "HEAD")
	hash, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Get branch
	cmd = exec.Command("git", "-C", sm.repoPath, "branch", "--show-current")
	branch, _ := cmd.Output()

	return &Snapshot{
		ID:          string(hash)[:8],
		Name:        name,
		Description: description,
		Commit:      string(hash),
		Branch:      string(branch),
		CreatedAt:   time.Now(),
	}, nil
}

// List lists all snapshots
func (sm *SnapshotManager) List() ([]Snapshot, error) {
	// Search for commits with [helm-snapshot] tag
	cmd := exec.Command("git", "-C", sm.repoPath, "log", "--all", "--grep=\\[helm-snapshot\\]", "--pretty=format:%H|%s|%ai")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var snapshots []Snapshot
	lines := splitLines(string(output))
	for _, line := range lines {
		parts := splitN(line, "|", 3)
		if len(parts) >= 2 {
			name := parts[1]
			name = trimPrefix(name, "[helm-snapshot] ")

			snapshots = append(snapshots, Snapshot{
				ID:        parts[0][:8],
				Name:      name,
				Commit:    parts[0],
				CreatedAt: parseTime(parts[2]),
			})
		}
	}

	return snapshots, nil
}

// Restore restores to a snapshot
func (sm *SnapshotManager) Restore(snapshotID string) error {
	cmd := exec.Command("git", "-C", sm.repoPath, "checkout", snapshotID)
	return cmd.Run()
}

// Show shows what changed since a snapshot
func (sm *SnapshotManager) Show(snapshotID string) (string, error) {
	cmd := exec.Command("git", "-C", sm.repoPath, "diff", snapshotID+"..HEAD")
	output, err := cmd.Output()
	return string(output), err
}

// Compare compares two snapshots
func (sm *SnapshotManager) Compare(id1, id2 string) (string, error) {
	cmd := exec.Command("git", "-C", sm.repoPath, "diff", id1+".."+id2)
	output, err := cmd.Output()
	return string(output), err
}

// Undo undoes the last agent changes
func (sm *SnapshotManager) Undo() error {
	// Find last helm commit
	cmd := exec.Command("git", "-C", sm.repoPath, "log", "--grep=\\[helm-snapshot\\]", "-1", "--pretty=format:%H")
	hash, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("no helm commits found")
	}

	// Reset to before that commit
	cmd = exec.Command("git", "-C", sm.repoPath, "reset", "--hard", string(hash)+"~1")
	return cmd.Run()
}

func splitLines(s string) []string {
	var lines []string
	var current []rune
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, string(current))
			current = nil
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func splitN(s string, sep string, n int) []string {
	var parts []string
	for i := 0; i < n-1; i++ {
		idx := 0
		for j := 0; j < len(s); j++ {
			if s[j:j+1] == sep {
				idx = j
				break
			}
		}
		if idx == 0 {
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func parseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05 -0700", s)
	return t
}
