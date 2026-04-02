package provider

import (
	"context"
	"fmt"
	"strings"
)

// FallbackChain holds an ordered list of provider names to try.
type FallbackChain struct {
	names []string
}

// NewFallbackChain creates a fallback chain from provider names.
func NewFallbackChain(providers ...string) *FallbackChain {
	return &FallbackChain{names: providers}
}

// Names returns the ordered list of provider names.
func (fc *FallbackChain) Names() []string {
	return fc.names
}

// IsRateLimitError checks if an error is likely a rate limit error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "rate_limit") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "overloaded")
}

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return IsRateLimitError(err) ||
		strings.Contains(msg, "500") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused")
}

// ProviderConfig holds configuration for a single provider.
type ProviderConfig struct {
	Name         string
	APIKey       string
	BaseURL      string
	DefaultModel string
}

// RouterConfig holds the full configuration for the ProviderRouter.
type RouterConfig struct {
	Providers      []ProviderConfig
	FallbackChain  []string
	MaxRetries     int
	RateLimitRetry bool
}

// ChatWithFallback tries providers in order until one succeeds.
func ChatWithFallback(ctx context.Context, providers map[string]Provider, chain []string, req ChatRequest) (*ChatResponse, error) {
	var lastErr error
	for _, name := range chain {
		p, ok := providers[name]
		if !ok {
			lastErr = fmt.Errorf("provider %q not configured", name)
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
