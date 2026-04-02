// Package memory provides project memory management with FTS5 search.
package memory

import (
	"time"

	"github.com/yourname/helm/internal/db"
)

// MemoryType enumerates the types of memories.
type MemoryType string

const (
	TypeConvention MemoryType = "convention"
	TypeDecision   MemoryType = "decision"
	TypePreference MemoryType = "preference"
	TypeFact       MemoryType = "fact"
	TypeCorrection MemoryType = "correction"
	TypeSkill      MemoryType = "skill"
)

// MemoryEntry wraps the sqlc Memory model with typed methods.
type MemoryEntry struct {
	ID         string
	Project    string
	Type       MemoryType
	Key        string
	Value      string
	Source     string
	Confidence float64
	UsageCount int64
	LastUsedAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// FromDB converts a sqlc Memory to a MemoryEntry.
func FromDB(m db.Memory) MemoryEntry {
	var lastUsed time.Time
	if m.LastUsedAt.Valid {
		lastUsed, _ = time.Parse(time.RFC3339, m.LastUsedAt.String)
	}
	createdAt, _ := time.Parse(time.RFC3339, m.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, m.UpdatedAt)

	return MemoryEntry{
		ID:         m.ID,
		Project:    m.Project,
		Type:       MemoryType(m.Type),
		Key:        m.Key,
		Value:      m.Value,
		Source:     m.Source,
		Confidence: m.Confidence,
		UsageCount: m.UsageCount,
		LastUsedAt: lastUsed,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

// ToDB converts a MemoryEntry to sqlc CreateMemoryParams.
func (e *MemoryEntry) ToDB() db.CreateMemoryParams {
	return db.CreateMemoryParams{
		ID:         e.ID,
		Project:    e.Project,
		Type:       string(e.Type),
		Key:        e.Key,
		Value:      e.Value,
		Source:     e.Source,
		Confidence: e.Confidence,
	}
}

// AllTypes returns all memory types.
func AllTypes() []MemoryType {
	return []MemoryType{
		TypeConvention,
		TypeDecision,
		TypePreference,
		TypeFact,
		TypeCorrection,
		TypeSkill,
	}
}
