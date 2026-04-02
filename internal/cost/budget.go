// Package cost provides cost tracking and budget management.
package cost

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yourname/helm/internal/db"
)

type BudgetStatus struct {
	Project      string
	DailyCost    float64
	DailyLimit   float64
	WeeklyCost   float64
	WeeklyLimit  float64
	MonthlyCost  float64
	MonthlyLimit float64
	DailyPct     float64
	WeeklyPct    float64
	MonthlyPct   float64
	Warning      bool
	HardStop     bool
}

type BudgetEnforcer struct {
	q CostQuerier
}

func NewBudgetEnforcer(q CostQuerier) *BudgetEnforcer {
	return &BudgetEnforcer{q: q}
}

func (be *BudgetEnforcer) Check(ctx context.Context, project string) (*BudgetStatus, error) {
	budget, err := be.q.GetBudget(ctx, project)
	if err != nil {
		return &BudgetStatus{Project: project}, nil
	}

	status := &BudgetStatus{
		Project:      project,
		DailyLimit:   nullFloat64(budget.DailyLimit),
		WeeklyLimit:  nullFloat64(budget.WeeklyLimit),
		MonthlyLimit: nullFloat64(budget.MonthlyLimit),
	}

	warningPct := budget.WarningPct
	if warningPct == 0 {
		warningPct = 0.8
	}

	todayRow, _ := be.q.GetCostByProjectToday(ctx, project)
	weekRow, _ := be.q.GetCostByProjectWeek(ctx, project)
	monthRow, _ := be.q.GetCostByProjectMonth(ctx, project)

	status.DailyCost = todayRow.TotalCost
	status.WeeklyCost = weekRow.TotalCost
	status.MonthlyCost = monthRow.TotalCost

	if status.DailyLimit > 0 {
		status.DailyPct = status.DailyCost / status.DailyLimit
	}
	if status.WeeklyLimit > 0 {
		status.WeeklyPct = status.WeeklyCost / status.WeeklyLimit
	}
	if status.MonthlyLimit > 0 {
		status.MonthlyPct = status.MonthlyCost / status.MonthlyLimit
	}

	status.Warning = status.DailyPct >= warningPct || status.WeeklyPct >= warningPct || status.MonthlyPct >= warningPct
	status.HardStop = status.DailyPct >= 1.0 || status.WeeklyPct >= 1.0 || status.MonthlyPct >= 1.0

	return status, nil
}

func (be *BudgetEnforcer) SetBudget(ctx context.Context, project string, daily, weekly, monthly, warningPct float64) error {
	err := be.q.UpsertBudget(ctx, db.UpsertBudgetParams{
		Project:      project,
		DailyLimit:   sql.NullFloat64{Float64: daily, Valid: daily != 0},
		WeeklyLimit:  sql.NullFloat64{Float64: weekly, Valid: weekly != 0},
		MonthlyLimit: sql.NullFloat64{Float64: monthly, Valid: monthly != 0},
		WarningPct:   warningPct,
	})
	if err != nil {
		return fmt.Errorf("set budget: %w", err)
	}
	return nil
}
