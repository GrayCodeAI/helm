// Package agent provides the agent coordinator for multi-turn conversations.
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

// QueuedPrompt is a prompt waiting to be processed
type QueuedPrompt struct {
	ID        string
	Content   string
	Timestamp time.Time
}

// Coordinator manages the agent conversation loop
type Coordinator struct {
	provider      provider.Provider
	tools         map[string]ToolHandler
	maxTurns      int
	broker        *pubsub.Broker
	queue         []QueuedPrompt
	queueMu       sync.Mutex
	cancel        context.CancelFunc
	ctx           context.Context
	isBusy        bool
	summary       string
	largeModel    provider.Provider
	smallModel    provider.Provider
	contextWindow int
}

// ToolHandler handles tool execution
type ToolHandler func(ctx context.Context, args string) (string, error)

// Config configures the coordinator
type Config struct {
	Provider      provider.Provider
	MaxTurns      int
	LargeModel    provider.Provider
	SmallModel    provider.Provider
	ContextWindow int
}

// NewCoordinator creates a new agent coordinator
func NewCoordinator(cfg Config, broker *pubsub.Broker) *Coordinator {
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 50
	}
	if cfg.ContextWindow == 0 {
		cfg.ContextWindow = 128000 // Default context window
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		provider:      cfg.Provider,
		tools:         make(map[string]ToolHandler),
		maxTurns:      cfg.MaxTurns,
		broker:        broker,
		largeModel:    cfg.LargeModel,
		smallModel:    cfg.SmallModel,
		contextWindow: cfg.ContextWindow,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// RegisterTool registers a tool handler
func (c *Coordinator) RegisterTool(name string, handler ToolHandler) {
	c.tools[name] = handler
}

// QueuePrompt adds a prompt to the queue
func (c *Coordinator) QueuePrompt(content string) string {
	c.queueMu.Lock()
	defer c.queueMu.Unlock()

	id := fmt.Sprintf("prompt-%d", time.Now().UnixNano())
	c.queue = append(c.queue, QueuedPrompt{
		ID:        id,
		Content:   content,
		Timestamp: time.Now(),
	})

	if c.broker != nil {
		c.broker.Publish("prompt.queued", map[string]interface{}{
			"id":      id,
			"content": content,
		})
	}

	return id
}

// GetQueuedPrompts returns all queued prompts
func (c *Coordinator) GetQueuedPrompts() []QueuedPrompt {
	c.queueMu.Lock()
	defer c.queueMu.Unlock()
	return c.queue
}

// ClearQueue clears the prompt queue
func (c *Coordinator) ClearQueue() {
	c.queueMu.Lock()
	defer c.queueMu.Unlock()
	c.queue = nil
}

// Run executes the conversation loop
func (c *Coordinator) Run(ctx context.Context, systemPrompt string, messages []Message) ([]Message, error) {
	c.isBusy = true
	defer func() { c.isBusy = false }()

	allMessages := append([]Message{{
		Role:    "system",
		Content: systemPrompt,
	}}, messages...)

	for turn := 0; turn < c.maxTurns; turn++ {
		// Check context
		select {
		case <-ctx.Done():
			return allMessages, ctx.Err()
		case <-c.ctx.Done():
			return allMessages, fmt.Errorf("coordinator cancelled")
		default:
		}

		// Check context window and summarize if needed
		if c.needsSummarization(allMessages) {
			summary, err := c.summarize(ctx, allMessages)
			if err == nil {
				c.summary = summary
				allMessages = c.trimMessages(allMessages, summary)
			}
		}

		// Select model based on task complexity
		model := c.selectModel(allMessages)

		// Call provider
		req := provider.ChatRequest{
			Model:    model,
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
	if c.cancel != nil {
		c.cancel()
	}
	c.isBusy = false
}

// IsBusy returns true if coordinator is processing
func (c *Coordinator) IsBusy() bool {
	return c.isBusy
}

// GetSummary returns the current conversation summary
func (c *Coordinator) GetSummary() string {
	return c.summary
}

// needsSummarization checks if messages need summarization
func (c *Coordinator) needsSummarization(messages []Message) bool {
	totalTokens := 0
	for _, m := range messages {
		// Rough estimate: 1 token per 4 characters
		totalTokens += len(m.Content) / 4
	}
	return totalTokens > c.contextWindow/2
}

// summarize generates a summary of the conversation
func (c *Coordinator) summarize(ctx context.Context, messages []Message) (string, error) {
	// Use small model for summarization
	if c.smallModel != nil {
		summaryReq := provider.ChatRequest{
			Model: "default",
			Messages: []provider.Message{
				{Role: "system", Content: "Summarize this conversation concisely."},
				{Role: "user", Content: formatMessagesForSummary(messages)},
			},
		}
		resp, err := c.smallModel.Chat(ctx, summaryReq)
		if err == nil {
			return resp.Content, nil
		}
	}

	// Fallback: simple truncation
	return "Conversation summarized due to length", nil
}

// trimMessages trims messages keeping only recent ones and summary
func (c *Coordinator) trimMessages(messages []Message, summary string) []Message {
	// Keep system message, summary, and last few messages
	var trimmed []Message
	for _, m := range messages {
		if m.Role == "system" {
			trimmed = append(trimmed, m)
		}
	}

	// Add summary
	trimmed = append(trimmed, Message{
		Role:    "assistant",
		Content: fmt.Sprintf("[Summary of previous conversation: %s]", summary),
	})

	// Keep last 10 messages
	if len(messages) > 10 {
		trimmed = append(trimmed, messages[len(messages)-10:]...)
	} else {
		trimmed = append(trimmed, messages[1:]...)
	}

	return trimmed
}

// selectModel selects the appropriate model based on task complexity
func (c *Coordinator) selectModel(messages []Message) string {
	// Simple heuristic: use large model for complex tasks
	totalContent := 0
	for _, m := range messages {
		totalContent += len(m.Content)
	}

	if totalContent > 5000 && c.largeModel != nil {
		return "large"
	}
	return "default"
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

func formatMessagesForSummary(messages []Message) string {
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}
	return sb.String()
}
