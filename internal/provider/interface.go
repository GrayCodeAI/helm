// Package provider provides LLM provider adapters and routing.
package provider

import (
	"context"
)

// Provider is the unified interface for all LLM provider adapters.
type Provider interface {
	// Name returns the provider identifier (e.g., "anthropic", "openai").
	Name() string

	// Chat sends a non-streaming chat request.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// Message represents a single message in a chat conversation.
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// ToolCall represents a tool invocation by the assistant.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// ChatRequest is the unified request format for all providers.
type ChatRequest struct {
	Model     string
	Messages  []Message
	MaxTokens int
}

// Usage contains token usage metadata from a provider response.
type Usage struct {
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

// ChatResponse is the unified response format from all providers.
type ChatResponse struct {
	Content   string
	ToolCalls []ToolCall
	Usage     Usage
	Provider  string
	Model     string
}
