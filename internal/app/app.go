// Package app provides the main application orchestration.
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourname/helm/internal/config"
	"github.com/yourname/helm/internal/cost"
	"github.com/yourname/helm/internal/db"
	"github.com/yourname/helm/internal/memory"
	"github.com/yourname/helm/internal/prompt"
	"github.com/yourname/helm/internal/provider"
	"github.com/yourname/helm/internal/session"
	"github.com/yourname/helm/internal/ui"
)

// App is the main application container.
type App struct {
	Config         *config.Config
	DB             *db.DB
	ProviderRouter *provider.ProviderRouter
	SessionManager *session.Manager
	MemoryEngine   *memory.Engine
	PromptLibrary  *prompt.PromptLibrary
	CostTracker    *cost.Tracker
}

// New creates a new App instance.
func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// Initialize database
	database, err := db.Open(dbFilePath())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	queries := database

	// Initialize provider router
	router, err := provider.NewProviderRouter(providerConfigFromConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("initialize provider router: %w", err)
	}

	// Initialize session manager
	sessionManager := session.NewManager(queries)

	// Initialize memory engine
	memoryEngine := memory.NewEngine(queries)

	// Initialize prompt library
	promptLibrary := prompt.NewLibrary()

	// Initialize cost tracker
	costTracker := cost.NewTracker(queries)

	// Ensure budget record exists
	project, _ := os.Getwd()
	_, _ = queries.GetBudget(ctx, project)

	return &App{
		Config:         cfg,
		DB:             queries,
		ProviderRouter: router,
		SessionManager: sessionManager,
		MemoryEngine:   memoryEngine,
		PromptLibrary:  promptLibrary,
		CostTracker:    costTracker,
	}, nil
}

// providerConfigFromConfig converts config.Config to provider.RouterConfig.
func providerConfigFromConfig(cfg *config.Config) provider.RouterConfig {
	var providers []provider.ProviderConfig
	if cfg.Providers.Anthropic.APIKey != "" {
		providers = append(providers, provider.ProviderConfig{
			Name:    "anthropic",
			APIKey:  cfg.Providers.Anthropic.APIKey,
			BaseURL: cfg.Providers.Anthropic.BaseURL,
		})
	}
	if cfg.Providers.OpenAI.APIKey != "" {
		providers = append(providers, provider.ProviderConfig{
			Name:    "openai",
			APIKey:  cfg.Providers.OpenAI.APIKey,
			BaseURL: cfg.Providers.OpenAI.BaseURL,
		})
	}
	if cfg.Providers.Google.APIKey != "" {
		providers = append(providers, provider.ProviderConfig{
			Name:   "google",
			APIKey: cfg.Providers.Google.APIKey,
		})
	}
	if cfg.Providers.OpenRouter.APIKey != "" {
		providers = append(providers, provider.ProviderConfig{
			Name:   "openrouter",
			APIKey: cfg.Providers.OpenRouter.APIKey,
		})
	}

	return provider.RouterConfig{
		Providers:     providers,
		FallbackChain: cfg.Router.FallbackChain,
		MaxRetries:    cfg.Router.MaxRetries,
	}
}

// RunTUI starts the TUI application.
func (a *App) RunTUI() error {
	return ui.Run(a.DB)
}

// Close cleans up resources.
func (a *App) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}

// dbFilePath returns the database file path.
func dbFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".helm/helm.db"
	}
	return filepath.Join(home, ".helm", "helm.db")
}
