// Package cost provides cost tracking and budget management.
package cost

import (
	"github.com/yourname/helm/internal/provider"
)

// Calculator computes costs from token usage using the provider price catalog.
type Calculator struct {
	pc *provider.PriceCatalog
}

// NewCalculator creates a cost calculator.
func NewCalculator() *Calculator {
	return &Calculator{
		pc: provider.NewPriceCatalog(),
	}
}

// Calculate computes the cost for a given model and token usage.
func (c *Calculator) Calculate(model string, input, output, cacheRead, cacheWrite int64) float64 {
	return c.pc.Calculate(model, provider.Usage{
		InputTokens:      int(input),
		OutputTokens:     int(output),
		CacheReadTokens:  int(cacheRead),
		CacheWriteTokens: int(cacheWrite),
	})
}

// Estimate returns an estimated cost for expected token usage.
func (c *Calculator) Estimate(model string, expectedInput, expectedOutput int) float64 {
	return c.pc.Calculate(model, provider.Usage{
		InputTokens:  expectedInput,
		OutputTokens: expectedOutput,
	})
}
