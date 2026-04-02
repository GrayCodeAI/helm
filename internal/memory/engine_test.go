package memory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/helm/internal/db"
)

type mockQuerier struct {
	memories map[string]db.Memory
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{memories: make(map[string]db.Memory)}
}

type mockResult struct{}

func (m mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m mockResult) RowsAffected() (int64, error) { return 1, nil }

func (m *mockQuerier) CreateMemory(ctx context.Context, arg db.CreateMemoryParams) (sql.Result, error) {
	m.memories[arg.ID] = db.Memory{
		ID:         arg.ID,
		Project:    arg.Project,
		Type:       arg.Type,
		Key:        arg.Key,
		Value:      arg.Value,
		Source:     arg.Source,
		Confidence: arg.Confidence,
		UpdatedAt:  time.Now().Format(time.RFC3339),
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	return mockResult{}, nil
}

func (m *mockQuerier) GetMemory(ctx context.Context, arg db.GetMemoryParams) (db.Memory, error) {
	for _, mem := range m.memories {
		if mem.Project == arg.Project && mem.Key == arg.Key {
			return mem, nil
		}
	}
	return db.Memory{}, sql.ErrNoRows
}

func (m *mockQuerier) ListMemories(ctx context.Context, project string) ([]db.Memory, error) {
	var result []db.Memory
	for _, mem := range m.memories {
		if mem.Project == project {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockQuerier) ListMemoriesByType(ctx context.Context, arg db.ListMemoriesByTypeParams) ([]db.Memory, error) {
	var result []db.Memory
	for _, mem := range m.memories {
		if mem.Project == arg.Project && mem.Type == arg.Type {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockQuerier) SearchMemories(ctx context.Context, arg db.SearchMemoriesParams) ([]db.Memory, error) {
	var result []db.Memory
	for _, mem := range m.memories {
		if mem.Project == arg.Project {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockQuerier) UpdateMemory(ctx context.Context, arg db.UpdateMemoryParams) error {
	if mem, ok := m.memories[arg.ID]; ok {
		mem.Value = arg.Value
		mem.Confidence = arg.Confidence
		m.memories[arg.ID] = mem
	}
	return nil
}

func (m *mockQuerier) DeleteMemory(ctx context.Context, id string) error {
	delete(m.memories, id)
	return nil
}

func (m *mockQuerier) UpsertMemory(ctx context.Context, arg db.UpsertMemoryParams) error {
	m.memories[arg.ID] = db.Memory{
		ID:         arg.ID,
		Project:    arg.Project,
		Type:       arg.Type,
		Key:        arg.Key,
		Value:      arg.Value,
		Source:     arg.Source,
		Confidence: arg.Confidence,
		UpdatedAt:  time.Now().Format(time.RFC3339),
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	return nil
}

func TestEngineStoreAndGet(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	err := e.Store(ctx, "/project", TypeConvention, "naming", "use camelCase", "manual")
	require.NoError(t, err)

	mem, err := e.Get(ctx, "/project", "naming")
	require.NoError(t, err)
	assert.Equal(t, TypeConvention, mem.Type)
	assert.Equal(t, "use camelCase", mem.Value)
	assert.Equal(t, 0.5, mem.Confidence)
}

func TestEngineList(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	require.NoError(t, e.Store(ctx, "/project", TypeConvention, "naming", "camelCase", "manual"))
	require.NoError(t, e.Store(ctx, "/project", TypeFact, "language", "Go", "manual"))

	memories, err := e.List(ctx, "/project")
	require.NoError(t, err)
	assert.Equal(t, 2, len(memories))
}

func TestEngineListByType(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	require.NoError(t, e.Store(ctx, "/project", TypeConvention, "naming", "camelCase", "manual"))
	require.NoError(t, e.Store(ctx, "/project", TypeFact, "language", "Go", "manual"))

	conventions, err := e.ListByType(ctx, "/project", TypeConvention)
	require.NoError(t, err)
	assert.Equal(t, 1, len(conventions))
	assert.Equal(t, "naming", conventions[0].Key)
}

func TestEngineUpdate(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	require.NoError(t, e.Store(ctx, "/project", TypeConvention, "naming", "camelCase", "manual"))

	mem, _ := e.Get(ctx, "/project", "naming")
	require.NoError(t, e.Update(ctx, mem.ID, "use PascalCase", 0.8))

	mem, err := e.Get(ctx, "/project", "naming")
	require.NoError(t, err)
	assert.Equal(t, "use PascalCase", mem.Value)
	assert.Equal(t, 0.8, mem.Confidence)
}

func TestEngineDelete(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	require.NoError(t, e.Store(ctx, "/project", TypeConvention, "naming", "camelCase", "manual"))

	mem, _ := e.Get(ctx, "/project", "naming")
	require.NoError(t, e.Delete(ctx, mem.ID))

	_, err := e.Get(ctx, "/project", "naming")
	require.Error(t, err)
}

func TestEngineUpsert(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	require.NoError(t, e.Upsert(ctx, "/project", TypeConvention, "naming", "camelCase", "auto", 0.5))
	require.NoError(t, e.Upsert(ctx, "/project", TypeConvention, "naming", "PascalCase", "auto", 0.7))

	memories, err := e.List(ctx, "/project")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(memories), 1)
}

func TestRecallRelevance(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	ctx := context.Background()

	require.NoError(t, e.Store(ctx, "/project", TypeConvention, "naming", "use camelCase for variables", "manual"))
	require.NoError(t, e.Store(ctx, "/project", TypeFact, "framework", "React with TypeScript", "manual"))

	results, err := e.Recall(ctx, "/project", "add new variable for the naming convention", 5)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)

	if len(results) > 0 {
		assert.Equal(t, "naming", results[0].Memory.Key)
	}
}

func TestFormatForContext(t *testing.T) {
	t.Parallel()

	memories := []MemoryEntry{
		{Type: TypeConvention, Key: "naming", Value: "use camelCase"},
		{Type: TypeFact, Key: "language", Value: "Go"},
	}

	output := FormatForContext(memories)
	assert.Contains(t, output, "convention")
	assert.Contains(t, output, "naming")
	assert.Contains(t, output, "use camelCase")
}

func TestFormatForContextEmpty(t *testing.T) {
	t.Parallel()

	output := FormatForContext([]MemoryEntry{})
	assert.Empty(t, output)
}

func TestAutoLearn(t *testing.T) {
	t.Parallel()
	q := newMockQuerier()
	e := NewEngine(q)
	al := NewAutoLearn(e)
	ctx := context.Background()

	require.NoError(t, al.LearnFromRejectedDiff(ctx, "/project", "auth.go", "don't use global state"))
	require.NoError(t, al.LearnFromConvention(ctx, "/project", "error-handling", "return errors explicitly"))
	require.NoError(t, al.LearnFromFact(ctx, "/project", "db", "PostgreSQL 15"))
	require.NoError(t, al.LearnFromSkill(ctx, "/project", "testing", "use table-driven tests"))

	memories, err := e.List(ctx, "/project")
	require.NoError(t, err)
	assert.Equal(t, 4, len(memories))
}

func TestAllTypes(t *testing.T) {
	t.Parallel()

	types := AllTypes()
	assert.Equal(t, 6, len(types))
	assert.Contains(t, types, TypeConvention)
	assert.Contains(t, types, TypeCorrection)
	assert.Contains(t, types, TypeSkill)
}
