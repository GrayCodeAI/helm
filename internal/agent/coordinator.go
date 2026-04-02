// Package agent provides the agent coordinator for multi-turn conversations.
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/yourname/helm/internal/provider"
	"github.com/yourname/helm/internal/pubsub"
)

// Message represents a conversation message
type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Timestamp time.Time  `json:"timestamp"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult represents a tool execution result
type ToolResult struct {
	ID     string `json:"id"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// Coordinator manages the agent conversation loop
type Coordinator struct {
	provider provider.Provider
	tools    map[string]ToolHandler
	maxTurns int
	broker   *pubsub.Broker
}

// ToolHandler handles tool execution
type ToolHandler func(ctx context.Context, args string) (string, error)

// Config configures the coordinator
type Config struct {
	Provider provider.Provider
	MaxTurns int
}

// NewCoordinator creates a new agent coordinator
func NewCoordinator(cfg Config, broker *pubsub.Broker) *Coordinator {
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 50
	}
	return &Coordinator{
		provider: cfg.Provider,
		tools:    make(map[string]ToolHandler),
		maxTurns: cfg.MaxTurns,
		broker:   broker,
	}
}

// RegisterTool registers a tool handler
func (c *Coordinator) RegisterTool(name string, handler ToolHandler) {
	c.tools[name] = handler
}

// Run executes the conversation loop
func (c *Coordinator) Run(ctx context.Context, systemPrompt string, messages []Message) ([]Message, error) {
	allMessages := append([]Message{{
		Role:    "system",
		Content: systemPrompt,
	}}, messages...)

	for turn := 0; turn < c.maxTurns; turn++ {
		// Check context
		select {
		case <-ctx.Done():
			return allMessages, ctx.Err()
		default:
		}

		// Call provider
		req := provider.ChatRequest{
			Model:    "default",
			Messages: toProviderMessages(allMessages),
		}

		resp, err := c.provider.Chat(ctx, req)
		if err != nil {
			return allMessages, fmt.Errorf("provider chat: %w", err)
		}

		// Add assistant response
		assistantMsg := Message{
			Role:      "assistant",
			Content:   resp.Content,
			Timestamp: time.Now(),
		}
		allMessages = append(allMessages, assistantMsg)

		// Publish event
		if c.broker != nil {
			c.broker.Publish(pubsub.EventSessionUpdated, map[string]interface{}{
				"turn":    turn,
				"content": resp.Content,
			})
		}

		// Check if response has tool calls
		if len(resp.ToolCalls) == 0 {
			// No tool calls, conversation complete
			break
		}

		// Execute tool calls
		for _, tc := range resp.ToolCalls {
			handler, ok := c.tools[tc.Name]
			if !ok {
				toolResult := Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: tool %s not found", tc.Name),
				}
				allMessages = append(allMessages, toolResult)
				continue
			}

			output, err := handler(ctx, tc.Arguments)
			if err != nil {
				toolResult := Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: %v", err),
				}
				allMessages = append(allMessages, toolResult)
			} else {
				toolResult := Message{
					Role:    "tool",
					Content: output,
				}
				allMessages = append(allMessages, toolResult)
			}
		}
	}

	return allMessages, nil
}

// Cancel stops the coordinator
func (c *Coordinator) Cancel() {
	// Cancel any running operations
}

func toProviderMessages(messages []Message) []provider.Message {
	result := make([]provider.Message, len(messages))
	for i, m := range messages {
		result[i] = provider.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}
