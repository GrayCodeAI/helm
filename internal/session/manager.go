// Package session provides session management and forking capabilities.
package session

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourname/helm/internal/db"
)

// SessionQuerier is the DB interface for sessions.
type SessionQuerier interface {
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (sql.Result, error)
	GetSession(ctx context.Context, id string) (db.Session, error)
	ListSessions(ctx context.Context, arg db.ListSessionsParams) ([]db.Session, error)
	ListRecentSessions(ctx context.Context, limit int64) ([]db.Session, error)
	ListSessionsByStatus(ctx context.Context, arg db.ListSessionsByStatusParams) ([]db.Session, error)
	UpdateSessionStatus(ctx context.Context, arg db.UpdateSessionStatusParams) error
	UpdateSessionCost(ctx context.Context, arg db.UpdateSessionCostParams) error
	UpdateSessionSummary(ctx context.Context, arg db.UpdateSessionSummaryParams) error
	DeleteSession(ctx context.Context, id string) error
	CountSessions(ctx context.Context, project string) (int64, error)
	SearchSessions(ctx context.Context, arg db.SearchSessionsParams) ([]db.Session, error)
	CreateMessage(ctx context.Context, arg db.CreateMessageParams) (sql.Result, error)
	GetMessagesBySession(ctx context.Context, sessionID string) ([]db.Message, error)
}

// Manager handles session CRUD operations.
type Manager struct {
	q SessionQuerier
}

// NewManager creates a new session manager.
func NewManager(q SessionQuerier) *Manager {
	return &Manager{q: q}
}

// Create starts a new session.
func (m *Manager) Create(ctx context.Context, sess *Session) error {
	if sess.ID == "" {
		sess.ID = uuid.New().String()
	}
	if sess.Status == "" {
		sess.Status = "running"
	}

	_, err := m.q.CreateSession(ctx, sess.ToDB())
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// Get retrieves a session by ID.
func (m *Manager) Get(ctx context.Context, id string) (*Session, error) {
	s, err := m.q.GetSession(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	sess := FromDB(s)
	return &sess, nil
}

// List returns sessions for a project with pagination.
func (m *Manager) List(ctx context.Context, project string, limit, offset int64) ([]Session, error) {
	sessions, err := m.q.ListSessions(ctx, db.ListSessionsParams{
		Project: project,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	result := make([]Session, len(sessions))
	for i, s := range sessions {
		result[i] = FromDB(s)
	}
	return result, nil
}

// Recent returns the most recent sessions across all projects.
func (m *Manager) Recent(ctx context.Context, limit int64) ([]Session, error) {
	sessions, err := m.q.ListRecentSessions(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("recent sessions: %w", err)
	}

	result := make([]Session, len(sessions))
	for i, s := range sessions {
		result[i] = FromDB(s)
	}
	return result, nil
}

// UpdateStatus changes a session's status.
func (m *Manager) UpdateStatus(ctx context.Context, id, status string) error {
	err := m.q.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// UpdateCost updates token counts and cost for a session.
func (m *Manager) UpdateCost(ctx context.Context, id string, input, output, cacheRead, cacheWrite int64, cost float64) error {
	err := m.q.UpdateSessionCost(ctx, db.UpdateSessionCostParams{
		ID:               id,
		InputTokens:      input,
		OutputTokens:     output,
		CacheReadTokens:  cacheRead,
		CacheWriteTokens: cacheWrite,
		Cost:             cost,
	})
	if err != nil {
		return fmt.Errorf("update cost: %w", err)
	}
	return nil
}

// UpdateSummary sets the AI-generated summary.
func (m *Manager) UpdateSummary(ctx context.Context, id, summary string) error {
	err := m.q.UpdateSessionSummary(ctx, db.UpdateSessionSummaryParams{
		ID:      id,
		Summary: nullStr(summary),
	})
	if err != nil {
		return fmt.Errorf("update summary: %w", err)
	}
	return nil
}

// Delete removes a session and its messages.
func (m *Manager) Delete(ctx context.Context, id string) error {
	err := m.q.DeleteSession(ctx, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// Count returns the number of sessions for a project.
func (m *Manager) Count(ctx context.Context, project string) (int64, error) {
	count, err := m.q.CountSessions(ctx, project)
	if err != nil {
		return 0, fmt.Errorf("count sessions: %w", err)
	}
	return count, nil
}
