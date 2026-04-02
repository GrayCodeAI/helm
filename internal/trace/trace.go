// Package trace provides distributed tracing capabilities.
package trace

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourname/helm/internal/logger"
)

// Config configures tracing
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	SampleRate     float64 // 0-1
}

// DefaultConfig returns default config
func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		ServiceName:    "helm",
		ServiceVersion: "0.1.0",
		Environment:    "development",
		SampleRate:     1.0,
	}
}

// Tracer provides tracing functionality
type Tracer struct {
	config   Config
	logger   *logger.Logger
	exporter Exporter
	mu       sync.RWMutex
	spans    []*Span
}

// Exporter exports trace data
type Exporter interface {
	Export(span *Span) error
}

// LoggerExporter exports to logger
type LoggerExporter struct {
	logger *logger.Logger
}

// Export exports span to logger
func (e *LoggerExporter) Export(span *Span) error {
	e.logger.Info("[TRACE] %s %s %s %v %v",
		span.TraceID, span.ID, span.Name, span.Duration, span.Tags)
	return nil
}

// Span represents a trace span
type Span struct {
	ID        string
	TraceID   string
	ParentID  string
	Name      string
	StartTime time.Time
	EndTime   *time.Time
	Duration  time.Duration
	Tags      map[string]string
	Events    []Event
	Error     error
}

// Event represents a span event
type Event struct {
	Name      string
	Timestamp time.Time
	Tags      map[string]string
}

// New creates a new tracer
func New(config Config, log *logger.Logger) *Tracer {
	if log == nil {
		log = logger.GetDefault()
	}

	return &Tracer{
		config:   config,
		logger:   log,
		exporter: &LoggerExporter{logger: log},
		spans:    make([]*Span, 0),
	}
}

// Start starts a new span
func (t *Tracer) Start(ctx context.Context, name string) (context.Context, *Span) {
	if !t.config.Enabled {
		return ctx, nil
	}

	// Sample
	if t.config.SampleRate < 1.0 {
		// Simple sampling - in production use better algorithm
	}

	span := &Span{
		ID:        generateID(),
		TraceID:   getOrCreateTraceID(ctx),
		Name:      name,
		StartTime: time.Now(),
		Tags:      make(map[string]string),
		Events:    make([]Event, 0),
	}

	// Check for parent span
	if parent := SpanFromContext(ctx); parent != nil {
		span.ParentID = parent.ID
		span.TraceID = parent.TraceID
	}

	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()

	return ContextWithSpan(ctx, span), span
}

// End ends the span
func (s *Span) End() {
	if s == nil {
		return
	}
	now := time.Now()
	s.EndTime = &now
	s.Duration = now.Sub(s.StartTime)
}

// SetError marks span as error
func (s *Span) SetError(err error) {
	if s == nil {
		return
	}
	s.Error = err
	s.Tags["error"] = err.Error()
}

// SetTag sets a tag
func (s *Span) SetTag(key, value string) {
	if s == nil {
		return
	}
	s.Tags[key] = value
}

// AddEvent adds an event
func (s *Span) AddEvent(name string, tags map[string]string) {
	if s == nil {
		return
	}
	s.Events = append(s.Events, Event{
		Name:      name,
		Timestamp: time.Now(),
		Tags:      tags,
	})
}

// Context helpers

// contextKey is the key type for context

type contextKey struct{}

var spanKey = contextKey{}

// ContextWithSpan adds span to context
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	if span == nil {
		return ctx
	}
	return context.WithValue(ctx, spanKey, span)
}

// SpanFromContext gets span from context
func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	if span, ok := ctx.Value(spanKey).(*Span); ok {
		return span
	}
	return nil
}

// TraceIDFromContext gets trace ID from context
func TraceIDFromContext(ctx context.Context) string {
	if span := SpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

// Helper functions

func generateID() string {
	return uuid.New().String()[:8]
}

func getOrCreateTraceID(ctx context.Context) string {
	if span := SpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return uuid.New().String()
}

// TracedFunc wraps a function with tracing
func TracedFunc(tracer *Tracer, name string, fn func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ctx, span := tracer.Start(ctx, name)
		defer func() {
			if span != nil {
				span.End()
			}
		}()

		start := time.Now()
		err := fn(ctx)
		duration := time.Since(start)

		if span != nil {
			span.SetTag("duration", duration.String())
			if err != nil {
				span.SetError(err)
			}
		}

		return err
	}
}

// TracedFuncWithResult wraps a function with result and tracing
func TracedFuncWithResult[T any](tracer *Tracer, name string, fn func(ctx context.Context) (T, error)) func(ctx context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		ctx, span := tracer.Start(ctx, name)
		defer func() {
			if span != nil {
				span.End()
			}
		}()

		start := time.Now()
		result, err := fn(ctx)
		duration := time.Since(start)

		if span != nil {
			span.SetTag("duration", duration.String())
			if err != nil {
				span.SetError(err)
			}
		}

		return result, err
	}
}

// StartSpan starts a child span
func StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if tracer := Global(); tracer != nil {
		return tracer.Start(ctx, name)
	}
	return ctx, nil
}

// Global tracer instance
var (
	globalTracer *Tracer
	tracerMu     sync.RWMutex
)

// SetGlobal sets global tracer
func SetGlobal(tracer *Tracer) {
	tracerMu.Lock()
	defer tracerMu.Unlock()
	globalTracer = tracer
}

// Global returns global tracer
func Global() *Tracer {
	tracerMu.RLock()
	defer tracerMu.RUnlock()
	return globalTracer
}

// Enabled returns true if tracing is enabled
func (t *Tracer) Enabled() bool {
	return t != nil && t.config.Enabled
}

// Shutdown shuts down tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t == nil {
		return nil
	}
	// Flush remaining spans
	return nil
}
