// Package automation provides task scheduling and automation capabilities
package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SelfHealer automatically fixes CI failures
type SelfHealer struct {
	ciWatcher     *CIWatcher
	repoPath      string
	githubToken   string
	maxIterations int
}

// HealingResult represents the result of a healing attempt
type HealingResult struct {
	Success      bool
	Iterations   int
	FixesApplied []string
	Error        error
	CommitSHA    string
}

// NewSelfHealer creates a new self-healer
func NewSelfHealer(watcher *CIWatcher, repoPath, githubToken string) *SelfHealer {
	return &SelfHealer{
		ciWatcher:     watcher,
		repoPath:      repoPath,
		githubToken:   githubToken,
		maxIterations: 3,
	}
}

// Heal attempts to fix CI failures
func (h *SelfHealer) Heal(ctx context.Context, owner, repo, branch string) (*HealingResult, error) {
	result := &HealingResult{
		Success:      false,
		Iterations:   0,
		FixesApplied: []string{},
	}

	for i := 0; i < h.maxIterations; i++ {
		result.Iterations++

		// Get latest CI status
		statuses, err := h.ciWatcher.WatchGitHubActions(ctx, owner, repo, branch)
		if err != nil {
			result.Error = err
			return result, err
		}

		if len(statuses) == 0 {
			return result, nil
		}

		latest := statuses[0]

		if latest.IsSuccess() {
			result.Success = true
			return result, nil
		}

		if !latest.IsFailure() {
			continue
		}

		// Fetch CI logs
		logs, err := h.fetchCILogs(ctx, owner, repo, latest.ID)
		if err != nil {
			result.Error = fmt.Errorf("fetch CI logs: %w", err)
			return result, result.Error
		}

		// Analyze failure
		analysis := h.Analyze(logs)
		if analysis.Confidence < 0.5 {
			result.Error = fmt.Errorf("unable to determine fix for failure")
			return result, result.Error
		}

		// Apply fix
		fix, err := h.applyFix(ctx, analysis)
		if err != nil {
			result.Error = fmt.Errorf("apply fix: %w", err)
			return result, result.Error
		}

		result.FixesApplied = append(result.FixesApplied, fix)

		// Commit and push
		commitSHA, err := h.commitAndPushFix(ctx, analysis.Description)
		if err != nil {
			result.Error = fmt.Errorf("commit fix: %w", err)
			return result, result.Error
		}
		result.CommitSHA = commitSHA

		// Trigger CI re-run
		if err := h.triggerRerun(ctx, owner, repo, latest.ID); err != nil {
			result.Error = fmt.Errorf("trigger rerun: %w", err)
			return result, result.Error
		}

		// Wait for CI to complete
		time.Sleep(2 * time.Minute)
	}

	result.Error = fmt.Errorf("max iterations reached without success")
	return result, result.Error
}

// fetchCILogs fetches CI logs for a run
func (h *SelfHealer) fetchCILogs(ctx context.Context, owner, repo, runID string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "run", "view", runID,
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--log-failed")
	cmd.Dir = h.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", h.githubToken))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh run view: %s", string(output))
	}
	return string(output), nil
}

// applyFix applies the suggested fix
func (h *SelfHealer) applyFix(ctx context.Context, analysis *FailureAnalysis) (string, error) {
	switch analysis.Type {
	case "dependency":
		if err := exec.CommandContext(ctx, "go", "mod", "tidy").Run(); err != nil {
			return "", err
		}
		return "Ran go mod tidy", nil

	case "test":
		// Run tests to identify failing ones
		cmd := exec.CommandContext(ctx, "go", "test", "-v", "./...")
		cmd.Dir = h.repoPath
		output, _ := cmd.CombinedOutput()
		return fmt.Sprintf("Identified test failures: %s", string(output)[:200]), nil

	case "lint":
		cmd := exec.CommandContext(ctx, "golangci-lint", "run", "--fix")
		cmd.Dir = h.repoPath
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return "Ran golangci-lint --fix", nil

	default:
		return "Manual investigation required", nil
	}
}

// commitAndPushFix commits and pushes the fix
func (h *SelfHealer) commitAndPushFix(ctx context.Context, description string) (string, error) {
	if err := exec.CommandContext(ctx, "git", "add", "-A").Run(); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("fix(ci): auto-fix CI failure - %s", description)
	if err := exec.CommandContext(ctx, "git", "commit", "-m", msg).Run(); err != nil {
		return "", err
	}

	if err := exec.CommandContext(ctx, "git", "push").Run(); err != nil {
		return "", err
	}

	// Get commit SHA
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = h.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// triggerRerun triggers a CI re-run
func (h *SelfHealer) triggerRerun(ctx context.Context, owner, repo, runID string) error {
	cmd := exec.CommandContext(ctx, "gh", "run", "rerun", runID,
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--failed")
	cmd.Dir = h.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", h.githubToken))
	return cmd.Run()
}

// FailureAnalysis represents an analysis of a CI failure
type FailureAnalysis struct {
	Type         string
	Description  string
	SuggestedFix string
	Confidence   float64
}

// Analyze analyzes CI failure logs
func (h *SelfHealer) Analyze(logs string) *FailureAnalysis {
	if strings.Contains(logs, "go: downloading") && strings.Contains(logs, "error") {
		return &FailureAnalysis{
			Type:         "dependency",
			Description:  "Dependency download or compilation error",
			SuggestedFix: "Run go mod tidy and update dependencies",
			Confidence:   0.8,
		}
	}

	if strings.Contains(logs, "FAIL") && strings.Contains(logs, "Test") {
		return &FailureAnalysis{
			Type:         "test",
			Description:  "Test failure",
			SuggestedFix: "Run tests locally and fix failing tests",
			Confidence:   0.9,
		}
	}

	if strings.Contains(logs, "gofmt") || strings.Contains(logs, "golangci-lint") {
		return &FailureAnalysis{
			Type:         "lint",
			Description:  "Lint or formatting error",
			SuggestedFix: "Run linter with auto-fix enabled",
			Confidence:   0.85,
		}
	}

	if strings.Contains(logs, "undefined:") || strings.Contains(logs, "cannot find package") {
		return &FailureAnalysis{
			Type:         "compilation",
			Description:  "Compilation error - undefined symbol or missing package",
			SuggestedFix: "Check imports and fix undefined references",
			Confidence:   0.7,
		}
	}

	if strings.Contains(logs, "timeout") || strings.Contains(logs, "timed out") {
		return &FailureAnalysis{
			Type:         "timeout",
			Description:  "CI job timed out",
			SuggestedFix: "Optimize slow tests or increase timeout",
			Confidence:   0.6,
		}
	}

	return &FailureAnalysis{
		Type:         "unknown",
		Description:  "Unable to determine failure type",
		SuggestedFix: "Manual investigation required",
		Confidence:   0.0,
	}
}

// AutoHealWorkflow runs the complete auto-heal workflow
func (h *SelfHealer) AutoHealWorkflow(ctx context.Context, owner, repo, branch string) error {
	result, err := h.Heal(ctx, owner, repo, branch)
	if err != nil {
		return fmt.Errorf("heal failed: %w", err)
	}

	if result.Success {
		return nil
	}

	// If healing failed, create an issue
	return h.createHealingFailedIssue(ctx, owner, repo, branch, result)
}

// createHealingFailedIssue creates an issue when auto-healing fails
func (h *SelfHealer) createHealingFailedIssue(ctx context.Context, owner, repo, branch string, result *HealingResult) error {
	body := fmt.Sprintf(`## CI Auto-Healing Failed

**Branch:** %s
**Iterations:** %d
**Fixes Applied:** %s
**Error:** %v

### Manual intervention required

The auto-healing system attempted to fix the CI failure but was unsuccessful after %d iterations.

Please review the CI logs and apply the necessary fix manually.`,
		branch, result.Iterations,
		strings.Join(result.FixesApplied, ", "),
		result.Error,
		result.Iterations)

	cmd := exec.CommandContext(ctx, "gh", "issue", "create",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--title", fmt.Sprintf("CI Auto-Healing Failed on %s", branch),
		"--body", body)
	cmd.Dir = h.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", h.githubToken))
	return cmd.Run()
}

// GetHealingStats returns healing statistics
func (h *SelfHealer) GetHealingStats() map[string]interface{} {
	return map[string]interface{}{
		"max_iterations": h.maxIterations,
		"repo_path":      h.repoPath,
	}
}
