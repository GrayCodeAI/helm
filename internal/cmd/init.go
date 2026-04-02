// Package cmd provides the CLI commands for HELM.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourname/helm/internal/config"
	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/setup"
)

var (
	initForce bool
	helmDir   = ".helm"
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize HELM in the current directory",
	Long: `Initialize HELM by:
1. Detecting project language/framework
2. Building initial project memory
3. Configuring providers from environment
4. Creating .helm/ directory with config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runInit(); err != nil {
			return err
		}
		fmt.Println("\n✅ HELM ready! Run `helm` to start.")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Force re-initialization")
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	// Check if already initialized
	if _, err := os.Stat(helmDir); err == nil && !initForce {
		return fmt.Errorf("HELM already initialized. Use --force to reinitialize")
	}

	fmt.Println("🔍 Analyzing project...")

	// Detect project info
	projectInfo := setup.DetectProject()
	fmt.Printf("  Language: %s\n", projectInfo.Language)
	fmt.Printf("  Framework: %s\n", projectInfo.Framework)
	fmt.Printf("  Package Manager: %s\n", projectInfo.PackageManager)

	// Create .helm directory
	if err := os.MkdirAll(helmDir, 0755); err != nil {
		return fmt.Errorf("create .helm directory: %w", err)
	}

	// Create global config if not exists
	configPath := config.ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("save global config: %w", err)
		}
		fmt.Printf("\n⚙️  Created global config: %s\n", configPath)
	}

	// Load or create local config
	localConfigPath := filepath.Join(helmDir, "helm.toml")
	cfg := config.DefaultConfig()

	// Check for provider API keys
	providers := setup.CheckProviders()
	fmt.Println("\n⚙️  Configuring providers...")
	for name, configured := range providers {
		if configured {
			fmt.Printf("  ✓ %s (from environment)\n", name)
		} else {
			fmt.Printf("  ○ %s (not configured)\n", name)
		}
	}

	// Save local config
	if err := cfg.Save(localConfigPath); err != nil {
		return fmt.Errorf("save local config: %w", err)
	}

	// Initialize database
	fmt.Println("\n📦 Initializing database...")
	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".helm")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "helm.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	// Build initial project memory
	fmt.Println("\n📝 Building project memory...")
	memories := setup.BuildProjectMemory(projectInfo)
	fmt.Printf("  Found %d conventions from existing code\n", len(memories))

	// Create prompt suggestions
	fmt.Println("\n💡 Suggested prompts:")
	suggestions := setup.SuggestPrompts(projectInfo)
	for _, s := range suggestions {
		fmt.Printf("  - %s\n", s)
	}

	return nil
}
