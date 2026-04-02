package setup

import (
	"context"
	"fmt"
	"os"

	"github.com/yourname/helm/internal/config"
	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/memory"
)

// InitResult contains the results of helm init.
type InitResult struct {
	ProjectInfo    ProjectInfo
	Providers      map[string]bool
	MemoriesAdded  int
	ConfigPath     string
	PromptsSuggested []string
}

// Init initializes HELM for the current project.
func Init(ctx context.Context) (*InitResult, error) {
	result := &InitResult{}

	// Detect project info
	result.ProjectInfo = DetectProject()
	fmt.Printf("\n🔍 Analyzing project...\n")
	fmt.Printf("  Language: %s\n", result.ProjectInfo.Language)
	fmt.Printf("  Framework: %s\n", result.ProjectInfo.Framework)
	fmt.Printf("  Package Manager: %s\n", result.ProjectInfo.PackageManager)
	fmt.Printf("  Test Framework: %s\n", result.ProjectInfo.TestFramework)

	// Check providers
	result.Providers = CheckProviders()

	// Initialize database
	database, err := db.Open("")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	// Create project memory
	memEngine := memory.NewEngine(database.Queries)
	memItems := BuildProjectMemory(result.ProjectInfo)

	project, _ := os.Getwd()
	for _, item := range memItems {
		err := memEngine.Store(ctx, project, memory.MemoryType(item.Type), item.Key, item.Value, "auto")
		if err == nil {
			result.MemoriesAdded++
		}
	}
	fmt.Printf("\n📋 Building project memory...\n")
	fmt.Printf("  Found %d initial memories\n", result.MemoriesAdded)

	// Create config if it doesn't exist
	cfgPath := config.ConfigPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		if err := cfg.Save(cfgPath); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		result.ConfigPath = cfgPath
	}

	// Suggest prompts
	result.PromptsSuggested = SuggestPrompts(result.ProjectInfo)

	return result, nil
}

// PrintResults prints the init results to stdout.
func PrintResults(result *InitResult) {
	fmt.Printf("\n⚙️ Configuring providers...\n")
	for name, configured := range result.Providers {
		status := "○"
		if configured {
			status = "✓"
		}
		fmt.Printf("  %s %s\n", status, name)
	}

	fmt.Printf("\n💡 Suggested prompts:\n")
	for _, prompt := range result.PromptsSuggested {
		fmt.Printf("  - %s\n", prompt)
	}

	fmt.Printf("\n✅ HELM ready! Run `helm` to start.\n")
}
