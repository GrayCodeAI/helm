// Package retry provides auto-retry with learning capabilities
// Package retry provides auto-retry with learning capabilities and exponential backoff.
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/yourname/helm/internal/errors"
	"github.com/yourname/helm/internal/logger"
	"github.com/yourname/helm/internal/mistake"
)

// Strategy defines the retry approach
type Strategy int

const (
	StrategySameModel Strategy = iota
	StrategyFallbackModel
	StrategyAdjustedPrompt
	StrategyDifferentProvider
)

func (s Strategy) String() string {
	switch s {
	case StrategySameModel:
		return "same_model"
	case StrategyFallbackModel:
		return "fallback_model"
	case StrategyAdjustedPrompt:
		return "adjusted_prompt"
	case StrategyDifferentProvider:
		return "different_provider"
	default:
		return "unknown"
	}
}

// Config configures retry behavior
type Config struct {
	MaxRetries             int
	BaseDelay              time.Duration
	MaxDelay               time.Duration
	EnablePromptAdjustment bool
	EnableModelFallback    bool
}

// DefaultConfig returns default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:             3,
		BaseDelay:              5 * time.Second,
		MaxDelay:               5 * time.Minute,
		EnablePromptAdjustment: true,
		EnableModelFallback:    true,
	}
}

// Engine decides when and how to retry
type Engine struct {
	config   Config
	journal  *mistake.Journal
	attempts map[string]*Attempt // sessionID -> attempt
}

// Attempt tracks retry state for a session
type Attempt struct {
	SessionID       string
	CurrentAttempt  int
	LastError       error
	StrategiesTried []Strategy
	StartTime       time.Time
}

// Result represents the outcome of a retry decision
type Result struct {
	ShouldRetry    bool
	Strategy       Strategy
	Delay          time.Duration
	AdjustedPrompt string
	NewModel       string
	Message        string
}

// NewEngine creates a new retry engine
func NewEngine(config Config, journal *mistake.Journal) *Engine {
	return &Engine{
		config:   config,
		journal:  journal,
		attempts: make(map[string]*Attempt),
	}
}

// ShouldRetry determines if a session should be retried
func (e *Engine) ShouldRetry(ctx context.Context, sessionID string, err error) *Result {
	attempt := e.getAttempt(sessionID)
	attempt.LastError = err
	attempt.CurrentAttempt++

	// Check max retries
	if attempt.CurrentAttempt > e.config.MaxRetries {
		return &Result{
			ShouldRetry: false,
			Message:     fmt.Sprintf("Max retries (%d) exceeded", e.config.MaxRetries),
		}
	}

	// Check if error is retryable
	if !isRetryableError(err) {
		return &Result{
			ShouldRetry: false,
			Message:     "Error is not retryable",
		}
	}

	// Calculate delay with exponential backoff
	delay := e.calculateDelay(attempt.CurrentAttempt)

	// Select strategy
	strategy := e.selectStrategy(ctx, attempt)
	attempt.StrategiesTried = append(attempt.StrategiesTried, strategy)

	result := &Result{
		ShouldRetry: true,
		Strategy:    strategy,
		Delay:       delay,
	}

	// Build adjusted context if needed
	switch strategy {
	case StrategyAdjustedPrompt:
		result.AdjustedPrompt = e.buildCorrectedPrompt(ctx, sessionID, err)
		result.Message = "Retrying with adjusted prompt based on mistake history"
	case StrategyFallbackModel:
		result.NewModel = e.selectFallbackModel(attempt)
		result.Message = fmt.Sprintf("Retrying with fallback model: %s", result.NewModel)
	case StrategyDifferentProvider:
		result.Message = "Retrying with different provider"
	default:
		result.Message = fmt.Sprintf("Retrying (attempt %d/%d)", attempt.CurrentAttempt, e.config.MaxRetries)
	}

	return result
}

// RecordSuccess marks an attempt as successful
func (e *Engine) RecordSuccess(sessionID string) {
	delete(e.attempts, sessionID)
}

// RecordFailure records a failed attempt
func (e *Engine) RecordFailure(ctx context.Context, sessionID string, err error, filePath string) {
	// Record in mistake journal
	if e.journal != nil {
		var mistakeType mistake.Type
		switch {
		case isTimeoutError(err):
			mistakeType = mistake.TypeTimeout
		case isRateLimitError(err):
			mistakeType = mistake.TypeCompileError
		default:
			mistakeType = mistake.TypeRuntimeError
		}

		e.journal.Record(ctx, sessionID, mistakeType, err.Error(), "", "", filePath)
	}
}

// getAttempt gets or creates an attempt tracker
func (e *Engine) getAttempt(sessionID string) *Attempt {
	if attempt, ok := e.attempts[sessionID]; ok {
		return attempt
	}

	attempt := &Attempt{
		SessionID:       sessionID,
		CurrentAttempt:  0,
		StartTime:       time.Now(),
		StrategiesTried: []Strategy{},
	}
	e.attempts[sessionID] = attempt
	return attempt
}

// calculateDelay calculates backoff delay
func (e *Engine) calculateDelay(attempt int) time.Duration {
	delay := e.config.BaseDelay * (1 << (attempt - 1))
	if delay > e.config.MaxDelay {
		delay = e.config.MaxDelay
	}
	return delay
}

// selectStrategy chooses the retry strategy
func (e *Engine) selectStrategy(ctx context.Context, attempt *Attempt) Strategy {
	// First retry: same model
	if attempt.CurrentAttempt == 1 {
		return StrategySameModel
	}

	// Second retry: adjusted prompt if enabled
	if attempt.CurrentAttempt == 2 && e.config.EnablePromptAdjustment {
		if e.hasMistakeHistory(ctx, attempt.SessionID) {
			return StrategyAdjustedPrompt
		}
	}

	// Third retry: fallback model if enabled
	if e.config.EnableModelFallback {
		return StrategyFallbackModel
	}

	return StrategySameModel
}

// hasMistakeHistory checks if there are past mistakes for this session
func (e *Engine) hasMistakeHistory(ctx context.Context, sessionID string) bool {
	if e.journal == nil {
		return false
	}

	mistakes, err := e.journal.List(ctx, sessionID)
	return err == nil && len(mistakes) > 0
}

// buildCorrectedPrompt creates an adjusted prompt based on mistake history
func (e *Engine) buildCorrectedPrompt(ctx context.Context, sessionID string, err error) string {
	if e.journal == nil {
		return ""
	}

	// Get similar past mistakes
	similar, err := e.journal.FindSimilar(ctx, "", err.Error(), 3)
	if err != nil || len(similar) == 0 {
		return ""
	}

	// Build correction context
	var corrections []string
	for _, m := range similar {
		if m.Correction != "" {
			corrections = append(corrections, fmt.Sprintf("- %s: %s", m.Type, m.Correction))
		}
	}

	if len(corrections) == 0 {
		return ""
	}

	return fmt.Sprintf("\n\nNote: Previous similar attempts failed. Avoid these issues:\n%s",
		joinStrings(corrections, "\n"))
}

// selectFallbackModel chooses a fallback model
func (e *Engine) selectFallbackModel(attempt *Attempt) string {
	// Simple fallback selection
	// In production, this would query the model catalog
	models := []string{
		"claude-opus-4-20250514",
		"gpt-4o",
		"gemini-2.5-pro",
	}

	// Convert tried strategies to strings for comparison
	triedModels := make([]string, 0)
	for _, s := range attempt.StrategiesTried {
		if s == StrategyDifferentProvider {
			// This would check for provider switches
			continue
		}
		// For fallback model strategy, we've tried the fallback
		// We'll track actual model names separately in production
		_ = s
	}

	// Return a model not yet tried
	for _, model := range models {
		if !containsString(triedModels, model) {
			return model
		}
	}

	return models[0]
}

// isRetryableError checks if an error should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors
	if containsString([]string{errStr}, "timeout") ||
		containsString([]string{errStr}, "connection refused") ||
		containsString([]string{errStr}, "temporary") {
		return true
	}

	// Rate limiting
	if isRateLimitError(err) {
		return true
	}

	// Server errors
	if containsString([]string{errStr}, "500") ||
		containsString([]string{errStr}, "503") ||
		containsString([]string{errStr}, "overloaded") {
		return true
	}

	return false
}

// isTimeoutError checks if error is a timeout
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return containsString([]string{err.Error()}, "timeout")
}

// isRateLimitError checks if error is rate limiting
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsString([]string{errStr}, "rate limit") ||
		containsString([]string{errStr}, "429") ||
		containsString([]string{errStr}, "too many requests")
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// Func is a function that can be retried
type Func func(ctx context.Context) error

// Do executes a function with retry logic
func Do(ctx context.Context, fn Func, opts ...RetryOption) error {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	var lastErr error

	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt == config.MaxRetries {
			break
		}

		if !isRetryableError(err) {
			return err
		}

		delay := calculateExponentialDelay(attempt, config.BaseDelay, config.MaxDelay)

		logger.GetDefault().Warn("Attempt %d/%d failed, retrying in %s: %v",
			attempt, config.MaxRetries, delay, err)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}
	}

	return fmt.Errorf("max attempts (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// DoWithResult executes a function that returns a result with retry logic
func DoWithResult[T any](ctx context.Context, fn func(ctx context.Context) (T, error), opts ...RetryOption) (T, error) {
	var result T

	err := Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}, opts...)

	return result, err
}

// RetryOption configures retry behavior
type RetryOption func(*Config)

// WithMaxRetries sets max retries
func WithMaxRetries(n int) RetryOption {
	return func(c *Config) {
		c.MaxRetries = n
	}
}

// WithBaseDelay sets base delay
func WithBaseDelay(d time.Duration) RetryOption {
	return func(c *Config) {
		c.BaseDelay = d
	}
}

// WithMaxDelay sets max delay
func WithMaxDelay(d time.Duration) RetryOption {
	return func(c *Config) {
		c.MaxDelay = d
	}
}

// calculateExponentialDelay calculates delay with exponential backoff and jitter
func calculateExponentialDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: base * 2^(attempt-1)
	delay := float64(baseDelay) * math.Pow(2, float64(attempt-1))

	// Cap at max delay
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	// Add jitter (10%)
	jitter := delay * 0.1 * (rand.Float64()*2 - 1)
	delay += jitter

	return time.Duration(delay)
}

// IsRetryable checks if an error is retryable using error codes
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	code := errors.Code(err)
	switch code {
	case errors.CodeTimeout,
		errors.CodeUnavailable,
		errors.CodeRateLimited,
		errors.CodeNetwork:
		return true
	}

	return isRetryableError(err)
}

// Policy defines a retry policy
type Policy struct {
	Config Config
}

// NewPolicy creates a new retry policy
func NewPolicy(opts ...RetryOption) *Policy {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return &Policy{Config: config}
}

// Execute executes a function with the policy
func (p *Policy) Execute(ctx context.Context, fn Func) error {
	return Do(ctx, fn, func(c *Config) {
		*c = p.Config
	})
}

// Common policies

// FastRetry returns a policy for fast retries
func FastRetry() *Policy {
	return NewPolicy(WithMaxRetries(3), WithBaseDelay(100*time.Millisecond))
}

// SlowRetry returns a policy for slow retries
func SlowRetry() *Policy {
	return NewPolicy(WithMaxRetries(5), WithBaseDelay(1*time.Second))
}

// DatabaseRetry returns a policy optimized for database operations
func DatabaseRetry() *Policy {
	return NewPolicy(
		WithMaxRetries(5),
		WithBaseDelay(100*time.Millisecond),
		WithMaxDelay(5*time.Second),
	)
}

// HTTPRetry returns a policy optimized for HTTP requests
func HTTPRetry() *Policy {
	return NewPolicy(
		WithMaxRetries(3),
		WithBaseDelay(1*time.Second),
		WithMaxDelay(10*time.Second),
	)
}

// NoRetry returns a policy that doesn't retry
func NoRetry() *Policy {
	return NewPolicy(WithMaxRetries(1))
}

// LinearBackoff returns linear backoff function
func LinearBackoff(base time.Duration) func(int, Config) time.Duration {
	return func(attempt int, config Config) time.Duration {
		return base * time.Duration(attempt)
	}
}

// FixedBackoff returns fixed backoff function
func FixedBackoff(delay time.Duration) func(int, Config) time.Duration {
	return func(attempt int, config Config) time.Duration {
		return delay
	}
}
