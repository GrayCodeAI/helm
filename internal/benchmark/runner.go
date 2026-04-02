// Package benchmark provides memory system benchmarking.
package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/yourname/helm/internal/db"
)

// Result represents a benchmark result
type Result struct {
	Operation    string
	Duration     time.Duration
	OpsPerSecond float64
	Count        int
}

// Runner runs memory benchmarks
type Runner struct {
	querier db.Querier
}

// NewRunner creates a new benchmark runner
func NewRunner(q db.Querier) *Runner {
	return &Runner{querier: q}
}

// RunWriteBenchmark benchmarks memory write operations
func (r *Runner) RunWriteBenchmark(ctx context.Context, project string, count int) (*Result, error) {
	start := time.Now()

	for i := 0; i < count; i++ {
		_, err := r.querier.CreateMemory(ctx, db.CreateMemoryParams{
			ID:         fmt.Sprintf("bench-%d", i),
			Project:    project,
			Type:       "benchmark",
			Key:        fmt.Sprintf("key-%d", i),
			Value:      fmt.Sprintf("value-%d", rand.Intn(1000)),
			Source:     "benchmark",
			Confidence: rand.Float64(),
		})
		if err != nil {
			return nil, err
		}
	}

	duration := time.Since(start)
	return &Result{
		Operation:    "write",
		Duration:     duration,
		OpsPerSecond: float64(count) / duration.Seconds(),
		Count:        count,
	}, nil
}

// RunReadBenchmark benchmarks memory read operations
func (r *Runner) RunReadBenchmark(ctx context.Context, project string, count int) (*Result, error) {
	start := time.Now()

	for i := 0; i < count; i++ {
		_, err := r.querier.ListMemories(ctx, project)
		if err != nil {
			return nil, err
		}
	}

	duration := time.Since(start)
	return &Result{
		Operation:    "read",
		Duration:     duration,
		OpsPerSecond: float64(count) / duration.Seconds(),
		Count:        count,
	}, nil
}

// RunSearchBenchmark benchmarks memory search operations
func (r *Runner) RunSearchBenchmark(ctx context.Context, project string, count int) (*Result, error) {
	start := time.Now()

	for i := 0; i < count; i++ {
		_, err := r.querier.SearchMemories(ctx, db.SearchMemoriesParams{
			Project: project,
			Limit:   10,
		})
		if err != nil {
			return nil, err
		}
	}

	duration := time.Since(start)
	return &Result{
		Operation:    "search",
		Duration:     duration,
		OpsPerSecond: float64(count) / duration.Seconds(),
		Count:        count,
	}, nil
}

// RunFullBenchmark runs all benchmarks
func (r *Runner) RunFullBenchmark(ctx context.Context, project string) ([]Result, error) {
	var results []Result

	writeResult, err := r.RunWriteBenchmark(ctx, project, 100)
	if err != nil {
		return nil, err
	}
	results = append(results, *writeResult)

	readResult, err := r.RunReadBenchmark(ctx, project, 100)
	if err != nil {
		return nil, err
	}
	results = append(results, *readResult)

	searchResult, err := r.RunSearchBenchmark(ctx, project, 100)
	if err != nil {
		return nil, err
	}
	results = append(results, *searchResult)

	return results, nil
}

// FormatResults formats benchmark results for display
func FormatResults(results []Result) string {
	var output string
	for _, r := range results {
		output += fmt.Sprintf("%s: %d ops in %s (%.1f ops/sec)\n",
			r.Operation, r.Count, r.Duration.Round(time.Millisecond), r.OpsPerSecond)
	}
	return output
}
