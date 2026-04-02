// Package autonomy provides progressive autonomy capabilities
package autonomy

import (
	"sync"
	"time"
)

// TrustScore tracks agent reliability per task type
type TrustScore struct {
	TaskType     string
	Attempts     int
	Successes    int
	Failures     int
	Rejections   int
	LastSuccess  *time.Time
	LastFailure  *time.Time
	AvgCost      float64
	AvgDuration  time.Duration
	UpdatedAt    time.Time
}

// SuccessRate returns the success rate (0-1)
func (ts *TrustScore) SuccessRate() float64 {
	if ts.Attempts == 0 {
		return 0.0
	}
	return float64(ts.Successes) / float64(ts.Attempts)
}

// CanPromote checks if the agent can be promoted to the next autonomy level
func (ts *TrustScore) CanPromote(minSuccessRate float64, minAttempts int) bool {
	return ts.Attempts >= minAttempts && ts.SuccessRate() >= minSuccessRate
}

// TrustTracker tracks trust scores for all task types
type TrustTracker struct {
	scores map[string]*TrustScore // taskType -> score
	mu     sync.RWMutex
}

// NewTrustTracker creates a new trust tracker
func NewTrustTracker() *TrustTracker {
	return &TrustTracker{
		scores: make(map[string]*TrustScore),
	}
}

// RecordSuccess records a successful task completion
func (tt *TrustTracker) RecordSuccess(taskType string, cost float64, duration time.Duration) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	now := time.Now()
	score, exists := tt.scores[taskType]
	if !exists {
		score = &TrustScore{TaskType: taskType}
		tt.scores[taskType] = score
	}

	score.Attempts++
	score.Successes++
	score.LastSuccess = &now
	score.UpdatedAt = now

	// Update averages
	if score.Attempts == 1 {
		score.AvgCost = cost
		score.AvgDuration = duration
	} else {
		score.AvgCost = (score.AvgCost*float64(score.Attempts-1) + cost) / float64(score.Attempts)
		score.AvgDuration = (score.AvgDuration*time.Duration(score.Attempts-1) + duration) / time.Duration(score.Attempts)
	}
}

// RecordFailure records a failed task
func (tt *TrustTracker) RecordFailure(taskType string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	now := time.Now()
	score, exists := tt.scores[taskType]
	if !exists {
		score = &TrustScore{TaskType: taskType}
		tt.scores[taskType] = score
	}

	score.Attempts++
	score.Failures++
	score.LastFailure = &now
	score.UpdatedAt = now
}

// RecordRejection records a user rejection
func (tt *TrustTracker) RecordRejection(taskType string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	score, exists := tt.scores[taskType]
	if !exists {
		score = &TrustScore{TaskType: taskType}
		tt.scores[taskType] = score
	}

	score.Rejections++
	score.UpdatedAt = time.Now()
}

// GetScore gets the trust score for a task type
func (tt *TrustTracker) GetScore(taskType string) *TrustScore {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	return tt.scores[taskType]
}

// GetAllScores returns all trust scores
func (tt *TrustTracker) GetAllScores() []*TrustScore {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	var scores []*TrustScore
	for _, score := range tt.scores {
		scores = append(scores, score)
	}
	return scores
}

// GetBestTaskTypes returns task types with highest trust
func (tt *TrustTracker) GetBestTaskTypes(minAttempts int) []string {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	var best []string
	for taskType, score := range tt.scores {
		if score.Attempts >= minAttempts && score.SuccessRate() >= 0.8 {
			best = append(best, taskType)
		}
	}
	return best
}

// GetWorstTaskTypes returns task types needing improvement
func (tt *TrustTracker) GetWorstTaskTypes() []string {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	var worst []string
	for taskType, score := range tt.scores {
		if score.Attempts >= 3 && score.SuccessRate() < 0.5 {
			worst = append(worst, taskType)
		}
	}
	return worst
}
