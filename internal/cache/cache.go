// Package cache provides Redis caching layer for hot data.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Cache interface for caching
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
}

// MemoryCache is an in-memory cache implementation
type MemoryCache struct {
	data     map[string]*cacheItem
	mu       sync.RWMutex
	maxItems int
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(maxItems int) *MemoryCache {
	if maxItems == 0 {
		maxItems = 10000
	}
	return &MemoryCache{
		data:     make(map[string]*cacheItem),
		maxItems: maxItems,
	}
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.data[key]
	if !ok {
		return nil, fmt.Errorf("cache miss: %s", key)
	}

	if time.Now().After(item.expiresAt) {
		delete(c.data, key)
		return nil, fmt.Errorf("cache expired: %s", key)
	}

	return item.value, nil
}

// Set stores a value in cache
func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.data) >= c.maxItems {
		c.evict()
	}

	c.data[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

// Exists checks if a key exists in cache
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.data[key]
	if !ok {
		return false, nil
	}

	if time.Now().After(item.expiresAt) {
		delete(c.data, key)
		return false, nil
	}

	return true, nil
}

// Close closes the cache
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*cacheItem)
	return nil
}

// evict removes expired or least recently used items
func (c *MemoryCache) evict() {
	now := time.Now()

	// First, remove expired items
	for key, item := range c.data {
		if now.After(item.expiresAt) {
			delete(c.data, key)
			if len(c.data) < c.maxItems {
				return
			}
		}
	}

	// If still at capacity, remove oldest
	var oldestKey string
	var oldestTime time.Time
	for key, item := range c.data {
		if oldestKey == "" || item.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

// CachedFunc wraps a function with caching
func CachedFunc[T any](cache Cache, key string, ttl time.Duration, fn func(ctx context.Context) (T, error)) func(ctx context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		var zero T

		// Try cache
		data, err := cache.Get(ctx, key)
		if err == nil {
			var result T
			if err := json.Unmarshal(data, &result); err == nil {
				return result, nil
			}
		}

		// Call function
		result, err := fn(ctx)
		if err != nil {
			return zero, err
		}

		// Cache result
		data, err = json.Marshal(result)
		if err == nil {
			cache.Set(ctx, key, data, ttl)
		}

		return result, nil
	}
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"items":     len(c.data),
		"max_items": c.maxItems,
		"type":      "memory",
	}
}

// RedisCache is a Redis cache implementation (stub)
type RedisCache struct {
	addr   string
	client interface{}
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(addr string) *RedisCache {
	return &RedisCache{addr: addr}
}

// Get retrieves from Redis
func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Would use redis client
	return nil, fmt.Errorf("redis not configured")
}

// Set stores in Redis
func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return fmt.Errorf("redis not configured")
}

// Delete removes from Redis
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Exists checks Redis
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// Close closes Redis connection
func (r *RedisCache) Close() error {
	return nil
}

// NewCache creates a cache based on configuration
func NewCache(cacheType string, addr string, maxItems int) Cache {
	switch cacheType {
	case "redis":
		return NewRedisCache(addr)
	default:
		return NewMemoryCache(maxItems)
	}
}
