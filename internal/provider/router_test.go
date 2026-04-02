package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	name  string
	resp  *ChatResponse
	err   error
	calls int
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	m.calls++
	return m.resp, m.err
}

func TestProviderRouter_Route(t *testing.T) {
	t.Parallel()

	mockAnthropic := &mockProvider{
		name: "anthropic",
		resp: &ChatResponse{
			Content:  "Hello from Anthropic",
			Provider: "anthropic",
			Model:    "claude-sonnet-4",
			Usage:    Usage{InputTokens: 10, OutputTokens: 20},
		},
	}

	router := NewProviderRouterWithProviders(
		map[string]Provider{"anthropic": mockAnthropic},
		[]string{"anthropic"},
	)

	resp, err := router.Route(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello from Anthropic", resp.Content)
	assert.Equal(t, "anthropic", resp.Provider)
	assert.Equal(t, 1, mockAnthropic.calls)
}

func TestProviderRouter_Fallback(t *testing.T) {
	t.Parallel()

	mockPrimary := &mockProvider{
		name: "anthropic",
		err:  fmt.Errorf("rate limit: 429"),
	}
	mockFallback := &mockProvider{
		name: "openai",
		resp: &ChatResponse{
			Content:  "Hello from OpenAI",
			Provider: "openai",
		},
	}

	router := NewProviderRouterWithProviders(
		map[string]Provider{
			"anthropic": mockPrimary,
			"openai":    mockFallback,
		},
		[]string{"anthropic", "openai"},
	)

	resp, err := router.Route(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello from OpenAI", resp.Content)
	assert.Equal(t, 1, mockPrimary.calls)
	assert.Equal(t, 1, mockFallback.calls)
}

func TestProviderRouter_FallbackNonRetryable(t *testing.T) {
	t.Parallel()

	mockPrimary := &mockProvider{
		name: "anthropic",
		err:  fmt.Errorf("invalid API key"),
	}
	mockFallback := &mockProvider{
		name: "openai",
		resp: &ChatResponse{Content: "fallback"},
	}

	router := NewProviderRouterWithProviders(
		map[string]Provider{
			"anthropic": mockPrimary,
			"openai":    mockFallback,
		},
		[]string{"anthropic", "openai"},
	)

	_, err := router.Route(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	require.Error(t, err)
	assert.Equal(t, 1, mockPrimary.calls)
	assert.Equal(t, 0, mockFallback.calls)
}

func TestProviderRouter_RouteTo(t *testing.T) {
	t.Parallel()

	mockOpenAI := &mockProvider{
		name: "openai",
		resp: &ChatResponse{
			Content:  "Hello from OpenAI",
			Provider: "openai",
		},
	}

	router := NewProviderRouterWithProviders(
		map[string]Provider{"openai": mockOpenAI},
		[]string{"openai"},
	)

	resp, err := router.RouteTo(context.Background(), "openai", ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello from OpenAI", resp.Content)
}

func TestProviderRouter_RouteToNotFound(t *testing.T) {
	t.Parallel()

	router := NewProviderRouterWithProviders(
		map[string]Provider{},
		[]string{},
	)

	_, err := router.RouteTo(context.Background(), "unknown", ChatRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIsRateLimitError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"429 status", fmt.Errorf("status 429: too many requests"), true},
		{"rate limit text", fmt.Errorf("rate limit exceeded"), true},
		{"rate_limit text", fmt.Errorf("rate_limit_error"), true},
		{"too many requests", fmt.Errorf("too many requests"), true},
		{"overloaded", fmt.Errorf("overloaded"), true},
		{"other error", fmt.Errorf("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsRateLimitError(tt.err))
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"rate limit", fmt.Errorf("429 rate limit"), true},
		{"500 error", fmt.Errorf("status 500"), true},
		{"503 error", fmt.Errorf("status 503"), true},
		{"timeout", fmt.Errorf("context deadline exceeded: timeout"), true},
		{"connection refused", fmt.Errorf("connection refused"), true},
		{"invalid key", fmt.Errorf("invalid API key"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsRetryableError(tt.err))
		})
	}
}

func TestModelCatalog(t *testing.T) {
	t.Parallel()

	mc := NewModelCatalog()

	info, ok := mc.Get("claude-sonnet-4-20250514")
	require.True(t, ok)
	assert.Equal(t, "anthropic", info.Provider)
	assert.Equal(t, 200000, info.ContextWindow)

	_, ok = mc.Get("unknown-model")
	assert.False(t, ok)

	assert.Equal(t, "anthropic", mc.Provider("claude-sonnet-4-20250514"))
	assert.Equal(t, "openai", mc.Provider("gpt-4o"))
	assert.Equal(t, "google", mc.Provider("gemini-2.5-pro"))
}

func TestPriceCatalog(t *testing.T) {
	t.Parallel()

	pc := NewPriceCatalog()

	price, ok := pc.Get("claude-sonnet-4-20250514")
	require.True(t, ok)
	assert.Equal(t, 3.0, price.InputPerM)
	assert.Equal(t, 15.0, price.OutputPerM)

	cost := pc.Calculate("claude-sonnet-4-20250514", Usage{
		InputTokens:  1000,
		OutputTokens: 500,
	})
	expected := float64(1000)/1_000_000*3.0 + float64(500)/1_000_000*15.0
	assert.InDelta(t, expected, cost, 0.0001)

	_, ok = pc.Get("unknown-model")
	assert.False(t, ok)
}

func TestNormalizeModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input, want string
	}{
		{"claude-sonnet-4-20250514", "claude-sonnet-4"},
		{"gpt-4o", "gpt-4o"},
		{"claude-opus-4-20250514", "claude-opus-4"},
		{"gemini-2.5-flash", "gemini-2.5-flash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, normalizeModel(tt.input))
		})
	}
}

func TestChatWithFallback(t *testing.T) {
	t.Parallel()

	p1 := &mockProvider{name: "p1", err: fmt.Errorf("500 error")}
	p2 := &mockProvider{name: "p2", resp: &ChatResponse{Content: "ok"}}

	providers := map[string]Provider{"p1": p1, "p2": p2}
	chain := []string{"p1", "p2"}

	resp, err := ChatWithFallback(context.Background(), providers, chain, ChatRequest{})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Content)
	assert.Equal(t, 1, p1.calls)
	assert.Equal(t, 1, p2.calls)
}

func TestChatWithFallbackAllFail(t *testing.T) {
	t.Parallel()

	p1 := &mockProvider{name: "p1", err: fmt.Errorf("500 error 1")}
	p2 := &mockProvider{name: "p2", err: fmt.Errorf("503 error 2")}

	providers := map[string]Provider{"p1": p1, "p2": p2}
	chain := []string{"p1", "p2"}

	_, err := ChatWithFallback(context.Background(), providers, chain, ChatRequest{})
	require.Error(t, err)
}
