// Package perf provides performance budget tracking
package perf

import (
	"context"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/db"
)

// Budget tracks performance budgets
type Budget struct {
	querier db.Querier
}

// NewBudget creates a new performance budget tracker
func NewBudget(querier db.Querier) *Budget {
	return &Budget{querier: querier}
}

// PerformanceBudget defines budget constraints
type PerformanceBudget struct {
	Project           string
	MaxResponseTime   time.Duration
	MaxTokensPerReq   int64
	MaxCostPerReq     float64
	MaxMemoryMB       int64
	MaxConcurrentReqs int
}

// BudgetMetrics tracks actual vs budget
type BudgetMetrics struct {
	Project         string
	Budget          PerformanceBudget
	AvgResponseTime time.Duration
	AvgTokensPerReq int64
	AvgCostPerReq   float64
	PeakMemoryMB    int64
	ConcurrentReqs  int
	Violations      []BudgetViolation
}

// BudgetViolation represents a budget violation
type BudgetViolation struct {
	Timestamp   time.Time
	Metric      string
	BudgetValue interface{}
	ActualValue interface{}
	Severity    string // "warning", "critical"
}

// CheckBudget checks if performance is within budget
func (b *Budget) CheckBudget(ctx context.Context, project string, budget PerformanceBudget) (*BudgetMetrics, error) {
	metrics := &BudgetMetrics{
		Project: project,
		Budget:  budget,
	}

	// Get sessions for project
	sessions, err := b.querier.ListSessions(ctx, db.ListSessionsParams{
		Project: project,
		Limit:   100,
	})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return metrics, nil
	}

	// Calculate averages
	var totalCost float64
	var totalTokens int64
	for _, session := range sessions {
		totalCost += session.Cost
		totalTokens += session.InputTokens + session.OutputTokens
	}

	metrics.AvgCostPerReq = totalCost / float64(len(sessions))
	metrics.AvgTokensPerReq = totalTokens / int64(len(sessions))

	// Check for violations
	metrics.Violations = b.checkViolations(metrics, budget)

	return metrics, nil
}

func (b *Budget) checkViolations(metrics *BudgetMetrics, budget PerformanceBudget) []BudgetViolation {
	var violations []BudgetViolation

	if metrics.AvgCostPerReq > budget.MaxCostPerReq {
		violations = append(violations, BudgetViolation{
			Timestamp:   time.Now(),
			Metric:      "cost_per_request",
			BudgetValue: budget.MaxCostPerReq,
			ActualValue: metrics.AvgCostPerReq,
			Severity:    "warning",
		})
	}

	if metrics.AvgTokensPerReq > budget.MaxTokensPerReq {
		violations = append(violations, BudgetViolation{
			Timestamp:   time.Now(),
			Metric:      "tokens_per_request",
			BudgetValue: budget.MaxTokensPerReq,
			ActualValue: metrics.AvgTokensPerReq,
			Severity:    "warning",
		})
	}

	return violations
}

// DefaultBudget returns default performance budget
func DefaultBudget(project string) PerformanceBudget {
	return PerformanceBudget{
		Project:           project,
		MaxResponseTime:   30 * time.Second,
		MaxTokensPerReq:   100000,
		MaxCostPerReq:     5.0,
		MaxMemoryMB:       1024,
		MaxConcurrentReqs: 10,
	}
}

// BudgetAlert represents a budget alert
type BudgetAlert struct {
	ID        string
	Project   string
	Metric    string
	Message   string
	Severity  string
	CreatedAt time.Time
}

// AlertManager manages budget alerts
type AlertManager struct {
	alerts []BudgetAlert
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts: []BudgetAlert{},
	}
}

// AddAlert adds a new alert
func (am *AlertManager) AddAlert(alert BudgetAlert) {
	am.alerts = append(am.alerts, alert)
}

// GetAlerts gets all alerts for a project
func (am *AlertManager) GetAlerts(project string) []BudgetAlert {
	var result []BudgetAlert
	for _, alert := range am.alerts {
		if alert.Project == project {
			result = append(result, alert)
		}
	}
	return result
}

// ClearAlerts clears alerts for a project
func (am *AlertManager) ClearAlerts(project string) {
	var filtered []BudgetAlert
	for _, alert := range am.alerts {
		if alert.Project != project {
			filtered = append(filtered, alert)
		}
	}
	am.alerts = filtered
}
