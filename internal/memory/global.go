// Package memory provides global memory management across projects
package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GlobalMemoryStore manages memories across all projects
type GlobalMemoryStore struct {
	basePath string
}

// NewGlobalMemoryStore creates a new global memory store
func NewGlobalMemoryStore() (*GlobalMemoryStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	basePath := filepath.Join(homeDir, ".helm", "memory")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}

	return &GlobalMemoryStore{basePath: basePath}, nil
}

// GlobalMemoryEntry represents a global memory
type GlobalMemoryEntry struct {
	ID            string    `json:"id"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	Type          string    `json:"type"`
	SourceProject string    `json:"source_project"`
	Tags          []string  `json:"tags"`
	Confidence    float64   `json:"confidence"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	UsageCount    int       `json:"usage_count"`
}

// Store stores a global memory
func (gms *GlobalMemoryStore) Store(ctx context.Context, entry GlobalMemoryEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	now := time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	filePath := gms.getFilePath(entry.ID)
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// Get retrieves a global memory by ID
func (gms *GlobalMemoryStore) Get(id string) (*GlobalMemoryEntry, error) {
	filePath := gms.getFilePath(id)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var entry GlobalMemoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// List lists all global memories
func (gms *GlobalMemoryStore) List(ctx context.Context) ([]GlobalMemoryEntry, error) {
	entries, err := os.ReadDir(gms.basePath)
	if err != nil {
		return nil, err
	}

	var memories []GlobalMemoryEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		mem, err := gms.Get(id)
		if err != nil {
			continue
		}

		memories = append(memories, *mem)
	}

	return memories, nil
}

// Search searches global memories by keyword
func (gms *GlobalMemoryStore) Search(ctx context.Context, query string) ([]GlobalMemoryEntry, error) {
	memories, err := gms.List(ctx)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []GlobalMemoryEntry

	for _, mem := range memories {
		if strings.Contains(strings.ToLower(mem.Key), queryLower) ||
			strings.Contains(strings.ToLower(mem.Value), queryLower) ||
			containsAny(mem.Tags, queryLower) {
			results = append(results, mem)
		}
	}

	return results, nil
}

// FindRelevantForProject finds memories relevant to a new project
func (gms *GlobalMemoryStore) FindRelevantForProject(ctx context.Context, projectType, language, framework string) ([]GlobalMemoryEntry, error) {
	memories, err := gms.List(ctx)
	if err != nil {
		return nil, err
	}

	var relevant []GlobalMemoryEntry

	for _, mem := range memories {
		score := 0.0

		// Check tags for matches
		for _, tag := range mem.Tags {
			tagLower := strings.ToLower(tag)
			if tagLower == strings.ToLower(language) {
				score += 3.0
			}
			if tagLower == strings.ToLower(framework) {
				score += 2.0
			}
			if tagLower == strings.ToLower(projectType) {
				score += 2.0
			}
		}

		// Universal memories (coding conventions, best practices)
		if mem.Type == "convention" || mem.Type == "best_practice" {
			score += 1.0
		}

		// High confidence memories
		score *= mem.Confidence

		if score >= 2.0 {
			relevant = append(relevant, mem)
		}
	}

	return relevant, nil
}

// Delete deletes a global memory
func (gms *GlobalMemoryStore) Delete(id string) error {
	filePath := gms.getFilePath(id)
	return os.Remove(filePath)
}

// SyncToProject syncs global memories to a project
func (gms *GlobalMemoryStore) SyncToProject(ctx context.Context, project string, entries []GlobalMemoryEntry) error {
	projectMemPath := filepath.Join(project, ".helm", "memory")
	if err := os.MkdirAll(projectMemPath, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		// Translate if needed (e.g., Go conventions to Python)
		translated := gms.translateEntry(entry, project)

		filePath := filepath.Join(projectMemPath, entry.ID+".json")
		data, err := json.MarshalIndent(translated, "", "  ")
		if err != nil {
			continue
		}

		os.WriteFile(filePath, data, 0644)
	}

	return nil
}

// translateEntry translates a memory entry to a different project context
func (gms *GlobalMemoryStore) translateEntry(entry GlobalMemoryEntry, targetProject string) GlobalMemoryEntry {
	// In a real implementation, this would use LLM to translate
	// e.g., "Use Zod for validation" -> "Use go-playground/validator for validation"
	// for a Go project

	// For now, just mark as imported
	entry.Tags = append(entry.Tags, "imported")
	entry.SourceProject = targetProject
	return entry
}

func (gms *GlobalMemoryStore) getFilePath(id string) string {
	return filepath.Join(gms.basePath, id+".json")
}

func containsAny(tags []string, query string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

// ConflictResolver resolves conflicts between global and project memories
type ConflictResolver struct{}

// Conflict represents a memory conflict
type Conflict struct {
	GlobalMemory  GlobalMemoryEntry
	ProjectMemory GlobalMemoryEntry
	ConflictType  string // "value_mismatch", "confidence_diff"
}

// Resolve resolves a conflict
func (cr *ConflictResolver) Resolve(conflict Conflict, strategy string) (*GlobalMemoryEntry, error) {
	switch strategy {
	case "prefer_global":
		return &conflict.GlobalMemory, nil
	case "prefer_project":
		// Update global with project value
		updated := conflict.GlobalMemory
		updated.Value = conflict.ProjectMemory.Value
		updated.Confidence = conflict.ProjectMemory.Confidence
		return &updated, nil
	case "merge":
		// Merge values (in real implementation, use LLM)
		merged := conflict.GlobalMemory
		merged.Value = conflict.GlobalMemory.Value + " | " + conflict.ProjectMemory.Value
		merged.Confidence = (conflict.GlobalMemory.Confidence + conflict.ProjectMemory.Confidence) / 2
		return &merged, nil
	default:
		return &conflict.GlobalMemory, nil
	}
}
