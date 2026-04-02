// Package search provides full-text session search using FTS5.
package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Result represents a search result
type Result struct {
	SessionID string
	Score     float64
	Snippet   string
	MatchType string // session, message, content
}

// Engine provides full-text search
type Engine struct {
	db *sql.DB
}

// NewEngine creates a new search engine
func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// SearchSessions searches sessions by query
func (e *Engine) SearchSessions(ctx context.Context, query string, project string, limit int) ([]Result, error) {
	// Use FTS5 if available, fallback to LIKE
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	// Try FTS5 search first
	results, err := e.searchFTS5(ctx, query, project, limit)
	if err == nil {
		return results, nil
	}

	// Fallback to LIKE search
	return e.searchLike(ctx, query, project, limit)
}

func (e *Engine) searchFTS5(ctx context.Context, query, project string, limit int) ([]Result, error) {
	queryStr := `
		SELECT session_id, rank, snippet(sessions_fts, 2, '<b>', '</b>', '...', 30) as snippet
		FROM sessions_fts
		WHERE sessions_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := e.db.QueryContext(ctx, queryStr, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var score float64
		if err := rows.Scan(&r.SessionID, &score, &r.Snippet); err != nil {
			continue
		}
		r.Score = score
		r.MatchType = "session"
		results = append(results, r)
	}

	return results, nil
}

func (e *Engine) searchLike(ctx context.Context, query, project string, limit int) ([]Result, error) {
	likeQuery := "%" + query + "%"

	queryStr := `
		SELECT id, prompt, status, model
		FROM sessions
		WHERE (prompt LIKE ? OR status LIKE ? OR model LIKE ?)
		AND project = ?
		LIMIT ?
	`

	rows, err := e.db.QueryContext(ctx, queryStr, likeQuery, likeQuery, likeQuery, project, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var prompt, status, model sql.NullString
		if err := rows.Scan(&r.SessionID, &prompt, &status, &model); err != nil {
			continue
		}

		if prompt.Valid && strings.Contains(strings.ToLower(prompt.String), strings.ToLower(query)) {
			r.Snippet = prompt.String
			r.MatchType = "session"
		} else if model.Valid {
			r.Snippet = model.String
			r.MatchType = "session"
		}

		r.Score = 1.0
		results = append(results, r)
	}

	return results, nil
}

// SearchMessages searches messages by query
func (e *Engine) SearchMessages(ctx context.Context, query string, sessionID string, limit int) ([]Result, error) {
	likeQuery := "%" + query + "%"

	queryStr := `
		SELECT id, content, role
		FROM messages
		WHERE content LIKE ?
		AND session_id = ?
		LIMIT ?
	`

	rows, err := e.db.QueryContext(ctx, queryStr, likeQuery, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		var content, role sql.NullString
		if err := rows.Scan(&r.SessionID, &content, &role); err != nil {
			continue
		}

		if content.Valid {
			r.Snippet = content.String
			if len(r.Snippet) > 100 {
				r.Snippet = r.Snippet[:100] + "..."
			}
		}
		r.MatchType = "message"
		r.Score = 1.0
		results = append(results, r)
	}

	return results, nil
}

// CreateFTS5Tables creates FTS5 virtual tables
func (e *Engine) CreateFTS5Tables() error {
	queries := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS sessions_fts USING fts5(
			id, project, prompt, status, model, provider,
			content='sessions', content_rowid='rowid'
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
			id, session_id, content, role,
			content='messages', content_rowid='rowid'
		)`,
	}

	for _, q := range queries {
		if _, err := e.db.Exec(q); err != nil {
			return fmt.Errorf("create FTS5 table: %w", err)
		}
	}

	return nil
}
