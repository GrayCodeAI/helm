// Package mistake provides mistake tracking and learning capabilities
package mistake

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/helm/internal/db"
)

// DBQuerier defines the database operations needed by the mistake journal
type DBQuerier interface {
	CreateMistake(ctx context.Context, arg db.CreateMistakeParams) (sql.Result, error)
	ListMistakes(ctx context.Context, sessionID string) ([]db.Mistake, error)
	ListMistakesByType(ctx context.Context, arg db.ListMistakesByTypeParams) ([]db.Mistake, error)
	CountMistakesByType(ctx context.Context, type_ string) (int64, error)
}

// Type categorizes different types of mistakes
type Type string

const (
	TypeRejectedDiff   Type = "rejected_diff"
	TypeTestFailure    Type = "test_failure"
	TypeLintError      Type = "lint_error"
	TypeCompileError   Type = "compile_error"
	TypeTimeout        Type = "timeout"
	TypeLoopDetected   Type = "loop_detected"
	TypeWrongFile      Type = "wrong_file"
	TypeRuntimeError   Type = "runtime_error"
	TypeSecurityIssue  Type = "security_issue"
)

// Entry represents a single mistake in the journal
type Entry struct {
	ID          string
	SessionID   string
	Type        Type
	Description string
	Context     string
	Correction  string
	FilePath    string
	CreatedAt   time.Time
}

// Journal tracks and manages mistakes
type Journal struct {
	q DBQuerier
}

// NewJournal creates a new mistake journal
func NewJournal(q db.Querier) *Journal {
	return &Journal{q: q}
}

// Record adds a new mistake to the journal
func (j *Journal) Record(ctx context.Context, sessionID string, mistakeType Type, description, context, correction, filePath string) (*Entry, error) {
	entry := &Entry{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Type:        mistakeType,
		Description: description,
		Context:     context,
		Correction:  correction,
		FilePath:    filePath,
		CreatedAt:   time.Now(),
	}

	_, err := j.q.CreateMistake(ctx, db.CreateMistakeParams{
		ID:          entry.ID,
		SessionID:   entry.SessionID,
		Type:        string(entry.Type),
		Description: entry.Description,
		Context:     toNullString(entry.Context),
		Correction:  toNullString(entry.Correction),
		FilePath:    toNullString(entry.FilePath),
	})
	if err != nil {
		return nil, fmt.Errorf("create mistake: %w", err)
	}

	return entry, nil
}

// List returns all mistakes for a session
func (j *Journal) List(ctx context.Context, sessionID string) ([]*Entry, error) {
	mistakes, err := j.q.ListMistakes(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list mistakes: %w", err)
	}

	entries := make([]*Entry, len(mistakes))
	for i, m := range mistakes {
		entries[i] = fromDB(m)
	}

	return entries, nil
}

// ListByType returns mistakes filtered by type
func (j *Journal) ListByType(ctx context.Context, mistakeType Type) ([]*Entry, error) {
	mistakes, err := j.q.ListMistakesByType(ctx, db.ListMistakesByTypeParams{
		Type:  string(mistakeType),
		Limit: 1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list mistakes by type: %w", err)
	}

	entries := make([]*Entry, len(mistakes))
	for i, m := range mistakes {
		entries[i] = fromDB(m)
	}

	return entries, nil
}

// FindSimilar searches for similar past mistakes
func (j *Journal) FindSimilar(ctx context.Context, filePath, description string, limit int) ([]*Entry, error) {
	// Get all mistakes and filter
	// Note: This is a simplified implementation
	allMistakes := make(map[string][]db.Mistake)

	// Query by type to get recent mistakes
	for _, t := range []Type{TypeRejectedDiff, TypeTestFailure, TypeCompileError} {
		mistakes, err := j.ListByType(ctx, t)
		if err == nil {
			for _, m := range mistakes {
				allMistakes[m.SessionID] = append(allMistakes[m.SessionID], db.Mistake{
					ID:          m.ID,
					SessionID:   m.SessionID,
					Type:        string(m.Type),
					Description: m.Description,
					Context:     toNullString(m.Context),
					Correction:  toNullString(m.Correction),
					FilePath:    toNullString(m.FilePath),
				})
			}
		}
	}

	var similar []*Entry
	for _, mistakes := range allMistakes {
		for _, m := range mistakes {
			// Simple similarity check based on file path
			if m.FilePath.Valid && m.FilePath.String == filePath {
				similar = append(similar, fromDB(m))
				continue
			}
			// Check for description overlap
			if hasOverlap(m.Description, description) {
				similar = append(similar, fromDB(m))
			}
		}
	}

	if len(similar) > limit {
		similar = similar[:limit]
	}

	return similar, nil
}

// Stats returns statistics about mistakes
func (j *Journal) Stats(ctx context.Context) (map[Type]int64, error) {
	stats := make(map[Type]int64)

	// Count by type
	for _, t := range []Type{TypeRejectedDiff, TypeTestFailure, TypeLintError, TypeCompileError, TypeTimeout} {
		count, err := j.q.CountMistakesByType(ctx, string(t))
		if err == nil {
			stats[t] = count
		}
	}

	return stats, nil
}

func fromDB(m db.Mistake) *Entry {
	createdAt, _ := time.Parse(time.RFC3339, m.CreatedAt)
	return &Entry{
		ID:          m.ID,
		SessionID:   m.SessionID,
		Type:        Type(m.Type),
		Description: m.Description,
		Context:     fromNullString(m.Context),
		Correction:  fromNullString(m.Correction),
		FilePath:    fromNullString(m.FilePath),
		CreatedAt:   createdAt,
	}
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func fromNullString(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func hasOverlap(a, b string) bool {
	// Simple word overlap check
	wordsA := make(map[string]bool)
	for _, w := range splitWords(a) {
		wordsA[w] = true
	}

	overlap := 0
	for _, w := range splitWords(b) {
		if wordsA[w] {
			overlap++
		}
	}

	return overlap >= 3
}

func splitWords(s string) []string {
	var words []string
	var current []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			current = append(current, r)
		} else if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// dbQuerier interface for database operations
type dbQuerier interface {
	CreateMistake(ctx context.Context, arg db.CreateMistakeParams) (sql.Result, error)
	ListMistakes(ctx context.Context, sessionID string) ([]db.Mistake, error)
	ListMistakesByType(ctx context.Context, arg db.ListMistakesByTypeParams) ([]db.Mistake, error)
	CountMistakesByType(ctx context.Context, type_ string) (int64, error)
}
