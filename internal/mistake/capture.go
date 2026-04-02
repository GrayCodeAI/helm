// Package mistake provides mistake tracking and learning capabilities
package mistake

import (
	"context"
	"strings"
)

// CaptureConfig configures mistake capture behavior
type CaptureConfig struct {
	AutoCaptureRejected bool
	AutoCaptureTestFail bool
	AutoCaptureLint     bool
	AutoCaptureCompile  bool
	AutoCaptureTimeout  bool
}

// DefaultCaptureConfig returns default capture configuration
func DefaultCaptureConfig() CaptureConfig {
	return CaptureConfig{
		AutoCaptureRejected: true,
		AutoCaptureTestFail: true,
		AutoCaptureLint:     true,
		AutoCaptureCompile:  true,
		AutoCaptureTimeout:  true,
	}
}

// Capture captures mistakes from various sources
type Capture struct {
	journal *Journal
	config  CaptureConfig
}

// NewCapture creates a new mistake capture
func NewCapture(journal *Journal, config CaptureConfig) *Capture {
	return &Capture{
		journal: journal,
		config:  config,
	}
}

// CaptureRejectedDiff records a rejected diff
func (c *Capture) CaptureRejectedDiff(ctx context.Context, sessionID, filePath, reason string) (*Entry, error) {
	if !c.config.AutoCaptureRejected {
		return nil, nil
	}

	return c.journal.Record(ctx, sessionID, TypeRejectedDiff,
		"User rejected diff changes",
		reason,
		"Review changes before applying",
		filePath)
}

// CaptureTestFailure records a test failure
func (c *Capture) CaptureTestFailure(ctx context.Context, sessionID, testName, output string) (*Entry, error) {
	if !c.config.AutoCaptureTestFail {
		return nil, nil
	}

	// Extract relevant part of test output
	context := output
	if len(output) > 500 {
		context = output[:500]
	}

	return c.journal.Record(ctx, sessionID, TypeTestFailure,
		"Test failed: "+testName,
		context,
		"Run tests before submitting changes",
		"")
}

// CaptureLintError records a lint error
func (c *Capture) CaptureLintError(ctx context.Context, sessionID, filePath, message string) (*Entry, error) {
	if !c.config.AutoCaptureLint {
		return nil, nil
	}

	return c.journal.Record(ctx, sessionID, TypeLintError,
		"Lint error: "+message,
		"",
		"Run linter before completing task",
		filePath)
}

// CaptureCompileError records a compile error
func (c *Capture) CaptureCompileError(ctx context.Context, sessionID, filePath, errorMsg string) (*Entry, error) {
	if !c.config.AutoCaptureCompile {
		return nil, nil
	}

	return c.journal.Record(ctx, sessionID, TypeCompileError,
		"Compilation failed: "+truncate(errorMsg, 200),
		errorMsg,
		"Verify code compiles before submitting",
		filePath)
}

// CaptureTimeout records a timeout
func (c *Capture) CaptureTimeout(ctx context.Context, sessionID, operation string, duration int) (*Entry, error) {
	if !c.config.AutoCaptureTimeout {
		return nil, nil
	}

	return c.journal.Record(ctx, sessionID, TypeTimeout,
		"Operation timed out: "+operation,
		"",
		"Break task into smaller chunks",
		"")
}

// CaptureLoopDetected records a detected loop
func (c *Capture) CaptureLoopDetected(ctx context.Context, sessionID, action string, count int) (*Entry, error) {
	description := "Agent appears to be in a loop"
	if action != "" {
		description = "Repeated action detected: " + action
	}

	return c.journal.Record(ctx, sessionID, TypeLoopDetected,
		description,
		"",
		"Consider adjusting prompt or taking manual control",
		"")
}

// CaptureWrongFile records modification of wrong files
func (c *Capture) CaptureWrongFile(ctx context.Context, sessionID, filePath, expectedPattern string) (*Entry, error) {
	return c.journal.Record(ctx, sessionID, TypeWrongFile,
		"Agent modified unrelated file",
		"Expected files matching: "+expectedPattern,
		"Focus on files relevant to the task",
		filePath)
}

// CaptureRuntimeError records a runtime error
func (c *Capture) CaptureRuntimeError(ctx context.Context, sessionID, filePath, errorMsg string) (*Entry, error) {
	return c.journal.Record(ctx, sessionID, TypeRuntimeError,
		"Runtime error: "+truncate(errorMsg, 200),
		errorMsg,
		"Test code execution before completing",
		filePath)
}

// CaptureSecurityIssue records a security issue
func (c *Capture) CaptureSecurityIssue(ctx context.Context, sessionID, filePath, issue string) (*Entry, error) {
	return c.journal.Record(ctx, sessionID, TypeSecurityIssue,
		"Security issue detected: "+issue,
		"",
		"Review security best practices",
		filePath)
}

// AutoCaptureFromSession analyzes session output and captures relevant mistakes
func (c *Capture) AutoCaptureFromSession(ctx context.Context, sessionID string, output string) []*Entry {
	var entries []*Entry

	// Check for common error patterns
	if strings.Contains(output, "error:") || strings.Contains(output, "Error:") {
		// Try to extract file path from error
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, ".go:") && strings.Contains(line, "error") {
				parts := strings.Split(line, ":")
				if len(parts) > 0 {
					filePath := strings.TrimSpace(parts[0])
					entry, _ := c.CaptureCompileError(ctx, sessionID, filePath, output)
					if entry != nil {
						entries = append(entries, entry)
					}
					break
				}
			}
		}
	}

	// Check for timeout patterns
	if strings.Contains(output, "timeout") || strings.Contains(output, "deadline exceeded") {
		entry, _ := c.CaptureTimeout(ctx, sessionID, "session", 0)
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	// Check for loop patterns (repeated identical lines)
	if hasRepeatedLines(output, 3) {
		entry, _ := c.CaptureLoopDetected(ctx, sessionID, "", 3)
		if entry != nil {
			entries = append(entries, entry)
		}
	}

	return entries
}

func hasRepeatedLines(text string, threshold int) bool {
	lines := strings.Split(text, "\n")
	counts := make(map[string]int)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && len(line) > 10 {
			counts[line]++
			if counts[line] >= threshold {
				return true
			}
		}
	}

	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
