package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourname/helm/internal/db"
)

// Archive manages session file storage and parsing from provider JSONL files.
type Archive struct {
	manager *Manager
	parsers map[string]Parser
}

// NewArchive creates a session archive with registered parsers.
func NewArchive(manager *Manager) *Archive {
	a := &Archive{
		manager: manager,
		parsers: make(map[string]Parser),
	}
	a.RegisterParser(NewClaudeParser())
	a.RegisterParser(NewCodexParser())
	a.RegisterParser(NewGeminiParser())
	a.RegisterParser(NewOpenCodeParser())
	return a
}

// RegisterParser adds a parser for a provider.
func (a *Archive) RegisterParser(p Parser) {
	a.parsers[p.Provider()] = p
}

// Ingest parses a JSONL file and stores the session in the database.
func (a *Archive) Ingest(ctx context.Context, path, project string) (*Session, error) {
	parser, err := a.detectParser(path)
	if err != nil {
		return nil, err
	}

	sess, _, err := parser.ParseFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	sess.Project = project
	sess.RawPath = path

	if sess.ID == "" {
		sess.ID = filepath.Base(path)
		sess.ID = strings.TrimSuffix(sess.ID, filepath.Ext(sess.ID))
	}

	if err := a.manager.Create(ctx, sess); err != nil {
		return nil, fmt.Errorf("store session: %w", err)
	}

	return sess, nil
}

// IngestDir scans a directory for JSONL files and ingests them.
func (a *Archive) IngestDir(ctx context.Context, dir, project string) ([]*Session, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var sessions []*Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(dir, e.Name())
		sess, err := a.Ingest(ctx, path, project)
		if err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// Search finds sessions matching a query.
func (a *Archive) Search(ctx context.Context, project, query string, limit int64) ([]Session, error) {
	sessions, err := a.manager.q.SearchSessions(ctx, db.SearchSessionsParams{
		Column1: nullStr(query),
		Column2: nullStr(query),
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search sessions: %w", err)
	}

	result := make([]Session, len(sessions))
	for i, s := range sessions {
		result[i] = FromDB(s)
	}
	return result, nil
}

func (a *Archive) detectParser(path string) (Parser, error) {
	name := strings.ToLower(filepath.Base(path))

	for provider, parser := range a.parsers {
		if strings.Contains(name, provider) {
			return parser, nil
		}
	}

	if strings.Contains(name, "claude") {
		return a.parsers["anthropic"], nil
	}
	if strings.Contains(name, "codex") || strings.Contains(name, "openai") {
		return a.parsers["openai"], nil
	}
	if strings.Contains(name, "gemini") || strings.Contains(name, "google") {
		return a.parsers["google"], nil
	}

	return nil, fmt.Errorf("no parser found for: %s", path)
}
