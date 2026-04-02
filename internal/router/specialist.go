// Package router provides specialist routing capabilities
package router

import (
	"fmt"
	"strings"
)

// SpecialistRouter routes tasks to the best model based on historical performance
type SpecialistRouter struct {
	performance *PerformanceTracker
	models      []ModelSpecialty
}

// ModelSpecialty represents a model's specialty
type ModelSpecialty struct {
	Model       string
	Provider    string
	Specialties []string // "frontend", "backend", "tests", "docs", "security"
	AvgCost     float64
	AvgQuality  float64
	AvgSpeed    float64 // tokens per second
}

// PerformanceTracker tracks model performance
type PerformanceTracker struct {
	records map[string]*PerformanceRecord // model:taskType -> record
}

// PerformanceRecord tracks performance for a model on a task type
type PerformanceRecord struct {
	Model        string
	TaskType     string
	Attempts     int
	Successes    int
	TotalCost    float64
	AvgCost      float64
	AvgDuration  float64 // seconds
	SuccessRate  float64
}

// NewSpecialistRouter creates a new specialist router
func NewSpecialistRouter() *SpecialistRouter {
	return &SpecialistRouter{
		performance: NewPerformanceTracker(),
		models: []ModelSpecialty{
			{
				Model:       "claude-sonnet-4-20250514",
				Provider:    "anthropic",
				Specialties: []string{"frontend", "backend", "tests", "architecture"},
				AvgCost:     0.003,
				AvgQuality:  0.92,
				AvgSpeed:    150,
			},
			{
				Model:       "gpt-4o",
				Provider:    "openai",
				Specialties: []string{"backend", "tests", "docs"},
				AvgCost:     0.005,
				AvgQuality:  0.88,
				AvgSpeed:    180,
			},
			{
				Model:       "gemini-2.5-pro",
				Provider:    "google",
				Specialties: []string{"frontend", "docs", "tests"},
				AvgCost:     0.0015,
				AvgQuality:  0.85,
				AvgSpeed:    200,
			},
			{
				Model:       "claude-haiku-3-5-20241022",
				Provider:    "anthropic",
				Specialties: []string{"quick_fixes", "refactoring", "simple_tasks"},
				AvgCost:     0.0005,
				AvgQuality:  0.78,
				AvgSpeed:    300,
			},
		},
	}
}

// RouteResult represents a routing decision
type RouteResult struct {
	Model       string
	Provider    string
	Reason      string
	Confidence  float64
}

// Route selects the best model for a task
func (sr *SpecialistRouter) Route(prompt string, priority string) (*RouteResult, error) {
	taskType := sr.classifyTask(prompt)

	switch priority {
	case "quality":
		return sr.routeForQuality(taskType)
	case "cost":
		return sr.routeForCost(taskType)
	case "speed":
		return sr.routeForSpeed(taskType)
	default:
		return sr.routeBalanced(taskType)
	}
}

// classifyTask classifies the task type from the prompt
func (sr *SpecialistRouter) classifyTask(prompt string) string {
	promptLower := strings.ToLower(prompt)

	if containsAny(promptLower, []string{"ui", "component", "button", "css", "react", "vue", "html", "style"}) {
		return "frontend"
	}
	if containsAny(promptLower, []string{"api", "endpoint", "database", "sql", "handler", "middleware", "server"}) {
		return "backend"
	}
	if containsAny(promptLower, []string{"test", "spec", "unit test", "integration test"}) {
		return "tests"
	}
	if containsAny(promptLower, []string{"doc", "readme", "documentation", "comment"}) {
		return "docs"
	}
	if containsAny(promptLower, []string{"security", "auth", "encrypt", "vulnerability"}) {
		return "security"
	}
	if containsAny(promptLower, []string{"fix", "bug", "small", "quick", "simple"}) {
		return "quick_fixes"
	}
	if containsAny(promptLower, []string{"refactor", "clean up", "reorganize"}) {
		return "refactoring"
	}

	return "general"
}

// routeForQuality selects the model with best quality for the task
func (sr *SpecialistRouter) routeForQuality(taskType string) (*RouteResult, error) {
	var best *ModelSpecialty
	bestScore := 0.0

	for _, model := range sr.models {
		if !hasSpecialty(model.Specialties, taskType) {
			continue
		}

		// Get performance data
		record := sr.performance.GetRecord(model.Model, taskType)
		score := model.AvgQuality

		if record != nil && record.Attempts > 0 {
			// Blend static score with actual performance
			score = (score + record.SuccessRate) / 2
		}

		if score > bestScore {
			bestScore = score
			best = &model
		}
	}

	if best == nil {
		// Fallback to most capable model
		return &RouteResult{
			Model:      "claude-sonnet-4-20250514",
			Provider:   "anthropic",
			Reason:     "No specialist found, using default high-capability model",
			Confidence: 0.5,
		}, nil
	}

	return &RouteResult{
		Model:      best.Model,
		Provider:   best.Provider,
		Reason:     fmt.Sprintf("Best quality for %s tasks", taskType),
		Confidence: bestScore,
	}, nil
}

// routeForCost selects the cheapest capable model
func (sr *SpecialistRouter) routeForCost(taskType string) (*RouteResult, error) {
	var best *ModelSpecialty
	bestCost := 999.0

	for _, model := range sr.models {
		if !hasSpecialty(model.Specialties, taskType) {
			continue
		}

		// Get performance data
		record := sr.performance.GetRecord(model.Model, taskType)
		cost := model.AvgCost

		if record != nil && record.Attempts > 0 {
			cost = record.AvgCost
		}

		if cost < bestCost && model.AvgQuality >= 0.75 { // Minimum quality threshold
			bestCost = cost
			best = &model
		}
	}

	if best == nil {
		return sr.routeBalanced(taskType)
	}

	return &RouteResult{
		Model:      best.Model,
		Provider:   best.Provider,
		Reason:     fmt.Sprintf("Most cost-effective for %s tasks", taskType),
		Confidence: 0.7,
	}, nil
}

// routeForSpeed selects the fastest model
func (sr *SpecialistRouter) routeForSpeed(taskType string) (*RouteResult, error) {
	var best *ModelSpecialty
	bestSpeed := 0.0

	for _, model := range sr.models {
		if !hasSpecialty(model.Specialties, taskType) {
			continue
		}

		if model.AvgSpeed > bestSpeed && model.AvgQuality >= 0.70 {
			bestSpeed = model.AvgSpeed
			best = &model
		}
	}

	if best == nil {
		return sr.routeBalanced(taskType)
	}

	return &RouteResult{
		Model:      best.Model,
		Provider:   best.Provider,
		Reason:     fmt.Sprintf("Fastest for %s tasks", taskType),
		Confidence: 0.7,
	}, nil
}

// routeBalanced selects a balanced option
func (sr *SpecialistRouter) routeBalanced(taskType string) (*RouteResult, error) {
	var best *ModelSpecialty
	bestScore := 0.0

	for _, model := range sr.models {
		if !hasSpecialty(model.Specialties, taskType) {
			continue
		}

		// Balanced score: quality * speed / cost
		score := (model.AvgQuality * (model.AvgSpeed / 100)) / (model.AvgCost * 1000)

		if score > bestScore {
			bestScore = score
			best = &model
		}
	}

	if best == nil {
		return &RouteResult{
			Model:      "claude-sonnet-4-20250514",
			Provider:   "anthropic",
			Reason:     "Using default model (no specialist match)",
			Confidence: 0.5,
		}, nil
	}

	return &RouteResult{
		Model:      best.Model,
		Provider:   best.Provider,
		Reason:     fmt.Sprintf("Best balance for %s tasks", taskType),
		Confidence: 0.8,
	}, nil
}

// RecordResult records the result of a routing decision
func (sr *SpecialistRouter) RecordResult(model, taskType string, success bool, cost, duration float64) {
	sr.performance.Record(model, taskType, success, cost, duration)
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func hasSpecialty(specialties []string, taskType string) bool {
	for _, s := range specialties {
		if s == taskType {
			return true
		}
	}
	return false
}

// NewPerformanceTracker creates a new performance tracker
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		records: make(map[string]*PerformanceRecord),
	}
}

// GetRecord gets performance record for a model on a task type
func (pt *PerformanceTracker) GetRecord(model, taskType string) *PerformanceRecord {
	key := model + ":" + taskType
	return pt.records[key]
}

// Record records a performance result
func (pt *PerformanceTracker) Record(model, taskType string, success bool, cost, duration float64) {
	key := model + ":" + taskType
	record, exists := pt.records[key]
	if !exists {
		record = &PerformanceRecord{
			Model:    model,
			TaskType: taskType,
		}
		pt.records[key] = record
	}

	record.Attempts++
	if success {
		record.Successes++
	}
	record.TotalCost += cost

	// Update averages
	record.AvgCost = record.TotalCost / float64(record.Attempts)
	record.AvgDuration = (record.AvgDuration*float64(record.Attempts-1) + duration) / float64(record.Attempts)
	record.SuccessRate = float64(record.Successes) / float64(record.Attempts)
}
