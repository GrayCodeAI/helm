package provider

// ModelInfo contains metadata about a model including pricing and capabilities.
type ModelInfo struct {
	ID                string
	Provider          string
	DisplayName       string
	ContextWindow     int
	MaxOutputTokens   int
	SupportsImages    bool
	SupportsStreaming bool
}

// ModelCatalog provides model metadata and pricing lookup.
type ModelCatalog struct {
	models map[string]ModelInfo
}

// NewModelCatalog creates a catalog with built-in models.
func NewModelCatalog() *ModelCatalog {
	mc := &ModelCatalog{
		models: make(map[string]ModelInfo),
	}
	mc.loadBuiltIn()
	return mc
}

func (mc *ModelCatalog) loadBuiltIn() {
	// Anthropic Claude models
	mc.add(ModelInfo{
		ID: "claude-sonnet-4-20250514", Provider: "anthropic",
		DisplayName: "Claude Sonnet 4", ContextWindow: 200000,
		MaxOutputTokens: 64000, SupportsStreaming: true,
	})
	mc.add(ModelInfo{
		ID: "claude-opus-4-20250514", Provider: "anthropic",
		DisplayName: "Claude Opus 4", ContextWindow: 200000,
		MaxOutputTokens: 64000, SupportsStreaming: true,
	})
	mc.add(ModelInfo{
		ID: "claude-haiku-3-5-20241022", Provider: "anthropic",
		DisplayName: "Claude Haiku 3.5", ContextWindow: 200000,
		MaxOutputTokens: 8192, SupportsStreaming: true,
	})

	// OpenAI models
	mc.add(ModelInfo{
		ID: "gpt-4o", Provider: "openai",
		DisplayName: "GPT-4o", ContextWindow: 128000,
		MaxOutputTokens: 16384, SupportsImages: true, SupportsStreaming: true,
	})
	mc.add(ModelInfo{
		ID: "gpt-4o-mini", Provider: "openai",
		DisplayName: "GPT-4o Mini", ContextWindow: 128000,
		MaxOutputTokens: 16384, SupportsImages: true, SupportsStreaming: true,
	})

	// Google Gemini models
	mc.add(ModelInfo{
		ID: "gemini-2.5-pro", Provider: "google",
		DisplayName: "Gemini 2.5 Pro", ContextWindow: 1000000,
		MaxOutputTokens: 65536, SupportsImages: true, SupportsStreaming: true,
	})
	mc.add(ModelInfo{
		ID: "gemini-2.5-flash", Provider: "google",
		DisplayName: "Gemini 2.5 Flash", ContextWindow: 1000000,
		MaxOutputTokens: 65536, SupportsImages: true, SupportsStreaming: true,
	})
}

func (mc *ModelCatalog) add(m ModelInfo) {
	mc.models[m.ID] = m
}

// Get returns model info by ID.
func (mc *ModelCatalog) Get(id string) (ModelInfo, bool) {
	m, ok := mc.models[id]
	return m, ok
}

// List returns all known models.
func (mc *ModelCatalog) List() []ModelInfo {
	result := make([]ModelInfo, 0, len(mc.models))
	for _, m := range mc.models {
		result = append(result, m)
	}
	return result
}

// Provider returns the provider name for a given model ID.
func (mc *ModelCatalog) Provider(modelID string) string {
	if m, ok := mc.models[modelID]; ok {
		return m.Provider
	}
	// Fallback: try to detect from model name
	switch {
	case containsAny(modelID, "claude", "sonnet", "opus", "haiku"):
		return "anthropic"
	case containsAny(modelID, "gpt", "o1", "o3"):
		return "openai"
	case containsAny(modelID, "gemini"):
		return "google"
	default:
		return "unknown"
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
