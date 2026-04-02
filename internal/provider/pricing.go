package provider

// ModelPrice stores per-million-token pricing in USD cents.
type ModelPrice struct {
	InputPerM      float64
	OutputPerM     float64
	CacheReadPerM  float64
	CacheWritePerM float64
}

// PriceCatalog provides per-model pricing lookup.
type PriceCatalog struct {
	prices map[string]ModelPrice
}

// NewPriceCatalog creates a catalog with built-in pricing.
func NewPriceCatalog() *PriceCatalog {
	pc := &PriceCatalog{
		prices: make(map[string]ModelPrice),
	}
	pc.loadBuiltIn()
	return pc
}

func (pc *PriceCatalog) loadBuiltIn() {
	// Anthropic Claude pricing (per 1M tokens, USD)
	pc.set("claude-sonnet-4-20250514", ModelPrice{
		InputPerM: 3.0, OutputPerM: 15.0,
		CacheReadPerM: 0.3, CacheWritePerM: 3.75,
	})
	pc.set("claude-opus-4-20250514", ModelPrice{
		InputPerM: 15.0, OutputPerM: 75.0,
		CacheReadPerM: 1.5, CacheWritePerM: 18.75,
	})
	pc.set("claude-haiku-3-5-20241022", ModelPrice{
		InputPerM: 0.25, OutputPerM: 1.25,
		CacheReadPerM: 0.03, CacheWritePerM: 0.3,
	})

	// OpenAI pricing (per 1M tokens, USD)
	pc.set("gpt-4o", ModelPrice{
		InputPerM: 2.5, OutputPerM: 10.0,
		CacheReadPerM: 1.25, CacheWritePerM: 2.5,
	})
	pc.set("gpt-4o-mini", ModelPrice{
		InputPerM: 0.15, OutputPerM: 0.6,
		CacheReadPerM: 0.075, CacheWritePerM: 0.15,
	})

	// Google Gemini pricing (per 1M tokens, USD)
	pc.set("gemini-2.5-pro", ModelPrice{
		InputPerM: 1.25, OutputPerM: 10.0,
		CacheReadPerM: 0, CacheWritePerM: 0,
	})
	pc.set("gemini-2.5-flash", ModelPrice{
		InputPerM: 0.15, OutputPerM: 0.6,
		CacheReadPerM: 0, CacheWritePerM: 0,
	})
}

func (pc *PriceCatalog) set(id string, price ModelPrice) {
	pc.prices[id] = price
}

// Get returns pricing for a model. Falls back to keyword matching.
func (pc *PriceCatalog) Get(modelID string) (ModelPrice, bool) {
	if p, ok := pc.prices[modelID]; ok {
		return p, true
	}
	// Fallback: keyword match
	for id, p := range pc.prices {
		if containsAny(modelID, normalizeModel(id)) {
			return p, true
		}
	}
	return ModelPrice{}, false
}

// Calculate computes the cost for a given token usage.
func (pc *PriceCatalog) Calculate(modelID string, usage Usage) float64 {
	price, ok := pc.Get(modelID)
	if !ok {
		return 0
	}
	inputCost := float64(usage.InputTokens) / 1_000_000 * price.InputPerM
	outputCost := float64(usage.OutputTokens) / 1_000_000 * price.OutputPerM
	cacheReadCost := float64(usage.CacheReadTokens) / 1_000_000 * price.CacheReadPerM
	cacheWriteCost := float64(usage.CacheWriteTokens) / 1_000_000 * price.CacheWritePerM
	return inputCost + outputCost + cacheReadCost + cacheWriteCost
}

// normalizeModel strips date suffixes for matching.
func normalizeModel(model string) string {
	// Strip -YYYYMMDD suffix
	for i := len(model) - 1; i >= 0; i-- {
		if model[i] == '-' {
			suffix := model[i+1:]
			if len(suffix) == 8 {
				allDigits := true
				for _, c := range suffix {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					return model[:i]
				}
			}
			break
		}
	}
	return model
}
