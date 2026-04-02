// Package analytics provides analytics and reporting capabilities
package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/yourname/helm/internal/db"
)

// ROIDashboard provides model ROI analysis
type ROIDashboard struct {
	querier db.Querier
}

// NewROIDashboard creates a new ROI dashboard
func NewROIDashboard(querier db.Querier) *ROIDashboard {
	return &ROIDashboard{querier: querier}
}

// ModelROI represents ROI data for a model
type ModelROI struct {
	Model           string
	Provider        string
	TotalCost       float64
	TotalTasks      int
	SuccessfulTasks int
	FailedTasks     int
	RejectedTasks   int
	SuccessRate     float64
	RejectionRate   float64
	AvgCostPerTask  float64
	AvgTimeSeconds  float64
	ROI             float64
}

// GetModelROI calculates ROI for all models
func (rd *ROIDashboard) GetModelROI(ctx context.Context) ([]ModelROI, error) {
	// Get model performance data from database
	records, err := rd.querier.ListModelPerformance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get model performance: %w", err)
	}

	roiMap := make(map[string]*ModelROI)

	for _, record := range records {
		key := record.Model
		roi, exists := roiMap[key]
		if !exists {
			roi = &ModelROI{
				Model: record.Model,
			}
			roiMap[key] = roi
		}

		roi.TotalTasks += int(record.Attempts)
		roi.SuccessfulTasks += int(record.Successes)
		roi.TotalCost += record.TotalCost
		roi.AvgCostPerTask = record.TotalCost / float64(record.Attempts)
		roi.AvgTimeSeconds = float64(record.AvgTimeSeconds)
	}

	// Calculate derived metrics
	var results []ModelROI
	for _, roi := range roiMap {
		if roi.TotalTasks > 0 {
			roi.SuccessRate = float64(roi.SuccessfulTasks) / float64(roi.TotalTasks)
			roi.RejectionRate = float64(roi.RejectedTasks) / float64(roi.TotalTasks)
			roi.FailedTasks = roi.TotalTasks - roi.SuccessfulTasks - roi.RejectedTasks
			// ROI = success rate / cost per task (higher is better)
			if roi.AvgCostPerTask > 0 {
				roi.ROI = roi.SuccessRate / roi.AvgCostPerTask
			}
		}
		results = append(results, *roi)
	}

	// Sort by ROI (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].ROI > results[j].ROI
	})

	return results, nil
}

// GetModelROIByTaskType gets ROI breakdown by task type
func (rd *ROIDashboard) GetModelROIByTaskType(ctx context.Context, model string) (map[string]*ModelROI, error) {
	records, err := rd.querier.GetModelPerformance(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("get model performance: %w", err)
	}

	result := make(map[string]*ModelROI)
	for _, record := range records {
		roi := &ModelROI{
			Model:           record.Model,
			TotalTasks:      int(record.Attempts),
			SuccessfulTasks: int(record.Successes),
			TotalCost:       record.TotalCost,
			AvgCostPerTask:  0,
			AvgTimeSeconds:  float64(record.AvgTimeSeconds),
		}

		if record.Attempts > 0 {
			roi.SuccessRate = float64(record.Successes) / float64(record.Attempts)
			roi.AvgCostPerTask = record.TotalCost / float64(record.Attempts)
			if roi.AvgCostPerTask > 0 {
				roi.ROI = roi.SuccessRate / roi.AvgCostPerTask
			}
		}

		result[record.TaskType] = roi
	}

	return result, nil
}

// GetRecommendations generates model recommendations based on ROI
func (rd *ROIDashboard) GetRecommendations(ctx context.Context) ([]Recommendation, error) {
	roiData, err := rd.GetModelROI(ctx)
	if err != nil {
		return nil, err
	}

	var recommendations []Recommendation

	for _, roi := range roiData {
		var rec Recommendation
		rec.Model = roi.Model
		rec.SuccessRate = roi.SuccessRate
		rec.AvgCost = roi.AvgCostPerTask

		if roi.SuccessRate >= 0.9 && roi.AvgCostPerTask <= 0.5 {
			rec.Category = "preferred"
			rec.Reason = "High success rate at low cost - use as default"
		} else if roi.SuccessRate >= 0.8 {
			rec.Category = "recommended"
			rec.Reason = "Reliable performance - use for critical tasks"
		} else if roi.AvgCostPerTask <= 0.3 {
			rec.Category = "budget"
			rec.Reason = "Low cost - use for simple tasks or experimentation"
		} else if roi.SuccessRate < 0.5 {
			rec.Category = "avoid"
			rec.Reason = "Low success rate - consider alternative models"
		} else {
			rec.Category = "neutral"
			rec.Reason = "Average performance"
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// Recommendation represents a model recommendation
type Recommendation struct {
	Model       string
	Category    string // "preferred", "recommended", "budget", "avoid", "neutral"
	Reason      string
	SuccessRate float64
	AvgCost     float64
}

// ROISummary provides a summary of ROI data
type ROISummary struct {
	TotalModels        int
	PreferredModels    []string
	TotalCost          float64
	TotalTasks         int
	OverallSuccessRate float64
	TopPerformer       string
	BestValue          string
}

// GetSummary generates an ROI summary
func (rd *ROIDashboard) GetSummary(ctx context.Context) (*ROISummary, error) {
	roiData, err := rd.GetModelROI(ctx)
	if err != nil {
		return nil, err
	}

	summary := &ROISummary{
		TotalModels: len(roiData),
	}

	var totalCost float64
	var totalTasks int
	var totalSuccesses int
	var topROI float64
	var bestValue float64 = 999999

	for _, roi := range roiData {
		totalCost += roi.TotalCost
		totalTasks += roi.TotalTasks
		totalSuccesses += roi.SuccessfulTasks

		if roi.ROI > topROI {
			topROI = roi.ROI
			summary.TopPerformer = roi.Model
		}

		if roi.AvgCostPerTask < bestValue && roi.SuccessRate >= 0.7 {
			bestValue = roi.AvgCostPerTask
			summary.BestValue = roi.Model
		}

		if roi.SuccessRate >= 0.85 && roi.AvgCostPerTask <= 0.5 {
			summary.PreferredModels = append(summary.PreferredModels, roi.Model)
		}
	}

	summary.TotalCost = totalCost
	summary.TotalTasks = totalTasks
	if totalTasks > 0 {
		summary.OverallSuccessRate = float64(totalSuccesses) / float64(totalTasks)
	}

	return summary, nil
}

// TrackSession records session data for ROI tracking
func (rd *ROIDashboard) TrackSession(ctx context.Context, model, taskType string, success bool, cost float64, duration time.Duration) error {
	// Update model performance
	params := db.UpsertModelPerformanceParams{
		Model:          model,
		TaskType:       taskType,
		Attempts:       1,
		TotalCost:      cost,
		AvgTimeSeconds: int64(duration.Seconds()),
	}

	if success {
		params.Successes = 1
	}

	return rd.querier.UpsertModelPerformance(ctx, params)
}
