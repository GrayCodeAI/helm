package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCacheSetAndGet(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	err := c.Set(ctx, "key1", []byte("value1"), 1*time.Hour)
	require.NoError(t, err)

	val, err := c.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), val)
}

func TestMemoryCacheMiss(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	_, err := c.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestMemoryCacheExpiration(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	err := c.Set(ctx, "key1", []byte("value1"), 10*time.Millisecond)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)

	_, err = c.Get(ctx, "key1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestMemoryCacheDelete(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	c.Set(ctx, "key1", []byte("value1"), 1*time.Hour)
	err := c.Delete(ctx, "key1")
	require.NoError(t, err)

	_, err = c.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryCacheExists(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	c.Set(ctx, "key1", []byte("value1"), 1*time.Hour)

	exists, err := c.Exists(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = c.Exists(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMemoryCacheEviction(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(5)
	ctx := context.Background()

	// Fill cache
	for i := 0; i < 10; i++ {
		c.Set(ctx, string(rune('a'+i)), []byte("value"), 1*time.Hour)
	}

	// Should have evicted some items
	assert.LessOrEqual(t, len(c.data), 5)
}

func TestMemoryCacheClose(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	c.Set(ctx, "key1", []byte("value1"), 1*time.Hour)
	err := c.Close()
	require.NoError(t, err)

	_, err = c.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryCacheStats(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	c.Set(ctx, "key1", []byte("value1"), 1*time.Hour)
	c.Set(ctx, "key2", []byte("value2"), 1*time.Hour)

	stats := c.Stats()
	assert.Equal(t, 2, stats["items"])
	assert.Equal(t, 100, stats["max_items"])
	assert.Equal(t, "memory", stats["type"])
}

func TestCachedFunc(t *testing.T) {
	t.Parallel()

	c := NewMemoryCache(100)
	ctx := context.Background()

	callCount := 0
	fn := CachedFunc(c, "test-key", 1*time.Hour, func(ctx context.Context) (string, error) {
		callCount++
		return "result", nil
	})

	// First call
	result, err := fn(ctx)
	require.NoError(t, err)
	assert.Equal(t, "result", result)
	assert.Equal(t, 1, callCount)

	// Second call (should use cache)
	result, err = fn(ctx)
	require.NoError(t, err)
	assert.Equal(t, "result", result)
	assert.Equal(t, 1, callCount) // Should not increment
}

func TestNewCache(t *testing.T) {
	t.Parallel()

	// Memory cache
	c := NewCache("memory", "", 100)
	assert.IsType(t, &MemoryCache{}, c)

	// Redis cache (stub)
	c = NewCache("redis", "localhost:6379", 100)
	assert.IsType(t, &RedisCache{}, c)

	// Default (memory)
	c = NewCache("", "", 100)
	assert.IsType(t, &MemoryCache{}, c)
}

func TestRedisCacheStub(t *testing.T) {
	t.Parallel()

	c := NewRedisCache("localhost:6379")
	ctx := context.Background()

	_, err := c.Get(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")

	err = c.Set(ctx, "key", []byte("value"), 1*time.Hour)
	assert.Error(t, err)
}
