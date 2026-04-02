// Package automation provides task scheduling and automation capabilities
package automation

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ReleaseAutomation handles release automation
type ReleaseAutomation struct {
	gitRemote string
	gitBranch string
}

// ReleaseConfig configures release automation
type ReleaseConfig struct {
	AutoChangelog    bool
	AutoVersionBump  bool
	AutoTag          bool
	AutoDraftRelease bool
	VersionType      string // "patch", "minor", "major"
}

// NewReleaseAutomation creates a new release automation
func NewReleaseAutomation(remote, branch string) *ReleaseAutomation {
	return &ReleaseAutomation{
		gitRemote: remote,
		gitBranch: branch,
	}
}

// ReleaseResult represents the result of a release
type ReleaseResult struct {
	Version     string
	Tag         string
	Changelog   string
	ReleaseURL  string
	Error       error
}

// Run executes the release pipeline
func (r *ReleaseAutomation) Run(ctx context.Context, config ReleaseConfig) (*ReleaseResult, error) {
	result := &ReleaseResult{}

	// Step 1: Analyze commits
	commits, err := r.getCommitsSinceLastTag(ctx)
	if err != nil {
		result.Error = err
		return result, err
	}

	// Step 2: Generate changelog
	if config.AutoChangelog {
		changelog := r.generateChangelog(commits)
		result.Changelog = changelog
	}

	// Step 3: Bump version
	if config.AutoVersionBump {
		version, err := r.bumpVersion(ctx, config.VersionType, commits)
		if err != nil {
			result.Error = err
			return result, err
		}
		result.Version = version
		result.Tag = "v" + version
	}

	// Step 4: Create tag
	if config.AutoTag && result.Tag != "" {
		if err := r.createTag(ctx, result.Tag, result.Changelog); err != nil {
			result.Error = err
			return result, err
		}
	}

	// Step 5: Draft release
	if config.AutoDraftRelease {
		url, err := r.draftRelease(ctx, result.Tag, result.Changelog)
		if err != nil {
			result.Error = err
			return result, err
		}
		result.ReleaseURL = url
	}

	return result, nil
}

// Commit represents a git commit
type Commit struct {
	Hash    string
	Subject string
	Body    string
	Type    string // "feat", "fix", "chore", "docs", "breaking"
}

func (r *ReleaseAutomation) getCommitsSinceLastTag(ctx context.Context) ([]Commit, error) {
	// Get the latest tag
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	out, err := cmd.Output()
	if err != nil {
		// No previous tags
		return r.getAllCommits(ctx)
	}

	lastTag := strings.TrimSpace(string(out))

	// Get commits since tag
	cmd = exec.CommandContext(ctx, "git", "log", lastTag+"..HEAD", "--pretty=format:%H|%s|%b---END---")
	out, err = cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseCommits(string(out)), nil
}

func (r *ReleaseAutomation) getAllCommits(ctx context.Context) ([]Commit, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--pretty=format:%H|%s|%b---END---")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseCommits(string(out)), nil
}

func parseCommits(output string) []Commit {
	var commits []Commit

	entries := strings.Split(output, "---END---")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, "|", 3)
		if len(parts) < 2 {
			continue
		}

		commit := Commit{
			Hash:    parts[0],
			Subject: parts[1],
		}

		if len(parts) > 2 {
			commit.Body = parts[2]
		}

		commit.Type = classifyCommit(commit.Subject)
		commits = append(commits, commit)
	}

	return commits
}

func classifyCommit(subject string) string {
	subject = strings.ToLower(subject)

	if strings.Contains(subject, "breaking") || strings.Contains(subject, "breaking change") {
		return "breaking"
	}
	if strings.HasPrefix(subject, "feat") || strings.HasPrefix(subject, "feature") {
		return "feat"
	}
	if strings.HasPrefix(subject, "fix") {
		return "fix"
	}
	if strings.HasPrefix(subject, "docs") {
		return "docs"
	}
	return "chore"
}

func (r *ReleaseAutomation) generateChangelog(commits []Commit) string {
	var feat, fix, docs, chore, breaking []string

	for _, c := range commits {
		switch c.Type {
		case "feat":
			feat = append(feat, "- "+c.Subject)
		case "fix":
			fix = append(fix, "- "+c.Subject)
		case "docs":
			docs = append(docs, "- "+c.Subject)
		case "breaking":
			breaking = append(breaking, "- "+c.Subject)
		default:
			chore = append(chore, "- "+c.Subject)
		}
	}

	var sections []string
	sections = append(sections, "## Changelog\n")

	if len(breaking) > 0 {
		sections = append(sections, "### Breaking Changes\n"+strings.Join(breaking, "\n"))
	}
	if len(feat) > 0 {
		sections = append(sections, "### Features\n"+strings.Join(feat, "\n"))
	}
	if len(fix) > 0 {
		sections = append(sections, "### Bug Fixes\n"+strings.Join(fix, "\n"))
	}
	if len(docs) > 0 {
		sections = append(sections, "### Documentation\n"+strings.Join(docs, "\n"))
	}
	if len(chore) > 0 {
		sections = append(sections, "### Chores\n"+strings.Join(chore, "\n"))
	}

	return strings.Join(sections, "\n\n")
}

func (r *ReleaseAutomation) bumpVersion(ctx context.Context, versionType string, commits []Commit) (string, error) {
	// Get current version from latest tag
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	out, err := cmd.Output()

	var major, minor, patch int
	if err != nil {
		// No previous tags, start at 0.0.0
		major, minor, patch = 0, 0, 0
	} else {
		// Parse version from tag
		tag := strings.TrimSpace(string(out))
		tag = strings.TrimPrefix(tag, "v")
		fmt.Sscanf(tag, "%d.%d.%d", &major, &minor, &patch)
	}

	// Determine version bump type from commits if not specified
	if versionType == "" {
		versionType = r.determineVersionBump(commits)
	}

	// Bump version
	switch versionType {
	case "major":
		major++
		minor, patch = 0, 0
	case "minor":
		minor++
		patch = 0
	default: // patch
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

func (r *ReleaseAutomation) determineVersionBump(commits []Commit) string {
	hasBreaking := false
	hasFeature := false

	for _, c := range commits {
		if c.Type == "breaking" {
			hasBreaking = true
			break
		}
		if c.Type == "feat" {
			hasFeature = true
		}
	}

	if hasBreaking {
		return "major"
	}
	if hasFeature {
		return "minor"
	}
	return "patch"
}

func (r *ReleaseAutomation) createTag(ctx context.Context, tag, message string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", "-a", tag, "-m", message)
	return cmd.Run()
}

func (r *ReleaseAutomation) draftRelease(ctx context.Context, tag, changelog string) (string, error) {
	// In a real implementation, this would use the GitHub/GitLab API
	// to create a draft release
	return "https://github.com/user/repo/releases/tag/" + tag, nil
}
