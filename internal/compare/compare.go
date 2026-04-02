// Package compare provides session comparison functionality
package compare

import (
	"context"
	"fmt"

	"github.com/yourname/helm/internal/db"
)

// Comparator compares sessions
type Comparator struct {
	querier db.Querier
}

// NewComparator creates a new comparator
func NewComparator(querier db.Querier) *Comparator {
	return &Comparator{querier: querier}
}

// SessionComparison compares two sessions
type SessionComparison struct {
	SessionA    db.Session
	SessionB    db.Session
	Metrics     ComparisonMetrics
	Differences []Difference
	Similarity  float64 // 0-1 similarity score
}

// ComparisonMetrics contains comparison metrics
type ComparisonMetrics struct {
	CostDelta         float64
	TokenDelta        int64
	DurationDelta     float64
	MessageCountDelta int
	FileChangeDelta   int
}

// Difference represents a difference between sessions
type Difference struct {
	Type         string // "cost", "token", "file", "message"
	Description  string
	ValueA       interface{}
	ValueB       interface{}
	Significance float64 // 0-1 how significant
}

// CompareSessions compares two sessions
func (c *Comparator) CompareSessions(ctx context.Context, sessionAID, sessionBID string) (*SessionComparison, error) {
	sessionA, err := c.querier.GetSession(ctx, sessionAID)
	if err != nil {
		return nil, fmt.Errorf("get session A: %w", err)
	}

	sessionB, err := c.querier.GetSession(ctx, sessionBID)
	if err != nil {
		return nil, fmt.Errorf("get session B: %w", err)
	}

	comparison := &SessionComparison{
		SessionA: sessionA,
		SessionB: sessionB,
	}

	// Calculate metrics
	comparison.Metrics = c.calculateMetrics(sessionA, sessionB)

	// Find differences
	comparison.Differences = c.findDifferences(sessionA, sessionB)

	// Calculate similarity
	comparison.Similarity = c.calculateSimilarity(comparison.Differences)

	return comparison, nil
}

func (c *Comparator) calculateMetrics(a, b db.Session) ComparisonMetrics {
	return ComparisonMetrics{
		CostDelta:         a.Cost - b.Cost,
		TokenDelta:        (a.InputTokens + a.OutputTokens) - (b.InputTokens + b.OutputTokens),
		MessageCountDelta: 0, // Would need to query messages
	}
}

func (c *Comparator) findDifferences(a, b db.Session) []Difference {
	var diffs []Difference

	if a.Model != b.Model {
		diffs = append(diffs, Difference{
			Type:         "model",
			Description:  fmt.Sprintf("Different models: %s vs %s", a.Model, b.Model),
			ValueA:       a.Model,
			ValueB:       b.Model,
			Significance: 0.8,
		})
	}

	if a.Provider != b.Provider {
		diffs = append(diffs, Difference{
			Type:         "provider",
			Description:  fmt.Sprintf("Different providers: %s vs %s", a.Provider, b.Provider),
			ValueA:       a.Provider,
			ValueB:       b.Provider,
			Significance: 0.7,
		})
	}

	if a.Cost != b.Cost {
		diffs = append(diffs, Difference{
			Type:         "cost",
			Description:  fmt.Sprintf("Cost difference: $%.4f vs $%.4f", a.Cost, b.Cost),
			ValueA:       a.Cost,
			ValueB:       b.Cost,
			Significance: 0.9,
		})
	}

	return diffs
}

func (c *Comparator) calculateSimilarity(diffs []Difference) float64 {
	if len(diffs) == 0 {
		return 1.0
	}

	totalSignificance := 0.0
	for _, d := range diffs {
		totalSignificance += d.Significance
	}

	return 1.0 - (totalSignificance / float64(len(diffs)))
}

// ComparePrompts compares different prompts for the same task
type PromptComparison struct {
	PromptA      string
	PromptB      string
	ResultsA     []db.Session
	ResultsB     []db.Session
	AvgCostA     float64
	AvgCostB     float64
	AvgTimeA     float64
	AvgTimeB     float64
	SuccessRateA float64
	SuccessRateB float64
}

// ComparePrompts compares sessions using different prompts
func (c *Comparator) ComparePrompts(ctx context.Context, promptA, promptB string, limit int32) (*PromptComparison, error) {
	// This would search for sessions using each prompt
	// and compare their outcomes

	return &PromptComparison{
		PromptA: promptA,
		PromptB: promptB,
	}, nil
}

// CompareModels compares different models for similar tasks
type ModelComparison struct {
	ModelA       string
	ModelB       string
	TaskType     string
	SessionsA    []db.Session
	SessionsB    []db.Session
	AvgCost      float64
	AvgTokens    int64
	AvgTime      float64
	SuccessRate  float64
	QualityScore float64
}

// CompareModels compares sessions by model
func (c *Comparator) CompareModels(ctx context.Context, modelA, modelB, taskType string) (*ModelComparison, error) {
	// This would find comparable sessions and compare model performance

	return &ModelComparison{
		ModelA:   modelA,
		ModelB:   modelB,
		TaskType: taskType,
	}, nil
}

// BatchComparison compares multiple sessions
type BatchComparison struct {
	Sessions []db.Session
	Best     db.Session
	Worst    db.Session
	Median   db.Session
	Average  ComparisonMetrics
}

// CompareBatch compares a batch of sessions
func (c *Comparator) CompareBatch(ctx context.Context, sessionIDs []string) (*BatchComparison, error) {
	var sessions []db.Session

	for _, id := range sessionIDs {
		session, err := c.querier.GetSession(ctx, id)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	return &BatchComparison{
		Sessions: sessions,
		Best:     sessions[0],
		Worst:    sessions[0],
		Median:   sessions[len(sessions)/2],
	}, nil
}
