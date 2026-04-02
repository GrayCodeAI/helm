// Package cost provides cost tracking and budget management.
package cost

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/helm/internal/db"
)

type CostQuerier interface {
	CreateCostRecord(ctx context.Context, arg db.CreateCostRecordParams) (sql.Result, error)
	GetCostBySession(ctx context.Context, sessionID string) (db.GetCostBySessionRow, error)
	GetCostByProject(ctx context.Context, project string) (db.GetCostByProjectRow, error)
	GetCostByProjectToday(ctx context.Context, project string) (db.GetCostByProjectTodayRow, error)
	GetCostByProjectWeek(ctx context.Context, project string) (db.GetCostByProjectWeekRow, error)
	GetCostByProjectMonth(ctx context.Context, project string) (db.GetCostByProjectMonthRow, error)
	ListCostRecords(ctx context.Context, arg db.ListCostRecordsParams) ([]db.CostRecord, error)
	ListCostRecordsByDate(ctx context.Context, project string) ([]db.ListCostRecordsByDateRow, error)
	GetBudget(ctx context.Context, project string) (db.Budget, error)
	UpsertBudget(ctx context.Context, arg db.UpsertBudgetParams) error
}

type Record struct {
	ID               string
	SessionID        string
	Project          string
	Provider         string
	Model            string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	TotalCost        float64
	RecordedAt       time.Time
}

type DailyCost struct {
	Day          string
	TotalCost    float64
	InputTokens  int64
	OutputTokens int64
}

type Tracker struct {
	q          CostQuerier
	calculator *Calculator
}

func NewTracker(q CostQuerier) *Tracker {
	return &Tracker{
		q:          q,
		calculator: NewCalculator(),
	}
}

func (t *Tracker) RecordCost(ctx context.Context, sessionID, project, provider, model string, input, output, cacheRead, cacheWrite int64) error {
	cost := t.calculator.Calculate(model, input, output, cacheRead, cacheWrite)

	record := db.CreateCostRecordParams{
		ID:               uuid.New().String(),
		SessionID:        sessionID,
		Project:          project,
		Provider:         provider,
		Model:            model,
		InputTokens:      sql.NullInt64{Int64: input, Valid: input != 0},
		OutputTokens:     sql.NullInt64{Int64: output, Valid: output != 0},
		CacheReadTokens:  sql.NullInt64{Int64: cacheRead, Valid: cacheRead != 0},
		CacheWriteTokens: sql.NullInt64{Int64: cacheWrite, Valid: cacheWrite != 0},
		TotalCost:        sql.NullFloat64{Float64: cost, Valid: cost != 0},
	}

	_, err := t.q.CreateCostRecord(ctx, record)
	if err != nil {
		return fmt.Errorf("record cost: %w", err)
	}
	return nil
}

func (t *Tracker) SessionCost(ctx context.Context, sessionID string) (cost float64, input, output int64, err error) {
	row, err := t.q.GetCostBySession(ctx, sessionID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("session cost: %w", err)
	}
	return row.TotalCost, row.InputTokens, row.OutputTokens, nil
}

func (t *Tracker) ProjectCostToday(ctx context.Context, project string) (cost float64, input, output int64, err error) {
	row, err := t.q.GetCostByProjectToday(ctx, project)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("project cost today: %w", err)
	}
	return row.TotalCost, row.InputTokens, row.OutputTokens, nil
}

func (t *Tracker) ProjectCostWeek(ctx context.Context, project string) (cost float64, input, output int64, err error) {
	row, err := t.q.GetCostByProjectWeek(ctx, project)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("project cost week: %w", err)
	}
	return row.TotalCost, row.InputTokens, row.OutputTokens, nil
}

func (t *Tracker) ProjectCostMonth(ctx context.Context, project string) (cost float64, input, output int64, err error) {
	row, err := t.q.GetCostByProjectMonth(ctx, project)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("project cost month: %w", err)
	}
	return row.TotalCost, row.InputTokens, row.OutputTokens, nil
}

func (t *Tracker) DailyBreakdown(ctx context.Context, project string) ([]DailyCost, error) {
	rows, err := t.q.ListCostRecordsByDate(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("daily breakdown: %w", err)
	}

	result := make([]DailyCost, len(rows))
	for i, r := range rows {
		result[i] = DailyCost{
			Day:          fmt.Sprintf("%v", r.Day),
			TotalCost:    nullFloat64(r.Total),
			InputTokens:  int64(r.Input.Float64),
			OutputTokens: int64(r.Output.Float64),
		}
	}
	return result, nil
}

func (t *Tracker) Records(ctx context.Context, project string, limit int64) ([]Record, error) {
	rows, err := t.q.ListCostRecords(ctx, db.ListCostRecordsParams{
		Project: project,
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("cost records: %w", err)
	}

	result := make([]Record, len(rows))
	for i, r := range rows {
		result[i] = Record{
			ID:               r.ID,
			SessionID:        r.SessionID,
			Project:          r.Project,
			Provider:         r.Provider,
			Model:            r.Model,
			InputTokens:      nullInt64(r.InputTokens),
			OutputTokens:     nullInt64(r.OutputTokens),
			CacheReadTokens:  nullInt64(r.CacheReadTokens),
			CacheWriteTokens: nullInt64(r.CacheWriteTokens),
			TotalCost:        nullFloat64(r.TotalCost),
			RecordedAt:       parseTime(r.RecordedAt),
		}
	}
	return result, nil
}

func nullInt64(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

func nullFloat64(v sql.NullFloat64) float64 {
	if !v.Valid {
		return 0
	}
	return v.Float64
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
