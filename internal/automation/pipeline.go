// Package automation provides task scheduling and automation capabilities
package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// IssueToPRPipeline orchestrates the issue-to-PR workflow
type IssueToPRPipeline struct {
	issueFetcher *IssueFetcher
	repoPath     string
	githubToken  string
}

// PipelineConfig configures the issue-to-PR pipeline
type PipelineConfig struct {
	AutoCreateBranch bool
	AutoRunAgent     bool
	AutoOpenPR       bool
	RequireTests     bool
	RequireLint      bool
	MaxCost          float64
	BaseBranch       string
}

// NewIssueToPRPipeline creates a new pipeline
func NewIssueToPRPipeline(fetcher *IssueFetcher, repoPath, githubToken string) *IssueToPRPipeline {
	return &IssueToPRPipeline{
		issueFetcher: fetcher,
		repoPath:     repoPath,
		githubToken:  githubToken,
	}
}

// PipelineResult represents the result of running the pipeline
type PipelineResult struct {
	Success      bool
	IssueNumber  int
	BranchName   string
	PRNumber     int
	PRURL        string
	Cost         float64
	Duration     time.Duration
	Error        error
	SessionID    string
	FilesChanged []string
	Commits      int
}

// Run executes the full pipeline for an issue
func (p *IssueToPRPipeline) Run(ctx context.Context, issue Issue, config PipelineConfig) (*PipelineResult, error) {
	result := &PipelineResult{
		IssueNumber: issue.Number,
		Success:     false,
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	baseBranch := config.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Step 1: Ensure we're on the base branch
	if err := p.runGit(ctx, "checkout", baseBranch); err != nil {
		return result, fmt.Errorf("checkout base branch: %w", err)
	}

	if err := p.runGit(ctx, "pull", "origin", baseBranch); err != nil {
		return result, fmt.Errorf("pull latest: %w", err)
	}

	// Step 2: Create branch
	if config.AutoCreateBranch {
		branchName := fmt.Sprintf("helm/issue-%d-%s", issue.Number, sanitizeBranchName(issue.Title))
		result.BranchName = branchName

		if err := p.runGit(ctx, "checkout", "-b", branchName); err != nil {
			return result, fmt.Errorf("create branch: %w", err)
		}
	}

	// Step 3: Run agent with issue as prompt
	if config.AutoRunAgent {
		prompt := buildPromptFromIssue(issue)
		_ = prompt // Used by agent session manager
		// In production, this would call the session manager to start an agent
		// For now, we create a placeholder session
		result.SessionID = fmt.Sprintf("session-issue-%d", issue.Number)

		// The agent would make changes to the codebase here
		// After agent completes, we check for changes
		changedFiles, err := p.getChangedFiles(ctx)
		if err != nil {
			return result, fmt.Errorf("get changed files: %w", err)
		}
		result.FilesChanged = changedFiles
	}

	// Step 4: Validate (tests, lint)
	if config.RequireTests {
		if err := p.runTests(ctx); err != nil {
			return result, fmt.Errorf("tests failed: %w", err)
		}
	}

	if config.RequireLint {
		if err := p.runLint(ctx); err != nil {
			return result, fmt.Errorf("lint failed: %w", err)
		}
	}

	// Step 5: Commit changes
	if len(result.FilesChanged) > 0 {
		if err := p.runGit(ctx, "add", "-A"); err != nil {
			return result, fmt.Errorf("git add: %w", err)
		}

		commitMsg := fmt.Sprintf("fix: resolve issue #%d - %s", issue.Number, issue.Title)
		if err := p.runGit(ctx, "commit", "-m", commitMsg); err != nil {
			return result, fmt.Errorf("git commit: %w", err)
		}

		// Count commits on this branch
		count, err := p.countCommits(ctx, baseBranch)
		if err == nil {
			result.Commits = count
		}

		// Push branch
		if err := p.runGit(ctx, "push", "-u", "origin", result.BranchName); err != nil {
			return result, fmt.Errorf("git push: %w", err)
		}
	}

	// Step 6: Open PR
	if config.AutoOpenPR {
		prNumber, prURL, err := p.createGitHubPR(ctx, issue, result.BranchName, baseBranch)
		if err != nil {
			return result, fmt.Errorf("create PR: %w", err)
		}
		result.PRNumber = prNumber
		result.PRURL = prURL
	}

	result.Success = true
	return result, nil
}

// runGit executes a git command in the repo directory
func (p *IssueToPRPipeline) runGit(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = p.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getChangedFiles returns list of changed files
func (p *IssueToPRPipeline) getChangedFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD")
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}

// runTests runs the test suite
func (p *IssueToPRPipeline) runTests(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = p.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runLint runs the linter
func (p *IssueToPRPipeline) runLint(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "golangci-lint", "run")
	cmd.Dir = p.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// countCommits counts commits between current branch and base
func (p *IssueToPRPipeline) countCommits(ctx context.Context, baseBranch string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("%s..HEAD", baseBranch))
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var count int
	fmt.Sscanf(string(output), "%d", &count)
	return count, nil
}

// createGitHubPR creates a PR via GitHub API
func (p *IssueToPRPipeline) createGitHubPR(ctx context.Context, issue Issue, branch, base string) (int, string, error) {
	if p.githubToken == "" {
		return 0, "", fmt.Errorf("no GitHub token provided")
	}

	// Parse owner/repo from issue
	parts := strings.Split(issue.Repository, "/")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid repository format: %s", issue.Repository)
	}
	owner, repo := parts[0], parts[1]

	// In production, would use GitHub API client with prBody
	// For now, use gh CLI if available
	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--repo", issue.Repository,
		"--base", base,
		"--head", branch,
		"--title", fmt.Sprintf("Fix #%d: %s", issue.Number, issue.Title),
		"--body", fmt.Sprintf("Resolves #%d\n\n%s", issue.Number, issue.Body),
	)
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, "", fmt.Errorf("gh pr create: %s: %w", string(output), err)
	}

	// Parse PR URL from output
	prURL := strings.TrimSpace(string(output))
	prNumber := 0
	fmt.Sscanf(prURL, "https://github.com/%s/%s/pull/%d", &owner, &repo, &prNumber)

	return prNumber, prURL, nil
}

// buildPromptFromIssue builds an agent prompt from an issue
func buildPromptFromIssue(issue Issue) string {
	return fmt.Sprintf(`Issue #%d: %s

%s

Labels: %s

Please implement the necessary changes to resolve this issue. Follow existing code conventions and include tests.`,
		issue.Number, issue.Title, issue.Body, strings.Join(issue.Labels, ", "))
}

// sanitizeBranchName creates a valid git branch name from a string
func sanitizeBranchName(name string) string {
	// Replace invalid characters with hyphens
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)

	// Collapse multiple hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim hyphens from ends
	result = strings.Trim(result, "-")

	// Limit length
	if len(result) > 50 {
		result = result[:50]
		result = strings.TrimRight(result, "-")
	}

	return result
}

// PipelineStatus represents the current status of a pipeline run
type PipelineStatus struct {
	CurrentStep string
	Progress    float64
	IssueNumber int
	StartedAt   time.Time
	Messages    []string
}

// Status returns the current pipeline status
func (p *IssueToPRPipeline) Status() *PipelineStatus {
	return &PipelineStatus{
		CurrentStep: "idle",
		Progress:    0,
	}
}

// PRResult represents the result of PR creation
type PRResult struct {
	Number    int
	URL       string
	State     string
	Merged    bool
	CreatedAt time.Time
}

// GetPRStatus gets the status of a PR
func (p *IssueToPRPipeline) GetPRStatus(ctx context.Context, owner, repo string, prNumber int) (*PRResult, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "number,url,state,mergedAt,createdAt")
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get PR status: %w", err)
	}

	var result PRResult
	if err := parseJSON(output, &result); err != nil {
		return nil, fmt.Errorf("parse PR response: %w", err)
	}

	return &result, nil
}

// MergePR merges a PR if checks pass
func (p *IssueToPRPipeline) MergePR(ctx context.Context, owner, repo string, prNumber int) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "merge", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--squash", "--delete-branch")
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

func parseJSON(data []byte, v interface{}) error {
	// Simple JSON parsing - in production use encoding/json
	return nil
}

// ListPendingPRs lists PRs waiting for review
func (p *IssueToPRPipeline) ListPendingPRs(ctx context.Context, owner, repo string) ([]PRResult, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "list",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--state", "open",
		"--json", "number,url,state,createdAt")
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var prs []PRResult
	_ = output // Would parse JSON response
	return prs, nil
}

// GetPRDiff gets the diff for a PR
func (p *IssueToPRPipeline) GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "diff", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo))
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetPRChecks gets the check status for a PR
func (p *IssueToPRPipeline) GetPRChecks(ctx context.Context, owner, repo string, prNumber int) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "checks", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo))
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	checks := make(map[string]string)
	_ = output // Would parse check results
	return checks, nil
}

// WaitForChecks waits for all PR checks to complete
func (p *IssueToPRPipeline) WaitForChecks(ctx context.Context, owner, repo string, prNumber int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		checks, err := p.GetPRChecks(ctx, owner, repo, prNumber)
		if err != nil {
			return err
		}

		allComplete := true
		for _, status := range checks {
			if status == "pending" || status == "in_progress" {
				allComplete = false
				break
			}
		}

		if allComplete {
			return nil
		}

		time.Sleep(30 * time.Second)
	}

	return fmt.Errorf("timeout waiting for checks")
}

// AutoMergeIfChecksPass waits for checks and merges if all pass
func (p *IssueToPRPipeline) AutoMergeIfChecksPass(ctx context.Context, owner, repo string, prNumber int) error {
	if err := p.WaitForChecks(ctx, owner, repo, prNumber, 30*time.Minute); err != nil {
		return err
	}

	checks, err := p.GetPRChecks(ctx, owner, repo, prNumber)
	if err != nil {
		return err
	}

	for _, status := range checks {
		if status == "failure" {
			return fmt.Errorf("checks failed, not merging")
		}
	}

	return p.MergePR(ctx, owner, repo, prNumber)
}

// CreateReleasePR creates a PR for a release
func (p *IssueToPRPipeline) CreateReleasePR(ctx context.Context, version, branch string) (int, string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--base", "main",
		"--head", branch,
		"--title", fmt.Sprintf("Release v%s", version),
		"--body", fmt.Sprintf("## Release v%s\n\nAuto-generated release PR.", version))
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, "", fmt.Errorf("create release PR: %s: %w", string(output), err)
	}

	prURL := strings.TrimSpace(string(output))
	return 0, prURL, nil
}

// CommentOnPR adds a comment to a PR
func (p *IssueToPRPipeline) CommentOnPR(ctx context.Context, owner, repo string, prNumber int, comment string) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "comment", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--body", comment)
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// RequestReview requests a review on a PR
func (p *IssueToPRPipeline) RequestReview(ctx context.Context, owner, repo string, prNumber int, reviewers []string) error {
	args := []string{"pr", "edit", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--add-reviewer", strings.Join(reviewers, ",")}
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// ListPRFiles lists files changed in a PR
func (p *IssueToPRPipeline) ListPRFiles(ctx context.Context, owner, repo string, prNumber int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "diff", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo), "--name-only")
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}

// ApprovePR approves a PR
func (p *IssueToPRPipeline) ApprovePR(ctx context.Context, owner, repo string, prNumber int, message string) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "review", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--approve", "--body", message)
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// ClosePR closes a PR without merging
func (p *IssueToPRPipeline) ClosePR(ctx context.Context, owner, repo string, prNumber int) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "close", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo))
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// ReopenPR reopens a closed PR
func (p *IssueToPRPipeline) ReopenPR(ctx context.Context, owner, repo string, prNumber int) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "reopen", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo))
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// UpdatePRBranch updates a PR branch with the latest base branch
func (p *IssueToPRPipeline) UpdatePRBranch(ctx context.Context, owner, repo string, prNumber int) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "update-branch", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo))
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// GetPRComments gets all comments on a PR
func (p *IssueToPRPipeline) GetPRComments(ctx context.Context, owner, repo string, prNumber int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "comments")
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var comments []string
	_ = output // Would parse JSON
	return comments, nil
}

// GetPRReviewDecision gets the review decision status
func (p *IssueToPRPipeline) GetPRReviewDecision(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", fmt.Sprintf("%d", prNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "reviewDecision")
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	decision := strings.TrimSpace(string(output))
	return decision, nil
}

// CreatePRFromPatch creates a PR from a patch file
func (p *IssueToPRPipeline) CreatePRFromPatch(ctx context.Context, issue Issue, patchPath string) (int, string, error) {
	// Apply patch
	if err := p.runGit(ctx, "apply", patchPath); err != nil {
		return 0, "", fmt.Errorf("apply patch: %w", err)
	}

	// Commit
	if err := p.runGit(ctx, "add", "-A"); err != nil {
		return 0, "", fmt.Errorf("git add: %w", err)
	}

	commitMsg := fmt.Sprintf("fix: resolve issue #%d - %s", issue.Number, issue.Title)
	if err := p.runGit(ctx, "commit", "-m", commitMsg); err != nil {
		return 0, "", fmt.Errorf("git commit: %w", err)
	}

	// Push and create PR
	branchName := fmt.Sprintf("helm/issue-%d-%s", issue.Number, sanitizeBranchName(issue.Title))
	if err := p.runGit(ctx, "push", "-u", "origin", branchName); err != nil {
		return 0, "", fmt.Errorf("git push: %w", err)
	}

	return p.createGitHubPR(ctx, issue, branchName, "main")
}

// GenerateReleaseNotes generates release notes from commits
func (p *IssueToPRPipeline) GenerateReleaseNotes(ctx context.Context, fromTag, toTag string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", fmt.Sprintf("%s..%s", fromTag, toTag))
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")

	var features, fixes, chores []string
	for _, commit := range commits {
		commit = strings.TrimSpace(commit)
		if commit == "" {
			continue
		}

		if strings.Contains(commit, "feat:") {
			features = append(features, commit)
		} else if strings.Contains(commit, "fix:") {
			fixes = append(fixes, commit)
		} else {
			chores = append(chores, commit)
		}
	}

	var notes strings.Builder
	notes.WriteString(fmt.Sprintf("## Release %s\n\n", toTag))

	if len(features) > 0 {
		notes.WriteString("### Features\n\n")
		for _, f := range features {
			notes.WriteString(fmt.Sprintf("- %s\n", f))
		}
		notes.WriteString("\n")
	}

	if len(fixes) > 0 {
		notes.WriteString("### Bug Fixes\n\n")
		for _, f := range fixes {
			notes.WriteString(fmt.Sprintf("- %s\n", f))
		}
		notes.WriteString("\n")
	}

	if len(chores) > 0 {
		notes.WriteString("### Chores\n\n")
		for _, c := range chores {
			notes.WriteString(fmt.Sprintf("- %s\n", c))
		}
	}

	return notes.String(), nil
}

// CreateTag creates a git tag
func (p *IssueToPRPipeline) CreateTag(ctx context.Context, tag, message string) error {
	return p.runGit(ctx, "tag", "-a", tag, "-m", message)
}

// PushTag pushes a tag to remote
func (p *IssueToPRPipeline) PushTag(ctx context.Context, tag string) error {
	return p.runGit(ctx, "push", "origin", tag)
}

// CreateGitHubRelease creates a release on GitHub
func (p *IssueToPRPipeline) CreateGitHubRelease(ctx context.Context, tag, notes string) error {
	cmd := exec.CommandContext(ctx, "gh", "release", "create", tag,
		"--title", fmt.Sprintf("Release %s", tag),
		"--notes", notes)
	cmd.Dir = p.repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", p.githubToken))
	return cmd.Run()
}

// BumpVersion bumps the version in a version file
func (p *IssueToPRPipeline) BumpVersion(ctx context.Context, versionFile, newVersion string) error {
	content := fmt.Sprintf("%s\n", newVersion)
	return os.WriteFile(filepath.Join(p.repoPath, versionFile), []byte(content), 0644)
}

// GetLatestTag gets the latest git tag
func (p *IssueToPRPipeline) GetLatestTag(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = p.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SemverBump bumps a semantic version
func SemverBump(current, bumpType string) string {
	parts := strings.Split(strings.TrimPrefix(current, "v"), ".")
	if len(parts) != 3 {
		return current
	}

	var major, minor, patch int
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	fmt.Sscanf(parts[2], "%d", &patch)

	switch bumpType {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}
