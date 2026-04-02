// Package tracker provides file tracking per session.
package tracker

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"time"
)

// FileChange represents a tracked file change
type FileChange struct {
	Path      string
	Hash      string
	Size      int64
	Modified  time.Time
	SessionID string
	Action    string // read, write, delete
}

// Tracker tracks file changes per session
type Tracker struct {
	mu      sync.RWMutex
	changes map[string][]FileChange // sessionID -> changes
}

// NewTracker creates a new file tracker
func NewTracker() *Tracker {
	return &Tracker{
		changes: make(map[string][]FileChange),
	}
}

// TrackRead tracks a file read
func (t *Tracker) TrackRead(sessionID, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	change := FileChange{
		Path:      path,
		Hash:      fmt.Sprintf("%x", sha256.Sum256(content)),
		Size:      info.Size(),
		Modified:  info.ModTime(),
		SessionID: sessionID,
		Action:    "read",
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.changes[sessionID] = append(t.changes[sessionID], change)
	return nil
}

// TrackWrite tracks a file write
func (t *Tracker) TrackWrite(sessionID, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	change := FileChange{
		Path:      path,
		Hash:      fmt.Sprintf("%x", sha256.Sum256(content)),
		Size:      info.Size(),
		Modified:  info.ModTime(),
		SessionID: sessionID,
		Action:    "write",
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.changes[sessionID] = append(t.changes[sessionID], change)
	return nil
}

// GetChanges gets all changes for a session
func (t *Tracker) GetChanges(sessionID string) []FileChange {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.changes[sessionID]
}

// GetFiles gets all unique files touched by a session
func (t *Tracker) GetFiles(sessionID string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	seen := make(map[string]bool)
	var files []string
	for _, c := range t.changes[sessionID] {
		if !seen[c.Path] {
			seen[c.Path] = true
			files = append(files, c.Path)
		}
	}
	return files
}

// HasChanged checks if a file has changed since last track
func (t *Tracker) HasChanged(sessionID, path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	currentHash := fmt.Sprintf("%x", sha256.Sum256(content))

	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, c := range t.changes[sessionID] {
		if c.Path == path && c.Hash != currentHash {
			return true, nil
		}
	}
	return false, nil
}
