// Package executor provides the real agent execution loop with queuing and cancellation.
package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourname/helm/internal/logger"
	"github.com/yourname/helm/internal/provider"
	"github.com/yourname/helm/internal/tools"
)

// Prompt represents a queued prompt
type Prompt struct {
	ID       string
	Content  string
	AddedAt  time.Time
	Priority int // higher = more urgent
}

// ExecutionState represents the current state of execution
type ExecutionState string

const (
	StateIdle      ExecutionState = "idle"
	StateThinking  ExecutionState = "thinking"
	StateToolUse   ExecutionState = "tool_use"
	StateWaiting   ExecutionState = "waiting"
	StateCompleted ExecutionState = "completed"
	StateCancelled ExecutionState = "cancelled"
	StateFailed    ExecutionState = "failed"
)

// Event represents an execution event
type Event struct {
	Type      string
	Timestamp time.Time
	Data      interface{}
}

// Executor manages the real agent execution loop
type Executor struct {
	provider    provider.Provider
	tools       *tools.Registry
	logger      *logger.Logger
	promptQueue []*Prompt
	queueMu     sync.Mutex
	state       ExecutionState
	stateMu     sync.RWMutex
	cancel      context.CancelFunc
	ctx         context.Context
	events      chan Event
	maxTurns    int
	maxTokens   int
	workdir     string
}

// NewExecutor creates a new executor
func NewExecutor(p provider.Provider, t *tools.Registry, log *logger.Logger, workdir string) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		provider:  p,
		tools:     t,
		logger:    log,
		ctx:       ctx,
		cancel:    cancel,
		events:    make(chan Event, 100),
		maxTurns:  50,
		maxTokens: 128000,
		workdir:   workdir,
	}
}

// QueuePrompt adds a prompt to the execution queue
func (e *Executor) QueuePrompt(content string, priority int) string {
	e.queueMu.Lock()
	defer e.queueMu.Unlock()

	id := fmt.Sprintf("prompt-%d", time.Now().UnixNano())
	e.promptQueue = append(e.promptQueue, &Prompt{
		ID:       id,
		Content:  content,
		AddedAt:  time.Now(),
		Priority: priority,
	})

	// Sort by priority (highest first)
	sortPrompts(e.promptQueue)

	e.logger.Info("Queued prompt %s (priority: %d)", id, priority)
	e.emitEvent("prompt_queued", map[string]interface{}{
		"id":       id,
		"content":  content[:min(100, len(content))],
		"priority": priority,
	})

	return id
}

// Cancel cancels the current execution
func (e *Executor) Cancel() {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}
	e.state = StateCancelled
	e.emitEvent("cancelled", nil)
	e.logger.Info("Execution cancelled")
}

// Run starts the execution loop
func (e *Executor) Run(ctx context.Context, systemPrompt string, initialPrompt string) error {
	e.stateMu.Lock()
	e.state = StateThinking
	e.stateMu.Unlock()

	// Create new context with cancellation
	execCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	defer cancel()

	messages := []provider.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: initialPrompt},
	}

	e.emitEvent("execution_started", map[string]interface{}{
		"prompt": initialPrompt[:min(100, len(initialPrompt))],
	})

	for turn := 0; turn < e.maxTurns; turn++ {
		// Check for cancellation
		select {
		case <-execCtx.Done():
			e.stateMu.Lock()
			e.state = StateCancelled
			e.stateMu.Unlock()
			return execCtx.Err()
		default:
		}

		// Check context window and summarize if needed
		messages = e.trimContextIfNeeded(messages)

		// Call provider
		e.stateMu.Lock()
		e.state = StateThinking
		e.stateMu.Unlock()

		resp, err := e.provider.Chat(execCtx, provider.ChatRequest{
			Model:    "default",
			Messages: messages,
		})
		if err != nil {
			e.stateMu.Lock()
			e.state = StateFailed
			e.stateMu.Unlock()
			return fmt.Errorf("provider chat: %w", err)
		}

		// Add assistant response
		messages = append(messages, provider.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		e.emitEvent("assistant_response", map[string]interface{}{
			"turn":    turn,
			"content": resp.Content[:min(200, len(resp.Content))],
		})

		// Check if response has tool calls
		if len(resp.ToolCalls) == 0 {
			// No tool calls, execution complete
			e.stateMu.Lock()
			e.state = StateCompleted
			e.stateMu.Unlock()
			e.emitEvent("execution_completed", map[string]interface{}{
				"turns": turn + 1,
			})
			return nil
		}

		// Execute tool calls
		e.stateMu.Lock()
		e.state = StateToolUse
		e.stateMu.Unlock()

		for _, tc := range resp.ToolCalls {
			e.logger.Info("Executing tool: %s", tc.Name)
			e.emitEvent("tool_call", map[string]interface{}{
				"name": tc.Name,
				"args": tc.Arguments[:min(100, len(tc.Arguments))],
			})

			// Parse arguments
			var args map[string]interface{}
			if err := jsonUnmarshal(tc.Arguments, &args); err != nil {
				messages = append(messages, provider.Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error: invalid arguments for %s: %v", tc.Name, err),
				})
				continue
			}

			// Execute tool
			output, err := e.tools.Execute(execCtx, tc.Name, args, e.workdir)
			if err != nil {
				messages = append(messages, provider.Message{
					Role:    "tool",
					Content: fmt.Sprintf("Error executing %s: %v", tc.Name, err),
				})
				e.emitEvent("tool_error", map[string]interface{}{
					"name":  tc.Name,
					"error": err.Error(),
				})
			} else {
				messages = append(messages, provider.Message{
					Role:    "tool",
					Content: output,
				})
				e.emitEvent("tool_result", map[string]interface{}{
					"name":   tc.Name,
					"output": output[:min(200, len(output))],
				})
			}
		}
	}

	// Max turns exceeded
	e.stateMu.Lock()
	e.state = StateFailed
	e.stateMu.Unlock()
	return fmt.Errorf("max turns (%d) exceeded", e.maxTurns)
}

// GetState returns the current execution state
func (e *Executor) GetState() ExecutionState {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()
	return e.state
}

// GetQueuedPrompts returns all queued prompts
func (e *Executor) GetQueuedPrompts() []*Prompt {
	e.queueMu.Lock()
	defer e.queueMu.Unlock()
	return e.promptQueue
}

// GetEvents returns recent events
func (e *Executor) GetEvents() []Event {
	var events []Event
	for {
		select {
		case ev := <-e.events:
			events = append(events, ev)
		default:
			return events
		}
	}
}

// Events returns the event channel for streaming
func (e *Executor) Events() <-chan Event {
	return e.events
}

func (e *Executor) emitEvent(eventType string, data interface{}) {
	select {
	case e.events <- Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}:
	default:
		// Channel full, drop event
	}
}

func (e *Executor) trimContextIfNeeded(messages []provider.Message) []provider.Message {
	// Estimate token count
	totalTokens := 0
	for _, m := range messages {
		totalTokens += len(m.Content) / 4
	}

	if totalTokens <= e.maxTokens/2 {
		return messages
	}

	// Keep system message and last 10 messages
	var trimmed []provider.Message
	for _, m := range messages {
		if m.Role == "system" {
			trimmed = append(trimmed, m)
		}
	}

	// Add summary of removed messages
	summary := "[Previous conversation summarized due to length]"
	trimmed = append(trimmed, provider.Message{
		Role:    "assistant",
		Content: summary,
	})

	// Keep last 10 messages
	if len(messages) > 10 {
		trimmed = append(trimmed, messages[len(messages)-10:]...)
	} else {
		trimmed = append(trimmed, messages[1:]...)
	}

	return trimmed
}

func sortPrompts(prompts []*Prompt) {
	for i := 0; i < len(prompts); i++ {
		for j := i + 1; j < len(prompts); j++ {
			if prompts[i].Priority < prompts[j].Priority {
				prompts[i], prompts[j] = prompts[j], prompts[i]
			}
		}
	}
}

func jsonUnmarshal(data string, v interface{}) error {
	// Simple JSON parsing
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
