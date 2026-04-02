// Package logger provides structured logging with slog.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Level represents log levels
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelFatal Level = "fatal"
)

// Config configures the logger
type Config struct {
	Level      Level
	Format     string // "json" or "text"
	Output     string // "stdout", "stderr", or file path
	AddSource  bool
	TimeFormat string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Level:      LevelInfo,
		Format:     "text",
		Output:     "stdout",
		AddSource:  false,
		TimeFormat: time.RFC3339,
	}
}

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	config Config
}

// New creates a new logger
func New(config Config) (*Logger, error) {
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output
		dir := filepath.Dir(config.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
		f, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		output = f
	}

	level := parseLevel(config.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
	}

	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		config: config,
	}, nil
}

// NewDefault creates a logger with default config
func NewDefault() *Logger {
	l, _ := New(DefaultConfig())
	return l
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract trace ID or other context values
	traceID := ctx.Value("trace_id")
	if traceID != nil {
		return &Logger{
			Logger: l.Logger.With("trace_id", traceID),
			config: l.config,
		}
	}
	return l
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return &Logger{
		Logger: slog.New(l.Logger.Handler().WithAttrs(attrs)),
		config: l.config,
	}
}

// WithError returns a logger with error field
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{
		Logger: l.Logger.With("error", err.Error()),
		config: l.config,
	}
}

// WithComponent returns a logger with component field
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
		config: l.config,
	}
}

// Debug logs debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(msg, args...))
}

// Info logs info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(msg, args...))
}

// Warn logs warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(msg, args...))
}

// Error logs error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(msg, args...))
}

// Fatal logs fatal message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(msg, args...))
	os.Exit(1)
}

// Log logs at specified level
func (l *Logger) Log(level Level, msg string, args ...interface{}) {
	switch level {
	case LevelDebug:
		l.Debug(msg, args...)
	case LevelInfo:
		l.Info(msg, args...)
	case LevelWarn:
		l.Warn(msg, args...)
	case LevelError, LevelFatal:
		l.Error(msg, args...)
	}
}

// Trace logs entry and exit of a function
func (l *Logger) Trace(operation string) func() {
	start := time.Now()
	l.Debug("Entering: %s", operation)
	return func() {
		duration := time.Since(start)
		l.Debug("Exiting: %s (took %s)", operation, duration)
	}
}

// LogError logs an error with full details
func (l *Logger) LogError(err error, msg string, args ...interface{}) {
	if err == nil {
		return
	}

	// Get stack trace if available
	_, file, line, _ := runtime.Caller(1)

	l.Logger.Error(
		fmt.Sprintf(msg, args...),
		"error", err.Error(),
		"file", filepath.Base(file),
		"line", line,
	)
}

// Structured logging methods

// DebugContext logs debug with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.Logger.DebugContext(ctx, msg, args...)
}

// InfoContext logs info with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.Logger.InfoContext(ctx, msg, args...)
}

// WarnContext logs warning with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.Logger.WarnContext(ctx, msg, args...)
}

// ErrorContext logs error with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.Logger.ErrorContext(ctx, msg, args...)
}

// Helpers

func parseLevel(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError, LevelFatal:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Global logger instance
var defaultLogger = NewDefault()

// SetDefault sets the global default logger
func SetDefault(l *Logger) {
	defaultLogger = l
	slog.SetDefault(l.Logger)
}

// GetDefault returns the global default logger
func GetDefault() *Logger {
	return defaultLogger
}

// Convenience functions using global logger

func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	defaultLogger.Fatal(msg, args...)
}

func WithContext(ctx context.Context) *Logger {
	return defaultLogger.WithContext(ctx)
}

func WithFields(fields map[string]interface{}) *Logger {
	return defaultLogger.WithFields(fields)
}

func WithError(err error) *Logger {
	return defaultLogger.WithError(err)
}

func WithComponent(component string) *Logger {
	return defaultLogger.WithComponent(component)
}

// Request logging

// RequestLogger logs HTTP requests
type RequestLogger struct {
	logger *Logger
}

// NewRequestLogger creates a request logger
func NewRequestLogger(l *Logger) *RequestLogger {
	return &RequestLogger{logger: l}
}

// LogRequest logs an HTTP request
func (rl *RequestLogger) LogRequest(method, path, clientIP string, status int, duration time.Duration, size int64) {
	level := LevelInfo
	if status >= 400 {
		level = LevelWarn
	}
	if status >= 500 {
		level = LevelError
	}

	rl.logger.Log(level,
		"%s %s %d %s %d bytes",
		method, path, status, duration, size,
	)
}

// Middleware returns HTTP middleware for logging
func (rl *RequestLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		rl.LogRequest(
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			wrapped.statusCode,
			duration,
			wrapped.size,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// Ensure http.Handler is implemented
var _ http.Handler = (*RequestLogger)(nil).Middleware(nil)
