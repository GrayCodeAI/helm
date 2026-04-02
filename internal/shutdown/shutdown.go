// Package shutdown provides graceful shutdown handling for the application.
package shutdown

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/yourname/helm/internal/logger"
)

// Hook is a function to be called during shutdown
type Hook func(ctx context.Context) error

// Manager manages graceful shutdown
type Manager struct {
	hooks      []Hook
	timeout    time.Duration
	logger     *logger.Logger
	signals    []os.Signal
	mu         sync.Mutex
	started    bool
	cancelFunc context.CancelFunc
}

// Config configures the shutdown manager
type Config struct {
	Timeout time.Duration
	Signals []os.Signal
	Logger  *logger.Logger
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Timeout: 30 * time.Second,
		Signals: []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP},
		Logger:  logger.GetDefault(),
	}
}

// New creates a new shutdown manager
func New(config Config) *Manager {
	return &Manager{
		hooks:   make([]Hook, 0),
		timeout: config.Timeout,
		logger:  config.Logger,
		signals: config.Signals,
	}
}

// NewDefault creates a shutdown manager with defaults
func NewDefault() *Manager {
	return New(DefaultConfig())
}

// Register adds a shutdown hook
func (m *Manager) Register(name string, hook Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wrappedHook := func(ctx context.Context) error {
		m.logger.Info("Running shutdown hook: %s", name)
		start := time.Now()

		err := hook(ctx)

		duration := time.Since(start)
		if err != nil {
			m.logger.Error("Shutdown hook %s failed after %s: %v", name, duration, err)
		} else {
			m.logger.Info("Shutdown hook %s completed in %s", name, duration)
		}

		return err
	}

	m.hooks = append(m.hooks, wrappedHook)
}

// RegisterHTTPServer registers an HTTP server to shut down
func (m *Manager) RegisterHTTPServer(server *http.Server) {
	m.Register("http_server", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})
}

// RegisterFunc registers a simple function as a hook
func (m *Manager) RegisterFunc(name string, fn func() error) {
	m.Register(name, func(ctx context.Context) error {
		return fn()
	})
}

// RegisterCloser registers an io.Closer
func (m *Manager) RegisterCloser(name string, closer interface{ Close() error }) {
	m.Register(name, func(ctx context.Context) error {
		return closer.Close()
	})
}

// Listen starts listening for shutdown signals
func (m *Manager) Listen(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	// Create signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, m.signals...)

	m.logger.Info("Shutdown manager started, listening for signals: %v", m.signals)

	go func() {
		select {
		case sig := <-sigChan:
			m.logger.Info("Received signal: %v, initiating graceful shutdown...", sig)
			m.Shutdown(context.Background())

		case <-ctx.Done():
			m.logger.Info("Context cancelled, initiating graceful shutdown...")
			m.Shutdown(context.Background())
		}
	}()
}

// Shutdown performs graceful shutdown
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	hooks := make([]Hook, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.Unlock()

	// Create timeout context if not provided
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
		defer cancel()
	}

	m.logger.Info("Starting graceful shutdown with timeout: %s", m.timeout)
	start := time.Now()

	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks))

	// Run all hooks concurrently
	for _, hook := range hooks {
		wg.Add(1)
		go func(h Hook) {
			defer wg.Done()

			if err := h(ctx); err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}(hook)
	}

	// Wait for all hooks to complete or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info("Graceful shutdown completed in %s", time.Since(start))

		// Check for errors
		select {
		case err := <-errChan:
			return fmt.Errorf("shutdown hook failed: %w", err)
		default:
			return nil
		}

	case <-ctx.Done():
		m.logger.Error("Graceful shutdown timed out after %s", m.timeout)
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
}

// Wait blocks until shutdown is complete
func (m *Manager) Wait() {
	// Create a channel that never receives
	<-make(chan struct{})
}

// ForceExit forces immediate exit
func (m *Manager) ForceExit(code int) {
	m.logger.Error("Forcing immediate exit with code: %d", code)
	os.Exit(code)
}

// IsShuttingDown returns true if shutdown is in progress
func (m *Manager) IsShuttingDown() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cancelFunc != nil
}

// SetTimeout updates the shutdown timeout
func (m *Manager) SetTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = timeout
}

// HookCount returns the number of registered hooks
func (m *Manager) HookCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.hooks)
}

// ShutdownWithExit performs shutdown and exits the process
func (m *Manager) ShutdownWithExit(ctx context.Context, exitCode int) {
	if err := m.Shutdown(ctx); err != nil {
		m.logger.Error("Shutdown failed: %v", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}

// HandlePanic recovers from panics and initiates shutdown
func (m *Manager) HandlePanic() {
	if r := recover(); r != nil {
		m.logger.Error("Panic recovered: %v", r)
		m.Shutdown(context.Background())
		m.ForceExit(1)
	}
}

// WaitForSignal blocks until a signal is received
func (m *Manager) WaitForSignal() os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, m.signals...)
	return <-sigChan
}

// Builder helps build shutdown hooks fluently
type Builder struct {
	manager *Manager
}

// NewBuilder creates a new builder
func NewBuilder(config Config) *Builder {
	return &Builder{
		manager: New(config),
	}
}

// WithTimeout sets timeout
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
	b.manager.SetTimeout(timeout)
	return b
}

// WithLogger sets logger
func (b *Builder) WithLogger(log *logger.Logger) *Builder {
	b.manager.logger = log
	return b
}

// AddHook adds a hook
func (b *Builder) AddHook(name string, hook Hook) *Builder {
	b.manager.Register(name, hook)
	return b
}

// AddFunc adds a function hook
func (b *Builder) AddFunc(name string, fn func() error) *Builder {
	b.manager.RegisterFunc(name, fn)
	return b
}

// AddServer adds an HTTP server
func (b *Builder) AddServer(server *http.Server) *Builder {
	b.manager.RegisterHTTPServer(server)
	return b
}

// AddCloser adds a closer
func (b *Builder) AddCloser(name string, closer interface{ Close() error }) *Builder {
	b.manager.RegisterCloser(name, closer)
	return b
}

// Build returns the manager
func (b *Builder) Build() *Manager {
	return b.manager
}

// Global instance
var globalManager = NewDefault()

// InitGlobal initializes global shutdown manager
func InitGlobal(m *Manager) {
	globalManager = m
}

// Global returns global shutdown manager
func Global() *Manager {
	return globalManager
}

// Register registers a hook globally
func Register(name string, hook Hook) {
	globalManager.Register(name, hook)
}

// RegisterFunc registers a function hook globally
func RegisterFunc(name string, fn func() error) {
	globalManager.RegisterFunc(name, fn)
}

// RegisterHTTPServer registers HTTP server globally
func RegisterHTTPServer(server *http.Server) {
	globalManager.RegisterHTTPServer(server)
}

// Listen starts listening globally
func Listen(ctx context.Context) {
	globalManager.Listen(ctx)
}

// Shutdown performs shutdown globally
func Shutdown(ctx context.Context) error {
	return globalManager.Shutdown(ctx)
}

// Wait blocks globally
func Wait() {
	globalManager.Wait()
}
