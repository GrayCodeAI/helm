package provider

import (
	"context"
	"testing"
)

func BenchmarkPriceCatalogCalculate(b *testing.B) {
	pc := NewPriceCatalog()
	usage := Usage{
		InputTokens:      1000,
		OutputTokens:     500,
		CacheReadTokens:  200,
		CacheWriteTokens: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Calculate("claude-sonnet-4-20250514", usage)
	}
}

func BenchmarkPriceCatalogGet(b *testing.B) {
	pc := NewPriceCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.Get("claude-sonnet-4-20250514")
	}
}

func BenchmarkModelCatalogGet(b *testing.B) {
	mc := NewModelCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.Get("claude-sonnet-4-20250514")
	}
}

func BenchmarkModelCatalogProvider(b *testing.B) {
	mc := NewModelCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.Provider("claude-sonnet-4-20250514")
	}
}

func BenchmarkProviderRouter(b *testing.B) {
	router := NewProviderRouterWithProviders(map[string]Provider{
		"anthropic": NewAnthropicProvider("test-key"),
		"openai":    NewOpenAIProvider("test-key"),
	}, []string{"anthropic", "openai"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = router.Route(context.Background(), ChatRequest{
			Model: "claude-sonnet-4",
		})
	}
}
