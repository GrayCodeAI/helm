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

// NightlyMaintenance runs automated maintenance tasks
type NightlyMaintenance struct {
	scheduler *Scheduler
	tasks     []MaintenanceTask
	repoPath  string
	maxCost   float64
	totalCost float64
	lastRun   *time.Time
	runCount  int
}

// MaintenanceTask represents a single maintenance task
type MaintenanceTask struct {
	ID           string
	Name         string
	Description  string
	Enabled      bool
	CostEstimate float64
	Run          func(ctx context.Context) error
}

// NewNightlyMaintenance creates a new nightly maintenance runner
func NewNightlyMaintenance(scheduler *Scheduler, repoPath string, maxCost float64) *NightlyMaintenance {
	nm := &NightlyMaintenance{
		scheduler: scheduler,
		repoPath:  repoPath,
		tasks:     make([]MaintenanceTask, 0),
		maxCost:   maxCost,
	}

	// Register built-in tasks
	nm.registerBuiltinTasks()

	return nm
}

// registerBuiltinTasks registers the built-in maintenance tasks
func (nm *NightlyMaintenance) registerBuiltinTasks() {
	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "deps",
		Name:         "Update Dependencies",
		Description:  "Update project dependencies to latest versions",
		Enabled:      true,
		CostEstimate: 0.50,
		Run:          nm.updateDependencies,
	})

	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "lint",
		Name:         "Fix Lint Issues",
		Description:  "Run linter and fix auto-fixable issues",
		Enabled:      true,
		CostEstimate: 0.30,
		Run:          nm.fixLintIssues,
	})

	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "types",
		Name:         "Regenerate Types",
		Description:  "Regenerate generated types and code",
		Enabled:      true,
		CostEstimate: 0.20,
		Run:          nm.regenerateTypes,
	})

	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "docs",
		Name:         "Update Documentation",
		Description:  "Update documentation from code changes",
		Enabled:      true,
		CostEstimate: 0.40,
		Run:          nm.updateDocumentation,
	})

	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "tests",
		Name:         "Run Tests",
		Description:  "Run test suite and report failures",
		Enabled:      true,
		CostEstimate: 0.60,
		Run:          nm.runTests,
	})

	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "tidy",
		Name:         "Go Mod Tidy",
		Description:  "Clean up go.mod and go.sum",
		Enabled:      true,
		CostEstimate: 0.10,
		Run:          nm.goModTidy,
	})

	nm.tasks = append(nm.tasks, MaintenanceTask{
		ID:           "format",
		Name:         "Format Code",
		Description:  "Run gofumpt/gofmt on all Go files",
		Enabled:      true,
		CostEstimate: 0.10,
		Run:          nm.formatCode,
	})
}

// Run executes all enabled maintenance tasks
func (nm *NightlyMaintenance) Run(ctx context.Context) error {
	if nm.totalCost >= nm.maxCost {
		return fmt.Errorf("max cost limit reached: $%.2f/$%.2f", nm.totalCost, nm.maxCost)
	}

	now := time.Now()
	nm.lastRun = &now
	nm.runCount++

	for _, task := range nm.tasks {
		if !task.Enabled {
			continue
		}

		if nm.totalCost+task.CostEstimate > nm.maxCost {
			continue
		}

		if err := task.Run(ctx); err != nil {
			continue
		}
		nm.totalCost += task.CostEstimate
	}

	return nil
}

// GetTasks returns all maintenance tasks
func (nm *NightlyMaintenance) GetTasks() []MaintenanceTask {
	return nm.tasks
}

// EnableTask enables a maintenance task
func (nm *NightlyMaintenance) EnableTask(taskID string) error {
	for i := range nm.tasks {
		if nm.tasks[i].ID == taskID {
			nm.tasks[i].Enabled = true
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// DisableTask disables a maintenance task
func (nm *NightlyMaintenance) DisableTask(taskID string) error {
	for i := range nm.tasks {
		if nm.tasks[i].ID == taskID {
			nm.tasks[i].Enabled = false
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// GetStats returns maintenance statistics
func (nm *NightlyMaintenance) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"run_count":  nm.runCount,
		"total_cost": nm.totalCost,
		"max_cost":   nm.maxCost,
		"last_run":   nm.lastRun,
	}
}

// Task implementations

func (nm *NightlyMaintenance) updateDependencies(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "get", "-u", "./...")
	cmd.Dir = nm.repoPath
	return cmd.Run()
}

func (nm *NightlyMaintenance) fixLintIssues(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "golangci-lint", "run", "--fix")
	cmd.Dir = nm.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (nm *NightlyMaintenance) regenerateTypes(ctx context.Context) error {
	// Try sqlc first
	cmd := exec.CommandContext(ctx, "sqlc", "generate")
	cmd.Dir = nm.repoPath
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Try protoc
	cmd = exec.CommandContext(ctx, "protoc", "--go_out=.", "--go-grpc_out=.", "proto/*.proto")
	cmd.Dir = nm.repoPath
	return cmd.Run()
}

func (nm *NightlyMaintenance) updateDocumentation(ctx context.Context) error {
	// Try swag for Go
	cmd := exec.CommandContext(ctx, "swag", "init", "-g", "cmd/helm/main.go")
	cmd.Dir = nm.repoPath
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Try godoc
	cmd = exec.CommandContext(ctx, "go", "doc", "-all", "./...")
	cmd.Dir = nm.repoPath
	return cmd.Run()
}

func (nm *NightlyMaintenance) runTests(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "./...")
	cmd.Dir = nm.repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (nm *NightlyMaintenance) goModTidy(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = nm.repoPath
	return cmd.Run()
}

func (nm *NightlyMaintenance) formatCode(ctx context.Context) error {
	// Try gofumpt first
	cmd := exec.CommandContext(ctx, "gofumpt", "-w", ".")
	cmd.Dir = nm.repoPath
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fall back to gofmt
	cmd = exec.CommandContext(ctx, "gofmt", "-w", ".")
	cmd.Dir = nm.repoPath
	return cmd.Run()
}

// Schedule configures the nightly maintenance schedule
type NightlyConfig struct {
	Enabled  bool
	Time     string // "02:00"
	Timezone string // "America/New_York"
	Tasks    []string
	MaxCost  float64
}

// DefaultNightlyConfig returns default nightly config
func DefaultNightlyConfig() NightlyConfig {
	return NightlyConfig{
		Enabled:  true,
		Time:     "02:00",
		Timezone: "America/New_York",
		Tasks:    []string{"deps", "lint", "docs", "tidy", "format"},
		MaxCost:  2.00,
	}
}

// ParseTime parses the time string and returns the next run time
func ParseTime(timeStr, timezoneStr string) (time.Time, error) {
	loc, err := time.LoadLocation(timezoneStr)
	if err != nil {
		loc = time.UTC
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	var hour, minute int
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)

	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	return next, nil
}

// GetNextRunTime returns the next scheduled run time
func GetNextRunTime(config NightlyConfig) (time.Time, error) {
	return ParseTime(config.Time, config.Timezone)
}

// CreateMaintenanceBranch creates a branch for maintenance changes
func CreateMaintenanceBranch(ctx context.Context, repoPath, branchName string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	cmd.Dir = repoPath
	return cmd.Run()
}

// CommitMaintenanceChanges commits maintenance changes
func CommitMaintenanceChanges(ctx context.Context, repoPath, message string) error {
	if err := exec.CommandContext(ctx, "git", "add", "-A").Run(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	cmd.Dir = repoPath
	return cmd.Run()
}

// CreateMaintenancePR creates a PR for maintenance changes
func CreateMaintenancePR(ctx context.Context, repoPath, branch, title, body string) error {
	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--base", "main",
		"--head", branch,
		"--title", title,
		"--body", body)
	cmd.Dir = repoPath
	return cmd.Run()
}

// RunMaintenanceTask runs a single maintenance task and returns results
func RunMaintenanceTask(ctx context.Context, repoPath, taskID string) (string, error) {
	var cmd *exec.Cmd

	switch taskID {
	case "deps":
		cmd = exec.CommandContext(ctx, "go", "get", "-u", "./...")
	case "lint":
		cmd = exec.CommandContext(ctx, "golangci-lint", "run")
	case "tests":
		cmd = exec.CommandContext(ctx, "go", "test", "./...")
	case "tidy":
		cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	case "format":
		cmd = exec.CommandContext(ctx, "gofumpt", "-d", ".")
	default:
		return "", fmt.Errorf("unknown task: %s", taskID)
	}

	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetMaintenanceReport generates a report of maintenance activities
func GetMaintenanceReport(ctx context.Context, repoPath string) (string, error) {
	var report strings.Builder

	report.WriteString("# Maintenance Report\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Check for outdated dependencies
	cmd := exec.CommandContext(ctx, "go", "list", "-u", "-m", "all")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		outdated := 0
		for _, line := range lines {
			if strings.Contains(line, "[") {
				outdated++
			}
		}
		report.WriteString(fmt.Sprintf("## Dependencies\n- Outdated: %d\n\n", outdated))
	}

	// Check test status
	cmd = exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		report.WriteString("## Tests\n- Status: FAILING\n\n")
	} else {
		report.WriteString("## Tests\n- Status: PASSING\n\n")
	}

	// Check lint status
	cmd = exec.CommandContext(ctx, "golangci-lint", "run")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		report.WriteString("## Lint\n- Status: ISSUES FOUND\n\n")
	} else {
		report.WriteString("## Lint\n- Status: CLEAN\n\n")
	}

	return report.String(), nil
}

// CleanupOldBranches removes old maintenance branches
func CleanupOldBranches(ctx context.Context, repoPath string, maxAge time.Duration) error {
	cmd := exec.CommandContext(ctx, "git", "branch", "--list", "helm/maintenance-*", "--format=%(refname:short) %(committerdate:iso)")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		branchName := parts[0]
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", parts[1])
		if err != nil {
			continue
		}

		if commitDate.Before(cutoff) {
			delCmd := exec.CommandContext(ctx, "git", "branch", "-D", branchName)
			delCmd.Dir = repoPath
			delCmd.Run()
		}
	}

	return nil
}

// AutoCommitAndPush commits and pushes changes if any exist
func AutoCommitAndPush(ctx context.Context, repoPath, message string) (bool, error) {
	// Check for changes
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return false, nil
	}

	// Stage and commit
	if err := exec.CommandContext(ctx, "git", "add", "-A").Run(); err != nil {
		return false, err
	}

	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commitCmd.Dir = repoPath
	if err := commitCmd.Run(); err != nil {
		return false, err
	}

	// Push
	pushCmd := exec.CommandContext(ctx, "git", "push")
	pushCmd.Dir = repoPath
	if err := pushCmd.Run(); err != nil {
		return false, err
	}

	return true, nil
}

// GetMaintenanceStatus returns the current maintenance status
func GetMaintenanceStatus(ctx context.Context, repoPath string) map[string]string {
	status := make(map[string]string)

	// Check if we're on main branch
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, _ := cmd.Output()
	status["branch"] = strings.TrimSpace(string(output))

	// Check for uncommitted changes
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, _ = cmd.Output()
	if len(strings.TrimSpace(string(output))) > 0 {
		status["uncommitted"] = "true"
	} else {
		status["uncommitted"] = "false"
	}

	// Check last commit
	cmd = exec.CommandContext(ctx, "git", "log", "-1", "--format=%s %cr")
	cmd.Dir = repoPath
	output, _ = cmd.Output()
	status["last_commit"] = strings.TrimSpace(string(output))

	return status
}

// RunScheduledMaintenance runs all scheduled maintenance tasks
func RunScheduledMaintenance(ctx context.Context, repoPath string, config NightlyConfig) error {
	if !config.Enabled {
		return nil
	}

	// Create maintenance branch
	branchName := fmt.Sprintf("helm/maintenance-%s", time.Now().Format("2006-01-02"))
	if err := CreateMaintenanceBranch(ctx, repoPath, branchName); err != nil {
		return fmt.Errorf("create branch: %w", err)
	}

	var totalCost float64
	for _, taskID := range config.Tasks {
		if totalCost >= config.MaxCost {
			break
		}

		output, err := RunMaintenanceTask(ctx, repoPath, taskID)
		if err != nil {
			continue
		}
		_ = output
		totalCost += 0.10 // Estimated cost per task
	}

	// Auto-commit if there are changes
	changed, err := AutoCommitAndPush(ctx, repoPath, fmt.Sprintf("chore: nightly maintenance (%s)", time.Now().Format("2006-01-02")))
	if err != nil {
		return err
	}

	if changed {
		return CreateMaintenancePR(ctx, repoPath, branchName,
			fmt.Sprintf("Nightly Maintenance - %s", time.Now().Format("2006-01-02")),
			"Automated maintenance changes.")
	}

	return nil
}

// CleanupOldMaintenanceBranches removes old maintenance branches
func CleanupOldMaintenanceBranches(ctx context.Context, repoPath string) error {
	return CleanupOldBranches(ctx, repoPath, 7*24*time.Hour)
}

// GetMaintenanceHistory returns history of maintenance runs
func GetMaintenanceHistory(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "--grep=maintenance", "-20")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}
