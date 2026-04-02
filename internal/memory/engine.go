// Package memory provides project memory management with FTS5 search.
package memory

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourname/helm/internal/db"
)

// DBQuerier is the minimal interface for database operations.
type DBQuerier interface {
	CreateMemory(ctx context.Context, arg db.CreateMemoryParams) (sql.Result, error)
	GetMemory(ctx context.Context, arg db.GetMemoryParams) (db.Memory, error)
	ListMemories(ctx context.Context, project string) ([]db.Memory, error)
	ListMemoriesByType(ctx context.Context, arg db.ListMemoriesByTypeParams) ([]db.Memory, error)
	SearchMemories(ctx context.Context, arg db.SearchMemoriesParams) ([]db.Memory, error)
	UpdateMemory(ctx context.Context, arg db.UpdateMemoryParams) error
	DeleteMemory(ctx context.Context, id string) error
	UpsertMemory(ctx context.Context, arg db.UpsertMemoryParams) error
}

// Engine manages project memory: store, retrieve, consolidate, forget.
type Engine struct {
	q DBQuerier
}

// NewEngine creates a new memory engine backed by the given querier.
func NewEngine(q DBQuerier) *Engine {
	return &Engine{q: q}
}

// Store adds a new memory entry.
func (e *Engine) Store(ctx context.Context, project string, memType MemoryType, key, value string, source string) error {
	entry := &MemoryEntry{
		ID:         uuid.New().String(),
		Project:    project,
		Type:       memType,
		Key:        key,
		Value:      value,
		Source:     source,
		Confidence: 0.5,
	}

	_, err := e.q.CreateMemory(ctx, entry.ToDB())
	if err != nil {
		return fmt.Errorf("store memory: %w", err)
	}
	return nil
}

// Get retrieves a memory by project and key.
func (e *Engine) Get(ctx context.Context, project, key string) (*MemoryEntry, error) {
	m, err := e.q.GetMemory(ctx, db.GetMemoryParams{
		Project: project,
		Key:     key,
	})
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	entry := FromDB(m)
	return &entry, nil
}

// List returns all memories for a project.
func (e *Engine) List(ctx context.Context, project string) ([]MemoryEntry, error) {
	memories, err := e.q.ListMemories(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}

	result := make([]MemoryEntry, len(memories))
	for i, m := range memories {
		result[i] = FromDB(m)
	}
	return result, nil
}

// ListByType returns memories for a project filtered by type.
func (e *Engine) ListByType(ctx context.Context, project string, memType MemoryType) ([]MemoryEntry, error) {
	memories, err := e.q.ListMemoriesByType(ctx, db.ListMemoriesByTypeParams{
		Project: project,
		Type:    string(memType),
	})
	if err != nil {
		return nil, fmt.Errorf("list memories by type: %w", err)
	}

	result := make([]MemoryEntry, len(memories))
	for i, m := range memories {
		result[i] = FromDB(m)
	}
	return result, nil
}

// Search finds memories matching a query string.
func (e *Engine) Search(ctx context.Context, project, query string, limit int) ([]MemoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	memories, err := e.q.SearchMemories(ctx, db.SearchMemoriesParams{
		Project: project,
		Limit:   int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}

	result := make([]MemoryEntry, len(memories))
	for i, m := range memories {
		result[i] = FromDB(m)
	}
	return result, nil
}

// Update modifies an existing memory.
func (e *Engine) Update(ctx context.Context, id, value string, confidence float64) error {
	err := e.q.UpdateMemory(ctx, db.UpdateMemoryParams{
		ID:         id,
		Value:      value,
		Confidence: confidence,
	})
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

// Delete removes a memory by ID.
func (e *Engine) Delete(ctx context.Context, id string) error {
	err := e.q.DeleteMemory(ctx, id)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

// Upsert stores or updates a memory by key.
func (e *Engine) Upsert(ctx context.Context, project string, memType MemoryType, key, value string, source string, confidence float64) error {
	id := uuid.New().String()
	err := e.q.UpsertMemory(ctx, db.UpsertMemoryParams{
		ID:         id,
		Project:    project,
		Type:       string(memType),
		Key:        key,
		Value:      value,
		Source:     source,
		Confidence: confidence,
	})
	if err != nil {
		return fmt.Errorf("upsert memory: %w", err)
	}
	return nil
}

// FormatForContext formats memories as a context string for agent prompts.
func FormatForContext(memories []MemoryEntry) string {
	if len(memories) == 0 {
		return ""
	}

	var result string
	for _, m := range memories {
		result += fmt.Sprintf("[%s] %s: %s\n", m.Type, m.Key, m.Value)
	}
	return result
}
