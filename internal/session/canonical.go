package session

import (
	"database/sql"
	"strings"
	"time"

	"github.com/yourname/helm/internal/db"
)

// Session is the canonical, provider-agnostic session model.
type Session struct {
	ID               string
	Provider         string
	Model            string
	Project          string
	Prompt           string
	Status           string
	StartedAt        time.Time
	EndedAt          time.Time
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	Cost             float64
	Summary          string
	Tags             []string
	RawPath          string
}

// Message is a canonical message within a session.
type Message struct {
	SessionID string
	Role      string
	Content   string
	ToolCalls []ToolCall
	Timestamp time.Time
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
	Result    string
}

// FromDB converts a sqlc Session to canonical Session.
func FromDB(s db.Session) Session {
	var prompt, summary, rawPath string
	if s.Prompt.Valid {
		prompt = s.Prompt.String
	}
	if s.Summary.Valid {
		summary = s.Summary.String
	}
	if s.RawPath.Valid {
		rawPath = s.RawPath.String
	}

	startedAt, _ := time.Parse(time.RFC3339, s.StartedAt)
	var endedAt time.Time
	if s.EndedAt.Valid {
		endedAt, _ = time.Parse(time.RFC3339, s.EndedAt.String)
	}

	var tags []string
	if s.Tags.Valid && s.Tags.String != "" {
		tags = strings.Split(s.Tags.String, ",")
	}

	return Session{
		ID:               s.ID,
		Provider:         s.Provider,
		Model:            s.Model,
		Project:          s.Project,
		Prompt:           prompt,
		Status:           s.Status,
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		InputTokens:      s.InputTokens,
		OutputTokens:     s.OutputTokens,
		CacheReadTokens:  s.CacheReadTokens,
		CacheWriteTokens: s.CacheWriteTokens,
		Cost:             s.Cost,
		Summary:          summary,
		Tags:             tags,
		RawPath:          rawPath,
	}
}

// ToDB converts canonical Session to sqlc CreateSessionParams.
func (s *Session) ToDB() db.CreateSessionParams {
	return db.CreateSessionParams{
		ID:               s.ID,
		Provider:         s.Provider,
		Model:            s.Model,
		Project:          s.Project,
		Prompt:           nullStr(s.Prompt),
		Status:           s.Status,
		InputTokens:      s.InputTokens,
		OutputTokens:     s.OutputTokens,
		CacheReadTokens:  s.CacheReadTokens,
		CacheWriteTokens: s.CacheWriteTokens,
		Cost:             s.Cost,
		Summary:          nullStr(s.Summary),
		Tags:             nullStr(strings.Join(s.Tags, ",")),
		RawPath:          nullStr(s.RawPath),
	}
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
