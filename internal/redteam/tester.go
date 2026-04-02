// Package redteam provides adversarial testing mode.
package redteam

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourname/helm/internal/provider"
)

// Tester performs red team testing
type Tester struct {
	provider provider.Provider
}

// NewTester creates a new red team tester
func NewTester(p provider.Provider) *Tester {
	return &Tester{provider: p}
}

// TestResult represents a red team test result
type TestResult struct {
	Vulnerability  string
	Severity       string // critical, high, medium, low
	Description    string
	Recommendation string
}

// RunTests runs red team tests on code
func (t *Tester) RunTests(ctx context.Context, code string) ([]TestResult, error) {
	prompt := fmt.Sprintf(`Analyze this code for security vulnerabilities, bugs, and quality issues.
For each issue found, provide:
1. The type of issue
2. Severity (critical/high/medium/low)
3. Description
4. Recommendation for fixing

Code to analyze:
%s

Respond with one issue per line in format: TYPE|SEVERITY|DESCRIPTION|RECOMMENDATION`, code)

	resp, err := t.provider.Chat(ctx, provider.ChatRequest{
		Model: "default",
		Messages: []provider.Message{
			{Role: "system", Content: "You are a security expert performing code review."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	return parseResults(resp.Content), nil
}

// RunInjectionTests tests for injection vulnerabilities
func (t *Tester) RunInjectionTests(ctx context.Context, code string) ([]TestResult, error) {
	tests := []string{
		"Check for SQL injection vulnerabilities",
		"Check for command injection vulnerabilities",
		"Check for XSS vulnerabilities",
		"Check for path traversal vulnerabilities",
	}

	var results []TestResult
	for _, test := range tests {
		prompt := fmt.Sprintf("%s in this code:\n%s", test, code)
		resp, err := t.provider.Chat(ctx, provider.ChatRequest{
			Model: "default",
			Messages: []provider.Message{
				{Role: "system", Content: "You are a security tester."},
				{Role: "user", Content: prompt},
			},
		})
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToLower(resp.Content), "vulnerab") ||
			strings.Contains(strings.ToLower(resp.Content), "injection") {
			results = append(results, TestResult{
				Vulnerability:  test,
				Severity:       "medium",
				Description:    resp.Content[:min(200, len(resp.Content))],
				Recommendation: "Review and sanitize all inputs",
			})
		}
	}

	return results, nil
}

func parseResults(content string) []TestResult {
	var results []TestResult
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			results = append(results, TestResult{
				Vulnerability:  strings.TrimSpace(parts[0]),
				Severity:       strings.TrimSpace(parts[1]),
				Description:    strings.TrimSpace(parts[2]),
				Recommendation: strings.TrimSpace(parts[3]),
			})
		}
	}

	return results
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
