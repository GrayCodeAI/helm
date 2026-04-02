// Package compare provides session comparison capabilities.
package compare

import (
	"fmt"
	"strings"
	"time"
)

// Session represents a comparable session
type Session struct {
	ID           string
	Provider     string
	Model        string
	Prompt       string
	Status       string
	InputTokens  int64
	OutputTokens int64
	Cost         float64
	Duration     time.Duration
	StartedAt    time.Time
}

// SessionDiff represents a difference between sessions
type SessionDiff struct {
	Field  string
	Value1 interface{}
	Value2 interface{}
	Better string // "session1", "session2", or "equal"
}

// SessionComparisonResult represents the result of comparing two sessions
type SessionComparisonResult struct {
	Session1     Session
	Session2     Session
	Differences  []SessionDiff
	Winner       string
	WinnerReason string
}

// CompareSessions compares two sessions
func CompareSessions(s1, s2 Session) SessionComparisonResult {
	result := SessionComparisonResult{
		Session1: s1,
		Session2: s2,
	}

	// Compare cost
	costDiff := SessionDiff{
		Field:  "Cost",
		Value1: s1.Cost,
		Value2: s2.Cost,
	}
	if s1.Cost < s2.Cost {
		costDiff.Better = "session1"
	} else if s2.Cost < s1.Cost {
		costDiff.Better = "session2"
	} else {
		costDiff.Better = "equal"
	}
	result.Differences = append(result.Differences, costDiff)

	// Compare tokens
	total1 := s1.InputTokens + s1.OutputTokens
	total2 := s2.InputTokens + s2.OutputTokens
	tokenDiff := SessionDiff{
		Field:  "Total Tokens",
		Value1: total1,
		Value2: total2,
	}
	if total1 < total2 {
		tokenDiff.Better = "session1"
	} else if total2 < total1 {
		tokenDiff.Better = "session2"
	} else {
		tokenDiff.Better = "equal"
	}
	result.Differences = append(result.Differences, tokenDiff)

	// Compare duration
	durationDiff := SessionDiff{
		Field:  "Duration",
		Value1: s1.Duration.String(),
		Value2: s2.Duration.String(),
	}
	if s1.Duration < s2.Duration {
		durationDiff.Better = "session1"
	} else if s2.Duration < s1.Duration {
		durationDiff.Better = "session2"
	} else {
		durationDiff.Better = "equal"
	}
	result.Differences = append(result.Differences, durationDiff)

	// Compare providers and models
	result.Differences = append(result.Differences, SessionDiff{
		Field:  "Provider",
		Value1: s1.Provider,
		Value2: s2.Provider,
		Better: "equal",
	})
	result.Differences = append(result.Differences, SessionDiff{
		Field:  "Model",
		Value1: s1.Model,
		Value2: s2.Model,
		Better: "equal",
	})

	// Determine winner
	costBetter := 0
	tokenBetter := 0
	durationBetter := 0

	for _, d := range result.Differences {
		switch d.Field {
		case "Cost":
			if d.Better == "session1" {
				costBetter++
			} else if d.Better == "session2" {
				costBetter--
			}
		case "Total Tokens":
			if d.Better == "session1" {
				tokenBetter++
			} else if d.Better == "session2" {
				tokenBetter--
			}
		case "Duration":
			if d.Better == "session1" {
				durationBetter++
			} else if d.Better == "session2" {
				durationBetter--
			}
		}
	}

	totalScore := costBetter + tokenBetter + durationBetter
	if totalScore > 0 {
		result.Winner = "session1"
		result.WinnerReason = "Lower cost, fewer tokens, or faster"
	} else if totalScore < 0 {
		result.Winner = "session2"
		result.WinnerReason = "Lower cost, fewer tokens, or faster"
	} else {
		result.Winner = "tie"
		result.WinnerReason = "Sessions are comparable"
	}

	return result
}

// GenerateReport generates a comparison report in markdown
func GenerateReport(result SessionComparisonResult) string {
	var sb strings.Builder

	sb.WriteString("# Session Comparison Report\n\n")
	sb.WriteString(fmt.Sprintf("## Session 1: %s\n", result.Session1.ID))
	sb.WriteString(fmt.Sprintf("- Provider: %s\n", result.Session1.Provider))
	sb.WriteString(fmt.Sprintf("- Model: %s\n", result.Session1.Model))
	sb.WriteString(fmt.Sprintf("- Cost: $%.4f\n", result.Session1.Cost))
	sb.WriteString(fmt.Sprintf("- Tokens: %d input, %d output\n", result.Session1.InputTokens, result.Session1.OutputTokens))
	sb.WriteString(fmt.Sprintf("- Duration: %s\n\n", result.Session1.Duration))

	sb.WriteString(fmt.Sprintf("## Session 2: %s\n", result.Session2.ID))
	sb.WriteString(fmt.Sprintf("- Provider: %s\n", result.Session2.Provider))
	sb.WriteString(fmt.Sprintf("- Model: %s\n", result.Session2.Model))
	sb.WriteString(fmt.Sprintf("- Cost: $%.4f\n", result.Session2.Cost))
	sb.WriteString(fmt.Sprintf("- Tokens: %d input, %d output\n", result.Session2.InputTokens, result.Session2.OutputTokens))
	sb.WriteString(fmt.Sprintf("- Duration: %s\n\n", result.Session2.Duration))

	sb.WriteString("## Differences\n\n")
	sb.WriteString("| Field | Session 1 | Session 2 | Better |\n")
	sb.WriteString("|-------|-----------|-----------|--------|\n")
	for _, d := range result.Differences {
		better := d.Better
		if better == "session1" {
			better = "✓ Session 1"
		} else if better == "session2" {
			better = "✓ Session 2"
		} else {
			better = "="
		}
		sb.WriteString(fmt.Sprintf("| %s | %v | %v | %s |\n", d.Field, d.Value1, d.Value2, better))
	}

	sb.WriteString(fmt.Sprintf("\n## Winner: %s\n", result.Winner))
	sb.WriteString(fmt.Sprintf("**Reason:** %s\n", result.WinnerReason))

	return sb.String()
}

// ComparePrompts compares two prompts for similarity
func ComparePrompts(prompt1, prompt2 string) float64 {
	words1 := strings.Fields(strings.ToLower(prompt1))
	words2 := strings.Fields(strings.ToLower(prompt2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	common := 0
	for _, w := range words2 {
		if set1[w] {
			common++
		}
	}

	return float64(common) / float64(len(words1)+len(words2)-common)
}

// CompareModels compares two models based on performance metrics
func CompareModels(model1, model2 string, cost1, cost2 float64, tokens1, tokens2 int64) map[string]interface{} {
	cheaper := model2
	if cost1 < cost2 {
		cheaper = model1
	}

	return map[string]interface{}{
		"model1":          model1,
		"model2":          model2,
		"cost1":           cost1,
		"cost2":           cost2,
		"tokens1":         tokens1,
		"tokens2":         tokens2,
		"cost_per_token1": cost1 / float64(tokens1),
		"cost_per_token2": cost2 / float64(tokens2),
		"cheaper_model":   cheaper,
	}
}
