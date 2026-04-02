// Package watch provides stuck session detection
package watch

import (
	"fmt"
	"strings"
	"time"
)

// StuckDetector detects when sessions are stuck
type StuckDetector struct {
	thresholds StuckThresholds
}

// StuckThresholds configures stuck detection
type StuckThresholds struct {
	RepeatedToolCalls int
	RepeatedErrors    int
	NoProgressTurns   int
	TokenSpikePercent float64
	MaxDuration       time.Duration
}

// DefaultStuckThresholds returns default thresholds
func DefaultStuckThresholds() StuckThresholds {
	return StuckThresholds{
		RepeatedToolCalls: 3,
		RepeatedErrors:    2,
		NoProgressTurns:   5,
		TokenSpikePercent: 2.0, // 2x average
		MaxDuration:       30 * time.Minute,
	}
}

// NewStuckDetector creates a stuck detector
func NewStuckDetector(thresholds StuckThresholds) *StuckDetector {
	return &StuckDetector{thresholds: thresholds}
}

// StuckSignal represents a stuck detection signal
type StuckSignal struct {
	Type            string
	Description     string
	Severity        string // "low", "medium", "high"
	SuggestedAction string
}

// Analyze analyzes session activity for stuck signals
func (sd *StuckDetector) Analyze(activities []Activity) []StuckSignal {
	var signals []StuckSignal

	if len(activities) == 0 {
		return signals
	}

	// Check for repeated tool calls
	if repeated := sd.detectRepeatedToolCalls(activities); repeated > 0 {
		signals = append(signals, StuckSignal{
			Type:            "repeated_tool_calls",
			Description:     fmt.Sprintf("Same tool called %d times", repeated),
			Severity:        "high",
			SuggestedAction: "Consider adjusting prompt or taking manual control",
		})
	}

	// Check for repeated errors
	if errors := sd.detectRepeatedErrors(activities); errors > 0 {
		signals = append(signals, StuckSignal{
			Type:            "repeated_errors",
			Description:     fmt.Sprintf("Same error occurred %d times", errors),
			Severity:        "high",
			SuggestedAction: "Review error message and adjust approach",
		})
	}

	// Check for no progress
	if sd.detectNoProgress(activities) {
		signals = append(signals, StuckSignal{
			Type:            "no_progress",
			Description:     "No file changes in recent turns",
			Severity:        "medium",
			SuggestedAction: "Verify task clarity or try different approach",
		})
	}

	// Check for token spike
	if sd.detectTokenSpike(activities) {
		signals = append(signals, StuckSignal{
			Type:            "token_spike",
			Description:     "Unusual token usage increase",
			Severity:        "medium",
			SuggestedAction: "Monitor for potential loop or verbose output",
		})
	}

	// Check duration
	if sd.detectLongDuration(activities) {
		signals = append(signals, StuckSignal{
			Type:            "long_duration",
			Description:     "Session running longer than expected",
			Severity:        "low",
			SuggestedAction: "Consider breaking task into smaller parts",
		})
	}

	return signals
}

// Activity represents a session activity
type Activity struct {
	Timestamp   time.Time
	Type        string // "tool_call", "error", "file_change", "message"
	Description string
	Tokens      int
}

func (sd *StuckDetector) detectRepeatedToolCalls(activities []Activity) int {
	counts := make(map[string]int)
	for _, a := range activities {
		if a.Type == "tool_call" {
			counts[a.Description]++
		}
	}

	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}

	if maxCount >= sd.thresholds.RepeatedToolCalls {
		return maxCount
	}
	return 0
}

func (sd *StuckDetector) detectRepeatedErrors(activities []Activity) int {
	counts := make(map[string]int)
	for _, a := range activities {
		if a.Type == "error" {
			// Group similar errors
			errorKey := extractErrorKey(a.Description)
			counts[errorKey]++
		}
	}

	maxCount := 0
	for _, count := range counts {
		if count > maxCount {
			maxCount = count
		}
	}

	if maxCount >= sd.thresholds.RepeatedErrors {
		return maxCount
	}
	return 0
}

func (sd *StuckDetector) detectNoProgress(activities []Activity) bool {
	recent := getRecentActivities(activities, sd.thresholds.NoProgressTurns)
	for _, a := range recent {
		if a.Type == "file_change" {
			return false
		}
	}
	return true
}

func (sd *StuckDetector) detectTokenSpike(activities []Activity) bool {
	if len(activities) < 3 {
		return false
	}

	// Calculate average
	var totalTokens int
	for _, a := range activities {
		totalTokens += a.Tokens
	}
	avg := float64(totalTokens) / float64(len(activities))

	// Check recent activity
	recent := activities[len(activities)-1]
	if avg > 0 && float64(recent.Tokens) > avg*sd.thresholds.TokenSpikePercent {
		return true
	}

	return false
}

func (sd *StuckDetector) detectLongDuration(activities []Activity) bool {
	if len(activities) == 0 {
		return false
	}

	start := activities[0].Timestamp
	duration := time.Since(start)

	return duration > sd.thresholds.MaxDuration
}

func extractErrorKey(errorMsg string) string {
	// Extract key part of error for grouping
	if idx := strings.Index(errorMsg, ":"); idx > 0 {
		return errorMsg[:idx]
	}
	return errorMsg
}

func getRecentActivities(activities []Activity, n int) []Activity {
	if len(activities) <= n {
		return activities
	}
	return activities[len(activities)-n:]
}

// AutoPauser automatically pauses stuck sessions
type AutoPauser struct {
	detector *StuckDetector
	actions  []PauseAction
}

// PauseAction handles pause events
type PauseAction interface {
	OnPause(sessionID string, signals []StuckSignal)
}

// NewAutoPauser creates an auto pauser
func NewAutoPauser(detector *StuckDetector) *AutoPauser {
	return &AutoPauser{
		detector: detector,
	}
}

// Check checks a session and pauses if stuck
func (ap *AutoPauser) Check(sessionID string, activities []Activity) (*PauseDecision, error) {
	signals := ap.detector.Analyze(activities)

	// Check if we should pause
	shouldPause := false
	var reason string

	for _, signal := range signals {
		if signal.Severity == "high" {
			shouldPause = true
			reason = signal.Description
			break
		}
	}

	if !shouldPause {
		// Also pause if multiple medium signals
		mediumCount := 0
		for _, signal := range signals {
			if signal.Severity == "medium" {
				mediumCount++
			}
		}
		if mediumCount >= 2 {
			shouldPause = true
			reason = "Multiple concerning signals detected"
		}
	}

	return &PauseDecision{
		ShouldPause: shouldPause,
		Reason:      reason,
		Signals:     signals,
	}, nil
}

// PauseDecision represents a pause decision
type PauseDecision struct {
	ShouldPause bool
	Reason      string
	Signals     []StuckSignal
}

// SuggestFixes suggests fixes based on signals
func SuggestFixes(signals []StuckSignal) []string {
	var suggestions []string

	for _, signal := range signals {
		suggestions = append(suggestions, signal.SuggestedAction)
	}

	return uniqueStrings(suggestions)
}

func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
