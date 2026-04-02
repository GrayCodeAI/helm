package session

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/helm/internal/db"
)

type mockSessionQuerier struct {
	sessions map[string]db.Session
	messages map[string][]db.Message
}

func newMockSessionQuerier() *mockSessionQuerier {
	return &mockSessionQuerier{
		sessions: make(map[string]db.Session),
		messages: make(map[string][]db.Message),
	}
}

func (m *mockSessionQuerier) CreateSession(ctx context.Context, arg db.CreateSessionParams) (sql.Result, error) {
	m.sessions[arg.ID] = db.Session{
		ID:       arg.ID,
		Provider: arg.Provider,
		Model:    arg.Model,
		Project:  arg.Project,
		Prompt:   arg.Prompt,
		Status:   arg.Status,
		Cost:     arg.Cost,
	}
	return nil, nil
}

func (m *mockSessionQuerier) GetSession(ctx context.Context, id string) (db.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return db.Session{}, sql.ErrNoRows
	}
	return s, nil
}

func (m *mockSessionQuerier) ListSessions(ctx context.Context, arg db.ListSessionsParams) ([]db.Session, error) {
	var result []db.Session
	for _, s := range m.sessions {
		if s.Project == arg.Project {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSessionQuerier) ListRecentSessions(ctx context.Context, limit int64) ([]db.Session, error) {
	var result []db.Session
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockSessionQuerier) ListSessionsByStatus(ctx context.Context, arg db.ListSessionsByStatusParams) ([]db.Session, error) {
	var result []db.Session
	for _, s := range m.sessions {
		if s.Status == arg.Status {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockSessionQuerier) UpdateSessionStatus(ctx context.Context, arg db.UpdateSessionStatusParams) error {
	if s, ok := m.sessions[arg.ID]; ok {
		s.Status = arg.Status
		m.sessions[arg.ID] = s
	}
	return nil
}

func (m *mockSessionQuerier) UpdateSessionCost(ctx context.Context, arg db.UpdateSessionCostParams) error {
	if s, ok := m.sessions[arg.ID]; ok {
		s.InputTokens = arg.InputTokens
		s.OutputTokens = arg.OutputTokens
		s.Cost = arg.Cost
		m.sessions[arg.ID] = s
	}
	return nil
}

func (m *mockSessionQuerier) UpdateSessionSummary(ctx context.Context, arg db.UpdateSessionSummaryParams) error {
	if s, ok := m.sessions[arg.ID]; ok {
		s.Summary = arg.Summary
		m.sessions[arg.ID] = s
	}
	return nil
}

func (m *mockSessionQuerier) DeleteSession(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionQuerier) CountSessions(ctx context.Context, project string) (int64, error) {
	var count int64
	for _, s := range m.sessions {
		if s.Project == project {
			count++
		}
	}
	return count, nil
}

func (m *mockSessionQuerier) SearchSessions(ctx context.Context, arg db.SearchSessionsParams) ([]db.Session, error) {
	var result []db.Session
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockSessionQuerier) CreateMessage(ctx context.Context, arg db.CreateMessageParams) (sql.Result, error) {
	m.messages[arg.SessionID] = append(m.messages[arg.SessionID], db.Message{
		SessionID: arg.SessionID,
		Role:      arg.Role,
		Content:   arg.Content,
	})
	return nil, nil
}

func (m *mockSessionQuerier) GetMessagesBySession(ctx context.Context, sessionID string) ([]db.Message, error) {
	return m.messages[sessionID], nil
}

func TestSessionManagerCRUD(t *testing.T) {
	t.Parallel()
	q := newMockSessionQuerier()
	m := NewManager(q)
	ctx := context.Background()

	sess := &Session{
		ID:       "test-1",
		Provider: "anthropic",
		Model:    "claude-sonnet-4",
		Project:  "/project",
		Prompt:   "add auth feature",
		Status:   "running",
	}

	require.NoError(t, m.Create(ctx, sess))

	got, err := m.Get(ctx, "test-1")
	require.NoError(t, err)
	assert.Equal(t, "anthropic", got.Provider)
	assert.Equal(t, "running", got.Status)

	require.NoError(t, m.UpdateStatus(ctx, "test-1", "done"))
	got, err = m.Get(ctx, "test-1")
	require.NoError(t, err)
	assert.Equal(t, "done", got.Status)

	require.NoError(t, m.Delete(ctx, "test-1"))
	_, err = m.Get(ctx, "test-1")
	require.Error(t, err)
}

func TestSessionManagerList(t *testing.T) {
	t.Parallel()
	q := newMockSessionQuerier()
	m := NewManager(q)
	ctx := context.Background()

	require.NoError(t, m.Create(ctx, &Session{ID: "s1", Project: "/p", Status: "done"}))
	require.NoError(t, m.Create(ctx, &Session{ID: "s2", Project: "/p", Status: "running"}))

	sessions, err := m.List(ctx, "/p", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, len(sessions))
}

func TestSessionManagerCount(t *testing.T) {
	t.Parallel()
	q := newMockSessionQuerier()
	m := NewManager(q)
	ctx := context.Background()

	require.NoError(t, m.Create(ctx, &Session{ID: "s1", Project: "/p"}))
	require.NoError(t, m.Create(ctx, &Session{ID: "s2", Project: "/p"}))

	count, err := m.Count(ctx, "/p")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestClaudeParser(t *testing.T) {
	t.Parallel()

	content := `{"type":"message","message":{"role":"user","content":"fix the bug"},"timestamp":"2026-01-01T00:00:00Z"}
{"type":"message","message":{"role":"assistant","content":"I'll fix it"},"timestamp":"2026-01-01T00:00:01Z"}
{"type":"result","status":"done","usage":{"input_tokens":100,"output_tokens":50}}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	p := NewClaudeParser()
	sess, messages, err := p.ParseFile(path)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", sess.Provider)
	assert.Equal(t, "done", sess.Status)
	assert.Equal(t, "fix the bug", sess.Prompt)
	assert.Equal(t, int64(100), sess.InputTokens)
	assert.Equal(t, int64(50), sess.OutputTokens)
	assert.Equal(t, 2, len(messages))
}

func TestCodexParser(t *testing.T) {
	t.Parallel()

	content := `{"role":"user","content":"add feature"}
{"role":"assistant","content":"done"}
{"type":"usage","usage":{"prompt_tokens":200,"completion_tokens":100}}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "codex-session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	p := NewCodexParser()
	sess, messages, err := p.ParseFile(path)
	require.NoError(t, err)
	assert.Equal(t, "openai", sess.Provider)
	assert.Equal(t, "add feature", sess.Prompt)
	assert.Equal(t, int64(200), sess.InputTokens)
	assert.Equal(t, int64(100), sess.OutputTokens)
	assert.Equal(t, 2, len(messages))
}

func TestGeminiParser(t *testing.T) {
	t.Parallel()

	content := `{"role":"user","parts":[{"text":"hello"}]}
{"role":"model","parts":[{"text":"hi there"}],"usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":25}}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "gemini-session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	p := NewGeminiParser()
	sess, messages, err := p.ParseFile(path)
	require.NoError(t, err)
	assert.Equal(t, "google", sess.Provider)
	assert.Equal(t, "hello", sess.Prompt)
	assert.Equal(t, int64(50), sess.InputTokens)
	assert.Equal(t, int64(25), sess.OutputTokens)
	assert.Equal(t, 2, len(messages))
}

func TestOpenCodeParser(t *testing.T) {
	t.Parallel()

	content := `{"role":"user","content":"test me"}
{"role":"assistant","content":"ok"}
{"status":"done","model":"qwen2.5"}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "opencode-session.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	p := NewOpenCodeParser()
	sess, messages, err := p.ParseFile(path)
	require.NoError(t, err)
	assert.Equal(t, "opencode", sess.Provider)
	assert.Equal(t, "test me", sess.Prompt)
	assert.Equal(t, "done", sess.Status)
	assert.Equal(t, 2, len(messages))
}

func TestFromDB(t *testing.T) {
	t.Parallel()

	dbSession := db.Session{
		ID:           "s1",
		Provider:     "anthropic",
		Model:        "claude-sonnet-4",
		Project:      "/project",
		Status:       "done",
		InputTokens:  100,
		OutputTokens: 50,
		Cost:         0.01,
		StartedAt:    time.Now().Format(time.RFC3339),
	}

	sess := FromDB(dbSession)
	assert.Equal(t, "s1", sess.ID)
	assert.Equal(t, "anthropic", sess.Provider)
	assert.Equal(t, int64(100), sess.InputTokens)
}

func TestSessionToDB(t *testing.T) {
	t.Parallel()

	sess := &Session{
		ID:           "s1",
		Provider:     "openai",
		Model:        "gpt-4o",
		Project:      "/project",
		Prompt:       "hello",
		Status:       "running",
		InputTokens:  100,
		OutputTokens: 50,
		Cost:         0.01,
		Tags:         []string{"test", "feature"},
	}

	params := sess.ToDB()
	assert.Equal(t, "s1", params.ID)
	assert.Equal(t, "openai", params.Provider)
	assert.True(t, params.Prompt.Valid)
	assert.Equal(t, "hello", params.Prompt.String)
}

func TestArchiveIngest(t *testing.T) {
	t.Parallel()

	content := `{"type":"message","message":{"role":"user","content":"fix bug"},"timestamp":"2026-01-01T00:00:00Z"}
{"type":"result","status":"done","usage":{"input_tokens":100,"output_tokens":50}}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-test.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	q := newMockSessionQuerier()
	m := NewManager(q)
	a := NewArchive(m)

	sess, err := a.Ingest(context.Background(), path, "/project")
	require.NoError(t, err)
	assert.Equal(t, "anthropic", sess.Provider)
	assert.Equal(t, "/project", sess.Project)
}
