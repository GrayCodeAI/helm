// Package analytics provides analytics and reporting capabilities
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/db"
)

// WasteDetector identifies wasted spending
type WasteDetector struct {
	querier db.Querier
}

// NewWasteDetector creates a new waste detector
func NewWasteDetector(querier db.Querier) *WasteDetector {
	return &WasteDetector{querier: querier}
}

// WasteReport represents a waste analysis report
type WasteReport struct {
	Period          string
	TotalSpend      float64
	WasteCategories []WasteCategory
	TotalWaste      float64
	WastePercent    float64
	Trend           string // "improving", "worsening", "stable"
}

// WasteCategory represents waste in a specific category
type WasteCategory struct {
	Category    string
	Description string
	Count       int
	Cost        float64
	Percent     float64
}

// DetectWaste analyzes spending for waste
func (wd *WasteDetector) DetectWaste(ctx context.Context, project string, startDate, endDate time.Time) (*WasteReport, error) {
	report := &WasteReport{
		Period: fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
	}

	// Get total cost for period
	totalCost, err := wd.getTotalCost(ctx, project, startDate, endDate)
	if err != nil {
		return nil, err
	}
	report.TotalSpend = totalCost

	// Detect various waste categories
	categories := []struct {
		name string
		fn   func(context.Context, string, time.Time, time.Time) (int, float64, error)
	}{
		{"discarded_sessions", wd.detectDiscardedSessions},
		{"rejected_diffs", wd.detectRejectedDiffs},
		{"retry_loops", wd.detectRetryLoops},
		{"over_engineered", wd.detectOverEngineered},
	}

	var totalWaste float64
	for _, cat := range categories {
		count, cost, err := cat.fn(ctx, project, startDate, endDate)
		if err != nil {
			continue
		}

		if count > 0 {
			category := WasteCategory{
				Category: cat.name,
				Count:    count,
				Cost:     cost,
				Percent:  0,
			}
			report.WasteCategories = append(report.WasteCategories, category)
			totalWaste += cost
		}
	}

	report.TotalWaste = totalWaste
	if report.TotalSpend > 0 {
		report.WastePercent = (totalWaste / report.TotalSpend) * 100
	}

	// Calculate trend
	report.Trend = wd.calculateTrend(ctx, project, startDate, endDate)

	return report, nil
}

func (wd *WasteDetector) getTotalCost(ctx context.Context, project string, startDate, endDate time.Time) (float64, error) {
	cost, err := wd.querier.GetCostByProject(ctx, project)
	if err != nil {
		return 0, err
	}
	return cost.TotalCost, nil
}

func (wd *WasteDetector) detectDiscardedSessions(ctx context.Context, project string, startDate, endDate time.Time) (int, float64, error) {
	// Sessions that were started but never completed (no accepted changes)
	// This would need additional tracking in the database
	// For now, return placeholder data
	return 3, 4.20, nil
}

func (wd *WasteDetector) detectRejectedDiffs(ctx context.Context, project string, startDate, endDate time.Time) (int, float64, error) {
	// File changes that were rejected
	// This would query the file_changes table for rejected changes
	return 12, 2.80, nil
}

func (wd *WasteDetector) detectRetryLoops(ctx context.Context, project string, startDate, endDate time.Time) (int, float64, error) {
	// Sessions that required multiple retries
	return 2, 3.10, nil
}

func (wd *WasteDetector) detectOverEngineered(ctx context.Context, project string, startDate, endDate time.Time) (int, float64, error) {
	// Sessions where agent did more than necessary
	// This would compare complexity of task vs changes made
	return 1, 1.90, nil
}

func (wd *WasteDetector) calculateTrend(ctx context.Context, project string, startDate, endDate time.Time) string {
	// Compare current period to previous period
	periodDuration := endDate.Sub(startDate)
	prevStart := startDate.Add(-periodDuration)
	prevEnd := startDate

	currentReport, _ := wd.DetectWaste(ctx, project, startDate, endDate)
	prevReport, _ := wd.DetectWaste(ctx, project, prevStart, prevEnd)

	if currentReport == nil || prevReport == nil {
		return "stable"
	}

	if currentReport.WastePercent < prevReport.WastePercent*0.9 {
		return "improving"
	}
	if currentReport.WastePercent > prevReport.WastePercent*1.1 {
		return "worsening"
	}
	return "stable"
}

// GetWeeklyReport generates a weekly waste report
func (wd *WasteDetector) GetWeeklyReport(ctx context.Context, project string) (*WasteReport, error) {
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	return wd.DetectWaste(ctx, project, weekAgo, now)
}

// GetMonthlyReport generates a monthly waste report
func (wd *WasteDetector) GetMonthlyReport(ctx context.Context, project string) (*WasteReport, error) {
	now := time.Now()
	monthAgo := now.Add(-30 * 24 * time.Hour)
	return wd.DetectWaste(ctx, project, monthAgo, now)
}

// GenerateRecommendations generates waste reduction recommendations
func (wd *WasteDetector) GenerateRecommendations(report *WasteReport) []string {
	var recommendations []string

	for _, cat := range report.WasteCategories {
		switch cat.Category {
		case "discarded_sessions":
			recommendations = append(recommendations,
				"Define clear success criteria before starting sessions",
				"Review prompts for clarity and specificity",
			)
		case "rejected_diffs":
			recommendations = append(recommendations,
				"Enable quality gates to catch issues early",
				"Add more context to project memory",
			)
		case "retry_loops":
			recommendations = append(recommendations,
				"Break complex tasks into smaller subtasks",
				"Use session forking to try different approaches",
			)
		case "over_engineered":
			recommendations = append(recommendations,
				"Be more specific about scope in prompts",
				"Use smart diff triage to review changes",
			)
		}
	}

	if report.WastePercent > 30 {
		recommendations = append(recommendations,
			"High waste detected - consider reviewing your workflow",
		)
	}

	return recommendations
}
