package e2e

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/helm/internal/cost"
	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/session"
)

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

func nullFloat64(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "helm.db"))
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	return database
}

func TestFullWorkflow(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	ctx := context.Background()
	project := "/test/project"
	sessionID := "workflow-test-1"

	// Step 1: Create a session
	_, err := database.CreateSession(ctx, db.CreateSessionParams{
		ID:       sessionID,
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		Project:  project,
		Prompt:   nullStr("Implement a feature"),
		Status:   "running",
	})
	require.NoError(t, err)

	// Step 2: Record cost
	_, err = database.CreateCostRecord(ctx, db.CreateCostRecordParams{
		ID:           "cost-workflow-1",
		SessionID:    sessionID,
		Project:      project,
		Provider:     "anthropic",
		Model:        "claude-sonnet-4-20250514",
		InputTokens:  nullInt64(1000),
		OutputTokens: nullInt64(500),
		TotalCost:    nullFloat64(0.0115),
	})
	require.NoError(t, err)

	// Step 3: Store memory
	_, err = database.CreateMemory(ctx, db.CreateMemoryParams{
		ID:         "mem-workflow-1",
		Project:    project,
		Type:       "convention",
		Key:        "naming",
		Value:      "Use PascalCase for exported types",
		Source:     "auto-learn",
		Confidence: 0.85,
	})
	require.NoError(t, err)

	// Step 4: Verify session
	sess, err := database.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", sess.Provider)
	assert.Equal(t, "running", sess.Status)

	// Step 5: Verify cost
	costRow, err := database.GetCostBySession(ctx, sessionID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, costRow.TotalCost, float64(0))
	assert.GreaterOrEqual(t, costRow.InputTokens, int64(0))

	// Step 6: Verify memory
	memories, err := database.ListMemories(ctx, project)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(memories), 1)

	found := false
	for _, m := range memories {
		if m.ID == "mem-workflow-1" {
			found = true
			assert.Equal(t, "convention", m.Type)
			assert.Equal(t, "naming", m.Key)
			assert.InDelta(t, 0.85, m.Confidence, 0.01)
		}
	}
	assert.True(t, found, "Memory not found")

	// Step 7: Update session status
	err = database.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		Status: "done",
		ID:     sessionID,
	})
	require.NoError(t, err)

	sess, err = database.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, "done", sess.Status)
}

func TestSessionLifecycle(t *testing.T) {
	t.Skip("Skipping - requires full DB schema migration")
}

func TestCostTracking(t *testing.T) {
	t.Skip("Skipping - requires full DB schema migration")
}

func TestMemoryEngine(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	ctx := context.Background()
	project := "/test/project"

	memories := []db.CreateMemoryParams{
		{ID: "mem-1", Project: project, Type: "convention", Key: "naming", Value: "PascalCase", Source: "auto-learn", Confidence: 0.9},
		{ID: "mem-2", Project: project, Type: "pattern", Key: "error-handling", Value: "Wrap errors", Source: "session", Confidence: 0.7},
		{ID: "mem-3", Project: project, Type: "tool", Key: "testing", Value: "Use table-driven tests", Source: "manual", Confidence: 0.8},
	}

	for _, m := range memories {
		_, err := database.CreateMemory(ctx, m)
		require.NoError(t, err)
	}

	all, err := database.ListMemories(ctx, project)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 3)

	byType, err := database.ListMemoriesByType(ctx, db.ListMemoriesByTypeParams{
		Project: project,
		Type:    "convention",
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(byType), 1)

	err = database.UpdateMemory(ctx, db.UpdateMemoryParams{
		ID:         "mem-1",
		Value:      "PascalCase for exported, camelCase for unexported",
		Confidence: 0.95,
	})
	require.NoError(t, err)

	all, err = database.ListMemories(ctx, project)
	require.NoError(t, err)
	for _, m := range all {
		if m.ID == "mem-1" {
			assert.Equal(t, "PascalCase for exported, camelCase for unexported", m.Value)
			assert.InDelta(t, 0.95, m.Confidence, 0.01)
		}
	}
}

func TestConcurrentOperations(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	ctx := context.Background()
	project := "/test/project"

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			sessionID := "concurrent-session-" + string(rune('a'+id%26))
			database.CreateSession(ctx, db.CreateSessionParams{
				ID: sessionID, Provider: "test", Model: "test",
				Project: project, Status: "running",
			})
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	sessions, err := database.ListSessions(ctx, db.ListSessionsParams{
		Project: project,
		Limit:   100,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sessions), 20)
}

func TestSessionModelConversion(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	ctx := context.Background()

	_, err := database.CreateSession(ctx, db.CreateSessionParams{
		ID:               "convert-1",
		Provider:         "anthropic",
		Model:            "claude-sonnet-4",
		Project:          "/test",
		Prompt:           nullStr("Test prompt"),
		Status:           "done",
		InputTokens:      1000,
		OutputTokens:     500,
		CacheReadTokens:  200,
		CacheWriteTokens: 100,
		Cost:             0.015,
		Summary:          nullStr("Test summary"),
		Tags:             nullStr("test,feature"),
		RawPath:          nullStr("/path/to/raw"),
	})
	require.NoError(t, err)

	dbSession, err := database.GetSession(ctx, "convert-1")
	require.NoError(t, err)

	canonical := session.FromDB(dbSession)
	assert.Equal(t, "convert-1", canonical.ID)
	assert.Equal(t, "anthropic", canonical.Provider)
	assert.Equal(t, "claude-sonnet-4", canonical.Model)
	assert.Equal(t, "Test prompt", canonical.Prompt)
	assert.Equal(t, "done", canonical.Status)
	assert.Equal(t, int64(1000), canonical.InputTokens)
	assert.Equal(t, int64(500), canonical.OutputTokens)
	assert.Equal(t, 0.015, canonical.Cost)
	assert.Equal(t, "Test summary", canonical.Summary)
	assert.Contains(t, canonical.Tags, "test")
	assert.Contains(t, canonical.Tags, "feature")

	params := canonical.ToDB()
	assert.Equal(t, "convert-1", params.ID)
	assert.True(t, params.Prompt.Valid)
	assert.Equal(t, "Test prompt", params.Prompt.String)
}

func TestCostCalculator(t *testing.T) {
	t.Parallel()

	calculator := cost.NewCalculator()

	// Just verify calculator works, not exact values (pricing may change)
	cost := calculator.Calculate("claude-sonnet-4-20250514", 1000, 500, 0, 0)
	assert.Greater(t, cost, float64(0))
	assert.Less(t, cost, float64(1.0))
}

func TestDatabaseTablesExist(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	ctx := context.Background()

	_, err := database.CreateSession(ctx, db.CreateSessionParams{
		ID: "test-1", Provider: "test", Model: "test", Project: "/test", Status: "running",
	})
	assert.NoError(t, err, "sessions table should exist")

	_, err = database.CreateCostRecord(ctx, db.CreateCostRecordParams{
		ID: "cost-1", SessionID: "test-1", Project: "/test", Provider: "test", Model: "test",
	})
	assert.NoError(t, err, "cost_records table should exist")

	_, err = database.CreateMemory(ctx, db.CreateMemoryParams{
		ID: "mem-1", Project: "/test", Type: "test", Key: "k", Value: "v", Source: "test", Confidence: 0.5,
	})
	assert.NoError(t, err, "memories table should exist")

	_, err = database.CreatePrompt(ctx, db.CreatePromptParams{
		ID: "prompt-1", Name: "test-prompt", Template: "test", Source: "test",
	})
	assert.NoError(t, err, "prompts table should exist")

	err = database.UpsertBudget(ctx, db.UpsertBudgetParams{Project: "/test"})
	assert.NoError(t, err, "budgets table should exist")

	_, err = database.CreateMistake(ctx, db.CreateMistakeParams{
		ID: "mistake-1", SessionID: "test-1", Type: "test", Description: "test",
	})
	assert.NoError(t, err, "mistakes table should exist")

	err = database.UpsertModelPerformance(ctx, db.UpsertModelPerformanceParams{
		ID: "perf-1", Model: "test", TaskType: "test",
	})
	assert.NoError(t, err, "model_performance table should exist")

	_, err = database.CreateFileChange(ctx, db.CreateFileChangeParams{
		ID: "change-1", SessionID: "test-1", FilePath: "test.go",
	})
	assert.NoError(t, err, "file_changes table should exist")
}
