package db

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	db := &DB{
		Queries: New(sqlDB),
		sqlDB:   sqlDB,
		path:    ":memory:",
	}

	if err := db.migrate(); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	return db
}

func TestOpen(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	if db.Path() != ":memory:" {
		t.Errorf("expected :memory:, got %s", db.Path())
	}
}

func TestSessionCRUD(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	result, err := db.CreateSession(ctx, CreateSessionParams{
		ID:       "test-session-1",
		Provider: "anthropic",
		Model:    "claude-sonnet-4",
		Project:  "/tmp/test-project",
		Prompt:   sql.NullString{String: "add auth feature", Valid: true},
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("expected 1 row affected, got %d", rowsAffected)
	}

	session, err := db.GetSession(ctx, "test-session-1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.Provider != "anthropic" {
		t.Errorf("expected anthropic, got %s", session.Provider)
	}

	err = db.UpdateSessionStatus(ctx, UpdateSessionStatusParams{
		Status:  "done",
		EndedAt: sql.NullString{String: "2026-04-01T12:00:00Z", Valid: true},
		ID:      "test-session-1",
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}

	session, err = db.GetSession(ctx, "test-session-1")
	if err != nil {
		t.Fatalf("get session after update: %v", err)
	}
	if session.Status != "done" {
		t.Errorf("expected done, got %s", session.Status)
	}

	sessions, err := db.ListSessions(ctx, ListSessionsParams{
		Project: "/tmp/test-project",
		Limit:   10,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	err = db.DeleteSession(ctx, "test-session-1")
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}

	_, err = db.GetSession(ctx, "test-session-1")
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestMemoryCRUD(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateMemory(ctx, CreateMemoryParams{
		ID:         "mem-1",
		Project:    "/tmp/test-project",
		Type:       "convention",
		Key:        "naming",
		Value:      "use camelCase for variables",
		Source:     "manual",
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("create memory: %v", err)
	}

	mem, err := db.GetMemory(ctx, GetMemoryParams{
		Project: "/tmp/test-project",
		Key:     "naming",
	})
	if err != nil {
		t.Fatalf("get memory: %v", err)
	}
	if mem.Value != "use camelCase for variables" {
		t.Errorf("wrong value: %s", mem.Value)
	}

	memories, err := db.ListMemories(ctx, "/tmp/test-project")
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}
	if len(memories) != 1 {
		t.Errorf("expected 1 memory, got %d", len(memories))
	}

	err = db.DeleteMemory(ctx, "mem-1")
	if err != nil {
		t.Fatalf("delete memory: %v", err)
	}
}

func TestCostTracking(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	// Create session first (required by FK constraint)
	_, err := db.CreateSession(ctx, CreateSessionParams{
		ID:       "session-1",
		Provider: "anthropic",
		Model:    "claude-sonnet-4",
		Project:  "/tmp/test-project",
		Status:   "running",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, err = db.CreateCostRecord(ctx, CreateCostRecordParams{
		ID:               "cost-1",
		SessionID:        "session-1",
		Project:          "/tmp/test-project",
		Provider:         "anthropic",
		Model:            "claude-sonnet-4",
		InputTokens:      sql.NullInt64{Int64: 1000, Valid: true},
		OutputTokens:     sql.NullInt64{Int64: 500, Valid: true},
		CacheReadTokens:  sql.NullInt64{Int64: 200, Valid: true},
		CacheWriteTokens: sql.NullInt64{Int64: 100, Valid: true},
		TotalCost:        sql.NullFloat64{Float64: 0.0115, Valid: true},
	})
	if err != nil {
		t.Fatalf("create cost record: %v", err)
	}

	row, err := db.GetCostByProject(ctx, "/tmp/test-project")
	if err != nil {
		t.Fatalf("get cost by project: %v", err)
	}
	if row.TotalCost != 0.0115 {
		t.Errorf("expected cost 0.0115, got %v", row.TotalCost)
	}
	if row.InputTokens != 1000 {
		t.Errorf("expected input 1000, got %v", row.InputTokens)
	}
	if row.OutputTokens != 500 {
		t.Errorf("expected output 500, got %v", row.OutputTokens)
	}
}
