// Package autonomy provides progressive autonomy capabilities
package autonomy

import (
	"fmt"
)

// Level represents the autonomy level
type Level string

const (
	// LevelSupervised requires approval for every change
	LevelSupervised Level = "supervised"
	// LevelSemi allows batch approval
	LevelSemi Level = "semi"
	// LevelFull allows auto-apply with notification
	LevelFull Level = "full"
)

// Config configures autonomy behavior
type Config struct {
	CurrentLevel   Level
	TaskType       string
	MinAttempts    int
	MinSuccessRate float64
}

// DefaultConfig returns default autonomy config
func DefaultConfig() Config {
	return Config{
		CurrentLevel:   LevelSupervised,
		MinAttempts:    5,
		MinSuccessRate: 0.8,
	}
}

// Manager manages autonomy levels
type Manager struct {
	tracker *TrustTracker
	levels  map[string]Level // taskType -> level
}

// NewManager creates a new autonomy manager
func NewManager(tracker *TrustTracker) *Manager {
	return &Manager{
		tracker: tracker,
		levels:  make(map[string]Level),
	}
}

// GetLevel gets the autonomy level for a task type
func (m *Manager) GetLevel(taskType string) Level {
	level, exists := m.levels[taskType]
	if !exists {
		return LevelSupervised // Default
	}
	return level
}

// SetLevel manually sets the autonomy level
func (m *Manager) SetLevel(taskType string, level Level) {
	m.levels[taskType] = level
}

// EvaluatePromotion evaluates if a task type should be promoted
func (m *Manager) EvaluatePromotion(taskType string) (bool, Level) {
	score := m.tracker.GetScore(taskType)
	if score == nil {
		return false, LevelSupervised
	}

	currentLevel := m.GetLevel(taskType)

	// Check if can promote from supervised to semi
	if currentLevel == LevelSupervised {
		if score.CanPromote(0.8, 5) {
			return true, LevelSemi
		}
	}

	// Check if can promote from semi to full
	if currentLevel == LevelSemi {
		if score.CanPromote(0.9, 10) {
			return true, LevelFull
		}
	}

	return false, currentLevel
}

// Promote promotes a task type to the next level
func (m *Manager) Promote(taskType string) (Level, error) {
	canPromote, newLevel := m.EvaluatePromotion(taskType)
	if !canPromote {
		return m.GetLevel(taskType), fmt.Errorf("cannot promote %s at this time", taskType)
	}

	m.levels[taskType] = newLevel
	return newLevel, nil
}

// Demote demotes a task type to a lower level
func (m *Manager) Demote(taskType string, level Level) {
	m.levels[taskType] = level
}

// ShouldRequireApproval checks if approval is required for a task
func (m *Manager) ShouldRequireApproval(taskType string) bool {
	level := m.GetLevel(taskType)
	return level == LevelSupervised
}

// ShouldBatchApprove checks if batch approval should be used
func (m *Manager) ShouldBatchApprove(taskType string) bool {
	level := m.GetLevel(taskType)
	return level == LevelSemi
}

// CanAutoApply checks if changes can be auto-applied
func (m *Manager) CanAutoApply(taskType string) bool {
	level := m.GetLevel(taskType)
	return level == LevelFull
}

// GetPromotionProgress returns progress toward next level
func (m *Manager) GetPromotionProgress(taskType string) map[string]interface{} {
	score := m.tracker.GetScore(taskType)
	if score == nil {
		return map[string]interface{}{
			"current_level": LevelSupervised,
			"progress":      0.0,
			"needed":        5,
		}
	}

	currentLevel := m.GetLevel(taskType)
	var needed int
	var targetRate float64

	switch currentLevel {
	case LevelSupervised:
		needed = 5
		targetRate = 0.8
	case LevelSemi:
		needed = 10
		targetRate = 0.9
	default:
		return map[string]interface{}{
			"current_level": currentLevel,
			"progress":      1.0,
			"message":       "Already at maximum autonomy",
		}
	}

	progress := float64(score.Attempts) / float64(needed)
	if progress > 1.0 {
		progress = 1.0
	}

	return map[string]interface{}{
		"current_level": currentLevel,
		"attempts":      score.Attempts,
		"successes":     score.Successes,
		"success_rate":  score.SuccessRate(),
		"target_rate":   targetRate,
		"needed":        needed,
		"progress":      progress,
		"can_promote":   score.CanPromote(targetRate, needed),
	}
}
