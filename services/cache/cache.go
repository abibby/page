package cache

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] struct {
	m   map[K]V
	mtx sync.RWMutex
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		m: map[K]V{},
	}
}

func (c *Cache[K, V]) Get(key K, ttl time.Duration, fn func() V) V {
	c.mtx.RLock()
	v, ok := c.m[key]
	c.mtx.RUnlock()

	if ok {
		return v
	}

	v = fn()

	c.mtx.Lock()
	c.m[key] = v
	c.mtx.Unlock()

	go func() {
		c.mtx.Lock()
		delete(c.m, key)
		c.mtx.Unlock()
	}()

	return v
}
