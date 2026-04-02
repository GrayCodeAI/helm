// Package errors provides enhanced error handling with codes and stack traces.
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorCode represents error categories
type ErrorCode string

const (
	// General errors
	CodeUnknown         ErrorCode = "UNKNOWN"
	CodeInternal        ErrorCode = "INTERNAL"
	CodeInvalidArg      ErrorCode = "INVALID_ARGUMENT"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	CodePermission      ErrorCode = "PERMISSION_DENIED"
	CodeUnauthenticated ErrorCode = "UNAUTHENTICATED"

	// Service errors
	CodeDatabase    ErrorCode = "DATABASE"
	CodeNetwork     ErrorCode = "NETWORK"
	CodeTimeout     ErrorCode = "TIMEOUT"
	CodeUnavailable ErrorCode = "UNAVAILABLE"
	CodeRateLimited ErrorCode = "RATE_LIMITED"

	// Provider errors
	CodeProvider      ErrorCode = "PROVIDER"
	CodeAPIKeyInvalid ErrorCode = "API_KEY_INVALID"
	CodeQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"
	CodeModelNotFound ErrorCode = "MODEL_NOT_FOUND"
)

// Error represents a structured error with code and context
type Error struct {
	Code      ErrorCode
	Message   string
	Cause     error
	Stack     []Frame
	Context   map[string]interface{}
	Retryable bool
}

// Frame represents a stack frame
type Frame struct {
	File     string
	Line     int
	Function string
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[%s] %s", e.Code, e.Message))

	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}

	return b.String()
}

// Unwrap returns the wrapped error
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRetryable marks error as retryable
func (e *Error) WithRetryable(retryable bool) *Error {
	e.Retryable = retryable
	return e
}

// HasCode checks if error matches code
func (e *Error) HasCode(code ErrorCode) bool {
	return e.Code == code
}

// StackTrace returns formatted stack trace
func (e *Error) StackTrace() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Error: %s\n", e.Error()))
	b.WriteString("Stack trace:\n")

	for i, frame := range e.Stack {
		b.WriteString(fmt.Sprintf("  %d. %s:%d %s\n", i, frame.File, frame.Line, frame.Function))
	}

	return b.String()
}

// New creates a new error
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Stack:   captureStack(2),
	}
}

// Newf creates a new error with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Stack:   captureStack(2),
	}
}

// Wrap wraps an existing error
func Wrap(err error, code ErrorCode, message string) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
		Stack:   captureStack(2),
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
		Stack:   captureStack(2),
	}
}

// Is checks if error matches target
func Is(err error, code ErrorCode) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Code == code
	}
	return false
}

// As finds error in chain
func As(err error, code ErrorCode) *Error {
	var e *Error
	if errors.As(err, &e) {
		if e.Code == code {
			return e
		}
	}
	return nil
}

// Code extracts error code
func Code(err error) ErrorCode {
	if err == nil {
		return CodeUnknown
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}

	return CodeUnknown
}

// IsRetryable checks if error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Retryable
	}

	// Default retryable errors
	code := Code(err)
	switch code {
	case CodeTimeout, CodeUnavailable, CodeRateLimited, CodeNetwork:
		return true
	}

	return false
}

// captureStack captures call stack
func captureStack(skip int) []Frame {
	var frames []Frame

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		fnName := "unknown"
		if fn != nil {
			fnName = fn.Name()
		}

		frames = append(frames, Frame{
			File:     file,
			Line:     line,
			Function: fnName,
		})

		if len(frames) >= 10 {
			break
		}
	}

	return frames
}

// Convenience functions for common error types

func Internal(message string) *Error {
	return New(CodeInternal, message)
}

func Internalf(format string, args ...interface{}) *Error {
	return Newf(CodeInternal, format, args...)
}

func NotFound(resource string) *Error {
	return New(CodeNotFound, fmt.Sprintf("%s not found", resource))
}

func AlreadyExists(resource string) *Error {
	return New(CodeAlreadyExists, fmt.Sprintf("%s already exists", resource))
}

func InvalidArg(name, reason string) *Error {
	return New(CodeInvalidArg, fmt.Sprintf("invalid argument %s: %s", name, reason))
}

func PermissionDenied(action string) *Error {
	return New(CodePermission, fmt.Sprintf("permission denied: %s", action))
}

func Unauthenticated() *Error {
	return New(CodeUnauthenticated, "unauthenticated")
}

func Database(msg string) *Error {
	return New(CodeDatabase, msg).WithRetryable(true)
}

func Network(msg string) *Error {
	return New(CodeNetwork, msg).WithRetryable(true)
}

func Timeout(operation string) *Error {
	return New(CodeTimeout, fmt.Sprintf("operation %s timed out", operation)).WithRetryable(true)
}

func Unavailable(service string) *Error {
	return New(CodeUnavailable, fmt.Sprintf("service %s unavailable", service)).WithRetryable(true)
}

func RateLimited() *Error {
	return New(CodeRateLimited, "rate limit exceeded").WithRetryable(true)
}

func Provider(msg string) *Error {
	return New(CodeProvider, msg)
}

func APIKeyInvalid() *Error {
	return New(CodeAPIKeyInvalid, "invalid API key")
}

func QuotaExceeded() *Error {
	return New(CodeQuotaExceeded, "quota exceeded")
}

func ModelNotFound(model string) *Error {
	return New(CodeModelNotFound, fmt.Sprintf("model %s not found", model))
}

// ErrorList represents multiple errors
type ErrorList struct {
	Errors []error
}

func (el *ErrorList) Error() string {
	var msgs []string
	for _, err := range el.Errors {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("multiple errors (%d): %s", len(el.Errors), strings.Join(msgs, "; "))
}

func (el *ErrorList) Add(err error) {
	if err != nil {
		el.Errors = append(el.Errors, err)
	}
}

func (el *ErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// Must panics if error is not nil
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// MustValue returns value or panics
func MustValue[T any](val T, err error) T {
	Must(err)
	return val
}
