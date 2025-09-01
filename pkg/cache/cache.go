package cache

import (
	"context"
	"sync"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, val string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type InMemoryCache struct {
	mu   sync.RWMutex
	data map[string]item
}

type item struct {
	val string
	exp time.Time
}

func NewInMemory() *InMemoryCache { return &InMemoryCache{data: make(map[string]item)} }

func (c *InMemoryCache) Get(_ context.Context, key string) (string, bool) {
	c.mu.RLock()
	it, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if !it.exp.IsZero() && time.Now().After(it.exp) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return "", false
	}
	return it.val, true
}

func (c *InMemoryCache) Set(_ context.Context, key string, val string, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.data[key] = item{val: val, exp: exp}
	c.mu.Unlock()
	return nil
}

func (c *InMemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
	return nil
}
