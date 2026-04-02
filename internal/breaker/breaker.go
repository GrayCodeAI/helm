// Package breaker implements circuit breaker pattern for external APIs.
package breaker

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config configures circuit breaker
type Config struct {
	Name             string
	MaxFailures      int
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int
	Timeout          time.Duration
}

// DefaultConfig returns default config
func DefaultConfig(name string) Config {
	return Config{
		Name:             name,
		MaxFailures:      5,
		ResetTimeout:     60 * time.Second,
		HalfOpenMaxCalls: 3,
		Timeout:          30 * time.Second,
	}
}

// Breaker implements circuit breaker pattern
type Breaker struct {
	config        Config
	state         State
	failures      int
	successes     int
	halfOpenCalls int
	lastFailure   time.Time
	mu            sync.RWMutex
	onStateChange func(State)
}

// New creates a new circuit breaker
func New(config Config) *Breaker {
	b := &Breaker{
		config: config,
		state:  StateClosed,
	}
	return b
}

// Execute executes a function with circuit breaker protection
func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()

	// Check if circuit is open
	if b.state == StateOpen {
		if time.Since(b.lastFailure) > b.config.ResetTimeout {
			b.state = StateHalfOpen
			b.halfOpenCalls = 0
			b.mu.Unlock()
			if b.onStateChange != nil {
				b.onStateChange(StateHalfOpen)
			}
		} else {
			b.mu.Unlock()
			return fmt.Errorf("circuit breaker %s is open", b.config.Name)
		}
	} else {
		b.mu.Unlock()
	}

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		b.recordResult(err == nil)
		return err
	case <-time.After(b.config.Timeout):
		b.recordResult(false)
		return fmt.Errorf("timeout: %s", b.config.Name)
	}
}

// recordResult records success or failure
func (b *Breaker) recordResult(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if success {
		if b.state == StateHalfOpen {
			b.halfOpenCalls++
			if b.halfOpenCalls >= b.config.HalfOpenMaxCalls {
				b.state = StateClosed
				b.failures = 0
				b.successes = 0
				if b.onStateChange != nil {
					b.onStateChange(StateClosed)
				}
			}
		} else {
			b.successes++
			b.failures = 0
		}
	} else {
		b.failures++
		b.lastFailure = time.Now()

		if b.state == StateHalfOpen {
			b.state = StateOpen
			if b.onStateChange != nil {
				b.onStateChange(StateOpen)
			}
		} else if b.failures >= b.config.MaxFailures {
			b.state = StateOpen
			if b.onStateChange != nil {
				b.onStateChange(StateOpen)
			}
		}
	}
}

// State returns current state
func (b *Breaker) State() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Failures returns failure count
func (b *Breaker) Failures() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.failures
}

// Reset resets the breaker
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateClosed
	b.failures = 0
	b.successes = 0
	b.halfOpenCalls = 0
}

// OnStateChange sets callback for state changes
func (b *Breaker) OnStateChange(fn func(State)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onStateChange = fn
}

// Stats returns breaker statistics
func (b *Breaker) Stats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return map[string]interface{}{
		"name":         b.config.Name,
		"state":        b.state.String(),
		"failures":     b.failures,
		"successes":    b.successes,
		"max_failures": b.config.MaxFailures,
	}
}

// ErrCircuitOpen is returned when circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// IsCircuitOpen checks if error is circuit open
func IsCircuitOpen(err error) bool {
	return err != nil && (err.Error() == ErrCircuitOpen.Error() ||
		(err.Error() != "" && err.Error()[:len("circuit breaker")] == "circuit breaker"))
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*Breaker
	mu       sync.RWMutex
}

// NewManager creates a new breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*Breaker),
	}
}

// GetOrCreate gets or creates a breaker
func (m *Manager) GetOrCreate(name string, config Config) *Breaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.breakers[name]; ok {
		return b
	}

	b := New(config)
	m.breakers[name] = b
	return b
}

// Get gets a breaker by name
func (m *Manager) Get(name string) (*Breaker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.breakers[name]
	return b, ok
}

// List lists all breakers
func (m *Manager) List() []*Breaker {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breakers := make([]*Breaker, 0, len(m.breakers))
	for _, b := range m.breakers {
		breakers = append(breakers, b)
	}
	return breakers
}

// Stats returns stats for all breakers
func (m *Manager) Stats() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for name, b := range m.breakers {
		stats[name] = b.Stats()
	}
	return stats
}
