package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/helm/internal/config"
	"github.com/yourname/helm/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.Open(dbPath)
	require.NoError(t, err)

	return database, dbPath
}

func TestRootCommand(t *testing.T) {
	t.Parallel()
	cmd := rootCmd
	assert.Equal(t, "helm", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()
	cmd := versionCmd
	assert.Equal(t, "version", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestStatusCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "status", statusCmd.Use)
	assert.NotEmpty(t, statusCmd.Short)
}

func TestMemoryCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "memory", memoryCmd.Use)
	assert.NotEmpty(t, memoryCmd.Short)
}

func TestSessionCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "session", sessionCmd.Use)
	assert.NotEmpty(t, sessionCmd.Short)
}

func TestPromptsCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "prompts", promptCmd.Use)
	assert.NotEmpty(t, promptCmd.Short)
}

func TestReportCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "report", reportCmd.Use)
	assert.NotEmpty(t, reportCmd.Short)
	assert.NotNil(t, reportCmd.Commands())
}

func TestExportCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "export", exportCmd.Use)
	assert.NotEmpty(t, exportCmd.Short)
	subcommands := exportCmd.Commands()
	assert.GreaterOrEqual(t, len(subcommands), 3)
}

func TestImportCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "import", importCmd.Use)
	assert.NotEmpty(t, importCmd.Short)
	subcommands := importCmd.Commands()
	assert.GreaterOrEqual(t, len(subcommands), 3)
}

func TestInitCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "init", initCmd.Use)
	assert.NotEmpty(t, initCmd.Short)
	forceFlag := initCmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestRunCommandStructure(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "run [prompt]", runCmd.Use)
	assert.NotEmpty(t, runCmd.Short)
}

func TestConfigLoading(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, "helm", cfg.AppName)
	assert.NotEmpty(t, cfg.Router.FallbackChain)
}

func TestDatabaseOperations(t *testing.T) {
	t.Parallel()
	database, _ := setupTestDB(t)
	defer database.Close()

	ctx := context.Background()

	params := db.CreateSessionParams{
		ID: "test-session-1", Provider: "anthropic",
		Model: "claude-sonnet-4", Project: "/test/project", Status: "running",
	}

	_, err := database.CreateSession(ctx, params)
	require.NoError(t, err)

	session, err := database.GetSession(ctx, "test-session-1")
	require.NoError(t, err)
	assert.Equal(t, "test-session-1", session.ID)
	assert.Equal(t, "anthropic", session.Provider)
	assert.Equal(t, "running", session.Status)

	err = database.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		Status: "done", ID: "test-session-1",
	})
	require.NoError(t, err)

	session, err = database.GetSession(ctx, "test-session-1")
	require.NoError(t, err)
	assert.Equal(t, "done", session.Status)
}

func TestMemoryOperations(t *testing.T) {
	t.Parallel()
	database, _ := setupTestDB(t)
	defer database.Close()

	ctx := context.Background()

	_, err := database.CreateMemory(ctx, db.CreateMemoryParams{
		ID: "mem-1", Project: "/test/project", Type: "convention",
		Key: "naming", Value: "Use PascalCase", Source: "auto-learn", Confidence: 0.8,
	})
	require.NoError(t, err)

	memories, err := database.ListMemories(ctx, "/test/project")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(memories), 1)

	found := false
	for _, m := range memories {
		if m.ID == "mem-1" {
			found = true
			assert.Equal(t, "convention", m.Type)
			assert.Equal(t, "naming", m.Key)
			assert.InDelta(t, 0.8, m.Confidence, 0.01)
		}
	}
	assert.True(t, found, "Memory entry not found")
}

func TestPromptOperations(t *testing.T) {
	t.Parallel()
	database, _ := setupTestDB(t)
	defer database.Close()

	ctx := context.Background()

	_, err := database.CreatePrompt(ctx, db.CreatePromptParams{
		ID: "prompt-1", Name: "test-prompt",
		Template: "Write a function that {{action}}", Source: "builtin",
	})
	require.NoError(t, err)

	prompts, err := database.ListPrompts(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(prompts), 1)

	found := false
	for _, p := range prompts {
		if p.Name == "test-prompt" {
			found = true
			assert.Equal(t, "Write a function that {{action}}", p.Template)
		}
	}
	assert.True(t, found, "Prompt not found")
}

func TestDatabaseTablesExist(t *testing.T) {
	t.Parallel()
	database, _ := setupTestDB(t)
	defer database.Close()

	ctx := context.Background()

	// Test by performing operations on each table
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
		ID: "prompt-1", Name: "test-prompt-2", Template: "test", Source: "test",
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

func TestConcurrentDatabaseAccess(t *testing.T) {
	t.Parallel()
	database, _ := setupTestDB(t)
	defer database.Close()

	ctx := context.Background()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			sessionID := "concurrent-" + string(rune('a'+id))
			params := db.CreateSessionParams{
				ID: sessionID, Provider: "test", Model: "test-model",
				Project: "/test", Status: "running",
			}
			database.CreateSession(ctx, params)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	sessions, err := database.ListSessions(ctx, db.ListSessionsParams{
		Project: "/test", Limit: 100,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sessions), 10)
}

func TestDatabaseCleanup(t *testing.T) {
	t.Skip("Skipping cleanup test - temp dir handles cleanup")
}
