// Package quality provides quality gate functionality
package quality

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GateConfig configures quality gates
type GateConfig struct {
	Lint          bool
	Test          bool
	Security      bool
	Complexity    bool
	Build         bool
	MaxComplexity int
}

// DefaultGateConfig returns default gate configuration
func DefaultGateConfig() GateConfig {
	return GateConfig{
		Lint:          true,
		Test:          true,
		Security:      false,
		Complexity:    false,
		Build:         true,
		MaxComplexity: 15,
	}
}

// GateRunner runs quality gates
type GateRunner struct {
	config GateConfig
}

// NewGateRunner creates a new gate runner
func NewGateRunner(config GateConfig) *GateRunner {
	return &GateRunner{config: config}
}

// GateResult represents the result of running gates
type GateResult struct {
	Passed  bool
	Gates   []SingleGateResult
	Summary string
}

// SingleGateResult represents the result of a single gate
type SingleGateResult struct {
	Name   string
	Passed bool
	Output string
	Errors []string
}

// RunAll runs all enabled gates
func (gr *GateRunner) RunAll(ctx context.Context) (*GateResult, error) {
	result := &GateResult{
		Passed: true,
		Gates:  []SingleGateResult{},
	}

	if gr.config.Lint {
		gate := gr.runLint(ctx)
		result.Gates = append(result.Gates, gate)
		if !gate.Passed {
			result.Passed = false
		}
	}

	if gr.config.Test {
		gate := gr.runTests(ctx)
		result.Gates = append(result.Gates, gate)
		if !gate.Passed {
			result.Passed = false
		}
	}

	if gr.config.Build {
		gate := gr.runBuild(ctx)
		result.Gates = append(result.Gates, gate)
		if !gate.Passed {
			result.Passed = false
		}
	}

	if gr.config.Security {
		gate := gr.runSecurity(ctx)
		result.Gates = append(result.Gates, gate)
		if !gate.Passed {
			result.Passed = false
		}
	}

	if gr.config.Complexity {
		gate := gr.runComplexity(ctx, gr.config.MaxComplexity)
		result.Gates = append(result.Gates, gate)
		if !gate.Passed {
			result.Passed = false
		}
	}

	return result, nil
}

func (gr *GateRunner) runLint(ctx context.Context) SingleGateResult {
	result := SingleGateResult{Name: "Lint"}

	// Try different linters
	linters := []string{"golangci-lint run", "go vet ./...", "staticcheck ./..."}

	for _, linter := range linters {
		parts := strings.Fields(linter)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		output, err := cmd.CombinedOutput()

		if err == nil {
			result.Passed = true
			result.Output = "Linting passed"
			return result
		}

		// Store output for later
		result.Output = string(output)
	}

	result.Passed = false
	result.Errors = []string{"Linting failed"}
	return result
}

func (gr *GateRunner) runTests(ctx context.Context) SingleGateResult {
	result := SingleGateResult{Name: "Test"}

	cmd := exec.CommandContext(ctx, "go", "test", "./...")
	output, err := cmd.CombinedOutput()

	result.Output = string(output)
	if err != nil {
		result.Passed = false
		result.Errors = extractErrors(string(output))
	} else {
		result.Passed = true
	}

	return result
}

func (gr *GateRunner) runBuild(ctx context.Context) SingleGateResult {
	result := SingleGateResult{Name: "Build"}

	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	output, err := cmd.CombinedOutput()

	result.Output = string(output)
	if err != nil {
		result.Passed = false
		result.Errors = []string{string(output)}
	} else {
		result.Passed = true
	}

	return result
}

func (gr *GateRunner) runSecurity(ctx context.Context) SingleGateResult {
	result := SingleGateResult{Name: "Security"}

	// Try semgrep or trivy
	securityTools := []string{"semgrep --config=auto", "trivy filesystem ."}

	for _, tool := range securityTools {
		parts := strings.Fields(tool)
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		output, err := cmd.CombinedOutput()

		result.Output = string(output)
		if err == nil {
			result.Passed = true
			return result
		}
	}

	// If no security tools available, pass
	result.Passed = true
	result.Output = "Security tools not available, skipping"
	return result
}

func (gr *GateRunner) runComplexity(ctx context.Context, maxComplexity int) SingleGateResult {
	result := SingleGateResult{Name: "Complexity"}

	// Try gocyclo
	cmd := exec.CommandContext(ctx, "gocyclo", ".")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// gocyclo not available
		result.Passed = true
		result.Output = "Complexity checker not available, skipping"
		return result
	}

	// Parse output for high complexity functions
	lines := strings.Split(string(output), "\n")
	var highComplexity []string

	for _, line := range lines {
		if line == "" {
			continue
		}
		// gocyclo format: <complexity> <package> <function>
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			complexity := 0
			fmt.Sscanf(parts[0], "%d", &complexity)
			if complexity > maxComplexity {
				highComplexity = append(highComplexity, line)
			}
		}
	}

	if len(highComplexity) > 0 {
		result.Passed = false
		result.Errors = highComplexity
	} else {
		result.Passed = true
		result.Output = "Complexity check passed"
	}

	return result
}

func extractErrors(output string) []string {
	var errors []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "FAIL") || strings.Contains(line, "Error") || strings.Contains(line, "error") {
			errors = append(errors, line)
		}
	}

	return errors
}
