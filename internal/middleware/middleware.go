// Package middleware provides HTTP middleware components.
package middleware

import (
	"compress/gzip"
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yourname/helm/internal/logger"
)

// Middleware is an HTTP middleware function
type Middleware func(http.Handler) http.Handler

// Chain chains multiple middleware together
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// Logging middleware
func Logging(log *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			log.Info("%s %s %d %s %d bytes",
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
				wrapped.bytesWritten,
			)
		})
	}
}

// Recovery middleware
func Recovery(log *logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered: %v", err)
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS middleware
func CORS(origins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, o := range origins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Auth middleware (simple token-based)
func Auth(validTokens []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Validate token
			token := parts[1]
			valid := false
			for _, t := range validTokens {
				if subtle.ConstantTimeCompare([]byte(token), []byte(t)) == 1 {
					valid = true
					break
				}
			}

			if !valid {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// APIKey middleware
func APIKey(keys []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			valid := false
			for _, key := range keys {
				if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 1 {
					valid = true
					break
				}
			}

			if !valid {
				http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter middleware
func RateLimiter(requests int, window time.Duration) Middleware {
	type client struct {
		count   int
		resetAt time.Time
	}

	var (
		clients = make(map[string]*client)
		mu      sync.RWMutex
	)

	// Cleanup routine
	go func() {
		for {
			time.Sleep(window)
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.After(c.resetAt) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			mu.Lock()
			c, exists := clients[ip]
			now := time.Now()

			if !exists || now.After(c.resetAt) {
				clients[ip] = &client{
					count:   1,
					resetAt: now.Add(window),
				}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			if c.count >= requests {
				mu.Unlock()
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requests))
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", c.resetAt.Unix()))
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			c.count++
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// Gzip middleware
func Gzip() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			gz := gzip.NewWriter(w)
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			next.ServeHTTP(&gzipWriter{ResponseWriter: w, Writer: gz}, r)
		})
	}
}

// Timeout middleware
func Timeout(duration time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				http.Error(w, `{"error":"request timeout"}`, http.StatusGatewayTimeout)
				return
			}
		})
	}
}

// RequestID middleware
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SecurityHeaders middleware
func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}

// Helper types and functions

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

type gzipWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-Ip")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// DefaultChain returns commonly used middleware chain
func DefaultChain(log *logger.Logger) Middleware {
	return Chain(
		RequestID(),
		Recovery(log),
		Logging(log),
		CORS([]string{"*"}),
		SecurityHeaders(),
		Gzip(),
	)
}
