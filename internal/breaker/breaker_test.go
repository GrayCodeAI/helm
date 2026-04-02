package breaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBreaker(t *testing.T) {
	t.Parallel()

	config := DefaultConfig("test")
	b := New(config)

	require.NotNil(t, b)
	assert.Equal(t, StateClosed, b.State())
	assert.Equal(t, 0, b.Failures())
}

func TestBreakerSuccess(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))

	err := b.Execute(func() error {
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, StateClosed, b.State())
	assert.Equal(t, 0, b.Failures())
}

func TestBreakerFailure(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))
	b.config.MaxFailures = 3

	for i := 0; i < 3; i++ {
		err := b.Execute(func() error {
			return errors.New("fail")
		})
		assert.Error(t, err)
	}

	assert.Equal(t, StateOpen, b.State())
	assert.Equal(t, 3, b.Failures())
}

func TestBreakerOpen(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))
	b.config.MaxFailures = 2

	// Trip the breaker
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errors.New("fail") })
	}

	assert.Equal(t, StateOpen, b.State())

	// Should reject immediately
	err := b.Execute(func() error { return nil })
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open")
}

func TestBreakerReset(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))
	b.config.MaxFailures = 2

	// Trip the breaker
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errors.New("fail") })
	}

	assert.Equal(t, StateOpen, b.State())

	// Reset
	b.Reset()
	assert.Equal(t, StateClosed, b.State())
	assert.Equal(t, 0, b.Failures())
}

func TestBreakerHalfOpen(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))
	b.config.MaxFailures = 2
	b.config.ResetTimeout = 50 * time.Millisecond

	// Trip the breaker
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errors.New("fail") })
	}

	assert.Equal(t, StateOpen, b.State())

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Next call should transition to half-open
	err := b.Execute(func() error { return nil })
	require.NoError(t, err)
	assert.Equal(t, StateHalfOpen, b.State())
}

func TestBreakerOnStateChange(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))
	b.config.MaxFailures = 2

	states := make([]State, 0)
	b.OnStateChange(func(s State) {
		states = append(states, s)
	})

	// Trip the breaker
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errors.New("fail") })
	}

	assert.Contains(t, states, StateOpen)
}

func TestBreakerStats(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))

	b.Execute(func() error { return nil })
	b.Execute(func() error { return errors.New("fail") })

	stats := b.Stats()
	assert.Equal(t, "test", stats["name"])
	assert.Equal(t, "closed", stats["state"])
	assert.Equal(t, 1, stats["failures"])
	assert.Equal(t, 1, stats["successes"])
}

func TestBreakerTimeout(t *testing.T) {
	t.Parallel()

	b := New(DefaultConfig("test"))
	b.config.Timeout = 10 * time.Millisecond

	err := b.Execute(func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestManager(t *testing.T) {
	t.Parallel()

	m := NewManager()

	// Get or create
	b1 := m.GetOrCreate("api", DefaultConfig("api"))
	require.NotNil(t, b1)

	// Get existing
	b2, ok := m.Get("api")
	require.True(t, ok)
	assert.Equal(t, b1, b2)

	// List
	breakers := m.List()
	assert.Len(t, breakers, 1)

	// Stats
	stats := m.Stats()
	assert.Contains(t, stats, "api")
}

func TestIsCircuitOpen(t *testing.T) {
	t.Parallel()

	assert.True(t, IsCircuitOpen(errors.New("circuit breaker is open")))
	assert.False(t, IsCircuitOpen(nil))
	assert.False(t, IsCircuitOpen(errors.New("some other error")))
}

func TestStateString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
}
