package cost

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/helm/internal/db"
)

type mockCostQuerier struct {
	costs   []db.CostRecord
	budgets map[string]db.Budget
}

func newMockCostQuerier() *mockCostQuerier {
	return &mockCostQuerier{budgets: make(map[string]db.Budget)}
}

func (m *mockCostQuerier) CreateCostRecord(ctx context.Context, arg db.CreateCostRecordParams) (sql.Result, error) {
	m.costs = append(m.costs, db.CostRecord{
		ID:               arg.ID,
		SessionID:        arg.SessionID,
		Project:          arg.Project,
		Provider:         arg.Provider,
		Model:            arg.Model,
		InputTokens:      arg.InputTokens,
		OutputTokens:     arg.OutputTokens,
		CacheReadTokens:  arg.CacheReadTokens,
		CacheWriteTokens: arg.CacheWriteTokens,
		TotalCost:        arg.TotalCost,
	})
	return nil, nil
}

func (m *mockCostQuerier) GetCostBySession(ctx context.Context, sessionID string) (db.GetCostBySessionRow, error) {
	var row db.GetCostBySessionRow
	for _, c := range m.costs {
		if c.SessionID == sessionID {
			row.TotalCost = nullFloat64(c.TotalCost)
			row.InputTokens = nullInt64(c.InputTokens)
			row.OutputTokens = nullInt64(c.OutputTokens)
		}
	}
	return row, nil
}

func (m *mockCostQuerier) GetCostByProject(ctx context.Context, project string) (db.GetCostByProjectRow, error) {
	var row db.GetCostByProjectRow
	for _, c := range m.costs {
		if c.Project == project {
			row.TotalCost = nullFloat64(c.TotalCost)
			row.InputTokens = nullInt64(c.InputTokens)
			row.OutputTokens = nullInt64(c.OutputTokens)
		}
	}
	return row, nil
}

func (m *mockCostQuerier) GetCostByProjectToday(ctx context.Context, project string) (db.GetCostByProjectTodayRow, error) {
	return db.GetCostByProjectTodayRow{}, nil
}

func (m *mockCostQuerier) GetCostByProjectWeek(ctx context.Context, project string) (db.GetCostByProjectWeekRow, error) {
	return db.GetCostByProjectWeekRow{}, nil
}

func (m *mockCostQuerier) GetCostByProjectMonth(ctx context.Context, project string) (db.GetCostByProjectMonthRow, error) {
	return db.GetCostByProjectMonthRow{}, nil
}

func (m *mockCostQuerier) ListCostRecords(ctx context.Context, arg db.ListCostRecordsParams) ([]db.CostRecord, error) {
	var result []db.CostRecord
	for _, c := range m.costs {
		if c.Project == arg.Project {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockCostQuerier) ListCostRecordsByDate(ctx context.Context, project string) ([]db.ListCostRecordsByDateRow, error) {
	return nil, nil
}

func (m *mockCostQuerier) GetBudget(ctx context.Context, project string) (db.Budget, error) {
	b, ok := m.budgets[project]
	if !ok {
		return db.Budget{}, sql.ErrNoRows
	}
	return b, nil
}

func (m *mockCostQuerier) UpsertBudget(ctx context.Context, arg db.UpsertBudgetParams) error {
	m.budgets[arg.Project] = db.Budget{
		Project:      arg.Project,
		DailyLimit:   arg.DailyLimit,
		WeeklyLimit:  arg.WeeklyLimit,
		MonthlyLimit: arg.MonthlyLimit,
		WarningPct:   arg.WarningPct,
	}
	return nil
}

func TestCalculator(t *testing.T) {
	t.Parallel()
	calc := NewCalculator()

	cost := calc.Calculate("claude-sonnet-4-20250514", 1000, 500, 0, 0)
	expected := float64(1000)/1_000_000*3.0 + float64(500)/1_000_000*15.0
	assert.InDelta(t, expected, cost, 0.0001)

	cost = calc.Calculate("gpt-4o", 1000, 500, 0, 0)
	expected = float64(1000)/1_000_000*2.5 + float64(500)/1_000_000*10.0
	assert.InDelta(t, expected, cost, 0.0001)
}

func TestTrackerRecordCost(t *testing.T) {
	t.Parallel()
	q := newMockCostQuerier()
	tracker := NewTracker(q)
	ctx := context.Background()

	err := tracker.RecordCost(ctx, "s1", "/project", "anthropic", "claude-sonnet-4", 1000, 500, 0, 0)
	require.NoError(t, err)

	records, err := tracker.Records(ctx, "/project", 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(records))
}

func TestBudgetEnforcer(t *testing.T) {
	t.Parallel()
	q := newMockCostQuerier()
	enforcer := NewBudgetEnforcer(q)
	ctx := context.Background()

	err := enforcer.SetBudget(ctx, "/project", 10.0, 50.0, 150.0, 0.8)
	require.NoError(t, err)

	status, err := enforcer.Check(ctx, "/project")
	require.NoError(t, err)
	assert.Equal(t, 10.0, status.DailyLimit)
	assert.Equal(t, 50.0, status.WeeklyLimit)
	assert.False(t, status.Warning)
	assert.False(t, status.HardStop)
}

func TestFormatReport(t *testing.T) {
	t.Parallel()

	report := &Report{
		Period: "Today",
		Entries: []ReportEntry{
			{SessionID: "abc123def456", Cost: 0.87, Tokens: 12450},
			{SessionID: "def456ghi789", Cost: 1.23, Tokens: 34200},
		},
		TotalCost:   2.10,
		TotalTokens: 46650,
	}

	output := FormatReport(report)
	assert.Contains(t, output, "Cost Report")
	assert.Contains(t, output, "2.10")
	assert.Contains(t, output, "46650")
}

func TestFormatBudgetStatus(t *testing.T) {
	t.Parallel()

	status := &BudgetStatus{
		Project:    "/project",
		DailyCost:  8.5,
		DailyLimit: 10.0,
		DailyPct:   0.85,
		Warning:    true,
		HardStop:   false,
	}

	output := FormatBudgetStatus(status)
	assert.Contains(t, output, "Budget")
	assert.Contains(t, output, "WARNING")
}

func TestShortID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc12345...", shortID("abc12345def67890"))
	assert.Equal(t, "short", shortID("short"))
}
