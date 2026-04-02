package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls int32

	err := Do(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestDoMaxRetriesExceeded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls int32

	// Use timeout error which is retryable
	err := Do(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("timeout: connection timed out")
	}, WithMaxRetries(3), WithBaseDelay(1*time.Millisecond))

	assert.Error(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
	assert.Contains(t, err.Error(), "max attempts")
}

func TestDoRetryOnFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls int32

	err := Do(ctx, func(ctx context.Context) error {
		c := atomic.AddInt32(&calls, 1)
		if c < 3 {
			return errors.New("timeout: connection timed out")
		}
		return nil
	}, WithMaxRetries(5), WithBaseDelay(1*time.Millisecond))

	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestDoNonRetryableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	err := Do(ctx, func(ctx context.Context) error {
		return errors.New("permanent error")
	}, WithMaxRetries(3), WithBaseDelay(1*time.Millisecond))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permanent error")
}

func TestDoWithContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Do(ctx, func(ctx context.Context) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestDoWithResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls int32

	result, err := DoWithResult(ctx, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestDoWithResultRetry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls int32

	result, err := DoWithResult(ctx, func(ctx context.Context) (string, error) {
		c := atomic.AddInt32(&calls, 1)
		if c < 2 {
			return "", errors.New("timeout: connection timed out")
		}
		return "ok", nil
	}, WithMaxRetries(3), WithBaseDelay(1*time.Millisecond))

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestCalculateExponentialDelay(t *testing.T) {
	t.Parallel()

	base := 100 * time.Millisecond
	max := 5 * time.Second

	d1 := calculateExponentialDelay(1, base, max)
	d2 := calculateExponentialDelay(2, base, max)
	d3 := calculateExponentialDelay(3, base, max)

	assert.True(t, d2 > d1, "delay should increase")
	assert.True(t, d3 > d2, "delay should increase")
	assert.True(t, d3 <= max, "delay should not exceed max")
}

func TestPolicy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	policy := NewPolicy(WithMaxRetries(3), WithBaseDelay(1*time.Millisecond))

	var calls int32
	err := policy.Execute(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestFastRetryPolicy(t *testing.T) {
	t.Parallel()

	policy := FastRetry()
	assert.Equal(t, 3, policy.Config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, policy.Config.BaseDelay)
}

func TestSlowRetryPolicy(t *testing.T) {
	t.Parallel()

	policy := SlowRetry()
	assert.Equal(t, 5, policy.Config.MaxRetries)
	assert.Equal(t, 1*time.Second, policy.Config.BaseDelay)
}

func TestDatabaseRetryPolicy(t *testing.T) {
	t.Parallel()

	policy := DatabaseRetry()
	assert.Equal(t, 5, policy.Config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, policy.Config.BaseDelay)
}

func TestHTTPRetryPolicy(t *testing.T) {
	t.Parallel()

	policy := HTTPRetry()
	assert.Equal(t, 3, policy.Config.MaxRetries)
	assert.Equal(t, 1*time.Second, policy.Config.BaseDelay)
}

func TestNoRetryPolicy(t *testing.T) {
	t.Parallel()

	policy := NoRetry()
	assert.Equal(t, 1, policy.Config.MaxRetries)
}

func TestLinearBackoff(t *testing.T) {
	t.Parallel()

	fn := LinearBackoff(100 * time.Millisecond)

	d1 := fn(1, Config{BaseDelay: 100 * time.Millisecond})
	d2 := fn(2, Config{BaseDelay: 100 * time.Millisecond})
	d3 := fn(3, Config{BaseDelay: 100 * time.Millisecond})

	assert.Equal(t, 100*time.Millisecond, d1)
	assert.Equal(t, 200*time.Millisecond, d2)
	assert.Equal(t, 300*time.Millisecond, d3)
}

func TestFixedBackoff(t *testing.T) {
	t.Parallel()

	fn := FixedBackoff(500 * time.Millisecond)

	d1 := fn(1, Config{})
	d2 := fn(5, Config{})
	d3 := fn(10, Config{})

	assert.Equal(t, 500*time.Millisecond, d1)
	assert.Equal(t, 500*time.Millisecond, d2)
	assert.Equal(t, 500*time.Millisecond, d3)
}
