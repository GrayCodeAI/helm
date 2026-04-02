package provider

import (
	"context"
	"fmt"
)

// ProviderRouter routes chat requests to the appropriate provider.
type ProviderRouter struct {
	providers     map[string]Provider
	fallbackChain []string
	maxRetries    int
	modelCatalog  *ModelCatalog
	priceCatalog  *PriceCatalog
}

// NewProviderRouter creates a new router from config.
func NewProviderRouter(cfg RouterConfig) (*ProviderRouter, error) {
	r := &ProviderRouter{
		providers:     make(map[string]Provider),
		fallbackChain: cfg.FallbackChain,
		maxRetries:    cfg.MaxRetries,
		modelCatalog:  NewModelCatalog(),
		priceCatalog:  NewPriceCatalog(),
	}
	if r.maxRetries == 0 {
		r.maxRetries = 3
	}

	for _, pc := range cfg.Providers {
		var p Provider
		switch pc.Name {
		case "anthropic":
			p = NewAnthropicProvider(pc.APIKey)
		case "openai":
			p = NewOpenAIProvider(pc.APIKey)
		case "google":
			p = NewGoogleProvider(pc.APIKey)
		case "ollama":
			p = NewOllamaProvider(pc.BaseURL)
		case "openrouter":
			p = NewOpenRouterProvider(pc.APIKey)
		case "custom":
			p = NewCustomProvider(pc.Name, pc.BaseURL, pc.APIKey)
		default:
			return nil, fmt.Errorf("unknown provider: %s", pc.Name)
		}
		r.providers[pc.Name] = p
	}

	return r, nil
}

// NewProviderRouterWithProviders creates a router with pre-built providers (for testing).
func NewProviderRouterWithProviders(providers map[string]Provider, fallbackChain []string) *ProviderRouter {
	return &ProviderRouter{
		providers:     providers,
		fallbackChain: fallbackChain,
		maxRetries:    3,
		modelCatalog:  NewModelCatalog(),
		priceCatalog:  NewPriceCatalog(),
	}
}

// Provider returns the provider by name.
func (r *ProviderRouter) Provider(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Providers returns all registered provider names.
func (r *ProviderRouter) Providers() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ModelCatalog returns the model catalog.
func (r *ProviderRouter) ModelCatalog() *ModelCatalog {
	return r.modelCatalog
}

// PriceCatalog returns the price catalog.
func (r *ProviderRouter) PriceCatalog() *PriceCatalog {
	return r.priceCatalog
}

// Route sends a chat request to the appropriate provider with fallback.
func (r *ProviderRouter) Route(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	providerName := r.modelCatalog.Provider(req.Model)
	if providerName == "unknown" {
		providerName = r.selectDefaultProvider()
	}

	resp, err := r.chatWithFallback(ctx, req, providerName)
	if err != nil {
		return nil, fmt.Errorf("route: %w", err)
	}
	return resp, nil
}

// RouteTo sends a chat request to a specific provider without fallback.
func (r *ProviderRouter) RouteTo(ctx context.Context, providerName string, req ChatRequest) (*ChatResponse, error) {
	p, ok := r.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("provider %s: %w", providerName, err)
	}
	return resp, nil
}

func (r *ProviderRouter) chatWithFallback(ctx context.Context, req ChatRequest, primary string) (*ChatResponse, error) {
	chain := []string{primary}
	for _, name := range r.fallbackChain {
		if name != primary {
			chain = append(chain, name)
		}
	}

	var lastErr error
	for _, name := range chain {
		p, ok := r.providers[name]
		if !ok {
			continue
		}

		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = fmt.Errorf("provider %s: %w", name, err)
		if !IsRetryableError(err) {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

func (r *ProviderRouter) selectDefaultProvider() string {
	if len(r.fallbackChain) > 0 {
		return r.fallbackChain[0]
	}
	for name := range r.providers {
		return name
	}
	return ""
}
