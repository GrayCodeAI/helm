// Package analytics provides analytics and reporting capabilities
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/db"
)

// TrendAnalyzer analyzes trends over time
type TrendAnalyzer struct {
	querier db.Querier
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer(querier db.Querier) *TrendAnalyzer {
	return &TrendAnalyzer{querier: querier}
}

// TrendReport represents trend analysis
type TrendReport struct {
	Period            string
	StartDate         time.Time
	EndDate           time.Time
	CostTrend         []DataPoint
	ProductivityTrend []DataPoint
	ModelTrends       map[string][]DataPoint
	Insights          []string
}

// DataPoint represents a single data point
type DataPoint struct {
	Date  time.Time
	Value float64
}

// GetCostTrend gets cost trend over time
func (ta *TrendAnalyzer) GetCostTrend(ctx context.Context, project string, days int) ([]DataPoint, error) {
	// Get cost records by date
	records, err := ta.querier.ListCostRecordsByDate(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("get cost records: %w", err)
	}

	// Aggregate by date
	byDate := make(map[string]float64)
	for _, record := range records {
		if record.Day != nil {
			dateStr := fmt.Sprintf("%v", record.Day)
			if len(dateStr) >= 10 {
				date := dateStr[:10]
				byDate[date] += record.Total.Float64
			}
		}
	}

	// Convert to data points
	var points []DataPoint
	for date, cost := range byDate {
		t, _ := time.Parse("2006-01-02", date)
		points = append(points, DataPoint{
			Date:  t,
			Value: cost,
		})
	}

	return points, nil
}

// GetProductivityTrend gets sessions completed over time
func (ta *TrendAnalyzer) GetProductivityTrend(ctx context.Context, project string, days int) ([]DataPoint, error) {
	sessions, err := ta.querier.ListSessions(ctx, db.ListSessionsParams{
		Project: project,
		Limit:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	byDate := make(map[string]int)
	for _, s := range sessions {
		if s.Status == "done" && len(s.StartedAt) >= 10 {
			date := s.StartedAt[:10]
			byDate[date]++
		}
	}

	var points []DataPoint
	for date, count := range byDate {
		t, _ := time.Parse("2006-01-02", date)
		points = append(points, DataPoint{
			Date:  t,
			Value: float64(count),
		})
	}

	return points, nil
}

// GetModelTrend gets usage trend for a specific model
func (ta *TrendAnalyzer) GetModelTrend(ctx context.Context, model string, days int) ([]DataPoint, error) {
	performances, err := ta.querier.ListModelPerformance(ctx)
	if err != nil {
		return nil, fmt.Errorf("list model performance: %w", err)
	}

	var points []DataPoint
	for _, p := range performances {
		if p.Model == model {
			points = append(points, DataPoint{
				Value: float64(p.Attempts),
			})
		}
	}

	return points, nil
}

// GenerateTrendReport generates a comprehensive trend report
func (ta *TrendAnalyzer) GenerateTrendReport(ctx context.Context, project string, days int) (*TrendReport, error) {
	now := time.Now()
	startDate := now.Add(-time.Duration(days) * 24 * time.Hour)

	report := &TrendReport{
		Period:      fmt.Sprintf("Last %d days", days),
		StartDate:   startDate,
		EndDate:     now,
		ModelTrends: make(map[string][]DataPoint),
	}

	// Cost trend
	costTrend, err := ta.GetCostTrend(ctx, project, days)
	if err == nil {
		report.CostTrend = costTrend
	}

	// Productivity trend
	prodTrend, err := ta.GetProductivityTrend(ctx, project, days)
	if err == nil {
		report.ProductivityTrend = prodTrend
	}

	// Generate insights
	report.Insights = ta.generateInsights(report)

	return report, nil
}

func (ta *TrendAnalyzer) generateInsights(report *TrendReport) []string {
	var insights []string

	// Analyze cost trend
	if len(report.CostTrend) >= 7 {
		recent := averageLastN(report.CostTrend, 7)
		previous := averagePreviousN(report.CostTrend, 7, 7)

		if recent > previous*1.2 {
			insights = append(insights, "Cost is trending upward - review usage patterns")
		} else if recent < previous*0.8 {
			insights = append(insights, "Cost is trending downward - efficiency improving")
		}
	}

	// Analyze productivity
	if len(report.ProductivityTrend) >= 7 {
		recent := averageLastN(report.ProductivityTrend, 7)
		if recent > 5 {
			insights = append(insights, "High productivity - averaging 5+ tasks per week")
		}
	}

	return insights
}

func averageLastN(points []DataPoint, n int) float64 {
	if len(points) == 0 {
		return 0
	}
	if n > len(points) {
		n = len(points)
	}

	var sum float64
	for i := len(points) - n; i < len(points); i++ {
		sum += points[i].Value
	}
	return sum / float64(n)
}

func averagePreviousN(points []DataPoint, n, offset int) float64 {
	if len(points) < n+offset {
		return 0
	}

	var sum float64
	for i := len(points) - n - offset; i < len(points)-offset; i++ {
		sum += points[i].Value
	}
	return sum / float64(n)
}

// TimeSavedEstimator estimates time saved by using agents
type TimeSavedEstimator struct{}

// Estimate estimates time saved
func (tse *TimeSavedEstimator) Estimate(sessions int, avgSessionDuration time.Duration) time.Duration {
	// Assume manual work takes 4x longer than agent work
	manualTime := avgSessionDuration * 4
	savedPerSession := manualTime - avgSessionDuration
	return savedPerSession * time.Duration(sessions)
}

// FormatDuration formats duration in human-readable form
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours >= 168 {
		weeks := hours / 168
		return fmt.Sprintf("%d weeks", weeks)
	}
	if hours >= 24 {
		days := hours / 24
		return fmt.Sprintf("%d days", days)
	}
	return fmt.Sprintf("%d hours", hours)
}
