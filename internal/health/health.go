// Package health provides health checks and readiness probes.
package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Status represents health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check
type Check struct {
	Name         string                 `json:"name"`
	Status       Status                 `json:"status"`
	Message      string                 `json:"message,omitempty"`
	ResponseTime time.Duration          `json:"response_time"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Result represents overall health result
type Result struct {
	Status     Status     `json:"status"`
	Version    string     `json:"version"`
	Timestamp  time.Time  `json:"timestamp"`
	Uptime     string     `json:"uptime"`
	Checks     []Check    `json:"checks"`
	SystemInfo SystemInfo `json:"system_info"`
}

// SystemInfo holds system information
type SystemInfo struct {
	GoVersion    string     `json:"go_version"`
	NumGoroutine int        `json:"num_goroutine"`
	NumCPU       int        `json:"num_cpu"`
	MemoryUsage  MemoryInfo `json:"memory_usage"`
}

// MemoryInfo holds memory information
type MemoryInfo struct {
	Alloc      uint64 `json:"alloc_bytes"`
	TotalAlloc uint64 `json:"total_alloc_bytes"`
	Sys        uint64 `json:"sys_bytes"`
	NumGC      uint32 `json:"num_gc"`
}

// Checker performs health checks
type Checker struct {
	version   string
	startTime time.Time
	checks    map[string]CheckFunc
	mu        sync.RWMutex
	db        *sql.DB
}

// CheckFunc is a health check function
type CheckFunc func(ctx context.Context) Check

// New creates a new health checker
func New(version string, db *sql.DB) *Checker {
	c := &Checker{
		version:   version,
		startTime: time.Now(),
		checks:    make(map[string]CheckFunc),
		db:        db,
	}

	// Register default checks
	c.Register("database", c.databaseCheck)
	c.Register("memory", c.memoryCheck)

	return c
}

// Register registers a health check
func (c *Checker) Register(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = fn
}

// Unregister removes a health check
func (c *Checker) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.checks, name)
}

// Check runs all health checks
func (c *Checker) Check(ctx context.Context) Result {
	c.mu.RLock()
	checks := make(map[string]CheckFunc, len(c.checks))
	for k, v := range c.checks {
		checks[k] = v
	}
	c.mu.RUnlock()

	result := Result{
		Version:    c.version,
		Timestamp:  time.Now(),
		Uptime:     time.Since(c.startTime).String(),
		Status:     StatusHealthy,
		SystemInfo: c.getSystemInfo(),
	}

	// Run checks concurrently
	var wg sync.WaitGroup
	checkResults := make(chan Check, len(checks))

	for name, fn := range checks {
		wg.Add(1)
		go func(n string, f CheckFunc) {
			defer wg.Done()
			check := f(ctx)
			check.Name = n
			checkResults <- check
		}(name, fn)
	}

	go func() {
		wg.Wait()
		close(checkResults)
	}()

	for check := range checkResults {
		result.Checks = append(result.Checks, check)

		// Determine overall status
		if check.Status == StatusUnhealthy {
			result.Status = StatusUnhealthy
		} else if check.Status == StatusDegraded && result.Status != StatusUnhealthy {
			result.Status = StatusDegraded
		}
	}

	return result
}

// CheckReadiness checks if service is ready
func (c *Checker) CheckReadiness(ctx context.Context) Result {
	// Only check critical components
	result := Result{
		Version:   c.version,
		Timestamp: time.Now(),
		Uptime:    time.Since(c.startTime).String(),
		Status:    StatusHealthy,
	}

	// Check database
	check := c.databaseCheck(ctx)
	check.Name = "database"
	result.Checks = append(result.Checks, check)

	if check.Status != StatusHealthy {
		result.Status = StatusUnhealthy
	}

	return result
}

// CheckLiveness checks if service is alive
func (c *Checker) CheckLiveness(ctx context.Context) Result {
	return Result{
		Version:   c.version,
		Timestamp: time.Now(),
		Uptime:    time.Since(c.startTime).String(),
		Status:    StatusHealthy,
		Checks:    []Check{},
	}
}

// Default checks

func (c *Checker) databaseCheck(ctx context.Context) Check {
	start := time.Now()

	if c.db == nil {
		return Check{
			Status:       StatusDegraded,
			Message:      "database not configured",
			ResponseTime: time.Since(start),
			Timestamp:    time.Now(),
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		return Check{
			Status:       StatusUnhealthy,
			Message:      fmt.Sprintf("database ping failed: %v", err),
			ResponseTime: time.Since(start),
			Timestamp:    time.Now(),
		}
	}

	return Check{
		Status:       StatusHealthy,
		ResponseTime: time.Since(start),
		Timestamp:    time.Now(),
	}
}

func (c *Checker) memoryCheck(ctx context.Context) Check {
	start := time.Now()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check if memory usage is high (>80% of limit)
	// For now, just report info
	status := StatusHealthy
	message := ""

	// Alert if GC pause is too high
	if m.PauseNs[(m.NumGC+255)%256] > 100000000 { // 100ms
		status = StatusDegraded
		message = "high GC pause time detected"
	}

	return Check{
		Status:       status,
		Message:      message,
		ResponseTime: time.Since(start),
		Timestamp:    time.Now(),
		Metadata: map[string]interface{}{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"num_gc":         m.NumGC,
		},
	}
}

func (c *Checker) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		MemoryUsage: MemoryInfo{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}
}

// HTTP handlers

// Handler returns the main health handler
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := c.Check(ctx)

		statusCode := http.StatusOK
		if result.Status == StatusDegraded {
			statusCode = http.StatusOK // Still 200 but flagged
		} else if result.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	}
}

// ReadinessHandler returns the readiness probe handler
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := c.CheckReadiness(ctx)

		statusCode := http.StatusOK
		if result.Status != StatusHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	}
}

// LivenessHandler returns the liveness probe handler
func (c *Checker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		result := c.CheckLiveness(ctx)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

// MetricsHandler returns metrics for monitoring
func (c *Checker) MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		uptime := time.Since(c.startTime).Seconds()

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# HELP helm_uptime_seconds Uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE helm_uptime_seconds gauge\n")
		fmt.Fprintf(w, "helm_uptime_seconds %.0f\n\n", uptime)

		fmt.Fprintf(w, "# HELP helm_memory_alloc_bytes Allocated memory\n")
		fmt.Fprintf(w, "# TYPE helm_memory_alloc_bytes gauge\n")
		fmt.Fprintf(w, "helm_memory_alloc_bytes %d\n\n", m.Alloc)

		fmt.Fprintf(w, "# HELP helm_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "# TYPE helm_goroutines gauge\n")
		fmt.Fprintf(w, "helm_goroutines %d\n\n", runtime.NumGoroutine())

		fmt.Fprintf(w, "# HELP helm_gc_total Total GC cycles\n")
		fmt.Fprintf(w, "# TYPE helm_gc_total counter\n")
		fmt.Fprintf(w, "helm_gc_total %d\n", m.NumGC)
	}
}

// SetupRoutes sets up all health routes
func (c *Checker) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", c.Handler())
	mux.HandleFunc("/health/ready", c.ReadinessHandler())
	mux.HandleFunc("/health/live", c.LivenessHandler())
	mux.HandleFunc("/metrics", c.MetricsHandler())
}

// Simple health check for quick checks
func SimpleHealth(version string) http.HandlerFunc {
	startTime := time.Now()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": version,
			"uptime":  time.Since(startTime).String(),
		})
	}
}
