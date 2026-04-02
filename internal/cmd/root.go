// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/app"
	"github.com/yourname/helm/internal/config"
)

var (
	cfgFile     string
	cfg         *config.Config
	application *app.App
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "helm",
	Short: "HELM — Personal Coding Agent Control Plane",
	Long: `HELM is a unified TUI-first control plane for managing AI coding agents
across all providers (Claude, Codex, Gemini, Ollama, etc.).

You steer. Agents row.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		application, err = app.New(cfg)
		if err != nil {
			return fmt.Errorf("initialize app: %w", err)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// No subcommand — launch TUI
		return application.RunTUI()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.helm/helm.toml)")

	// Subcommands are registered via init() in their respective files
}

// GetApp returns the initialized app instance.
func GetApp() *app.App {
	return application
}

// GetConfig returns the loaded config.
func GetConfig() *config.Config {
	return cfg
}
