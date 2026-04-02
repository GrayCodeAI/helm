// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/db"
)

var (
	runProvider string
	runModel    string
	runPrompt   string
)

// runCmd represents the run command.
var runCmd = &cobra.Command{
	Use:   "run [prompt]",
	Short: "Run an agent with a prompt",
	Long: `Launch an AI agent session with the given prompt.

Examples:
  helm run "fix the login bug"
  helm run --provider anthropic "add auth middleware"
  helm run --prompt add-feature`,
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		promptText := strings.Join(args, " ")
		if promptText == "" && runPrompt != "" {
			// Load from prompt library
			p, ok := application.PromptLibrary.Get(runPrompt)
			if !ok {
				return fmt.Errorf("prompt not found: %s", runPrompt)
			}
			promptText = p.Template
		}
		if promptText == "" {
			return fmt.Errorf("please provide a prompt or use --prompt to select from library")
		}

		return runSession(promptText)
	},
}

func init() {
	runCmd.Flags().StringVarP(&runProvider, "provider", "p", "", "Provider to use (anthropic, openai, google, ollama)")
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use")
	runCmd.Flags().StringVar(&runPrompt, "prompt", "", "Use a named prompt from the library")
	rootCmd.AddCommand(runCmd)
}

func runSession(promptText string) error {
	ctx := context.Background()

	// Get project path
	project, err := os.Getwd()
	if err != nil {
		project = "unknown"
	}

	// Select provider
	providerName := runProvider
	if providerName == "" {
		providerName = application.Config.Router.FallbackChain[0]
	}

	// Get model
	model := runModel
	if model == "" {
		model = getDefaultModel(providerName)
	}

	// Create session record
	sessionID := uuid.New().String()
	_, err = application.DB.CreateSession(ctx, db.CreateSessionParams{
		ID:               sessionID,
		Provider:         providerName,
		Model:            model,
		Project:          project,
		Prompt:           db.ToNullString(promptText),
		Status:           "running",
		InputTokens:      0,
		OutputTokens:     0,
		CacheReadTokens:  0,
		CacheWriteTokens: 0,
		Cost:             0,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("🚀 Starting session: %s\n", sessionID[:8])
	fmt.Printf("   Provider: %s\n", providerName)
	fmt.Printf("   Model: %s\n", model)
	fmt.Printf("   Prompt: %.50s...\n\n", promptText)

	// Launch the actual agent based on provider
	startTime := time.Now()
	exitCode, err := launchAgent(providerName, promptText)
	duration := time.Since(startTime)

	// Update session status
	status := "done"
	if err != nil || exitCode != 0 {
		status = "failed"
	}

	_ = application.DB.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
		ID:      sessionID,
		Status:  status,
		EndedAt: db.ToNullString(time.Now().Format(time.RFC3339)),
	})

	// Record cost (placeholder - would parse actual usage)
	_, _ = application.DB.CreateCostRecord(ctx, db.CreateCostRecordParams{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Project:   project,
		Provider:  providerName,
		Model:     model,
	})

	if err != nil {
		return fmt.Errorf("agent failed: %w", err)
	}

	fmt.Printf("\n✅ Session completed in %s\n", duration.Round(time.Second))
	return nil
}

func launchAgent(provider, prompt string) (int, error) {
	switch provider {
	case "anthropic", "claude":
		return launchClaude(prompt)
	case "openai", "codex":
		return launchCodex(prompt)
	case "google", "gemini":
		return launchGemini(prompt)
	default:
		return 0, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func launchClaude(prompt string) (int, error) {
	// Launch Claude Code with the prompt
	cmd := exec.Command("claude", "--prompt", prompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func launchCodex(prompt string) (int, error) {
	cmd := exec.Command("codex", "--prompt", prompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func launchGemini(prompt string) (int, error) {
	return 0, fmt.Errorf("Gemini CLI not yet implemented")
}

func getDefaultModel(provider string) string {
	switch provider {
	case "anthropic", "claude":
		return application.Config.Providers.Anthropic.DefaultModel
	case "openai", "codex":
		return application.Config.Providers.OpenAI.DefaultModel
	case "google", "gemini":
		return application.Config.Providers.Google.DefaultModel
	case "ollama":
		return application.Config.Providers.Ollama.DefaultModel
	default:
		return "unknown"
	}
}
