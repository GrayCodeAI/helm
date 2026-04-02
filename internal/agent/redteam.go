// Package agent provides agent orchestration capabilities
package agent

import (
	"context"
	"fmt"
	"time"
)

// RedTeam orchestrates adversarial testing between agents
type RedTeam struct {
	primaryAgent   *Agent
	adversaryAgent *Agent
	iterations     int
}

// Agent represents an agent in the red team
type Agent struct {
	ID       string
	Name     string
	Model    string
	Role     string // "writer" or "breaker"
	Output   string
	Feedback string
}

// NewRedTeam creates a new red team setup
func NewRedTeam(primaryModel, adversaryModel string) *RedTeam {
	return &RedTeam{
		primaryAgent: &Agent{
			ID:    "primary",
			Name:  "Code Writer",
			Model: primaryModel,
			Role:  "writer",
		},
		adversaryAgent: &Agent{
			ID:    "adversary",
			Name:  "Code Breaker",
			Model: adversaryModel,
			Role:  "breaker",
		},
		iterations: 3,
	}
}

// RedTeamResult represents the result of red team testing
type RedTeamResult struct {
	Task         string
	Iterations   int
	FinalOutput  string
	IssuesFound  []Issue
	Improvements []string
	Notes        string
	Duration     time.Duration
	TotalCost    float64
}

// Issue represents an issue found by the breaker agent
type Issue struct {
	Type        string // "security", "performance", "edge_case", "logic"
	Severity    string // "low", "medium", "high", "critical"
	Description string
	Suggestion  string
	LineNumber  int
}

// Run executes the red team testing workflow
func (rt *RedTeam) Run(ctx context.Context, task string) (*RedTeamResult, error) {
	startTime := time.Now()
	result := &RedTeamResult{
		Task:        task,
		IssuesFound: []Issue{},
	}

	fmt.Println("🔴 Starting Red Team Mode...")

	// Phase 1: Writer creates initial code
	fmt.Println("\n📝 Phase 1: Writer agent creating code...")
	rt.primaryAgent.Output = rt.generateCode(ctx, task)

	// Phase 2-4: Breaker reviews, writer fixes (iterative)
	for i := 0; i < rt.iterations; i++ {
		fmt.Printf("\n🔄 Iteration %d/%d\n", i+1, rt.iterations)

		// Breaker reviews
		issues := rt.reviewCode(ctx, rt.primaryAgent.Output, task)
		if len(issues) == 0 {
			fmt.Println("✅ No issues found!")
			break
		}

		result.IssuesFound = append(result.IssuesFound, issues...)
		fmt.Printf("🐛 Found %d issues\n", len(issues))

		// Writer fixes
		rt.primaryAgent.Output = rt.fixCode(ctx, rt.primaryAgent.Output, issues)
		result.Improvements = append(result.Improvements, fmt.Sprintf("Fixed %d issues in iteration %d", len(issues), i+1))
	}

	result.FinalOutput = rt.primaryAgent.Output
	result.Iterations = rt.iterations
	result.Duration = time.Since(startTime)
	result.Notes = rt.generateNotes(ctx, result.IssuesFound)

	fmt.Println("\n✅ Red Team complete!")
	fmt.Printf("Found and addressed %d issues\n", len(result.IssuesFound))

	return result, nil
}

// generateCode simulates code generation
func (rt *RedTeam) generateCode(ctx context.Context, task string) string {
	// In real implementation, this would call the LLM
	return fmt.Sprintf("// Generated code for: %s\n// Implementation here...", task)
}

// reviewCode simulates code review
func (rt *RedTeam) reviewCode(ctx context.Context, code, task string) []Issue {
	// In real implementation, this would call the LLM
	// For now, return sample issues based on task keywords
	var issues []Issue

	if contains(task, "auth") || contains(task, "login") {
		issues = append(issues, Issue{
			Type:        "security",
			Severity:    "high",
			Description: "Consider adding rate limiting to prevent brute force attacks",
			Suggestion:  "Add rate limiting middleware",
		})
	}

	if contains(task, "api") || contains(task, "endpoint") {
		issues = append(issues, Issue{
			Type:        "edge_case",
			Severity:    "medium",
			Description: "Missing input validation for edge cases",
			Suggestion:  "Add comprehensive input validation",
		})
	}

	return issues
}

// fixCode simulates code fixing
func (rt *RedTeam) fixCode(ctx context.Context, code string, issues []Issue) string {
	// In real implementation, this would call the LLM
	return code + fmt.Sprintf("\n// Fixed %d issues", len(issues))
}

// generateNotes generates red team notes
func (rt *RedTeam) generateNotes(ctx context.Context, issues []Issue) string {
	notes := "## Red Team Analysis\n\n"

	byType := make(map[string][]Issue)
	for _, issue := range issues {
		byType[issue.Type] = append(byType[issue.Type], issue)
	}

	for issueType, typeIssues := range byType {
		notes += fmt.Sprintf("### %s Issues (%d)\n", issueType, len(typeIssues))
		for _, issue := range typeIssues {
			notes += fmt.Sprintf("- **%s**: %s\n", issue.Severity, issue.Description)
			notes += fmt.Sprintf("  - Suggestion: %s\n", issue.Suggestion)
		}
		notes += "\n"
	}

	return notes
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
