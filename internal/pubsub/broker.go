// Package pubsub provides a simple publish/subscribe event system.
package pubsub

import (
	"sync"
)

// Event type constants
const (
	EventSessionCreated  = "session.created"
	EventSessionUpdated  = "session.updated"
	EventSessionDone     = "session.done"
	EventSessionFailed   = "session.failed"
	EventCostRecorded    = "cost.recorded"
	EventMemoryAdded     = "memory.added"
	EventMemoryUpdated   = "memory.updated"
	EventBudgetExceeded  = "budget.exceeded"
	EventMistakeCaptured = "mistake.captured"
	EventPromptAdded     = "prompt.added"
)

// Handler is a function that handles an event
type Handler func(eventType string, payload interface{})

// Broker manages event publishing and subscribing
type Broker struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewBroker creates a new event broker
func NewBroker() *Broker {
	return &Broker{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for an event type
func (b *Broker) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// SubscribeAll registers a handler for all event types
func (b *Broker) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers["*"] = append(b.handlers["*"], handler)
}

// Publish sends an event to all registered handlers
func (b *Broker) Publish(eventType string, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Call wildcard handlers
	for _, h := range b.handlers["*"] {
		go h(eventType, payload)
	}

	// Call specific handlers
	for _, h := range b.handlers[eventType] {
		go h(eventType, payload)
	}
}

// Unsubscribe removes a handler (not goroutine-safe with Publish)
func (b *Broker) Unsubscribe(eventType string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.handlers, eventType)
}

// HandlerCount returns the number of registered handlers
func (b *Broker) HandlerCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	count := 0
	for _, handlers := range b.handlers {
		count += len(handlers)
	}
	return count
}
